package webhook

import "time"

const (
	MaxRetries = 3
	RetryDelay = 2 * time.Second
)

type WebhookInterface interface {
	Send(payload *WebhookPayload) *WebhookResult
}

type WebhookPayload struct {
	PaymentID   string    `json:"payment_id"`
	MerchantID  string    `json:"merchant_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	PaymentType string    `json:"payment_type"` // e.g., "payment", "refund"
	Status      string    `json:"status"`       // e.g., "pending", "completed", "failed"
	Timestamp   time.Time `json:"timestamp"`
}

type WebhookResult struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	Error      error         `json:"error,omitempty"`
	Attempts   int           `json:"attempts"`
	Permanent  bool          `json:"permanent"`
	URL        string        `json:"url"`
}
