package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// InventoryRepository defines the interface for inventory operations
type InventoryRepository interface {
	CreateTransaction(ctx context.Context, txn *gormdb.InventoryTransaction) error
	GetTransactionByID(ctx context.Context, id string) (*gormdb.InventoryTransaction, error)
	ListTransactions(ctx context.Context, params InventoryListParams) ([]gormdb.InventoryTransaction, int64, error)
	ListTransactionsByProduct(ctx context.Context, productID string, params PaginationParams) ([]gormdb.InventoryTransaction, int64, error)
}

// InventoryListParams holds parameters for listing inventory transactions
type InventoryListParams struct {
	ProductID string
	StoreID   string
	Type      string
	Page      int32
	PerPage   int32
}

// InventoryRepo implements InventoryRepository using GORM
type InventoryRepo struct {
	*GormBase
}

// NewInventoryRepository creates a new InventoryRepo
func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &InventoryRepo{GormBase: NewGormBase(db)}
}

// CreateTransaction creates a new inventory transaction
func (r *InventoryRepo) CreateTransaction(ctx context.Context, txn *gormdb.InventoryTransaction) error {
	return r.WithContext(ctx).Create(txn).Error
}

// GetTransactionByID gets an inventory transaction by ID
func (r *InventoryRepo) GetTransactionByID(ctx context.Context, id string) (*gormdb.InventoryTransaction, error) {
	var txn gormdb.InventoryTransaction
	err := r.WithContext(ctx).Where("id = ?", id).First(&txn).Error
	if err != nil {
		return nil, err
	}
	return &txn, nil
}

// ListTransactions lists inventory transactions with filters
func (r *InventoryRepo) ListTransactions(ctx context.Context, params InventoryListParams) ([]gormdb.InventoryTransaction, int64, error) {
	var txns []gormdb.InventoryTransaction
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.InventoryTransaction{})

	if params.ProductID != "" {
		query = query.Where("product_id = ?", params.ProductID)
	}
	if params.StoreID != "" {
		query = query.Where("store_id = ?", params.StoreID)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&txns).Error

	return txns, totalCount, err
}

// ListTransactionsByProduct lists inventory transactions for a specific product
func (r *InventoryRepo) ListTransactionsByProduct(ctx context.Context, productID string, params PaginationParams) ([]gormdb.InventoryTransaction, int64, error) {
	var txns []gormdb.InventoryTransaction
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.InventoryTransaction{}).
		Where("product_id = ?", productID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := params.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&txns).Error

	return txns, totalCount, err
}
