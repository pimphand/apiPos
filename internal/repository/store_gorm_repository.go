package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// StoreRepository defines the interface for store operations
type StoreRepository interface {
	Create(ctx context.Context, store *gormdb.Store) error
	GetByID(ctx context.Context, id string) (*gormdb.Store, error)
	GetByName(ctx context.Context, name string) (*gormdb.Store, error)
	List(ctx context.Context, params StoreListParams) ([]gormdb.Store, int64, error)
	Update(ctx context.Context, store *gormdb.Store) error
}

// StoreListParams holds parameters for listing stores
type StoreListParams struct {
	IsVerified string // "" = all, "true" = verified, "false" = pending
	IsActive   string // "" = all, "true" = active, "false" = inactive
	Page       int32
	PerPage    int32
}

// StoreRepo implements StoreRepository using GORM
type StoreRepo struct {
	*GormBase
}

// NewStoreRepository creates a new StoreRepo
func NewStoreRepository(db *gorm.DB) StoreRepository {
	return &StoreRepo{GormBase: NewGormBase(db)}
}

// Create creates a new store
func (r *StoreRepo) Create(ctx context.Context, store *gormdb.Store) error {
	return r.WithContext(ctx).Create(store).Error
}

// GetByID gets a store by ID
func (r *StoreRepo) GetByID(ctx context.Context, id string) (*gormdb.Store, error) {
	var store gormdb.Store
	err := r.WithContext(ctx).Where("id = ?", id).First(&store).Error
	if err != nil {
		return nil, err
	}
	return &store, nil
}

// GetByName gets a store by name (store names are globally unique)
func (r *StoreRepo) GetByName(ctx context.Context, name string) (*gormdb.Store, error) {
	var store gormdb.Store
	err := r.WithContext(ctx).Where("name = ?", name).First(&store).Error
	if err != nil {
		return nil, err
	}
	return &store, nil
}

// List lists stores with filters and pagination (system-wide, superadmin only)
func (r *StoreRepo) List(ctx context.Context, params StoreListParams) ([]gormdb.Store, int64, error) {
	var stores []gormdb.Store
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Store{})

	if params.IsVerified == "true" {
		query = query.Where("is_verified = ?", true)
	} else if params.IsVerified == "false" {
		query = query.Where("is_verified = ?", false)
	}
	if params.IsActive == "true" {
		query = query.Where("is_active = ?", true)
	} else if params.IsActive == "false" {
		query = query.Where("is_active = ?", false)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("created_at ASC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&stores).Error

	return stores, totalCount, err
}

// Update updates a store
func (r *StoreRepo) Update(ctx context.Context, store *gormdb.Store) error {
	return r.WithContext(ctx).Save(store).Error
}
