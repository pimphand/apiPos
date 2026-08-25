package repository

import (
	"context"

	db "apiPos/internal/database/sqlc"
)

// CategoryRepositoryImpl implements CategoryRepository.
type CategoryRepositoryImpl struct {
	queries *db.Queries
}

// NewCategoryRepository creates a new CategoryRepositoryImpl.
func NewCategoryRepository(queries *db.Queries) CategoryRepository {
	return &CategoryRepositoryImpl{queries: queries}
}

func (r *CategoryRepositoryImpl) CreateCategory(ctx context.Context, arg db.CreateCategoryParams) (db.Category, error) {
	return r.queries.CreateCategory(ctx, arg)
}

func (r *CategoryRepositoryImpl) GetCategoryByID(ctx context.Context, id int64) (db.Category, error) {
	return r.queries.GetCategoryByID(ctx, id)
}

func (r *CategoryRepositoryImpl) ListCategories(ctx context.Context, isActive bool) ([]db.Category, error) {
	return r.queries.ListCategories(ctx, isActive)
}

func (r *CategoryRepositoryImpl) UpdateCategory(ctx context.Context, arg db.UpdateCategoryParams) (db.Category, error) {
	return r.queries.UpdateCategory(ctx, arg)
}

func (r *CategoryRepositoryImpl) DeleteCategory(ctx context.Context, id int64) error {
	return r.queries.DeleteCategory(ctx, id)
}

func (r *CategoryRepositoryImpl) DeactivateCategory(ctx context.Context, id int64) error {
	return r.queries.DeactivateCategory(ctx, id)
}

func (r *CategoryRepositoryImpl) CountCategories(ctx context.Context) (int64, error) {
	return r.queries.CountCategories(ctx)
}
