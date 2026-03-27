package repository

import (
	"context"
	"fmt"

	"github.com/JahidNishat/payment-gateway/services/payment/internal/errors"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
	"github.com/JahidNishat/payment-gateway/services/payment/pkg/dbhelper"
	"github.com/jmoiron/sqlx"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

var _ PaymentRepository = (*PostgresRepository)(nil)

// ========== Create Payment Repository ==========
func (p *PostgresRepository) CreatePayment(ctx context.Context, payment *model.Payment) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO payments (idempotency_key, merchant_id, amount, currency, status, description) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		payment.IdempotencyKey, payment.MerchantID, payment.Amount, payment.Currency, payment.Status, payment.Description).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		if dbhelper.IsDuplicate(err) {
			return &errors.ConflictError{
				Resource: "payment",
				Field:    "idempotency_key",
				Value:    payment.MerchantID + payment.IdempotencyKey,
			}
		}
		return fmt.Errorf("failed to create payment: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO payment_events (payment_id, from_status, to_status, reason) VALUES ($1, $2, $3, $4)`,
		payment.ID, "", payment.Status, "payment created")
	if err != nil {
		return fmt.Errorf("failed to create payment event: %w", err)
	}

	return tx.Commit()
}


// ========== Get Payment Repository ==========
func (p *PostgresRepository) GetPaymentByID(ctx context.Context, id string) (*model.Payment, error) {
	var payment model.Payment
	err := p.db.GetContext(ctx, &payment, "SELECT * FROM payments WHERE id = $1", id)
	if err != nil {
		if dbhelper.IsNotFound(err) {
			return nil, &errors.NotFoundError{
				Resource: "payment",
				ID:       id,
			}
		}
		return nil, fmt.Errorf("failed to get payment by ID: %w", err)
	}
	return &payment, nil
}


// ========== Update Payment Status Repository ==========
func (p *PostgresRepository) UpdatePaymentStatus(ctx context.Context, id, newStatus, reason string, txnID *string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM payments WHERE id = $1 FOR UPDATE", id).Scan(&currentStatus)
	if err != nil {
		if dbhelper.IsNotFound(err) {
			return &errors.NotFoundError{
				Resource: "payment",
				ID:       id,
			}
		}
		return fmt.Errorf("failed to get current payment status: %w", err)
	}

	if !model.IsValidTransition(currentStatus, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
	}

	_, err = tx.ExecContext(ctx, "UPDATE payments SET status = $1, txn_id = COALESCE($2, txn_id), updated_at = NOW() WHERE id = $3", newStatus, txnID, id)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO payment_events (payment_id, from_status, to_status, reason) VALUES ($1, $2, $3, $4)`,
		id, currentStatus, newStatus, reason)
	if err != nil {
		return fmt.Errorf("failed to create payment event: %w", err)
	}

	return tx.Commit()
}


// ========== List Payments Repository ==========
func (p *PostgresRepository) ListPaymentsByMerchantID(ctx context.Context, merchantID string, page, limit int) ([]*model.Payment, int, error) {
	var total int
	err := p.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM payments WHERE merchant_id = $1", merchantID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	var payments []*model.Payment
	err = p.db.SelectContext(ctx, &payments, "SELECT * FROM payments WHERE merchant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		merchantID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}

	return payments, total, nil
}


// ========== Create Refund Repository ==========
func (p *PostgresRepository) CreateRefund(ctx context.Context, refund *model.Refund) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var oldStatus string
	var amount int64
	err = tx.QueryRowContext(ctx, `SELECT status, amount FROM payments WHERE id = $1 FOR UPDATE`, refund.PaymentID).Scan(&oldStatus, &amount)
	if err != nil {
		if dbhelper.IsNotFound(err) {
			return &errors.NotFoundError{
				Resource: "payment",
				ID:       refund.PaymentID,
			}
		}
		return fmt.Errorf("failed to get payment for refund: %w", err)
	}

	if oldStatus != model.PaymentStatusCompleted && oldStatus != model.PaymentStatusRefunded {
		return fmt.Errorf("cannot refund payment with status %s", oldStatus)
	}

	var totalRefunded int64
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount), 0) FROM refunds WHERE payment_id = $1`, refund.PaymentID).Scan(&totalRefunded)
	if err != nil {
		return fmt.Errorf("failed to calculate total refunded amount: %w", err)
	}

	if totalRefunded+refund.Amount > amount {
		return fmt.Errorf("refund amount exceeds original payment amount")
	}

	err = tx.QueryRowContext(ctx,
		`INSERT INTO refunds (payment_id, amount, reason, status) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		refund.PaymentID, refund.Amount, refund.Reason, refund.Status).Scan(&refund.ID, &refund.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	// Update Payment status to REFUNDED
	if oldStatus == model.PaymentStatusCompleted {
		_, err = tx.ExecContext(ctx, "UPDATE payments SET status = $1, updated_at = NOW() WHERE id = $2", model.PaymentStatusRefunded, refund.PaymentID)
		if err != nil {
			return fmt.Errorf("failed to update payment status to refunded: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO payment_events (payment_id, from_status, to_status, reason) VALUES ($1, $2, $3, $4)`,
		refund.PaymentID, oldStatus, model.PaymentStatusRefunded, "refund created")
	if err != nil {
		return fmt.Errorf("failed to create payment event: %w", err)
	}

	return tx.Commit()
}


// ========== Get Refunds Repository ==========
func (p *PostgresRepository) GetRefundsByPaymentID(ctx context.Context, paymentID string) ([]*model.Refund, error) {
	var refunds []*model.Refund
	err := p.db.SelectContext(ctx, &refunds, "SELECT * FROM refunds WHERE payment_id = $1 ORDER BY created_at DESC", paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refunds by payment ID: %w", err)
	}
	return refunds, nil
}

func (p *PostgresRepository) GetPaymentByIdempotencyKey(ctx context.Context, merchantID, idempotencyKey string) (*model.Payment, error) {
	var payment model.Payment
	err := p.db.GetContext(ctx, &payment, "SELECT * FROM payments WHERE merchant_id = $1 AND idempotency_key = $2", merchantID, idempotencyKey)
	if err != nil {
		if dbhelper.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment by idempotency key: %w", err)
	}
	return &payment, nil
}
