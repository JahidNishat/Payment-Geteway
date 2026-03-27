package events

import "log/slog"

type LogPublisher struct{}

func NewLogPublisher() *LogPublisher { return &LogPublisher{} }

func (p *LogPublisher) Publish(event *PaymentEvent) error {
	// For simplicity, we just log the event. In a real implementation, this could be more structured.
	slog.Info("Event Published",
		"payment_id", event.PaymentID,
		"merchant_id", event.MerchantID,
		"amount", event.Amount,
		"currency", event.Currency,
		"payment_type", event.PaymentType,
		"status", event.Status,
		"timestamp", event.Timestamp,
	)
	return nil
}
