package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Payment method constants
const (
	PaymentMethodCash         = "CASH"
	PaymentMethodCard         = "CARD"
	PaymentMethodBankTransfer = "BANK_TRANSFER"
	PaymentMethodEWallet      = "EWALLET"
	PaymentMethodOther        = "OTHER"
)

// Service handles payment business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new payment Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreatePaymentRequest represents the request to create a payment
type CreatePaymentRequest struct {
	SaleID        string
	PaymentMethod string
	Amount        float64
	Reference     string
}

// PaymentSummary holds payment summary for a sale
type PaymentSummary struct {
	SaleID      string
	GrandTotal  float64
	TotalPaid   float64
	Remaining   float64
	Change      float64
	Status      string
	Payments    []gormdb.Payment
}

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListResult holds the result of listing payments
type ListResult struct {
	Payments   []gormdb.Payment
	Pagination PaginationResult
}

// CreatePayment creates a new payment for a sale
func (s *Service) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*gormdb.Payment, error) {
	// Validate inputs
	if !isValidUUID(req.SaleID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid sale ID", nil)
	}
	if req.Amount <= 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Amount must be greater than 0", nil)
	}

	// Validate payment method
	validMethods := map[string]bool{
		PaymentMethodCash:         true,
		PaymentMethodCard:         true,
		PaymentMethodBankTransfer: true,
		PaymentMethodEWallet:      true,
		PaymentMethodOther:        true,
	}
	if !validMethods[req.PaymentMethod] {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, fmt.Sprintf("Invalid payment method: %s", req.PaymentMethod), nil)
	}

	// Get sale
	sale, err := s.repos.Sale.GetByID(ctx, req.SaleID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Sale not found", err)
	}

	// Check if sale is already paid
	if sale.Status == "PAID" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Sale is already paid", nil)
	}
	if sale.Status == "CANCELLED" || sale.Status == "REFUNDED" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Cannot add payment to cancelled/refunded sale", nil)
	}

	// Get current total paid
	currentTotalPaid, err := s.repos.Payment.GetTotalBySaleID(ctx, req.SaleID)
	if err != nil {
		currentTotalPaid = 0
	}

	// Check if payment exceeds grand total (for non-cash payments)
	remaining := sale.GrandTotal - currentTotalPaid
	if req.PaymentMethod != PaymentMethodCash && req.Amount > remaining {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest,
			fmt.Sprintf("Payment amount (%.2f) exceeds remaining (%.2f)", req.Amount, remaining), nil)
	}

	// Create payment
	payment := &gormdb.Payment{
		SaleID:        sale.ID,
		PaymentMethod: req.PaymentMethod,
		Amount:        req.Amount,
		Reference:     req.Reference,
		PaidAt:        time.Now(),
	}

	if err := s.repos.Payment.Create(ctx, payment); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create payment", err)
	}

	// Check if sale is now fully paid
	newTotalPaid := currentTotalPaid + req.Amount
	if newTotalPaid >= sale.GrandTotal {
		sale.Status = "PAID"
		s.repos.Sale.Update(ctx, sale)
	}

	return payment, nil
}

// GetPayment gets a payment by ID
func (s *Service) GetPayment(ctx context.Context, id string) (*gormdb.Payment, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid payment ID", nil)
	}

	payment, err := s.repos.Payment.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Payment not found", err)
	}

	return payment, nil
}

// GetPaymentsBySale gets all payments for a sale
func (s *Service) GetPaymentsBySale(ctx context.Context, saleID string) ([]gormdb.Payment, error) {
	if !isValidUUID(saleID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid sale ID", nil)
	}

	payments, err := s.repos.Payment.GetBySaleID(ctx, saleID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get payments", err)
	}

	return payments, nil
}

// ListPayments lists payments with filters
func (s *Service) ListPayments(ctx context.Context, saleID, paymentMethod string, page, perPage int32) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	payments, totalCount, err := s.repos.Payment.List(ctx, repository.PaymentListParams{
		SaleID:        saleID,
		PaymentMethod: paymentMethod,
		Page:          page,
		PerPage:       perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list payments", err)
	}

	return &ListResult{
		Payments: payments,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// DeletePayment deletes a payment
func (s *Service) DeletePayment(ctx context.Context, id string) error {
	if !isValidUUID(id) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid payment ID", nil)
	}

	_, err := s.repos.Payment.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(apperror.ErrNotFound, "Payment not found", err)
	}

	return s.repos.Payment.Delete(ctx, id)
}

// GetPaymentSummary gets payment summary for a sale
func (s *Service) GetPaymentSummary(ctx context.Context, saleID string) (*PaymentSummary, error) {
	if !isValidUUID(saleID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid sale ID", nil)
	}

	sale, err := s.repos.Sale.GetByID(ctx, saleID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Sale not found", err)
	}

	payments, err := s.repos.Payment.GetBySaleID(ctx, saleID)
	if err != nil {
		payments = []gormdb.Payment{}
	}

	totalPaid, _ := s.repos.Payment.GetTotalBySaleID(ctx, saleID)
	remaining := sale.GrandTotal - totalPaid
	if remaining < 0 {
		remaining = 0
	}

	change := totalPaid - sale.GrandTotal
	if change < 0 {
		change = 0
	}

	status := "PENDING"
	if totalPaid >= sale.GrandTotal {
		status = "PAID"
	} else if totalPaid > 0 {
		status = "PARTIAL"
	}

	return &PaymentSummary{
		SaleID:     saleID,
		GrandTotal: sale.GrandTotal,
		TotalPaid:  totalPaid,
		Remaining:  remaining,
		Change:     change,
		Status:     status,
		Payments:   payments,
	}, nil
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
