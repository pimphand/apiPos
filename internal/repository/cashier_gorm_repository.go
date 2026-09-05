package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// CashierRepository defines the interface for cashier shift operations
type CashierRepository interface {
	OpenShift(ctx context.Context, shift *gormdb.CashierShift) error
	CloseShift(ctx context.Context, params CloseShiftParams) (bool, error)
	GetCurrentShift(ctx context.Context, userID string) (*gormdb.CashierShift, error)
	GetByID(ctx context.Context, id string) (*gormdb.CashierShift, error)
	List(ctx context.Context, params CashierListParams) ([]gormdb.CashierShift, int64, error)
	UpdateCashSales(ctx context.Context, id string, amount float64) error
	UpdateCashRefund(ctx context.Context, id string, amount float64) error
	CountOpenShifts(ctx context.Context) (int64, error)
	ListShiftItems(ctx context.Context, params ShiftItemListParams) ([]gormdb.ShiftItem, int64, error)
}

// ShiftItemListParams holds filters for listing the shift_items ledger.
type ShiftItemListParams struct {
	ShiftID   string
	Direction string
	Page      int32
	PerPage   int32
}

// CloseShiftParams carries everything needed to atomically close a shift:
// the user-supplied drawer counts plus the server-computed reconciliation and
// frozen snapshots.
type CloseShiftParams struct {
	ID                string
	ClosingCash       float64
	ActualCash        float64
	ExpectedCash      float64
	TotalTransactions int32
	TotalSalesAmount  float64
	TotalRevenue      float64
	RevenueCash       float64
	TotalItemsOut     float64
	TotalItemsIn      float64
}

// CashierListParams holds parameters for listing cashier shifts
type CashierListParams struct {
	UserID  string
	StoreID string
	Status  string
	Page    int32
	PerPage int32
}

// CashierRepo implements CashierRepository using GORM
type CashierRepo struct {
	*GormBase
}

// NewCashierRepository creates a new CashierRepo
func NewCashierRepository(db *gorm.DB) CashierRepository {
	return &CashierRepo{GormBase: NewGormBase(db)}
}

// OpenShift opens a new cashier shift
func (r *CashierRepo) OpenShift(ctx context.Context, shift *gormdb.CashierShift) error {
	return r.WithContext(ctx).Create(shift).Error
}

// CloseShift atomically closes a shift only while it is still OPEN. It returns
// (false, nil) when the shift was already closed (idempotent no-op) so the
// service can treat a double-close as a success instead of an error.
func (r *CashierRepo) CloseShift(ctx context.Context, params CloseShiftParams) (bool, error) {
	difference := params.ActualCash - params.ExpectedCash

	res := r.WithContext(ctx).
		Model(&gormdb.CashierShift{}).
		Where("id = ? AND status = ?", params.ID, "OPEN").
		Updates(map[string]interface{}{
			"closing_cash":       params.ClosingCash,
			"actual_cash":        params.ActualCash,
			"cash_sales":         params.RevenueCash,
			"expected_cash":      params.ExpectedCash,
			"cash_difference":    difference,
			"status":             "CLOSED",
			"closed_at":          gorm.Expr("NOW()"),
			"total_transactions": params.TotalTransactions,
			"total_sales_amount": params.TotalSalesAmount,
			"total_revenue":      params.TotalRevenue,
			"revenue_cash":       params.RevenueCash,
			"total_items_out":    params.TotalItemsOut,
			"total_items_in":     params.TotalItemsIn,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// GetCurrentShift gets the current open shift for a user
func (r *CashierRepo) GetCurrentShift(ctx context.Context, userID string) (*gormdb.CashierShift, error) {
	var shift gormdb.CashierShift
	err := r.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "OPEN").
		Order("opened_at DESC").
		First(&shift).Error
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

// GetByID gets a shift by ID
func (r *CashierRepo) GetByID(ctx context.Context, id string) (*gormdb.CashierShift, error) {
	var shift gormdb.CashierShift
	err := r.WithContext(ctx).Where("id = ?", id).First(&shift).Error
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

// List lists cashier shifts with filters
func (r *CashierRepo) List(ctx context.Context, params CashierListParams) ([]gormdb.CashierShift, int64, error) {
	var shifts []gormdb.CashierShift
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.CashierShift{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.StoreID != "" {
		query = query.Where("store_id = ?", params.StoreID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("opened_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&shifts).Error

	return shifts, totalCount, err
}

// UpdateCashSales updates cash sales for a shift
func (r *CashierRepo) UpdateCashSales(ctx context.Context, id string, amount float64) error {
	return r.WithContext(ctx).
		Model(&gormdb.CashierShift{}).
		Where("id = ?", id).
		Update("cash_sales", gorm.Expr("cash_sales + ?", amount)).Error
}

// UpdateCashRefund updates cash refund for a shift
func (r *CashierRepo) UpdateCashRefund(ctx context.Context, id string, amount float64) error {
	return r.WithContext(ctx).
		Model(&gormdb.CashierShift{}).
		Where("id = ?", id).
		Update("cash_refund", gorm.Expr("cash_refund + ?", amount)).Error
}

// CountOpenShifts counts open shifts
func (r *CashierRepo) CountOpenShifts(ctx context.Context) (int64, error) {
	var count int64
	err := r.WithContext(ctx).
		Model(&gormdb.CashierShift{}).
		Where("status = ?", "OPEN").
		Count(&count).Error
	return count, err
}

// ListShiftItems lists shift_items rows for a shift with pagination and an
// optional direction filter (OUT / IN).
func (r *CashierRepo) ListShiftItems(ctx context.Context, params ShiftItemListParams) ([]gormdb.ShiftItem, int64, error) {
	var items []gormdb.ShiftItem
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.ShiftItem{}).
		Where("shift_id = ?", params.ShiftID)
	if params.Direction != "" {
		query = query.Where("direction = ?", params.Direction)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&items).Error

	return items, totalCount, err
}
