package repository

import (
	"context"

	db "apiPos/internal/database/sqlc"
)

// CashierRepositoryImpl implements CashierRepository.
type CashierRepositoryImpl struct {
	queries *db.Queries
}

// NewCashierRepository creates a new CashierRepositoryImpl.
func NewCashierRepository(queries *db.Queries) CashierRepository {
	return &CashierRepositoryImpl{queries: queries}
}

func (r *CashierRepositoryImpl) OpenCashierShift(ctx context.Context, arg db.OpenCashierShiftParams) (db.CashierShift, error) {
	return r.queries.OpenCashierShift(ctx, arg)
}

func (r *CashierRepositoryImpl) CloseCashierShift(ctx context.Context, arg db.CloseCashierShiftParams) (db.CashierShift, error) {
	return r.queries.CloseCashierShift(ctx, arg)
}

func (r *CashierRepositoryImpl) GetCurrentShift(ctx context.Context, userID int64) (db.CashierShift, error) {
	return r.queries.GetCurrentShift(ctx, userID)
}

func (r *CashierRepositoryImpl) GetShiftByID(ctx context.Context, id int64) (db.CashierShift, error) {
	return r.queries.GetShiftByID(ctx, id)
}

func (r *CashierRepositoryImpl) ListShifts(ctx context.Context, arg db.ListShiftsParams) ([]db.CashierShift, error) {
	return r.queries.ListShifts(ctx, arg)
}

func (r *CashierRepositoryImpl) UpdateShiftCashSales(ctx context.Context, arg db.UpdateShiftCashSalesParams) error {
	return r.queries.UpdateShiftCashSales(ctx, arg)
}

func (r *CashierRepositoryImpl) UpdateShiftCashRefund(ctx context.Context, arg db.UpdateShiftCashRefundParams) error {
	return r.queries.UpdateShiftCashRefund(ctx, arg)
}

func (r *CashierRepositoryImpl) CountOpenShifts(ctx context.Context) (int64, error) {
	return r.queries.CountOpenShifts(ctx)
}
