package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/JahidNishat/payment-gateway/services/payment/internal/errors"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/events"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/processor"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/repository"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/validator"
	"github.com/google/uuid"
)

type PaymentService struct {
	repo      repository.PaymentRepository
	processor processor.PaymentProcessor
	publisher events.EventPublisher
}

func NewPaymentService(
	repo repository.PaymentRepository, processor processor.PaymentProcessor, publisher events.EventPublisher) *PaymentService {
	return &PaymentService{
		repo:      repo,
		processor: processor,
		publisher: publisher,
	}
}

// ========== Create Payment Service ==========
func (s *PaymentService) CreatePayment(ctx context.Context, req *model.CreatePaymentRequest) (*model.Payment, error) {
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := validator.ValidateStruct(req); err != nil {
		slog.Error("svc: validation failed", "error", err)
		return nil, err
	}

	// Step 2: Check for existing (idempotency)
	existing, err := s.getExistingPayment(processCtx, req.MerchantID, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Build payment object
	payment := &model.Payment{
		MerchantID:     req.MerchantID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
		Status:         model.PaymentStatusPending,
		// Metadata handling later
	}

	// Step 3: Create payment
	if err := s.repo.CreatePayment(processCtx, payment); err != nil {
		if _, ok := err.(*errors.ConflictError); ok {
			// Race condition — someone else created it
			existing, existingErr := s.getExistingPayment(processCtx, req.MerchantID, req.IdempotencyKey)
			if existingErr != nil {
				return nil, existingErr
			}
			if existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	// Step 4: Move to PROCESSING
	if err := s.updatePaymentStatus(processCtx, payment.ID.String(), model.PaymentStatusProcessing, "", nil); err != nil {
		return nil, err
	}

	// Step 5: Process payment
	var description string
	if payment.Description != nil {
		description = *payment.Description
	}
	process, err := s.processor.ProcessPayment(processCtx, &processor.PaymentProcessorRequest{
		PaymentID:      payment.ID.String(),
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		MerchantID:     payment.MerchantID,
		IdempotencyKey: payment.IdempotencyKey,
		Description:    description,
	})
	if err != nil {
		slog.Error("svc: processor failed", "error", err, "payment_id", payment.ID)
		updateErr := s.updatePaymentStatus(processCtx, payment.ID.String(), model.PaymentStatusFailed, err.Error(), nil)
		if updateErr != nil {
			slog.Error("svc: also failed to update status to FAILED", "error", updateErr)
		}
		return nil, err
	}

	// Step 6: Update final status
	if process.Status == processor.ProcessorStatusSuccess {
		if err := s.updatePaymentStatus(processCtx, payment.ID.String(), model.PaymentStatusCompleted, "", &process.TxnID); err != nil {
			slog.Error("svc: failed to update to COMPLETED after success", "error", err)
			// TODO: In production this needs manual reconciliation
			return nil, err
		}
		payment.TxnID = &process.TxnID
		payment.Status = model.PaymentStatusCompleted
	} else {
		reason := process.Reason
		if reason == "" {
			reason = "unknown processor failure"
		}
		if err := s.updatePaymentStatus(processCtx, payment.ID.String(), model.PaymentStatusFailed, reason, nil); err != nil {
			slog.Error("svc: failed to update to FAILED", "error", err)
		}
		payment.Status = model.PaymentStatusFailed
	}

	// Step 7: Publish event (fire and forget)
	s.publishEvent(processCtx, payment, payment.Status, "payment")

	return payment, nil
}

// ========== Get Payment Service ==========
func (s *PaymentService) GetPaymentByID(ctx context.Context, id string) (*model.Payment, error) {
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	payment, err := s.repo.GetPaymentByID(processCtx, id)
	if err != nil {
		slog.Error("svc: failed to get payment by ID", "error", err, "payment_id", id)
		return nil, err
	}
	return payment, nil
}

// ========== List Payments Service ==========
func (s *PaymentService) ListPayments(ctx context.Context, merchantID string, page, limit int) ([]*model.Payment, int, error) {
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	payments, total, err := s.repo.ListPaymentsByMerchantID(processCtx, merchantID, page, limit)
	if err != nil {
		slog.Error("svc: failed to list payments", "error", err, "merchant_id", merchantID)
		return nil, 0, err
	}
	return payments, total, nil
}

// ========== Refund Payment Service ==========
func (s *PaymentService) RefundPayment(ctx context.Context, req *model.RefundRequest) (*model.Refund, error) {
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := validator.ValidateStruct(req); err != nil {
		slog.Error("svc: refund validation failed", "error", err)
		return nil, err
	}

	resp, err := s.processor.RefundPayment(processCtx, &processor.RefundProcessorRequest{
		PaymentID: req.PaymentID,
		Amount:    req.Amount,
		Reason:    req.Reason,
	})
	if err != nil {
		slog.Error("svc: refund failed", "error", err)
		return nil, err
	}
	if resp.Status != processor.ProcessorStatusSuccess {
		slog.Error("svc: refund processor did not succeed", "status", resp.Status, "reason", resp.Reason)
		if resp.Reason == "" {
			resp.Reason = "refund rejected by processor"
		}
		return nil, fmt.Errorf("svc: refund failed: %s", resp.Reason)
	}

	paymentID, err := uuid.Parse(req.PaymentID)
	if err != nil {
		slog.Error("svc: invalid payment ID", "error", err, "payment_id", resp.PaymentID)
		return nil, fmt.Errorf("svc: invalid payment ID: %w", err)
	}

	refund := &model.Refund{
		Amount:    req.Amount,
		Reason:    &req.Reason,
		Status:    model.RefundStatusCompleted,
		PaymentID: paymentID,
	}

	refundErr := s.repo.CreateRefund(processCtx, refund)
	if refundErr != nil {
		slog.Error("svc: failed to create refund", "error", refundErr)
		return nil, refundErr
	}
	s.publisher.Publish(&events.PaymentEvent{
		PaymentID:   req.PaymentID,
		Amount:      refund.Amount,
		PaymentType: "refund",
		Status:      refund.Status,
		Timestamp:   refund.CreatedAt,
	})
	return refund, nil
}
