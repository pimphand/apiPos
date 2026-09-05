package cashier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
	StoreID     string
	OpeningCash float64
	UserID      string
}

// CloseShiftRequest represents the request to close a shift. UserID and
// StoreID come from the JWT claims so a shift can only be closed by its owner
// within the same store.
type CloseShiftRequest struct {
	ShiftID     string
	ClosingCash float64
	ActualCash  float64
	UserID      string
	StoreID     string
}

// ShiftDetail holds shift with summary info
type ShiftDetail struct {
	Shift             gormdb.CashierShift
	UserName          string
	StoreName         string
	TotalTransactions int32
	TotalSalesAmount  float64
	RevenueByMethod   map[string]float64
	CashSales         float64
	CashRefund        float64
	ExpectedCash      float64
	ItemsOutTotal     float64
	ItemsInTotal      float64
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

// ShiftItemsResult holds the result of listing shift item ledger rows
type ShiftItemsResult struct {
	Items      []gormdb.ShiftItem
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

	// Fast path: user already has an open shift.
	if shift, err := s.repos.Cashier.GetCurrentShift(ctx, req.UserID); err == nil && shift != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "You already have an open shift", nil)
	}

	storeID, _ := uuid.Parse(req.StoreID)
	userID, _ := uuid.Parse(req.UserID)

	// Create shift. The partial unique index
	// (uniq_cashier_shifts_one_open_per_user) makes a racing second insert
	// fail even if the check above races, so a double-open is impossible.
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

// CloseShift closes a cashier shift. It is safe against:
//   - closing another cashier's shift        -> PERMISSION_DENIED
//   - double-close (two parallel requests)   -> idempotent success
//   - reconciliation on wrong numbers        -> computed server-side
//   - ledger vs inventory drift (B11)        -> rejected
//
// The whole reconcile → validate → close section runs inside one DB
// transaction that locks the shift row FOR UPDATE. Sale/refund writers take
// the same lock (openShiftOfActor in the sale service), so they serialize
// with the closer: no sale, OUT or IN row can be committed between the
// reconciliation snapshot and the CLOSED update — a close never misses a
// transaction, and a closed shift can never receive new rows afterwards.
func (s *Service) CloseShift(ctx context.Context, req CloseShiftRequest) (*gormdb.CashierShift, error) {
	if !isValidUUID(req.ShiftID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid shift ID", nil)
	}
	if !isValidUUID(req.UserID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID", nil)
	}

	shift, err := s.repos.Cashier.GetByID(ctx, req.ShiftID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Shift not found", err)
	}

	// Ownership: a shift can only be closed by the cashier who opened it,
	// operating from the same store.
	if shift.UserID.String() != req.UserID {
		return nil, apperror.Wrap(apperror.ErrForbidden, "You can only close your own shift", nil)
	}
	if req.StoreID != "" && isValidUUID(req.StoreID) && shift.StoreID.String() != req.StoreID {
		return nil, apperror.Wrap(apperror.ErrForbidden, "Shift belongs to a different store", nil)
	}

	// Already closed: treat as idempotent success (the caller reloads the
	// closed state), not as an error.
	if shift.Status == ShiftStatusClosed {
		return shift, nil
	}

	var result *gormdb.CashierShift
	txErr := s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the shift row for the whole reconcile → validate → close
		// section (see comment on CloseShift above, B11).
		var locked gormdb.CashierShift
		if err := tx.Raw("SELECT * FROM cashier_shifts WHERE id = ? FOR UPDATE", req.ShiftID).Scan(&locked).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to lock shift", err)
		}
		if locked.ID == uuid.Nil {
			return apperror.Wrap(apperror.ErrShiftNotFound, "Shift not found", nil)
		}

		// Re-check inside the lock: a concurrent closer may have won the race.
		if locked.Status == ShiftStatusClosed {
			result = &locked
			return nil
		}

		// Reconciliation read inside the transaction so the numbers are
		// consistent with the lock point (single source of truth).
		recon, err := s.computeReconciliationWith(ctx, tx, &locked)
		if err != nil {
			return err
		}

		// B11: strict cross-validation before freezing the numbers. Refuse to
		// close when the OUT ledger disagrees with the SALE inventory ledger.
		if err := s.validateShiftIntegrity(ctx, tx, &locked); err != nil {
			return err
		}

		// Atomic close under the row lock: only an OPEN shift can transition.
		difference := req.ActualCash - recon.ExpectedCash
		res := tx.Model(&gormdb.CashierShift{}).
			Where("id = ?", locked.ID.String()).
			Updates(map[string]interface{}{
				"closing_cash":       req.ClosingCash,
				"actual_cash":        req.ActualCash,
				"cash_sales":         recon.RevenueCash,
				"expected_cash":      recon.ExpectedCash,
				"cash_difference":    difference,
				"status":             ShiftStatusClosed,
				"closed_at":          gorm.Expr("NOW()"),
				"total_transactions": recon.TotalTransactions,
				"total_sales_amount": recon.TotalSalesAmount,
				"total_revenue":      recon.TotalRevenue,
				"revenue_cash":       recon.RevenueCash,
				"total_items_out":    recon.TotalItemsOut,
				"total_items_in":     recon.TotalItemsIn,
			})
		if res.Error != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to close shift", res.Error)
		}
		if res.RowsAffected == 0 {
			// The lock serializes closers, but the shift may already be closed
			// from another path; reload and return it as idempotent success.
			var current gormdb.CashierShift
			if err := tx.Where("id = ?", locked.ID.String()).First(&current).Error; err == nil {
				result = &current
			} else {
				result = &locked
			}
			return nil
		}

		var updated gormdb.CashierShift
		if err := tx.Where("id = ?", locked.ID.String()).First(&updated).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to load closed shift", err)
		}
		result = &updated
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	if result == nil {
		// Row vanished between the fast-path read and the transaction.
		return nil, apperror.Wrap(apperror.ErrShiftNotFound, "Shift not found", nil)
	}
	return result, nil
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

// ListShiftItems lists the per-item ledger (shift_items) of a shift with
// optional direction filter and standard pagination.
func (s *Service) ListShiftItems(ctx context.Context, shiftID, direction string, page, perPage int32) (*ShiftItemsResult, error) {
	if !isValidUUID(shiftID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid shift ID", nil)
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	items, total, err := s.repos.Cashier.ListShiftItems(ctx, repository.ShiftItemListParams{
		ShiftID:   shiftID,
		Direction: direction,
		Page:      page,
		PerPage:   perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list shift items", err)
	}

	return &ShiftItemsResult{
		Items: items,
		Pagination: PaginationResult{
			TotalCount: total,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// GetShiftSummary gets shift summary with sales, cash reconciliation and item
// movement info, computed from data linked to the shift via sales.shift_id.
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

	recon, err := s.computeReconciliation(ctx, shift)
	if err != nil {
		return nil, err
	}
	detail.TotalTransactions = recon.TotalTransactions
	detail.TotalSalesAmount = recon.TotalSalesAmount
	detail.RevenueByMethod = recon.RevenueByMethod
	detail.CashSales = recon.RevenueCash
	detail.CashRefund = recon.CashRefund
	detail.ExpectedCash = recon.ExpectedCash
	detail.ItemsOutTotal = recon.TotalItemsOut
	detail.ItemsInTotal = recon.TotalItemsIn
	detail.Sales = recon.Sales

	return detail, nil
}

// reconciliation holds every number needed by the close dialog / snapshot.
type reconciliation struct {
	TotalTransactions int32
	TotalSalesAmount  float64
	TotalRevenue      float64
	RevenueByMethod   map[string]float64
	RevenueCash       float64
	CashRefund        float64
	ExpectedCash      float64
	TotalItemsOut     float64
	TotalItemsIn      float64
	Sales             []ShiftSale
}

// computeReconciliation derives all shift numbers from the shift's linked
// data (sales.shift_id, payments and shift_items) using the service DB. For
// legacy shifts without links it falls back to the bounded opened_at..closed_at
// window so it never picks up sales that happened after the shift was closed.
func (s *Service) computeReconciliation(ctx context.Context, shift *gormdb.CashierShift) (*reconciliation, error) {
	return s.computeReconciliationWith(ctx, s.repos.DB().WithContext(ctx), shift)
}

// computeReconciliationWith is computeReconciliation over an explicit query
// handle (tx during CloseShift) so the numbers and the close update share one
// consistent, locked transaction.
func (s *Service) computeReconciliationWith(ctx context.Context, db *gorm.DB, shift *gormdb.CashierShift) (*reconciliation, error) {
	recon := &reconciliation{
		RevenueByMethod: map[string]float64{},
		CashRefund:      shift.CashRefund,
	}

	endBound := time.Now()
	if shift.ClosedAt != nil {
		endBound = *shift.ClosedAt
	}

	// Sales linked to this shift (or covered by the legacy window when links
	// are absent). Money-in movements: PAID + REFUNDED (refunded sales keep
	// their inflow and the refund is tracked separately as an outflow).
	var sales []gormdb.Sale
	if err := db.
		Where("status IN ?", []string{"PAID", "REFUNDED"}).
		Where(
			`(shift_id = ?) OR
			 (shift_id IS NULL AND cashier_id = ? AND store_id = ? AND created_at >= ? AND created_at <= ?)`,
			shift.ID, shift.UserID, shift.StoreID, shift.OpenedAt, endBound,
		).
		Order("created_at DESC").
		Find(&sales).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to load shift sales", err)
	}

	recon.TotalTransactions = int32(len(sales))

	for _, sale := range sales {
		recon.TotalSalesAmount += sale.GrandTotal
		recon.TotalRevenue += sale.GrandTotal

		// Determine the sale's payment method from its payment rows.
		var payments []gormdb.Payment
		if err := db.Where("sale_id = ?", sale.ID).Find(&payments).Error; err != nil {
			payments = nil
		}
		paymentMethod := ""
		if len(payments) > 0 {
			paymentMethod = payments[0].PaymentMethod
		}

		recon.RevenueByMethod[paymentMethod] += sale.GrandTotal
		if paymentMethod == "CASH" {
			recon.RevenueCash += sale.GrandTotal
		}

		recon.Sales = append(recon.Sales, ShiftSale{
			ID:            sale.ID.String(),
			InvoiceNumber: sale.InvoiceNumber,
			GrandTotal:    sale.GrandTotal,
			PaymentMethod: paymentMethod,
			CreatedAt:     sale.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Goods OUT / IN from the insert-only shift_items ledger.
	var outTotal, inTotal float64
	if err := db.Model(&gormdb.ShiftItem{}).
		Where("shift_id = ? AND direction = ?", shift.ID, "OUT").
		Select("COALESCE(SUM(quantity), 0)").Scan(&outTotal).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to sum items out", err)
	}
	if err := db.Model(&gormdb.ShiftItem{}).
		Where("shift_id = ? AND direction = ?", shift.ID, "IN").
		Select("COALESCE(SUM(quantity), 0)").Scan(&inTotal).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to sum items in", err)
	}
	recon.TotalItemsOut = outTotal
	recon.TotalItemsIn = inTotal

	// Expected drawer cash = opening + cash sales - cash refunds.
	recon.ExpectedCash = shift.OpeningCash + recon.RevenueCash - recon.CashRefund

	return recon, nil
}

// saleMovementRow compares one sale's OUT ledger rows against its inventory
// SALE movements (B11 cross-check).
type saleMovementRow struct {
	Invoice      string
	LedgerOutQty float64
	InventoryQty float64
}

// validateShiftIntegrity runs the B11 strict cross-validation for a shift
// about to close: every sale linked to the shift must have OUT ledger rows
// whose quantities match its SALE inventory movements. Both are written in
// the same DB transaction as the sale, so a mismatch means the data drifted
// and the shift must not be closed on wrong numbers.
func (s *Service) validateShiftIntegrity(ctx context.Context, tx *gorm.DB, shift *gormdb.CashierShift) error {
	rows, err := s.loadSaleMovementRows(ctx, tx, shift)
	if err != nil {
		return err
	}
	issues := validateSaleMovementConsistency(rows)
	if len(issues) > 0 {
		return apperror.Wrap(apperror.ErrFailedPrecondition,
			"Cannot close shift: "+strings.Join(issues, "; "), nil)
	}
	return nil
}

// loadSaleMovementRows loads, per sale linked to the shift, the total OUT
// ledger quantity and the total SALE inventory quantity. All sales are
// included regardless of status: ledger OUT and SALE inventory rows are
// written at sale creation and never removed (refunds add IN/RETURN rows).
func (s *Service) loadSaleMovementRows(ctx context.Context, tx *gorm.DB, shift *gormdb.CashierShift) ([]saleMovementRow, error) {
	var sales []gormdb.Sale
	if err := tx.Where("shift_id = ?", shift.ID).Find(&sales).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to load shift sales for validation", err)
	}

	rows := make([]saleMovementRow, 0, len(sales))
	for _, sale := range sales {
		var ledgerOut float64
		if err := tx.Model(&gormdb.ShiftItem{}).
			Where("sale_id = ? AND direction = ?", sale.ID, "OUT").
			Select("COALESCE(SUM(quantity), 0)").
			Scan(&ledgerOut).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to sum ledger OUT rows", err)
		}

		var inventoryQty float64
		if err := tx.Model(&gormdb.InventoryTransaction{}).
			Where("reference_type = ? AND reference_id = ?", "SALE", sale.ID).
			Select("COALESCE(SUM(quantity), 0)").
			Scan(&inventoryQty).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to sum inventory SALE rows", err)
		}

		rows = append(rows, saleMovementRow{
			Invoice:      sale.InvoiceNumber,
			LedgerOutQty: ledgerOut,
			InventoryQty: inventoryQty,
		})
	}
	return rows, nil
}

// validateSaleMovementConsistency is a pure decision function (unit-tested):
// it returns one human-readable issue per sale whose OUT ledger quantity does
// not equal its SALE inventory quantity.
func validateSaleMovementConsistency(rows []saleMovementRow) []string {
	var issues []string
	for _, r := range rows {
		if r.LedgerOutQty != r.InventoryQty {
			issues = append(issues, fmt.Sprintf(
				"sale %s: ledger OUT %.3f != inventory SALE %.3f",
				r.Invoice, r.LedgerOutQty, r.InventoryQty))
		}
	}
	return issues
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
