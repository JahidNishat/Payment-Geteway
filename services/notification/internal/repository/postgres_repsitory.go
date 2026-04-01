package repository

import (
	"context"
	"fmt"

	"github.com/JahidNishat/payment-gateway/services/notification/internal/model"
	"github.com/jmoiron/sqlx"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

var _ WebhookRepository = (*PostgresRepository)(nil)

func (p *PostgresRepository) SaveAttempt(ctx context.Context, attempt *model.WebhookAttempt) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO webhook_attempts (
		payment_id, merchant_id, url, success, request_body, response_body, status_code, error, attempt, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		attempt.PaymentID, attempt.MerchantID, attempt.URL, attempt.Success, attempt.RequestBody, attempt.ResponseBody, attempt.StatusCode, attempt.Error, attempt.Attempt, attempt.DurationMs)
	if err != nil {
		return fmt.Errorf("repo: failed to save webhook attempt: %w", err)
	}
	return nil
}

func (p *PostgresRepository) HasSuccessfulAttempt(ctx context.Context, paymentID string) (bool, error) {
	var exists bool
	err := p.db.GetContext(ctx, &exists,
		`SELECT EXISTS (
			SELECT 1 FROM webhook_attempts
			WHERE payment_id = $1 AND success = true
		)`, paymentID)
	if err != nil {
		return false, fmt.Errorf("repo: failed to check successful attempt: %w", err)
	}
	return exists, nil
}
