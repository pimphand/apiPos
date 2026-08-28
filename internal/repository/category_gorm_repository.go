package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// CategoryRepository defines the interface for category operations
type CategoryRepository interface {
	Create(ctx context.Context, category *gormdb.Category) error
	GetByID(ctx context.Context, id string) (*gormdb.Category, error)
	GetByIDs(ctx context.Context, ids []string) ([]gormdb.Category, error)
	List(ctx context.Context, params CategoryListParams) ([]gormdb.Category, int64, error)
	Update(ctx context.Context, category *gormdb.Category) error
	Delete(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
}

// CategoryListParams holds parameters for listing categories
type CategoryListParams struct {
	IsActive string // "" = all, "true" = active only, "false" = inactive only
	Page     int32
	PerPage  int32
}

// CategoryRepo implements CategoryRepository using GORM
type CategoryRepo struct {
	*GormBase
}

// NewCategoryRepository creates a new CategoryRepo
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &CategoryRepo{GormBase: NewGormBase(db)}
}

// Create creates a new category
func (r *CategoryRepo) Create(ctx context.Context, category *gormdb.Category) error {
	return r.WithContext(ctx).Create(category).Error
}

// GetByID gets a category by ID
func (r *CategoryRepo) GetByID(ctx context.Context, id string) (*gormdb.Category, error) {
	var category gormdb.Category
	err := r.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// List lists categories with filters and pagination
func (r *CategoryRepo) List(ctx context.Context, params CategoryListParams) ([]gormdb.Category, int64, error) {
	var categories []gormdb.Category
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Category{})

	// Apply filters
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
		Find(&categories).Error

	return categories, totalCount, err
}

// Update updates a category
func (r *CategoryRepo) Update(ctx context.Context, category *gormdb.Category) error {
	return r.WithContext(ctx).Save(category).Error
}

// Delete deletes a category
func (r *CategoryRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Category{}).Error
}

// Deactivate soft-deletes a category
func (r *CategoryRepo) Deactivate(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.Category{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// GetByIDs gets multiple categories by their IDs
func (r *CategoryRepo) GetByIDs(ctx context.Context, ids []string) ([]gormdb.Category, error) {
	var categories []gormdb.Category
	if len(ids) == 0 {
		return categories, nil
	}
	err := r.WithContext(ctx).Where("id IN ?", ids).Find(&categories).Error
	return categories, err
}
