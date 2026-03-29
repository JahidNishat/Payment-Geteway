package consumer

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JahidNishat/payment-gateway/services/notification/internal/webhook"
	"github.com/nats-io/nats.go"
)

type NatsConsumer struct {
	js      nats.JetStreamContext
	nc      *nats.Conn
	sub     *nats.Subscription
	webhook webhook.WebhookInterface
}

func NewNatsConsumer(url string, webhookSender webhook.WebhookInterface) (*NatsConsumer, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &NatsConsumer{
		js:      js,
		nc:      nc,
		webhook: webhookSender,
	}, nil
}

func (c *NatsConsumer) handleMessage(msg *nats.Msg) {
	slog.Info("received message", "subject", msg.Subject, "data", string(msg.Data))
	var event PaymentEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("failed to unmarshal message", "error", err)
		msg.Nak()
		return
	}

	// TODO: Process the event (e.g., update database, send notification)
	whResp := c.webhook.Send(&webhook.WebhookPayload{
		PaymentID:   event.PaymentID,
		MerchantID:  event.MerchantID,
		Amount:      event.Amount,
		Currency:    event.Currency,
		PaymentType: event.PaymentType,
		Status:      event.Status,
		Timestamp:   event.Timestamp,
	})
	slog.Info("webhook sent",
		"success", whResp.Success,
		"status_code", whResp.StatusCode,
		"duration_ms", whResp.Duration.Milliseconds(),
		"error", whResp.Error,
	)
	msg.Ack()
}

func (c *NatsConsumer) Start() error {
	subject := "payments.>"
	sub, err := c.js.Subscribe(
		subject,
		c.handleMessage,
		nats.Durable("notification-worker"),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	if err != nil {
		slog.Error("failed to subscribe to subject", "error", err, "subject", subject)
		return fmt.Errorf("failed to subscribe to subject: %w", err)
	}
	c.sub = sub
	slog.Info("NATS consumer started", "subject", subject)
	return nil
}

func (c *NatsConsumer) Stop() {
	if c.sub != nil {
		c.sub.Unsubscribe()
	}
	c.nc.Close()
	slog.Info("NATS consumer stopped")
}
