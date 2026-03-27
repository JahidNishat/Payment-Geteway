package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/JahidNishat/payment-gateway/gen/go/payment/v1"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/config"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/events"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/handler"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/processor"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/repository"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/service"
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Config Load
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect DB
	db, err := sqlx.Connect("postgres", cfg.DBURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to database")

	// Initialize the layers
	repo := repository.NewPostgresRepository(db)
	paymentProcessor := processor.NewSimulatePaymentProcessor()
	eventPublisher := events.NewLogPublisher()
	svc := service.NewPaymentService(repo, paymentProcessor, eventPublisher)
	h := handler.NewPaymentHandler(svc)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	pb.RegisterPaymentServiceServer(grpcServer, h)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("payment service is running on port: " + cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve", "err", err)
			quit <- os.Interrupt
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	slog.Info("shutting down server")
	select {
	case <-ctx.Done():
		slog.Warn("timeout reached, forcing shutdown")
		grpcServer.Stop()
	case <-stopped:
		slog.Info("server stopped gracefully")
	}
}
