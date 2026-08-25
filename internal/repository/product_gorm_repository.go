package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// ProductRepository defines the interface for product operations
type ProductRepository interface {
	Create(ctx context.Context, product *gormdb.Product) error
	GetByID(ctx context.Context, id string) (*gormdb.Product, error)
	GetBySKU(ctx context.Context, sku string) (*gormdb.Product, error)
	GetByBarcode(ctx context.Context, barcode string) (*gormdb.Product, error)
	List(ctx context.Context, params ProductListParams) ([]gormdb.Product, int64, error)
	SearchByName(ctx context.Context, params ProductSearchParams) ([]gormdb.Product, int64, error)
	Update(ctx context.Context, product *gormdb.Product) error
	UpdateStock(ctx context.Context, id string, stock int32) error
	DecreaseStock(ctx context.Context, id string, quantity int32) error
	IncreaseStock(ctx context.Context, id string, quantity int32) error
	Activate(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	GetLowStock(ctx context.Context, params PaginationParams) ([]gormdb.Product, int64, error)
	GetStockForUpdate(ctx context.Context, id string) (*gormdb.Product, error)
}

// ProductListParams holds parameters for listing products
type ProductListParams struct {
	IsActive   string // "" = all, "true" = active only, "false" = inactive only
	CategoryID string
	Page       int32
	PerPage    int32
}

// ProductSearchParams holds parameters for searching products
type ProductSearchParams struct {
	Name     string
	IsActive string
	Page     int32
	PerPage  int32
}

// ProductRepo implements ProductRepository using GORM
type ProductRepo struct {
	*GormBase
}

// NewProductRepository creates a new ProductRepo
func NewProductRepository(db *gorm.DB) ProductRepository {
	return &ProductRepo{GormBase: NewGormBase(db)}
}

// Create creates a new product
func (r *ProductRepo) Create(ctx context.Context, product *gormdb.Product) error {
	return r.WithContext(ctx).Create(product).Error
}

// GetByID gets a product by ID
func (r *ProductRepo) GetByID(ctx context.Context, id string) (*gormdb.Product, error) {
	var product gormdb.Product
	err := r.WithContext(ctx).Where("id = ?", id).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetBySKU gets a product by SKU
func (r *ProductRepo) GetBySKU(ctx context.Context, sku string) (*gormdb.Product, error) {
	var product gormdb.Product
	err := r.WithContext(ctx).Where("sku = ?", sku).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetByBarcode gets a product by barcode
func (r *ProductRepo) GetByBarcode(ctx context.Context, barcode string) (*gormdb.Product, error) {
	var product gormdb.Product
	err := r.WithContext(ctx).Where("barcode = ? AND barcode != ''", barcode).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// List lists products with filters and pagination
func (r *ProductRepo) List(ctx context.Context, params ProductListParams) ([]gormdb.Product, int64, error) {
	var products []gormdb.Product
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Product{})

	// Apply filters
	if params.IsActive == "true" {
		query = query.Where("is_active = ?", true)
	} else if params.IsActive == "false" {
		query = query.Where("is_active = ?", false)
	}

	if params.CategoryID != "" {
		query = query.Where("category_id = ?", params.CategoryID)
	}

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&products).Error

	return products, totalCount, err
}

// SearchByName searches products by name
func (r *ProductRepo) SearchByName(ctx context.Context, params ProductSearchParams) ([]gormdb.Product, int64, error) {
	var products []gormdb.Product
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Product{})

	// Apply name filter
	if params.Name != "" {
		query = query.Where("name ILIKE ?", "%"+params.Name+"%")
	}

	// Apply active filter
	if params.IsActive == "true" {
		query = query.Where("is_active = ?", true)
	} else if params.IsActive == "false" {
		query = query.Where("is_active = ?", false)
	}

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("name ASC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&products).Error

	return products, totalCount, err
}

// Update updates a product
func (r *ProductRepo) Update(ctx context.Context, product *gormdb.Product) error {
	return r.WithContext(ctx).Save(product).Error
}

// UpdateStock updates a product's stock
func (r *ProductRepo) UpdateStock(ctx context.Context, id string, stock int32) error {
	return r.WithContext(ctx).
		Model(&gormdb.Product{}).
		Where("id = ?", id).
		Update("stock", stock).Error
}

// DecreaseStock decreases a product's stock
func (r *ProductRepo) DecreaseStock(ctx context.Context, id string, quantity int32) error {
	result := r.WithContext(ctx).
		Model(&gormdb.Product{}).
		Where("id = ? AND stock >= ?", id, quantity).
		Update("stock", gorm.Expr("stock - ?", quantity))
	return result.Error
}

// IncreaseStock increases a product's stock
func (r *ProductRepo) IncreaseStock(ctx context.Context, id string, quantity int32) error {
	return r.WithContext(ctx).
		Model(&gormdb.Product{}).
		Where("id = ?", id).
		Update("stock", gorm.Expr("stock + ?", quantity)).Error
}

// Activate activates a product
func (r *ProductRepo) Activate(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.Product{}).
		Where("id = ?", id).
		Update("is_active", true).Error
}

// Deactivate soft-deletes a product
func (r *ProductRepo) Deactivate(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.Product{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// Delete deletes a product
func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Product{}).Error
}

// GetLowStock gets products with low stock
func (r *ProductRepo) GetLowStock(ctx context.Context, params PaginationParams) ([]gormdb.Product, int64, error) {
	var products []gormdb.Product
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Product{}).
		Where("stock <= minimum_stock AND is_active = ?", true)

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	normalized := params.normalize()
	err := query.Order("stock ASC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&products).Error

	return products, totalCount, err
}

// GetStockForUpdate gets a product's stock with row locking
func (r *ProductRepo) GetStockForUpdate(ctx context.Context, id string) (*gormdb.Product, error) {
	var product gormdb.Product
	err := r.WithContext(ctx).
		Where("id = ?", id).
		Clauses(gorm.Expr("FOR UPDATE")).
		Select("id", "stock").
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}
