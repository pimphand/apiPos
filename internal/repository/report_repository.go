package repository

import (
	"context"

	db "apiPos/internal/database/sqlc"
)

// ReportRepositoryImpl implements ReportRepository.
type ReportRepositoryImpl struct {
	queries *db.Queries
}

// NewReportRepository creates a new ReportRepositoryImpl.
func NewReportRepository(queries *db.Queries) ReportRepository {
	return &ReportRepositoryImpl{queries: queries}
}

func (r *ReportRepositoryImpl) GetDailySalesReport(ctx context.Context, arg db.GetDailySalesReportParams) ([]db.GetDailySalesReportRow, error) {
	return r.queries.GetDailySalesReport(ctx, arg)
}

func (r *ReportRepositoryImpl) GetSalesByDateRange(ctx context.Context, arg db.GetSalesByDateRangeParams) ([]db.Sale, error) {
	return r.queries.GetSalesByDateRange(ctx, arg)
}

func (r *ReportRepositoryImpl) GetTopSellingProducts(ctx context.Context, arg db.GetTopSellingProductsParams) ([]db.GetTopSellingProductsRow, error) {
	return r.queries.GetTopSellingProducts(ctx, arg)
}

func (r *ReportRepositoryImpl) GetInventorySummaryReport(ctx context.Context) ([]db.GetInventorySummaryReportRow, error) {
	return r.queries.GetInventorySummaryReport(ctx)
}

func (r *ReportRepositoryImpl) GetPaymentMethodSummary(ctx context.Context, arg db.GetPaymentMethodSummaryParams) ([]db.GetPaymentMethodSummaryRow, error) {
	return r.queries.GetPaymentMethodSummary(ctx, arg)
}
