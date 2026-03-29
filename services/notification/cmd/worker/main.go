package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/JahidNishat/payment-gateway/services/notification/internal/config"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/consumer"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/webhook"
)

func main() {
	// slog
	logger := slog.New(slog.NewJSONHandler(log.Writer(), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return
	}

	// Initialize NATS consumer
	webhookSender := webhook.NewWebhookSender() // You can implement this to send actual webhooks
	consumer, err := consumer.NewNatsConsumer(cfg.NatsURL, webhookSender)
	if err != nil {
		slog.Error("failed to create NATS consumer", "error", err)
		return
	}
	defer consumer.Stop()

	// Start consuming messages
	cErr := consumer.Start()
	if cErr != nil {
		slog.Error("failed to start consumer", "error", cErr)
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	slog.Info("shutting down notification worker")
}
