package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListCategoriesResult holds the result of listing categories
type ListCategoriesResult struct {
	Categories []gormdb.Category
	Pagination PaginationResult
}

// Service handles category business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new category Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreateCategory creates a new category
func (s *Service) CreateCategory(ctx context.Context, name, description string) (*gormdb.Category, error) {
	// Check if category with same name exists
	result, err := s.ListCategories(ctx, "", 1, 1000)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check category", err)
	}

	for _, c := range result.Categories {
		if c.Name == name {
			return nil, apperror.ErrCategoryAlreadyExists
		}
	}

	category := &gormdb.Category{
		Name:        name,
		Description: description,
	}

	if err := s.repos.Category.Create(ctx, category); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create category", err)
	}

	return category, nil
}

// GetCategory gets a category by ID
func (s *Service) GetCategory(ctx context.Context, id string) (*gormdb.Category, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid category ID format", fmt.Errorf("id: %s", id))
	}

	category, err := s.repos.Category.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrCategoryNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get category", err)
	}

	return category, nil
}

// ListCategories lists categories with pagination
func (s *Service) ListCategories(ctx context.Context, isActive string, page, perPage int32) (*ListCategoriesResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	categories, totalCount, err := s.repos.Category.List(ctx, repository.CategoryListParams{
		IsActive: isActive,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list categories", err)
	}

	return &ListCategoriesResult{
		Categories: categories,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// UpdateCategory updates a category
func (s *Service) UpdateCategory(ctx context.Context, idStr, name, description string, isActive bool) (*gormdb.Category, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid category ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if category exists
	category, err := s.repos.Category.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrCategoryNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get category", err)
	}

	// Update fields
	if name != "" {
		category.Name = name
	}
	if description != "" {
		category.Description = description
	}
	category.IsActive = isActive

	if err := s.repos.Category.Update(ctx, category); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update category", err)
	}

	return category, nil
}

// DeleteCategory deletes a category
func (s *Service) DeleteCategory(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid category ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if category exists
	_, err := s.repos.Category.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrCategoryNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get category", err)
	}

	return s.repos.Category.Delete(ctx, idStr)
}

// DeactivateCategory soft-deletes a category
func (s *Service) DeactivateCategory(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid category ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if category exists
	_, err := s.repos.Category.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrCategoryNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get category", err)
	}

	return s.repos.Category.Deactivate(ctx, idStr)
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
