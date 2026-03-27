package handler

import (
	"context"

	pb "github.com/JahidNishat/payment-gateway/gen/go/payment/v1"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// ========== Create Payment ==========
func (h *PaymentHandler) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	payment := &model.CreatePaymentRequest{
		MerchantID:     req.GetMerchantId(),
		Amount:         req.GetAmount(),
		Currency:       req.GetCurrency(),
		Description:    description,
		IdempotencyKey: req.GetIdempotencyKey(),
	}
	resp, err := h.svc.CreatePayment(ctx, payment)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.CreatePaymentResponse{
		PaymentId: resp.ID.String(),
		Status:    paymentStatusToProto(resp.Status),
		Amount:    resp.Amount,
		Currency:  resp.Currency,
		CreatedAt: timestamppb.New(resp.CreatedAt),
	}, nil
}

// ========== Get Payment ==========
func (h *PaymentHandler) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
	resp, err := h.svc.GetPaymentByID(ctx, req.GetPaymentId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetPaymentResponse{
		PaymentId:   resp.ID.String(),
		Status:      paymentStatusToProto(resp.Status),
		Amount:      resp.Amount,
		Currency:    resp.Currency,
		CreatedAt:   timestamppb.New(resp.CreatedAt),
		MerchantId:  resp.MerchantID,
		Description: model.SafeString(resp.Description),
		UpdatedAt:   timestamppb.New(resp.UpdatedAt),
	}, nil
}

// ========== List Payments ==========
func (h *PaymentHandler) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	resp, total, err := h.svc.ListPayments(ctx, req.GetMerchantId(), page, limit)
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbPayments := make([]*pb.PaymentSummary, len(resp))
	for i, p := range resp {
		pbPayments[i] = &pb.PaymentSummary{
			PaymentId:  p.ID.String(),
			Status:     paymentStatusToProto(p.Status),
			Amount:     p.Amount,
			Currency:   p.Currency,
			CreatedAt:  timestamppb.New(p.CreatedAt),
			MerchantId: p.MerchantID,
		}
	}
	return &pb.ListPaymentsResponse{
		Payments:   pbPayments,
		TotalCount: int32(total),
		Page:       int32(page),
		Limit:      int32(limit),
	}, nil
}

// ========== Refund Payment ==========
func (h *PaymentHandler) RefundPayment(ctx context.Context, req *pb.RefundPaymentRequest) (*pb.RefundPaymentResponse, error) {
	refund := &model.RefundRequest{
		PaymentID: req.GetPaymentId(),
		Amount:    req.GetAmount(),
		Reason:    req.Reason,
	}

	resp, err := h.svc.RefundPayment(ctx, refund)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.RefundPaymentResponse{
		RefundId:  resp.ID.String(),
		PaymentId: resp.PaymentID.String(),
		Amount:    resp.Amount,
		Reason:    model.SafeString(resp.Reason),
		Status:    refundStatusToPb(resp.Status),
	}, nil
}
