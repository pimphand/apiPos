package handler

import (
	"context"

	paymentv1 "apiPos/gen/payment/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/payment"
	"apiPos/internal/response"
)

// PaymentHandler implements the PaymentService gRPC server
type PaymentHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	service *payment.Service
}

// NewPaymentHandler creates a new payment Handler
func NewPaymentHandler(service *payment.Service) *PaymentHandler {
	return &PaymentHandler{service: service}
}

// CreatePayment creates a new payment for a sale
func (h *PaymentHandler) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.PaymentResponse, error) {
	p, err := h.service.CreatePayment(ctx, payment.CreatePaymentRequest{
		SaleID:        req.SaleId,
		PaymentMethod: req.PaymentMethod,
		Amount:        req.Amount,
		Reference:     req.Reference,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &paymentv1.PaymentResponse{
		Payment: toProtoPayment(p),
	}, nil
}

// GetPayment gets a payment by ID
func (h *PaymentHandler) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.PaymentResponse, error) {
	p, err := h.service.GetPayment(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &paymentv1.PaymentResponse{
		Payment: toProtoPayment(p),
	}, nil
}

// GetPaymentsBySale gets all payments for a sale
func (h *PaymentHandler) GetPaymentsBySale(ctx context.Context, req *paymentv1.GetPaymentsBySaleRequest) (*paymentv1.ListPaymentsResponse, error) {
	payments, err := h.service.GetPaymentsBySale(ctx, req.SaleId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoPayments := make([]*paymentv1.Payment, 0, len(payments))
	for _, p := range payments {
		protoPayments = append(protoPayments, toProtoPayment(&p))
	}

	return &paymentv1.ListPaymentsResponse{
		Payments: protoPayments,
	}, nil
}

// ListPayments lists payments with filters
func (h *PaymentHandler) ListPayments(ctx context.Context, req *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	result, err := h.service.ListPayments(ctx, req.SaleId, req.PaymentMethod, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	payments := make([]*paymentv1.Payment, 0, len(result.Payments))
	for _, p := range result.Payments {
		payments = append(payments, toProtoPayment(&p))
	}

	return &paymentv1.ListPaymentsResponse{
		Payments:   payments,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// DeletePayment deletes a payment
func (h *PaymentHandler) DeletePayment(ctx context.Context, req *paymentv1.DeletePaymentRequest) (*paymentv1.DeletePaymentResponse, error) {
	err := h.service.DeletePayment(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &paymentv1.DeletePaymentResponse{
		Success: true,
		Message: response.MsgPaymentDeleted,
	}, nil
}

// GetPaymentSummary gets payment summary for a sale
func (h *PaymentHandler) GetPaymentSummary(ctx context.Context, req *paymentv1.GetPaymentSummaryRequest) (*paymentv1.PaymentSummaryResponse, error) {
	summary, err := h.service.GetPaymentSummary(ctx, req.SaleId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	payments := make([]*paymentv1.Payment, 0, len(summary.Payments))
	for _, p := range summary.Payments {
		payments = append(payments, toProtoPayment(&p))
	}

	return &paymentv1.PaymentSummaryResponse{
		SaleId:     summary.SaleID,
		GrandTotal: summary.GrandTotal,
		TotalPaid:  summary.TotalPaid,
		Remaining:  summary.Remaining,
		Change:     summary.Change,
		Status:     summary.Status,
		Payments:   payments,
	}, nil
}

// toProtoPayment converts a GORM Payment to proto Payment
func toProtoPayment(p *gormdb.Payment) *paymentv1.Payment {
	return &paymentv1.Payment{
		Id:            p.ID.String(),
		SaleId:        p.SaleID.String(),
		PaymentMethod: p.PaymentMethod,
		Amount:        p.Amount,
		Reference:     p.Reference,
		PaidAt:        p.PaidAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
