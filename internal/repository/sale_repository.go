package repository

import (
	"context"

	db "apiPos/internal/database/sqlc"
)

// SaleRepositoryImpl implements SaleRepository.
type SaleRepositoryImpl struct {
	queries *db.Queries
}

// NewSaleRepository creates a new SaleRepositoryImpl.
func NewSaleRepository(queries *db.Queries) SaleRepository {
	return &SaleRepositoryImpl{queries: queries}
}

func (r *SaleRepositoryImpl) CreateSale(ctx context.Context, arg db.CreateSaleParams) (db.Sale, error) {
	return r.queries.CreateSale(ctx, arg)
}

func (r *SaleRepositoryImpl) GetSaleByID(ctx context.Context, id int64) (db.Sale, error) {
	return r.queries.GetSaleByID(ctx, id)
}

func (r *SaleRepositoryImpl) GetSaleByInvoiceNumber(ctx context.Context, invoiceNumber string) (db.Sale, error) {
	return r.queries.GetSaleByInvoiceNumber(ctx, invoiceNumber)
}

func (r *SaleRepositoryImpl) ListSales(ctx context.Context, arg db.ListSalesParams) ([]db.Sale, error) {
	return r.queries.ListSales(ctx, arg)
}

func (r *SaleRepositoryImpl) UpdateSaleStatus(ctx context.Context, arg db.UpdateSaleStatusParams) error {
	return r.queries.UpdateSaleStatus(ctx, arg)
}

func (r *SaleRepositoryImpl) UpdateSale(ctx context.Context, arg db.UpdateSaleParams) (db.Sale, error) {
	return r.queries.UpdateSale(ctx, arg)
}

func (r *SaleRepositoryImpl) DeleteSale(ctx context.Context, id int64) error {
	return r.queries.DeleteSale(ctx, id)
}

func (r *SaleRepositoryImpl) CountSales(ctx context.Context) (int64, error) {
	return r.queries.CountSales(ctx)
}

func (r *SaleRepositoryImpl) CreateSaleItem(ctx context.Context, arg db.CreateSaleItemParams) (db.SaleItem, error) {
	return r.queries.CreateSaleItem(ctx, arg)
}

func (r *SaleRepositoryImpl) GetSaleItemsBySaleID(ctx context.Context, saleID int64) ([]db.SaleItem, error) {
	return r.queries.GetSaleItemsBySaleID(ctx, saleID)
}

func (r *SaleRepositoryImpl) DeleteSaleItemsBySaleID(ctx context.Context, saleID int64) error {
	return r.queries.DeleteSaleItemsBySaleID(ctx, saleID)
}
