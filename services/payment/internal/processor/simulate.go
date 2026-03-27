package processor

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type PaymentProcessorRequest struct {
	PaymentID      string `json:"payment_id"`
	MerchantID     string `json:"merchant_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Description    string `json:"description,omitempty"`
}

type RefundProcessorRequest struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason,omitempty"`
}

type ProcessorResponse struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	TxnID     string `json:"txn_id,omitempty"`
}

const (
	ProcessorStatusSuccess = "SUCCESS"
	ProcessorStatusFailed  = "FAILED"
)

const (
	ExceedsLimitAmount = 1000000
	ExceedsLimitReason = "amount exceeds limit"

	InsufficientFundsAmount = 66600
	InsufficientFundsReason = "insufficient funds"
)

type SimulatePaymentProcessor struct{}

func NewSimulatePaymentProcessor() *SimulatePaymentProcessor {
	return &SimulatePaymentProcessor{}
}

func (p *SimulatePaymentProcessor) ProcessPayment(ctx context.Context, req *PaymentProcessorRequest) (*ProcessorResponse, error) {
	// Simulate 500ms processing time
	time.Sleep(500 * time.Millisecond)

	switch req.Amount {
	case ExceedsLimitAmount:
		return &ProcessorResponse{
			PaymentID: req.PaymentID,
			Status:    ProcessorStatusFailed,
			Reason:    ExceedsLimitReason,
		}, nil
	case InsufficientFundsAmount:
		return &ProcessorResponse{
			PaymentID: req.PaymentID,
			Status:    ProcessorStatusFailed,
			Reason:    InsufficientFundsReason,
		}, nil
	}

	// Simulate random failure with 5% chance
	if rand.Intn(100) < 5 {
		return nil, fmt.Errorf("bank timeout")
	}

	return &ProcessorResponse{
		PaymentID: req.PaymentID,
		Status:    ProcessorStatusSuccess,
		TxnID:     uuid.NewString(),
	}, nil
}

func (p *SimulatePaymentProcessor) RefundPayment(ctx context.Context, req *RefundProcessorRequest) (*ProcessorResponse, error) {
	// Simulate 300ms processing time
	time.Sleep(300 * time.Millisecond)

	// Simulate random failure with 5% chance
	if rand.Intn(100) < 5 {
		return nil, fmt.Errorf("bank timeout")
	}

	// Simulate refund amount exceeds limit
	if req.Amount > 500000 {
		return &ProcessorResponse{
			PaymentID: req.PaymentID,
			Status:    ProcessorStatusFailed,
			Reason:    "refund amount exceeds processor limit",
		}, nil
	}

	// Simulate specific amount triggers failure (for testing)
	if req.Amount == 66600 {
		return &ProcessorResponse{
			PaymentID: req.PaymentID,
			Status:    ProcessorStatusFailed,
			Reason:    "refund rejected by bank",
		}, nil
	}

	return &ProcessorResponse{
		PaymentID: req.PaymentID,
		Status:    ProcessorStatusSuccess,
		TxnID:     uuid.NewString(),
	}, nil
}
