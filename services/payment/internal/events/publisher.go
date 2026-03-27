package events

import "time"

type EventPublisher interface {
	Publish(event *PaymentEvent) error
}

type PaymentEvent struct {
	PaymentID   string
	MerchantID  string
	Amount      int64
	Currency    string
	PaymentType string // e.g., "payment", "refund"
	Status      string // e.g., "pending", "completed", "failed"
	Timestamp   time.Time
}
