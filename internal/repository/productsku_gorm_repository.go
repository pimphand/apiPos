package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// ProductSKURepository defines the interface for product SKU data operations
type ProductSKURepository interface {
	Create(ctx context.Context, productSKU *gormdb.ProductSKU) error
	GetByID(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error)
	GetByBarcode(ctx context.Context, barcode string) (*gormdb.ProductSKU, error)
	GetBySKU(ctx context.Context, sku string) (*gormdb.ProductSKU, error)
	List(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	ListAll(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	Update(ctx context.Context, productSKU *gormdb.ProductSKU) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByProductIDs(ctx context.Context, productIDs []uuid.UUID) (map[string]int32, error)
}

type productSKURepository struct {
	*GormBase
}

// NewProductSKURepository creates a new ProductSKU repository
func NewProductSKURepository(db *gorm.DB) ProductSKURepository {
	return &productSKURepository{GormBase: NewGormBase(db)}
}

func (r *productSKURepository) Create(ctx context.Context, productSKU *gormdb.ProductSKU) error {
	productSKU.ID = uuid.New()
	productSKU.CreatedAt = time.Now()
	productSKU.UpdatedAt = time.Now()
	return r.WithContext(ctx).Create(productSKU).Error
}

func (r *productSKURepository) GetByID(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
	var productSKU gormdb.ProductSKU
	err := r.WithContext(ctx).Where("id = ?", id).First(&productSKU).Error
	if err != nil {
		return nil, err
	}
	return &productSKU, nil
}

func (r *productSKURepository) GetByBarcode(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
	var productSKU gormdb.ProductSKU
	err := r.WithContext(ctx).Where("barcode = ? AND is_active = ?", barcode, true).First(&productSKU).Error
	if err != nil {
		return nil, err
	}
	return &productSKU, nil
}

func (r *productSKURepository) GetBySKU(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
	var productSKU gormdb.ProductSKU
	err := r.WithContext(ctx).Where("sku = ? AND is_active = ?", sku, true).First(&productSKU).Error
	if err != nil {
		return nil, err
	}
	return &productSKU, nil
}

func (r *productSKURepository) List(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	var productSKUs []gormdb.ProductSKU
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.ProductSKU{})

	if productID != uuid.Nil {
		query = query.Where("product_id = ?", productID)
	}

	if search != "" {
		query = query.Where("(sku ILIKE ? OR variant ILIKE ? OR barcode ILIKE ?)", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if variant != "" {
		query = query.Where("variant = ?", variant)
	}

	if color != "" {
		query = query.Where("color = ?", color)
	}

	if size != "" {
		query = query.Where("size = ?", size)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	err = query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&productSKUs).Error
	if err != nil {
		return nil, 0, err
	}

	return productSKUs, totalCount, nil
}

func (r *productSKURepository) ListAll(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	var productSKUs []gormdb.ProductSKU
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.ProductSKU{})

	if productID != uuid.Nil {
		query = query.Where("product_id = ?", productID)
	}

	if search != "" {
		query = query.Where("(sku ILIKE ? OR variant ILIKE ? OR barcode ILIKE ?)", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if variant != "" {
		query = query.Where("variant = ?", variant)
	}

	if color != "" {
		query = query.Where("color = ?", color)
	}

	if size != "" {
		query = query.Where("size = ?", size)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	err = query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&productSKUs).Error
	if err != nil {
		return nil, 0, err
	}

	return productSKUs, totalCount, nil
}

func (r *productSKURepository) Update(ctx context.Context, productSKU *gormdb.ProductSKU) error {
	productSKU.UpdatedAt = time.Now()
	return r.WithContext(ctx).Save(productSKU).Error
}

func (r *productSKURepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.ProductSKU{}).Error
}

// CountByProductIDs counts SKUs for multiple product IDs in a single query
func (r *productSKURepository) CountByProductIDs(ctx context.Context, productIDs []uuid.UUID) (map[string]int32, error) {
	if len(productIDs) == 0 {
		return make(map[string]int32), nil
	}

	type countResult struct {
		ProductID uuid.UUID
		Count     int32
	}

	var results []countResult
	err := r.WithContext(ctx).Model(&gormdb.ProductSKU{}).
		Select("product_id, COUNT(*) as count").
		Where("product_id IN ?", productIDs).
		Group("product_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	countMap := make(map[string]int32, len(productIDs))
	for _, r := range results {
		countMap[r.ProductID.String()] = r.Count
	}
	return countMap, nil
}
