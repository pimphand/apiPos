package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// PaymentRepository defines the interface for payment operations
type PaymentRepository interface {
	Create(ctx context.Context, payment *gormdb.Payment) error
	GetByID(ctx context.Context, id string) (*gormdb.Payment, error)
	GetBySaleID(ctx context.Context, saleID string) ([]gormdb.Payment, error)
	List(ctx context.Context, params PaymentListParams) ([]gormdb.Payment, int64, error)
	Delete(ctx context.Context, id string) error
	GetTotalBySaleID(ctx context.Context, saleID string) (float64, error)
}

// PaymentListParams holds parameters for listing payments
type PaymentListParams struct {
	SaleID        string
	PaymentMethod string
	Page          int32
	PerPage       int32
}

// PaymentRepo implements PaymentRepository using GORM
type PaymentRepo struct {
	*GormBase
}

// NewPaymentRepository creates a new PaymentRepo
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &PaymentRepo{GormBase: NewGormBase(db)}
}

// Create creates a new payment
func (r *PaymentRepo) Create(ctx context.Context, payment *gormdb.Payment) error {
	return r.WithContext(ctx).Create(payment).Error
}

// GetByID gets a payment by ID
func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*gormdb.Payment, error) {
	var payment gormdb.Payment
	err := r.WithContext(ctx).Where("id = ?", id).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetBySaleID gets payments by sale ID
func (r *PaymentRepo) GetBySaleID(ctx context.Context, saleID string) ([]gormdb.Payment, error) {
	var payments []gormdb.Payment
	err := r.WithContext(ctx).Where("sale_id = ?", saleID).Order("paid_at ASC").Find(&payments).Error
	return payments, err
}

// List lists payments with filters
func (r *PaymentRepo) List(ctx context.Context, params PaymentListParams) ([]gormdb.Payment, int64, error) {
	var payments []gormdb.Payment
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Payment{})

	if params.SaleID != "" {
		query = query.Where("sale_id = ?", params.SaleID)
	}
	if params.PaymentMethod != "" {
		query = query.Where("payment_method = ?", params.PaymentMethod)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("paid_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&payments).Error

	return payments, totalCount, err
}

// Delete deletes a payment
func (r *PaymentRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Payment{}).Error
}

// GetTotalBySaleID gets total payments for a sale
func (r *PaymentRepo) GetTotalBySaleID(ctx context.Context, saleID string) (float64, error) {
	var total float64
	err := r.WithContext(ctx).
		Model(&gormdb.Payment{}).
		Where("sale_id = ?", saleID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
