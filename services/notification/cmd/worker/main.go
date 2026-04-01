package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/JahidNishat/payment-gateway/services/notification/internal/config"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/consumer"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/repository"
	"github.com/JahidNishat/payment-gateway/services/notification/internal/webhook"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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

	//Initialize DB
	db, err := sqlx.Connect("postgres", cfg.DBURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return
	}
	defer db.Close()
	slog.Info("connected to database")

	// Initialize NATS consumer
	ctx := context.Background()

	webhookSender := webhook.NewWebhookSender()
	webhookRepo := repository.NewPostgresRepository(db)
	consumer, err := consumer.NewNatsConsumer(ctx, cfg.NatsURL, webhookSender, webhookRepo)
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
