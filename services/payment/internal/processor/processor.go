package processor

import "context"

type PaymentProcessor interface {
	ProcessPayment(ctx context.Context, req *PaymentProcessorRequest) (*ProcessorResponse, error)
	RefundPayment(ctx context.Context, req *RefundProcessorRequest) (*ProcessorResponse, error)
}
