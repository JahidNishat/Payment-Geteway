package model

import (
	"time"

	"github.com/google/uuid"
)

type WebhookAttempt struct {
	ID           uuid.UUID `json:"id" db:"id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	PaymentID    string    `json:"payment_id" db:"payment_id"`
	MerchantID   string    `json:"merchant_id" db:"merchant_id"`
	URL          string    `json:"url" db:"url"`
	Success      bool      `json:"success" db:"success"`
	RequestBody  []byte    `json:"request_body" db:"request_body"`
	ResponseBody []byte    `json:"response_body" db:"response_body"`
	StatusCode   *int      `json:"status_code" db:"status_code"`
	Error        *string   `json:"error,omitempty" db:"error"`
	Attempt      int       `json:"attempt" db:"attempt"`
	DurationMs   int64     `json:"duration_ms" db:"duration_ms"`
}
