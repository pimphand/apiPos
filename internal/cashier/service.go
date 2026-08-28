package cashier

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Shift status constants
const (
	ShiftStatusOpen   = "OPEN"
	ShiftStatusClosed = "CLOSED"
)

// Service handles cashier shift business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new cashier Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// OpenShiftRequest represents the request to open a shift
type OpenShiftRequest struct {
	StoreID      string
	OpeningCash  float64
	UserID       string
}

// CloseShiftRequest represents the request to close a shift
type CloseShiftRequest struct {
	ShiftID     string
	ClosingCash float64
	ActualCash  float64
}

// ShiftDetail holds shift with summary info
type ShiftDetail struct {
	Shift             gormdb.CashierShift
	UserName          string
	StoreName         string
	TotalTransactions int32
	TotalSalesAmount  float64
	Sales             []ShiftSale
}

// ShiftSale holds sale info for shift summary
type ShiftSale struct {
	ID            string
	InvoiceNumber string
	GrandTotal    float64
	PaymentMethod string
	CreatedAt     string
}

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListResult holds the result of listing shifts
type ListResult struct {
	Shifts     []gormdb.CashierShift
	Pagination PaginationResult
}

// OpenShift opens a new cashier shift
func (s *Service) OpenShift(ctx context.Context, req OpenShiftRequest) (*gormdb.CashierShift, error) {
	// Validate inputs
	if !isValidUUID(req.StoreID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store ID", nil)
	}
	if !isValidUUID(req.UserID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID", nil)
	}
	if req.OpeningCash < 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Opening cash cannot be negative", nil)
	}

	// Check if user already has an open shift
	currentShift, err := s.repos.Cashier.GetCurrentShift(ctx, req.UserID)
	if err == nil && currentShift != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "You already have an open shift", nil)
	}

	storeID, _ := uuid.Parse(req.StoreID)
	userID, _ := uuid.Parse(req.UserID)

	// Create shift
	shift := &gormdb.CashierShift{
		UserID:      userID,
		StoreID:     storeID,
		OpeningCash: req.OpeningCash,
		Status:      ShiftStatusOpen,
		OpenedAt:    time.Now(),
	}

	if err := s.repos.Cashier.OpenShift(ctx, shift); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to open shift", err)
	}

	return shift, nil
}

// CloseShift closes a cashier shift
func (s *Service) CloseShift(ctx context.Context, req CloseShiftRequest) (*gormdb.CashierShift, error) {
	// Validate inputs
	if !isValidUUID(req.ShiftID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid shift ID", nil)
	}

	// Get current shift
	shift, err := s.repos.Cashier.GetByID(ctx, req.ShiftID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Shift not found", err)
	}

	if shift.Status == ShiftStatusClosed {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Shift is already closed", nil)
	}

	// Close shift
	if err := s.repos.Cashier.CloseShift(ctx, req.ShiftID, req.ClosingCash, req.ActualCash); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to close shift", err)
	}

	// Get updated shift
	updatedShift, err := s.repos.Cashier.GetByID(ctx, req.ShiftID)
	if err != nil {
		return shift, nil
	}

	return updatedShift, nil
}

// GetCurrentShift gets the current open shift for a user
func (s *Service) GetCurrentShift(ctx context.Context, userID string) (*gormdb.CashierShift, error) {
	if !isValidUUID(userID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID", nil)
	}

	shift, err := s.repos.Cashier.GetCurrentShift(ctx, userID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "No open shift found", err)
	}

	return shift, nil
}

// GetShift gets a shift by ID
func (s *Service) GetShift(ctx context.Context, id string) (*gormdb.CashierShift, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid shift ID", nil)
	}

	shift, err := s.repos.Cashier.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Shift not found", err)
	}

	return shift, nil
}

// ListShifts lists shifts with filters
func (s *Service) ListShifts(ctx context.Context, userID, storeID, status string, page, perPage int32) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	shifts, totalCount, err := s.repos.Cashier.List(ctx, repository.CashierListParams{
		UserID:  userID,
		StoreID: storeID,
		Status:  status,
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list shifts", err)
	}

	return &ListResult{
		Shifts: shifts,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// GetShiftSummary gets shift summary with sales and cash info
func (s *Service) GetShiftSummary(ctx context.Context, shiftID string) (*ShiftDetail, error) {
	if !isValidUUID(shiftID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid shift ID", nil)
	}

	shift, err := s.repos.Cashier.GetByID(ctx, shiftID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Shift not found", err)
	}

	detail := &ShiftDetail{
		Shift: *shift,
	}

	// Load user name
	var user gormdb.User
	if err := s.repos.DB().WithContext(ctx).Where("id = ?", shift.UserID).First(&user).Error; err == nil {
		detail.UserName = user.FullName
	}

	// Load store name
	var store gormdb.Store
	if err := s.repos.DB().WithContext(ctx).Where("id = ?", shift.StoreID).First(&store).Error; err == nil {
		detail.StoreName = store.Name
	}

	// Load sales for this shift
	var sales []gormdb.Sale
	if err := s.repos.DB().WithContext(ctx).
		Where("cashier_id = ? AND created_at >= ?", shift.UserID, shift.OpenedAt).
		Order("created_at DESC").
		Find(&sales).Error; err == nil {
		detail.TotalTransactions = int32(len(sales))
		var totalAmount float64
		for _, sale := range sales {
			totalAmount += sale.GrandTotal
		}
		detail.TotalSalesAmount = totalAmount

		// Convert sales to shift sales
		for _, sale := range sales {
			// Get payment method
			payments, _ := s.repos.Payment.GetBySaleID(ctx, sale.ID.String())
			paymentMethod := ""
			if len(payments) > 0 {
				paymentMethod = payments[0].PaymentMethod
			}

			detail.Sales = append(detail.Sales, ShiftSale{
				ID:            sale.ID.String(),
				InvoiceNumber: sale.InvoiceNumber,
				GrandTotal:    sale.GrandTotal,
				PaymentMethod: paymentMethod,
				CreatedAt:     sale.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	return detail, nil
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// Ensure fmt is used
var _ = fmt.Sprintf
