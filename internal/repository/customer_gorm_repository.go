package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// CustomerRepository defines the interface for customer operations
type CustomerRepository interface {
	Create(ctx context.Context, customer *gormdb.Customer) error
	GetByID(ctx context.Context, id string) (*gormdb.Customer, error)
	GetByCode(ctx context.Context, code string) (*gormdb.Customer, error)
	List(ctx context.Context, params CustomerListParams) ([]gormdb.Customer, int64, error)
	Search(ctx context.Context, params CustomerSearchParams) ([]gormdb.Customer, int64, error)
	Update(ctx context.Context, customer *gormdb.Customer) error
	Delete(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
	GetPurchaseHistory(ctx context.Context, customerID string, params PaginationParams) ([]gormdb.Sale, int64, error)
}

// CustomerListParams holds parameters for listing customers
type CustomerListParams struct {
	IsActive string
	Page     int32
	PerPage  int32
}

// CustomerSearchParams holds parameters for searching customers
type CustomerSearchParams struct {
	Query   string
	Page    int32
	PerPage int32
}

// CustomerRepo implements CustomerRepository using GORM
type CustomerRepo struct {
	*GormBase
}

// NewCustomerRepository creates a new CustomerRepo
func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &CustomerRepo{GormBase: NewGormBase(db)}
}

// Create creates a new customer
func (r *CustomerRepo) Create(ctx context.Context, customer *gormdb.Customer) error {
	return r.WithContext(ctx).Create(customer).Error
}

// GetByID gets a customer by ID
func (r *CustomerRepo) GetByID(ctx context.Context, id string) (*gormdb.Customer, error) {
	var customer gormdb.Customer
	err := r.WithContext(ctx).Where("id = ?", id).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetByCode gets a customer by code
func (r *CustomerRepo) GetByCode(ctx context.Context, code string) (*gormdb.Customer, error) {
	var customer gormdb.Customer
	err := r.WithContext(ctx).Where("code = ?", code).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// List lists customers with filters
func (r *CustomerRepo) List(ctx context.Context, params CustomerListParams) ([]gormdb.Customer, int64, error) {
	var customers []gormdb.Customer
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Customer{})

	if params.IsActive == "true" {
		query = query.Where("is_active = ?", true)
	} else if params.IsActive == "false" {
		query = query.Where("is_active = ?", false)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("name ASC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&customers).Error

	return customers, totalCount, err
}

// Search searches customers by name, code, email, or phone
func (r *CustomerRepo) Search(ctx context.Context, params CustomerSearchParams) ([]gormdb.Customer, int64, error) {
	var customers []gormdb.Customer
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Customer{}).
		Where("name ILIKE ? OR code ILIKE ? OR email ILIKE ? OR phone ILIKE ?",
			"%"+params.Query+"%", "%"+params.Query+"%", "%"+params.Query+"%", "%"+params.Query+"%")

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("name ASC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&customers).Error

	return customers, totalCount, err
}

// Update updates a customer
func (r *CustomerRepo) Update(ctx context.Context, customer *gormdb.Customer) error {
	return r.WithContext(ctx).Save(customer).Error
}

// Delete deletes a customer
func (r *CustomerRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Customer{}).Error
}

// Deactivate soft-deletes a customer
func (r *CustomerRepo) Deactivate(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.Customer{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// GetPurchaseHistory gets a customer's purchase history
func (r *CustomerRepo) GetPurchaseHistory(ctx context.Context, customerID string, params PaginationParams) ([]gormdb.Sale, int64, error) {
	var sales []gormdb.Sale
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Sale{}).
		Where("customer_id = ?", customerID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := params.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&sales).Error

	return sales, totalCount, err
}
