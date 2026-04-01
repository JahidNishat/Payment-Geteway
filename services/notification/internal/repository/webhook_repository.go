package repository

import (
	"context"

	"github.com/JahidNishat/payment-gateway/services/notification/internal/model"
)

type WebhookRepository interface {
	SaveAttempt(ctx context.Context, attempt *model.WebhookAttempt) error
	HasSuccessfulAttempt(ctx context.Context, paymentID string) (bool, error)
}
