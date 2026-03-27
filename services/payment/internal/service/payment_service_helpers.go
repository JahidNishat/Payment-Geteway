package service

import (
	"context"
	"log/slog"

	"github.com/JahidNishat/payment-gateway/services/payment/internal/events"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
)

func (s *PaymentService) updatePaymentStatus(ctx context.Context, paymentID, status, reason string, txnID *string) error {
	if err := s.repo.UpdatePaymentStatus(ctx, paymentID, status, reason, txnID); err != nil {
		slog.Error("svc: failed to update payment status", "error", err, "payment_id", paymentID, "status", status)
		return err
	}
	return nil
}

func (s *PaymentService) publishEvent(ctx context.Context, payment *model.Payment, status, paymentType string) {
	event := &events.PaymentEvent{
		PaymentID:   payment.ID.String(),
		MerchantID:  payment.MerchantID,
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		PaymentType: paymentType,
		Status:      status,
		Timestamp:   payment.UpdatedAt,
	}
	if err := s.publisher.Publish(event); err != nil {
		slog.Warn("svc: failed to publish event", "error", err, "event_type", paymentType)
	}
}

func (s *PaymentService) getExistingPayment(ctx context.Context, merchantID, idempotencyKey string) (*model.Payment, error) {
	payment, err := s.repo.GetPaymentByIdempotencyKey(ctx, merchantID, idempotencyKey)
	if err != nil {
		slog.Error("svc: failed to get payment by idempotency key", "error", err, "merchant_id", merchantID, "idempotency_key", idempotencyKey)
		return nil, err
	}
	if payment != nil {
		slog.Info("svc: payment already exists for idempotency key", "merchant_id", merchantID, "idempotency_key", idempotencyKey)
	}
	return payment, nil
}
