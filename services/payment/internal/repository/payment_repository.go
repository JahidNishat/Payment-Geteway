package repository

import (
	"context"

	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *model.Payment) error
	GetPaymentByID(ctx context.Context, id string) (*model.Payment, error)
	ListPaymentsByMerchantID(ctx context.Context, merchantID string, page, limit int) ([]*model.Payment, int, error)
	UpdatePaymentStatus(ctx context.Context, id, newStatus, reason string, txnID *string) error
	CreateRefund(ctx context.Context, refund *model.Refund) error
	GetRefundsByPaymentID(ctx context.Context, paymentID string) ([]*model.Refund, error)
	GetPaymentByIdempotencyKey(ctx context.Context, merchantID, idempotencyKey string) (*model.Payment, error)
}
