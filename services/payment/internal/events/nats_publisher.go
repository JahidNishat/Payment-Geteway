package events

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	js nats.JetStreamContext
	nc *nats.Conn
}

func NewNatsPublisher(url string) (*NatsPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "PAYMENTS",
		Subjects: []string{"payments.>"},
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to add stream: %w", err)
	}

	return &NatsPublisher{
		js: js,
		nc: nc,
	}, nil
}

func (p *NatsPublisher) Publish(event *PaymentEvent) error {
	subject := fmt.Sprintf("payments.%s.%s", strings.ToLower(event.PaymentType), strings.ToLower(event.Status))

	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}
	_, err = p.js.Publish(subject, jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}

func (p *NatsPublisher) Close() {
	p.nc.Close()
}
