package handler

import (
	"context"
	"errors"
	"log/slog"

	customErr "github.com/JahidNishat/payment-gateway/services/payment/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Check for wrapped errors using errors.As
	var notFoundErr *customErr.NotFoundError
	if errors.As(err, &notFoundErr) {
		return status.Error(codes.NotFound, notFoundErr.Error())
	}

	var validationErr *customErr.ValidationError
	if errors.As(err, &validationErr) {
		return status.Error(codes.InvalidArgument, validationErr.Error())
	}

	var conflictErr *customErr.ConflictError
	if errors.As(err, &conflictErr) {
		return status.Error(codes.AlreadyExists, conflictErr.Error())
	}

	// Handle context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "request timeout")
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request canceled")
	}

	// Default to internal error — don't leak internal details
	slog.Error("unhandled error type", "error", err)
	return status.Error(codes.Internal, "internal server error")
}
