package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/JahidNishat/payment-gateway/services/payment/internal/events"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/processor"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/repository"
	"github.com/google/uuid"
)

// ========== Fake Payment Repository ==========
type fakePaymentRepo struct {
	getByIdemKeyResp *model.Payment
	getByIdemKeyErr  error

	createPaymentErr error
	updateStatusErr  error

	createPaymentCalled bool
	statusUpdates       []string
}

var _ repository.PaymentRepository = (*fakePaymentRepo)(nil)

func (f *fakePaymentRepo) CreateRefund(ctx context.Context, refund *model.Refund) error {
	return nil
}

func (f *fakePaymentRepo) GetRefundsByPaymentID(ctx context.Context, paymentID string) ([]*model.Refund, error) {
	return nil, nil
}

func (f *fakePaymentRepo) ListPaymentsByMerchantID(ctx context.Context, merchantID string, page int, limit int) ([]*model.Payment, int, error) {
	return nil, 0, nil
}

func (f *fakePaymentRepo) GetPaymentByID(ctx context.Context, id string) (*model.Payment, error) {
	return nil, nil
}

func (f *fakePaymentRepo) CreatePayment(ctx context.Context, payment *model.Payment) error {
	f.createPaymentCalled = true

	return f.createPaymentErr
}

func (f *fakePaymentRepo) GetPaymentByIdempotencyKey(ctx context.Context, merchantID, idempotencyKey string) (*model.Payment, error) {
	return f.getByIdemKeyResp, f.getByIdemKeyErr
}

func (f *fakePaymentRepo) UpdatePaymentStatus(ctx context.Context, id, newStatus, reason string, txnID *string) error {
	f.statusUpdates = append(f.statusUpdates, newStatus)
	return f.updateStatusErr
}

// ========== Fake Payment Processor ==========
type fakePaymentProcessor struct {
	resp *processor.ProcessorResponse
	err  error
}

var _ processor.PaymentProcessor = (*fakePaymentProcessor)(nil)

func (f *fakePaymentProcessor) ProcessPayment(ctx context.Context, req *processor.PaymentProcessorRequest) (*processor.ProcessorResponse, error) {
	resp := &processor.ProcessorResponse{
		Status: processor.ProcessorStatusSuccess,
	}
	if f.resp != nil {
		resp = f.resp
	}
	return resp, f.err
}

func (f *fakePaymentProcessor) RefundPayment(ctx context.Context, req *processor.RefundProcessorRequest) (*processor.ProcessorResponse, error) {
	resp := &processor.ProcessorResponse{
		Status: processor.ProcessorStatusSuccess,
	}
	if f.resp != nil {
		resp = f.resp
	}
	return resp, f.err
}

// ========== Fake Event Publisher ==========
type fakeEventPublisher struct {
	called bool
	event  *events.PaymentEvent
}

var _ events.EventPublisher = (*fakeEventPublisher)(nil)

func (f *fakeEventPublisher) Publish(event *events.PaymentEvent) error {
	f.called = true
	f.event = event
	return nil
}

func TestCreatePayment_Validation(t *testing.T) {
	tests := []struct {
		name     string
		req      *model.CreatePaymentRequest
		checkErr func(t *testing.T, err error)
	}{
		{
			name: "empty request",
			req:  &model.CreatePaymentRequest{},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, but got nil")
				}
				expectedErrMsg := "is required"
				if !strings.Contains(err.Error(), expectedErrMsg) {
					t.Errorf("expected error message to contain %q, got %q", expectedErrMsg, err.Error())
				}
				if !strings.Contains(err.Error(), "merchantid") {
					t.Errorf("expected error message to mention missing merchant_id, got %q", err.Error())
				}
				if !strings.Contains(err.Error(), "idempotencykey") {
					t.Errorf("expected error message to mention missing idempotency_key, got %q", err.Error())
				}
				if !strings.Contains(err.Error(), "amount") {
					t.Errorf("expected error message to mention missing amount, got %q", err.Error())
				}
				if !strings.Contains(err.Error(), "currency") {
					t.Errorf("expected error message to mention missing currency, got %q", err.Error())
				}
			},
		},
		{
			name: "negative amount",
			req: &model.CreatePaymentRequest{
				MerchantID:     "merchant123",
				Currency:       "USD",
				IdempotencyKey: "idemkey123",
				Amount:         -1000,
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, but got nil")
				}
				expectedErrMsg := "must be greater than 0"
				if !strings.Contains(err.Error(), expectedErrMsg) {
					t.Errorf("expected error message to contain %q, got %q", expectedErrMsg, err.Error())
				}
				if !strings.Contains(err.Error(), "amount") {
					t.Errorf("expected error message to mention amount field, got %q", err.Error())
				}
			},
		},
		{
			name: "zero amount",
			req: &model.CreatePaymentRequest{
				MerchantID:     "merchant123",
				Currency:       "USD",
				IdempotencyKey: "idemkey123",
				Amount:         0,
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, but got nil")
				}
				expectedErrMsg := "is required"
				if !strings.Contains(err.Error(), expectedErrMsg) {
					t.Errorf("expected error message to contain %q, got %q", expectedErrMsg, err.Error())
				}
				if !strings.Contains(err.Error(), "amount") {
					t.Errorf("expected error message to mention amount field, got %q", err.Error())
				}
			},
		},
		{
			name: "missing merchant ID",
			req: &model.CreatePaymentRequest{
				Currency:       "USD",
				IdempotencyKey: "idemkey123",
				Amount:         1000,
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, but got nil")
				}
				expectedErrMsg := "is required"
				if !strings.Contains(err.Error(), expectedErrMsg) {
					t.Errorf("expected error message to contain %q, got %q", expectedErrMsg, err.Error())
				}
				if !strings.Contains(err.Error(), "merchantid") {
					t.Errorf("expected error message to mention missing merchant_id, got %q", err.Error())
				}
			},
		},
		{
			name: "invalid currency",
			req: &model.CreatePaymentRequest{
				MerchantID:     "merchant123",
				Currency:       "INVALID",
				IdempotencyKey: "idemkey123",
				Amount:         1000,
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, but got nil")
				}
				expectedErrMsg := "must be exactly 3 characters"
				if !strings.Contains(err.Error(), expectedErrMsg) {
					t.Errorf("expected error message to contain %q, got %q", expectedErrMsg, err.Error())
				}
				if !strings.Contains(err.Error(), "currency") {
					t.Errorf("expected error message to mention currency field, got %q", err.Error())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &fakePaymentRepo{}
			fakeProcessor := &fakePaymentProcessor{}
			fakePublisher := &fakeEventPublisher{}
			svc := NewPaymentService(fakeRepo, fakeProcessor, fakePublisher)
			_, err := svc.CreatePayment(context.Background(), tt.req)
			tt.checkErr(t, err)
		})
	}
}

func TestCreatePayment_HappyPath(t *testing.T) {
	fakeRepo := &fakePaymentRepo{}
	fakeProcessor := &fakePaymentProcessor{}
	fakePublisher := &fakeEventPublisher{}
	svc := NewPaymentService(fakeRepo, fakeProcessor, fakePublisher)
	req := &model.CreatePaymentRequest{
		MerchantID:     "merchant123",
		Currency:       "USD",
		IdempotencyKey: "test-idem-key-123",
		Amount:         1000,
	}
	resp, err := svc.CreatePayment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response, but got nil")
	}
	if resp.Status != model.PaymentStatusCompleted {
		t.Errorf("expected status %q, got %q", model.PaymentStatusCompleted, resp.Status)
	}
	if fakeRepo.createPaymentCalled == false {
		t.Error("expected CreatePayment to be called on repository, but it was not")
	}
	if !slices.Contains(fakeRepo.statusUpdates, model.PaymentStatusProcessing) {
		t.Error("expected status update to include processing status")
	}
	if !slices.Contains(fakeRepo.statusUpdates, model.PaymentStatusCompleted) {
		t.Error("expected status update to include completed status")
	}
	if fakePublisher.called == false {
		t.Error("expected Publish to be called on event publisher, but it was not")
	}
}

func TestCreatePayment_Idempotency(t *testing.T) {
	existingPayment := &model.Payment{
		ID:             uuid.New(),
		MerchantID:     "merchant123",
		Currency:       "USD",
		IdempotencyKey: "test-idem-key-123",
		Amount:         1000,
		Status:         model.PaymentStatusCompleted,
	}
	fakeRepo := &fakePaymentRepo{
		getByIdemKeyResp: existingPayment,
	}
	fakeProcessor := &fakePaymentProcessor{}
	fakePublisher := &fakeEventPublisher{}
	svc := NewPaymentService(fakeRepo, fakeProcessor, fakePublisher)
	req := &model.CreatePaymentRequest{
		MerchantID:     "merchant123",
		Currency:       "USD",
		IdempotencyKey: "test-idem-key-123",
		Amount:         1000,
	}
	resp, err := svc.CreatePayment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response, but got nil")
	}
	if resp.ID != existingPayment.ID {
		t.Errorf("expected payment ID %q, got %q", existingPayment.ID, resp.ID)
	}
	if resp.Status != existingPayment.Status {
		t.Errorf("expected status %q, got %q", existingPayment.Status, resp.Status)
	}
	if fakeRepo.createPaymentCalled == true {
		t.Error("expected CreatePayment to NOT be called on repository due to idempotency hit, but it was called")
	}
	if fakePublisher.called == true {
		t.Error("expected Publish to NOT be called on event publisher due to idempotency hit, but it was called")
	}
}

func TestCreatePayment_ProcessorFailure(t *testing.T) {
	fakeRepo := &fakePaymentRepo{}
	fakeProcessor := &fakePaymentProcessor{
		err: errors.New("processor failure"),
	}
	fakePublisher := &fakeEventPublisher{}
	svc := NewPaymentService(fakeRepo, fakeProcessor, fakePublisher)
	req := &model.CreatePaymentRequest{
		MerchantID:     "merchant123",
		Currency:       "USD",
		IdempotencyKey: "test-idem-key-123",
		Amount:         1000,
	}
	_, err := svc.CreatePayment(context.Background(), req)
	if err == nil {
		t.Fatal("expected error due to processor failure, but got nil")
	}
	if fakeRepo.createPaymentCalled == false {
		t.Error("expected CreatePayment to be called on repository even when processor fails, but it was not called")
	}
	if !slices.Contains(fakeRepo.statusUpdates, model.PaymentStatusProcessing) {
		t.Error("expected status update to include processing status even when processor fails")
	}
	if !slices.Contains(fakeRepo.statusUpdates, model.PaymentStatusFailed) {
		t.Error("expected status update to include failed status due to processor failure")
	}
}
