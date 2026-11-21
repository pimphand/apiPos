package payment

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"apiPos/internal/accounting"
	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/realtime"
	"apiPos/internal/repository"
	"apiPos/internal/selforder"
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
	repos   *repository.Repositories
	posting *accounting.PostingService
}

// NewService creates a new payment Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos, posting: accounting.NewPostingService(repos.DB())}
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
	SaleID     string
	GrandTotal float64
	TotalPaid  float64
	Remaining  float64
	Change     float64
	Status     string
	Payments   []gormdb.Payment
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
	if req.Amount <= 0 || math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
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

	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return nil, apperror.ErrUnauthorized
	}
	storeID, err := uuid.Parse(claims.StoreID)
	if err != nil || storeID == uuid.Nil {
		return nil, apperror.ErrForbidden
	}

	var sale gormdb.Sale
	var payment gormdb.Payment
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND store_id = ?", req.SaleID, storeID).
			First(&sale).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.Wrap(apperror.ErrNotFound, "Sale not found", nil)
		}
		if err != nil {
			return err
		}
		switch sale.Status {
		case "PENDING":
		case "PAID":
			return apperror.Wrap(apperror.ErrInvalidRequest, "Sale is already paid", nil)
		case "CANCELLED", "REFUNDED":
			return apperror.Wrap(apperror.ErrInvalidRequest, "Cannot add payment to cancelled/refunded sale", nil)
		default:
			return apperror.Wrap(apperror.ErrFailedPrecondition, "Sale is not awaiting payment", nil)
		}

		var paid float64
		if err := tx.Model(&gormdb.Payment{}).Where("sale_id = ? AND store_id = ?", sale.ID, storeID).
			Select("COALESCE(SUM(amount), 0)").Scan(&paid).Error; err != nil {
			return err
		}
		amount := accounting.COGSMoney(req.Amount)
		remaining := accounting.COGSMoney(sale.GrandTotal).Sub(accounting.COGSMoney(paid))
		if !amount.IsPositive() {
			return apperror.Wrap(apperror.ErrInvalidRequest, "Amount must be at least 0.01", nil)
		}
		isSelfOrder := sale.OrderSource == "SELF_ORDER"
		if req.PaymentMethod != PaymentMethodCash && req.Amount > remaining.InexactFloat64() {
			return apperror.Wrap(apperror.ErrInvalidRequest,
				fmt.Sprintf("Payment amount (%.2f) exceeds remaining (%.2f)", req.Amount, remaining.InexactFloat64()), nil)
		}
		if isSelfOrder {
			if time.Since(sale.CreatedAt) >= selforder.SelfOrderPaymentTimeout {
				return apperror.Wrap(apperror.ErrFailedPrecondition, "Order has expired", nil)
			}
			if err := selforder.ValidatePaymentMethod(req.PaymentMethod); err != nil {
				return err
			}
			if !amount.Equal(remaining) {
				return apperror.Wrap(apperror.ErrInvalidRequest, "Self orders require full payment", nil)
			}
		}
		payment = gormdb.Payment{
			SaleID: sale.ID, StoreID: storeID, PaymentMethod: req.PaymentMethod,
			Amount: req.Amount, Reference: req.Reference, PaidAt: time.Now(),
		}
		if err := tx.Create(&payment).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to create payment", err)
		}
		if amount.LessThan(remaining) {
			return nil
		}
		sale.Status = "PAID"
		if err := tx.Model(&sale).Where("store_id = ?", storeID).Update("status", sale.Status).Error; err != nil {
			return err
		}
		if isSelfOrder {
			var items []gormdb.SaleItem
			if err := tx.Where("sale_id = ? AND store_id = ?", sale.ID, storeID).Find(&items).Error; err != nil {
				return err
			}
			_, err := s.posting.PostSale(ctx, tx, &sale, items, []gormdb.Payment{payment})
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if sale.OrderSource == "SELF_ORDER" {
		realtime.Publish(realtime.Event{
			Event: realtime.EventSaleCreated, Resource: "sales",
			ResourceID: sale.ID.String(), StoreID: storeID.String(),
			Data: map[string]any{"order_source": sale.OrderSource, "status": sale.Status,
				"invoice_number": sale.InvoiceNumber, "table_no": sale.TableNo},
		})
		realtime.Publish(realtime.Event{
			Event: realtime.EventInventoryAdjusted, Resource: "sales",
			ResourceID: sale.ID.String(), StoreID: storeID.String(),
		})
	} else {
		_ = s.postPaymentJournal(ctx, &payment, &sale)
	}
	return &payment, nil
}

// postPaymentJournal posts the PAYMENT journal when the sale has an
// outstanding receivable. Returns nil when no receivable exists (fully
// settled sale) — the SALE entry already covered it.
func (s *Service) postPaymentJournal(ctx context.Context, payment *gormdb.Payment, sale *gormdb.Sale) error {
	// Check the sale's SALE journal: if it posted a Receivable leg, there is
	// an outstanding balance. We detect this by looking for an existing
	// journal with source_type SALE whose lines include account 1130.
	var saleJournal gormdb.JournalEntry
	err := s.repos.DB().WithContext(ctx).
		Where("store_id = ? AND source_type = ? AND source_id = ? AND status = ?",
			payment.StoreID, accounting.SourceTypeSale, sale.ID, accounting.JournalStatusPosted).
		First(&saleJournal).Error
	if err != nil {
		// No SALE journal yet (e.g. PENDING sale) — nothing to settle.
		return nil
	}
	var count int64
	s.repos.DB().WithContext(ctx).Model(&gormdb.JournalLine{}).
		Where("store_id = ? AND entry_id = ?", payment.StoreID, saleJournal.ID).
		Count(&count)
	// Heuristic: if the journal has 4+ lines, it includes the COGS leg (not
	// receivable). No receivable leg -> already settled.
	// Instead, check directly for a Receivable line on the SALE journal.
	var receivableLine gormdb.JournalLine
	err = s.repos.DB().WithContext(ctx).
		Where("store_id = ? AND entry_id = ?", payment.StoreID, saleJournal.ID).
		Joins("JOIN chart_of_accounts ON chart_of_accounts.id = journal_lines.account_id").
		Where("chart_of_accounts.code = ?", accounting.CodeReceivables).
		First(&receivableLine).Error
	if err != nil {
		return nil // no receivable leg -> already settled in SALE entry
	}
	// Outstanding receivable exists — post the settlement.
	_, err = s.posting.PostPayment(ctx, s.repos.DB().WithContext(ctx), payment, sale)
	return err
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
