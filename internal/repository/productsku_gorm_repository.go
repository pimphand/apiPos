package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// ProductSKURepository defines the interface for product SKU data operations
type ProductSKURepository interface {
	Create(productSKU *gormdb.ProductSKU) error
	GetByID(id uuid.UUID) (*gormdb.ProductSKU, error)
	GetByBarcode(barcode string) (*gormdb.ProductSKU, error)
	GetBySKU(sku string) (*gormdb.ProductSKU, error)
	List(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	ListAll(search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	Update(productSKU *gormdb.ProductSKU) error
	Delete(id uuid.UUID) error
	CountByProductIDs(productIDs []uuid.UUID) (map[string]int32, error)
}

type productSKURepository struct {
	db *gorm.DB
}

// NewProductSKURepository creates a new ProductSKU repository
func NewProductSKURepository(db *gorm.DB) ProductSKURepository {
	return &productSKURepository{db: db}
}

func (r *productSKURepository) Create(productSKU *gormdb.ProductSKU) error {
	productSKU.ID = uuid.New()
	productSKU.CreatedAt = time.Now()
	productSKU.UpdatedAt = time.Now()
	return r.db.Create(productSKU).Error
}

func (r *productSKURepository) GetByID(id uuid.UUID) (*gormdb.ProductSKU, error) {
	var productSKU gormdb.ProductSKU
	err := r.db.Where("id = ?", id).First(&productSKU).Error
	if err != nil {
		return nil, err
	}
	return &productSKU, nil
}

func (r *productSKURepository) GetByBarcode(barcode string) (*gormdb.ProductSKU, error) {
	var productSKU gormdb.ProductSKU
	err := r.db.Where("barcode = ? AND is_active = ?", barcode, true).First(&productSKU).Error
	if err != nil {
		return nil, err
	}
	return &productSKU, nil
}

func (r *productSKURepository) GetBySKU(sku string) (*gormdb.ProductSKU, error) {
	var productSKU gormdb.ProductSKU
	err := r.db.Where("sku = ? AND is_active = ?", sku, true).First(&productSKU).Error
	if err != nil {
		return nil, err
	}
	return &productSKU, nil
}

func (r *productSKURepository) List(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	var productSKUs []gormdb.ProductSKU
	var totalCount int64

	query := r.db.Model(&gormdb.ProductSKU{})

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

func (r *productSKURepository) ListAll(search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	var productSKUs []gormdb.ProductSKU
	var totalCount int64

	query := r.db.Model(&gormdb.ProductSKU{})

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

func (r *productSKURepository) Update(productSKU *gormdb.ProductSKU) error {
	productSKU.UpdatedAt = time.Now()
	return r.db.Save(productSKU).Error
}

func (r *productSKURepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&gormdb.ProductSKU{}).Error
}

// CountByProductIDs counts SKUs for multiple product IDs in a single query
func (r *productSKURepository) CountByProductIDs(productIDs []uuid.UUID) (map[string]int32, error) {
	if len(productIDs) == 0 {
		return make(map[string]int32), nil
	}

	type countResult struct {
		ProductID uuid.UUID
		Count     int32
	}

	var results []countResult
	err := r.db.Model(&gormdb.ProductSKU{}).
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
