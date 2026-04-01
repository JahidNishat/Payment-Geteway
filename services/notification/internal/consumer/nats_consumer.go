package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/JahidNishat/payment-gateway/services/notification/internal/model"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/repository"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/webhook"
	"github.com/nats-io/nats.go"
)

type NatsConsumer struct {
	ctx     context.Context
	js      nats.JetStreamContext
	nc      *nats.Conn
	sub     *nats.Subscription
	webhook webhook.WebhookInterface
	repo    repository.WebhookRepository
}

func NewNatsConsumer(ctx context.Context, url string, webhookSender webhook.WebhookInterface, repo repository.WebhookRepository) (*NatsConsumer, error) {
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
		ctx:     ctx,
		js:      js,
		nc:      nc,
		webhook: webhookSender,
		repo:    repo,
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

	hasSuccessfulAttempt, _ := c.repo.HasSuccessfulAttempt(c.ctx, event.PaymentID)
	if hasSuccessfulAttempt {
		slog.Info("skipping, webhook already succeeded", "payment_id", event.PaymentID)
		msg.Ack()
		return
	}

	// TODO: Process the event (e.g., update database, send notification)
	webhookPld := &webhook.WebhookPayload{
		PaymentID:   event.PaymentID,
		MerchantID:  event.MerchantID,
		Amount:      event.Amount,
		Currency:    event.Currency,
		PaymentType: event.PaymentType,
		Status:      event.Status,
		Timestamp:   event.Timestamp,
	}
	whResp := c.webhook.Send(webhookPld)
	reqBody, _ := json.Marshal(webhookPld)

	slog.Info("webhook sent",
		"success", whResp.Success,
		"status_code", whResp.StatusCode,
		"duration_ms", whResp.Duration.Milliseconds(),
		"error", whResp.Error,
	)

	respData := map[string]any{
		"success":     whResp.Success,
		"status_code": whResp.StatusCode,
		"duration_ms": whResp.Duration.Milliseconds(),
	}
	whRespErr := ""
	if whResp.Error != nil {
		respData["error"] = whResp.Error.Error()
		whRespErr = whResp.Error.Error()
	}
	respBody, _ := json.Marshal(respData)

	err := c.repo.SaveAttempt(c.ctx, &model.WebhookAttempt{
		PaymentID:    event.PaymentID,
		MerchantID:   event.MerchantID,
		URL:          whResp.URL,
		Success:      whResp.Success,
		RequestBody:  reqBody,
		ResponseBody: respBody,
		StatusCode:   &whResp.StatusCode,
		Error:        &whRespErr,
		Attempt:      whResp.Attempts,
		DurationMs:   whResp.Duration.Milliseconds(),
	})
	if err != nil {
		slog.Error("failed to save webhook attempt", "error", err)
		msg.Nak()
		return
	}
	msg.Ack()
}

func (c *NatsConsumer) Start() error {
	subject := "payments.>"
	sub, err := c.js.QueueSubscribe(
		subject,
		"notification-workers",
		c.handleMessage,
		nats.Durable("notification-worker"),
		nats.DeliverNew(),
		nats.AckExplicit(),
		nats.MaxDeliver(5),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		slog.Error("failed to subscribe to subject", "error", err, "subject", subject)
		return fmt.Errorf("failed to subscribe to subject: %w", err)
	}
	c.sub = sub
	slog.Info("NATS consumer started",
		"subject", subject,
		"queue_group", "notification-workers")
	return nil
}

func (c *NatsConsumer) Stop() {
	if c.sub != nil {
		c.sub.Unsubscribe()
	}
	c.nc.Close()
	slog.Info("NATS consumer stopped")
}
