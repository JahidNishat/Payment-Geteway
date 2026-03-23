package handler

import (
	"context"

	pb "github.com/JahidNishat/payment-gateway/gen/go/payment/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{}
}

func (h *PaymentHandler) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	return &pb.CreatePaymentResponse{
		PaymentId: uuid.New().String(),
		Status:    pb.PaymentStatus_PAYMENT_STATUS_PENDING,
		Amount:    req.GetAmount(),
		Currency:  req.GetCurrency(),
		CreatedAt: timestamppb.Now(),
	}, nil
}
