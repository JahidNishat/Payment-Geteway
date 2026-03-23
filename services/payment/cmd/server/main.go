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
	"github.com/JahidNishat/payment-gateway/services/payment/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	h := handler.NewPaymentHandler()
	pb.RegisterPaymentServiceServer(grpcServer, h)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("payment service is running on port: 50051")
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
