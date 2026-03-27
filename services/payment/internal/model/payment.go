package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	PaymentStatusPending    = "PENDING"
	PaymentStatusProcessing = "PROCESSING"
	PaymentStatusCompleted  = "COMPLETED"
	PaymentStatusFailed     = "FAILED"
	PaymentStatusRefunded   = "REFUNDED"
)

const (
	RefundStatusPending   = "PENDING"
	RefundStatusCompleted = "COMPLETED"
	RefundStatusFailed    = "FAILED"
)

var ValidTransitions = map[string][]string{
	PaymentStatusPending:    {PaymentStatusProcessing, PaymentStatusFailed},
	PaymentStatusProcessing: {PaymentStatusCompleted, PaymentStatusFailed},
	PaymentStatusCompleted:  {PaymentStatusRefunded},
	PaymentStatusFailed:     {}, // terminal state
	PaymentStatusRefunded:   {}, // terminal state
}

type Payment struct {
	ID             uuid.UUID `json:"id" db:"id"`
	IdempotencyKey string    `json:"idempotency_key" db:"idempotency_key"`
	MerchantID     string    `json:"merchant_id" db:"merchant_id"`
	Amount         int64     `json:"amount" db:"amount"`
	Currency       string    `json:"currency" db:"currency"`
	Status         string    `json:"status" db:"status"`
	Description    *string   `json:"description,omitempty" db:"description"`
	Metadata       []byte    `json:"metadata,omitempty" db:"metadata"`
	TxnID          *string   `json:"txn_id,omitempty" db:"txn_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type PaymentEvent struct {
	ID         uuid.UUID `json:"id" db:"id"`
	PaymentID  uuid.UUID `json:"payment_id" db:"payment_id"`
	FromStatus string    `json:"from_status" db:"from_status"`
	ToStatus   string    `json:"to_status" db:"to_status"`
	Reason     *string   `json:"reason,omitempty" db:"reason"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type Refund struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PaymentID uuid.UUID `json:"payment_id" db:"payment_id"`
	Amount    int64     `json:"amount" db:"amount"`
	Reason    *string   `json:"reason,omitempty" db:"reason"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func IsValidTransition(from, to string) bool {
	validTargets, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, target := range validTargets {
		if target == to {
			return true
		}
	}
	return false
}

type CreatePaymentRequest struct {
	IdempotencyKey string         `json:"idempotency_key" validate:"required,min=2,max=255"`
	MerchantID     string         `json:"merchant_id" validate:"required,min=2,max=255"`
	Amount         int64          `json:"amount" validate:"required,gt=0"`
	Currency       string         `json:"currency" validate:"required,len=3"`
	Description    *string        `json:"description,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type RefundRequest struct {
	PaymentID string `json:"payment_id" validate:"required,uuid4"`
	Amount    int64  `json:"amount" validate:"required,gt=0"`
	Reason    string `json:"reason,omitempty"`
}

func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
