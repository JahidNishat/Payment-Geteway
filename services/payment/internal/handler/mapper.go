package handler

import (
	pb "github.com/JahidNishat/payment-gateway/gen/go/payment/v1"
	"github.com/JahidNishat/payment-gateway/services/payment/internal/model"
)

func paymentStatusToProto(status string) pb.PaymentStatus {
	switch status {
	case model.PaymentStatusPending:
		return pb.PaymentStatus_PAYMENT_STATUS_PENDING
	case model.PaymentStatusProcessing:
		return pb.PaymentStatus_PAYMENT_STATUS_PROCESSING
	case model.PaymentStatusCompleted:
		return pb.PaymentStatus_PAYMENT_STATUS_COMPLETED
	case model.PaymentStatusFailed:
		return pb.PaymentStatus_PAYMENT_STATUS_FAILED
	case model.PaymentStatusRefunded:
		return pb.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return pb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func refundStatusToPb(status string) pb.RefundStatus {
	switch status {
	case model.RefundStatusPending:
		return pb.RefundStatus_REFUND_STATUS_PENDING
	case model.RefundStatusCompleted:
		return pb.RefundStatus_REFUND_STATUS_COMPLETED
	case model.RefundStatusFailed:
		return pb.RefundStatus_REFUND_STATUS_FAILED
	default:
		return pb.RefundStatus_REFUND_STATUS_UNSPECIFIED
	}
}
