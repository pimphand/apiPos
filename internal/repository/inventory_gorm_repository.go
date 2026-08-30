package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// InventoryRepository defines the interface for inventory operations
type InventoryRepository interface {
	CreateTransaction(ctx context.Context, txn *gormdb.InventoryTransaction) error
	GetTransactionByID(ctx context.Context, id string) (*gormdb.InventoryTransaction, error)
	ListTransactions(ctx context.Context, params InventoryListParams) ([]gormdb.InventoryTransaction, int64, error)
	ListTransactionsByProduct(ctx context.Context, productID string, params PaginationParams) ([]gormdb.InventoryTransaction, int64, error)
	// AdjustProductStock atomically adjusts a product's stock under a row lock
	// (SELECT ... FOR UPDATE) and returns the before/after stock. A negative
	// delta (decrease) is rejected when it would drive stock below zero.
	AdjustProductStock(ctx context.Context, productID uuid.UUID, delta int32) (beforeStock, afterStock int32, err error)
}

// ErrInsufficientStockAtRepo is returned by AdjustProductStock when a decrease
// would push stock below zero.
var ErrInsufficientStockAtRepo = errors.New("insufficient stock")

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

// AdjustProductStock atomically adjusts a product's stock under a row lock.
// The lock, read and update happen on the same connection so concurrent calls
// are serialized by PostgreSQL's SELECT ... FOR UPDATE.
func (r *InventoryRepo) AdjustProductStock(ctx context.Context, productID uuid.UUID, delta int32) (int32, int32, error) {
	var product gormdb.Product
	if err := r.WithContext(ctx).Raw(
		"SELECT * FROM products WHERE id = ? FOR UPDATE", productID,
	).Scan(&product).Error; err != nil {
		return 0, 0, err
	}

	before := product.Stock
	after := before + delta
	if after < 0 {
		return before, before, ErrInsufficientStockAtRepo
	}

	if err := r.WithContext(ctx).Model(&gormdb.Product{}).
		Where("id = ?", productID).
		Update("stock", after).Error; err != nil {
		return 0, 0, err
	}

	return before, after, nil
}
