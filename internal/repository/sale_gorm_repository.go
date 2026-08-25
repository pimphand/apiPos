package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// SaleRepository defines the interface for sale operations
type SaleRepository interface {
	Create(ctx context.Context, sale *gormdb.Sale) error
	GetByID(ctx context.Context, id string) (*gormdb.Sale, error)
	GetByInvoiceNumber(ctx context.Context, invoiceNumber string) (*gormdb.Sale, error)
	List(ctx context.Context, params SaleListParams) ([]gormdb.Sale, int64, error)
	UpdateStatus(ctx context.Context, id, status string) error
	Update(ctx context.Context, sale *gormdb.Sale) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
	CreateItem(ctx context.Context, item *gormdb.SaleItem) error
	GetItemsBySaleID(ctx context.Context, saleID string) ([]gormdb.SaleItem, error)
	DeleteItemsBySaleID(ctx context.Context, saleID string) error
}

// SaleListParams holds parameters for listing sales
type SaleListParams struct {
	StoreID    string
	CashierID  string
	CustomerID string
	Status     string
	StartDate  string
	EndDate    string
	Page       int32
	PerPage    int32
}

// SaleRepo implements SaleRepository using GORM
type SaleRepo struct {
	*GormBase
}

// NewSaleRepository creates a new SaleRepo
func NewSaleRepository(db *gorm.DB) SaleRepository {
	return &SaleRepo{GormBase: NewGormBase(db)}
}

// Create creates a new sale
func (r *SaleRepo) Create(ctx context.Context, sale *gormdb.Sale) error {
	return r.WithContext(ctx).Create(sale).Error
}

// GetByID gets a sale by ID
func (r *SaleRepo) GetByID(ctx context.Context, id string) (*gormdb.Sale, error) {
	var sale gormdb.Sale
	err := r.WithContext(ctx).Where("id = ?", id).First(&sale).Error
	if err != nil {
		return nil, err
	}
	return &sale, nil
}

// GetByInvoiceNumber gets a sale by invoice number
func (r *SaleRepo) GetByInvoiceNumber(ctx context.Context, invoiceNumber string) (*gormdb.Sale, error) {
	var sale gormdb.Sale
	err := r.WithContext(ctx).Where("invoice_number = ?", invoiceNumber).First(&sale).Error
	if err != nil {
		return nil, err
	}
	return &sale, nil
}

// List lists sales with filters
func (r *SaleRepo) List(ctx context.Context, params SaleListParams) ([]gormdb.Sale, int64, error) {
	var sales []gormdb.Sale
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Sale{})

	if params.StoreID != "" {
		query = query.Where("store_id = ?", params.StoreID)
	}
	if params.CashierID != "" {
		query = query.Where("cashier_id = ?", params.CashierID)
	}
	if params.CustomerID != "" {
		query = query.Where("customer_id = ?", params.CustomerID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.StartDate != "" {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("created_at <= ?", params.EndDate)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&sales).Error

	return sales, totalCount, err
}

// UpdateStatus updates a sale's status
func (r *SaleRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return r.WithContext(ctx).
		Model(&gormdb.Sale{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Update updates a sale
func (r *SaleRepo) Update(ctx context.Context, sale *gormdb.Sale) error {
	return r.WithContext(ctx).Save(sale).Error
}

// Delete deletes a sale
func (r *SaleRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Sale{}).Error
}

// Count counts all sales
func (r *SaleRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.WithContext(ctx).Model(&gormdb.Sale{}).Count(&count).Error
	return count, err
}

// CreateItem creates a new sale item
func (r *SaleRepo) CreateItem(ctx context.Context, item *gormdb.SaleItem) error {
	return r.WithContext(ctx).Create(item).Error
}

// GetItemsBySaleID gets sale items by sale ID
func (r *SaleRepo) GetItemsBySaleID(ctx context.Context, saleID string) ([]gormdb.SaleItem, error) {
	var items []gormdb.SaleItem
	err := r.WithContext(ctx).Where("sale_id = ?", saleID).Find(&items).Error
	return items, err
}

// DeleteItemsBySaleID deletes sale items by sale ID
func (r *SaleRepo) DeleteItemsBySaleID(ctx context.Context, saleID string) error {
	return r.WithContext(ctx).Where("sale_id = ?", saleID).Delete(&gormdb.SaleItem{}).Error
}
