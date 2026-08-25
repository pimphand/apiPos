package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "apiPos/internal/database/sqlc"
)

// ProductRepositoryImpl implements ProductRepository.
type ProductRepositoryImpl struct {
	queries *db.Queries
}

// NewProductRepository creates a new ProductRepositoryImpl.
func NewProductRepository(queries *db.Queries) ProductRepository {
	return &ProductRepositoryImpl{queries: queries}
}

func (r *ProductRepositoryImpl) CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error) {
	return r.queries.CreateProduct(ctx, arg)
}

func (r *ProductRepositoryImpl) GetProductByID(ctx context.Context, id int64) (db.Product, error) {
	return r.queries.GetProductByID(ctx, id)
}

func (r *ProductRepositoryImpl) GetProductBySKU(ctx context.Context, sku string) (db.Product, error) {
	return r.queries.GetProductBySKU(ctx, sku)
}

func (r *ProductRepositoryImpl) GetProductByBarcode(ctx context.Context, barcode pgtype.Text) (db.Product, error) {
	return r.queries.GetProductByBarcode(ctx, barcode)
}

func (r *ProductRepositoryImpl) ListProducts(ctx context.Context, arg db.ListProductsParams) ([]db.Product, error) {
	return r.queries.ListProducts(ctx, arg)
}

func (r *ProductRepositoryImpl) SearchProductsByName(ctx context.Context, arg db.SearchProductsByNameParams) ([]db.Product, error) {
	return r.queries.SearchProductsByName(ctx, arg)
}

func (r *ProductRepositoryImpl) UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.Product, error) {
	return r.queries.UpdateProduct(ctx, arg)
}

func (r *ProductRepositoryImpl) UpdateProductStock(ctx context.Context, arg db.UpdateProductStockParams) error {
	return r.queries.UpdateProductStock(ctx, arg)
}

func (r *ProductRepositoryImpl) DecreaseProductStock(ctx context.Context, arg db.DecreaseProductStockParams) error {
	return r.queries.DecreaseProductStock(ctx, arg)
}

func (r *ProductRepositoryImpl) IncreaseProductStock(ctx context.Context, arg db.IncreaseProductStockParams) error {
	return r.queries.IncreaseProductStock(ctx, arg)
}

func (r *ProductRepositoryImpl) ActivateProduct(ctx context.Context, id int64) error {
	return r.queries.ActivateProduct(ctx, id)
}

func (r *ProductRepositoryImpl) DeactivateProduct(ctx context.Context, id int64) error {
	return r.queries.DeactivateProduct(ctx, id)
}

func (r *ProductRepositoryImpl) DeleteProduct(ctx context.Context, id int64) error {
	return r.queries.DeleteProduct(ctx, id)
}

func (r *ProductRepositoryImpl) CountProducts(ctx context.Context) (int64, error) {
	return r.queries.CountProducts(ctx)
}

func (r *ProductRepositoryImpl) GetLowStockProducts(ctx context.Context) ([]db.Product, error) {
	return r.queries.GetLowStockProducts(ctx)
}

func (r *ProductRepositoryImpl) GetProductStockForUpdate(ctx context.Context, id int64) (db.GetProductStockForUpdateRow, error) {
	return r.queries.GetProductStockForUpdate(ctx, id)
}
