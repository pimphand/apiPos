package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListStoresResult holds the result of listing stores
type ListStoresResult struct {
	Stores     []gormdb.Store
	Pagination PaginationResult
}

// Service handles store business logic (superadmin).
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new store Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// GetStore gets a store by ID
func (s *Service) GetStore(ctx context.Context, id string) (*gormdb.Store, error) {
	if err := requireSuperadmin(ctx); err != nil {
		return nil, err
	}
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store ID format", fmt.Errorf("id: %s", id))
	}

	store, err := s.repos.Store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Store not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get store", err)
	}
	return store, nil
}

// ListStores lists stores with pagination and optional filters
func (s *Service) ListStores(ctx context.Context, isVerified, isActive string, page, perPage int32) (*ListStoresResult, error) {
	if err := requireSuperadmin(ctx); err != nil {
		return nil, err
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	stores, totalCount, err := s.repos.Store.List(ctx, repository.StoreListParams{
		IsVerified: isVerified,
		IsActive:   isActive,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list stores", err)
	}

	return &ListStoresResult{
		Stores: stores,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// VerifyStore verifies (approves) a pending store, activating it.
func (s *Service) VerifyStore(ctx context.Context, idStr string) (*gormdb.Store, error) {
	if err := requireSuperadmin(ctx); err != nil {
		return nil, err
	}
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store ID format", fmt.Errorf("id: %s", idStr))
	}

	store, err := s.repos.Store.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Store not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get store", err)
	}

	// Approve + activate the store.
	store.IsVerified = true
	store.IsActive = true

	if err := s.repos.Store.Update(ctx, store); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to verify store", err)
	}
	return store, nil
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// requireSuperadmin ensures the authenticated user is a superadmin.
func requireSuperadmin(ctx context.Context) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil || claims.RoleName != "superadmin" {
		return apperror.ErrForbidden
	}
	return nil
}
