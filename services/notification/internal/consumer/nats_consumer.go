package consumer

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type NatsConsumer struct {
	js  nats.JetStreamContext
	nc  *nats.Conn
	sub *nats.Subscription
}

func NewNatsConsumer(url string) (*NatsConsumer, error) {
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
		js: js,
		nc: nc,
	}, nil
}

func handleMessage(msg *nats.Msg) {
	slog.Info("received message", "subject", msg.Subject, "data", string(msg.Data))
	var event PaymentEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("failed to unmarshal message", "error", err)
		msg.Nak()
		return
	}

	// TODO: Process the event (e.g., update database, send notification)
	slog.Info("processed event", "payment_id", event.PaymentID, "status", event.Status)
	msg.Ack()
}

func (c *NatsConsumer) Start() error {
	subject := "payments.>"
	sub, err := c.js.Subscribe(
		subject,
		handleMessage,
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
