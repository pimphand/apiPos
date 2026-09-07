package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// TableRepository defines the interface for dine-in table operations.
// All queries are tenant-scoped via the GORM store_scope callbacks (the
// `tables` table carries NOT NULL store_id).
type TableRepository interface {
	Create(ctx context.Context, table *gormdb.Table) error
	GetByID(ctx context.Context, id uuid.UUID) (*gormdb.Table, error)
	GetByQRToken(ctx context.Context, token string) (*gormdb.Table, error)
	List(ctx context.Context, params PaginationParams) ([]gormdb.Table, int64, error)
	Update(ctx context.Context, table *gormdb.Table) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TableRepo implements TableRepository using GORM
type TableRepo struct {
	*GormBase
}

// NewTableRepository creates a new TableRepo
func NewTableRepository(db *gorm.DB) TableRepository {
	return &TableRepo{GormBase: NewGormBase(db)}
}

// Create inserts a new table
func (r *TableRepo) Create(ctx context.Context, table *gormdb.Table) error {
	return r.WithContext(ctx).Create(table).Error
}

// GetByID gets a table by id (store-scoped)
func (r *TableRepo) GetByID(ctx context.Context, id uuid.UUID) (*gormdb.Table, error) {
	var table gormdb.Table
	if err := r.WithContext(ctx).Where("id = ?", id).First(&table).Error; err != nil {
		return nil, err
	}
	return &table, nil
}

// GetByQRToken resolves a table by its globally-unique QR token. No store
// scoping is applied here on purpose: the token is globally unique, and the
// self-order flow resolves the tenant server-side from the resulting row.
func (r *TableRepo) GetByQRToken(ctx context.Context, token string) (*gormdb.Table, error) {
	var table gormdb.Table
	if err := r.WithContext(ctx).Unscoped().Where("qr_token = ?", token).First(&table).Error; err != nil {
		return nil, err
	}
	return &table, nil
}

// List lists the store's tables with pagination
func (r *TableRepo) List(ctx context.Context, params PaginationParams) ([]gormdb.Table, int64, error) {
	var tables []gormdb.Table
	var total int64

	query := r.WithContext(ctx).Model(&gormdb.Table{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.Paginate(query, params).Order("code ASC").Find(&tables).Error; err != nil {
		return nil, 0, err
	}
	return tables, total, nil
}

// Update saves a table row
func (r *TableRepo) Update(ctx context.Context, table *gormdb.Table) error {
	return r.WithContext(ctx).Save(table).Error
}

// Delete removes a table row. Sales referencing it are set to NULL via the
// ON DELETE SET NULL FK.
func (r *TableRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Table{}).Error
}