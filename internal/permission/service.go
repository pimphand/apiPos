package permission

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

// Service handles permission business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new permission Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreatePermission creates a new permission
func (s *Service) CreatePermission(ctx context.Context, name, resource, action, description string) (*gormdb.Permission, error) {
	// Check if permission with same name exists
	_, err := s.repos.Permission.GetByName(ctx, name)
	if err == nil {
		return nil, apperror.Wrap(apperror.ErrAlreadyExists, "Permission with this name already exists", nil)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check permission", err)
	}

	permission := &gormdb.Permission{
		Name:        name,
		Resource:    resource,
		Action:      action,
		Description: description,
	}

	if err := s.repos.Permission.Create(ctx, permission); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create permission", err)
	}

	return permission, nil
}

// GetPermission gets a permission by ID
func (s *Service) GetPermission(ctx context.Context, id string) (*gormdb.Permission, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid permission ID format", fmt.Errorf("id: %s", id))
	}

	permission, err := s.repos.Permission.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Permission not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get permission", err)
	}

	return permission, nil
}

// ListPermissions lists all permissions with pagination
func (s *Service) ListPermissions(ctx context.Context, page, perPage int32) ([]gormdb.Permission, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.repos.Permission.List(ctx, repository.PaginationParams{
		Page:    page,
		PerPage: perPage,
	})
}

// ListPermissionsByResource lists permissions filtered by resource
func (s *Service) ListPermissionsByResource(ctx context.Context, resource string, page, perPage int32) ([]gormdb.Permission, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.repos.Permission.ListByResource(ctx, resource, repository.PaginationParams{
		Page:    page,
		PerPage: perPage,
	})
}

// UpdatePermission updates a permission
func (s *Service) UpdatePermission(ctx context.Context, idStr, name, resource, action, description string) (*gormdb.Permission, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid permission ID format", fmt.Errorf("id: %s", idStr))
	}

	permission, err := s.repos.Permission.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Permission not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get permission", err)
	}

	if name != "" {
		permission.Name = name
	}
	if resource != "" {
		permission.Resource = resource
	}
	if action != "" {
		permission.Action = action
	}
	if description != "" {
		permission.Description = description
	}

	if err := s.repos.Permission.Update(ctx, permission); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update permission", err)
	}

	return permission, nil
}

// DeletePermission deletes a permission
func (s *Service) DeletePermission(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid permission ID format", fmt.Errorf("id: %s", idStr))
	}

	_, err := s.repos.Permission.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.Wrap(apperror.ErrNotFound, "Permission not found", err)
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get permission", err)
	}

	return s.repos.Permission.Delete(ctx, idStr)
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
