package interceptor

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	code := status.Code(err)

	slog.Info("grpc request",
		"method", info.FullMethod,
		"duration_ms", duration.Milliseconds(),
		"code", code.String(),
	)
	return resp, err
}
