package role

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

// Service handles role business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new role Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreateRole creates a new role
func (s *Service) CreateRole(ctx context.Context, name, description string) (*gormdb.Role, error) {
	// Check if role with same name exists
	_, err := s.repos.Role.GetByID(ctx, "")
	if err == nil {
		// Check by name - list and search
	}

	// Check by listing roles
	roles, err := s.repos.Role.List(ctx)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check role", err)
	}
	for _, r := range roles {
		if r.Name == name {
			return nil, apperror.Wrap(apperror.ErrAlreadyExists, "Role with this name already exists", nil)
		}
	}

	role := &gormdb.Role{
		Name:        name,
		Description: description,
	}

	if err := s.repos.Role.Create(ctx, role); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create role", err)
	}

	return role, nil
}

// GetRole gets a role by ID
func (s *Service) GetRole(ctx context.Context, id string) (*gormdb.Role, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role ID format", fmt.Errorf("id: %s", id))
	}

	role, err := s.repos.Role.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Role not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get role", err)
	}

	return role, nil
}

// ListRoles lists all roles with pagination
func (s *Service) ListRoles(ctx context.Context, page, perPage int32) ([]gormdb.Role, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	roles, err := s.repos.Role.List(ctx)
	if err != nil {
		return nil, 0, apperror.Wrap(apperror.ErrInternal, "Failed to list roles", err)
	}

	total := int64(len(roles))

	// Simple pagination
	start := int((page - 1) * perPage)
	if start > len(roles) {
		start = len(roles)
	}
	end := start + int(perPage)
	if end > len(roles) {
		end = len(roles)
	}

	return roles[start:end], total, nil
}

// UpdateRole updates a role
func (s *Service) UpdateRole(ctx context.Context, idStr, name, description string) (*gormdb.Role, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role ID format", fmt.Errorf("id: %s", idStr))
	}

	role, err := s.repos.Role.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Role not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get role", err)
	}

	if name != "" {
		role.Name = name
	}
	if description != "" {
		role.Description = description
	}

	if err := s.repos.Role.Update(ctx, role); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update role", err)
	}

	return role, nil
}

// DeleteRole deletes a role
func (s *Service) DeleteRole(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role ID format", fmt.Errorf("id: %s", idStr))
	}

	role, err := s.repos.Role.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.Wrap(apperror.ErrNotFound, "Role not found", err)
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get role", err)
	}

	// Prevent deleting default roles
	if role.Name == "admin" || role.Name == "manager" || role.Name == "cashier" {
		return apperror.Wrap(apperror.ErrFailedPrecondition, "Cannot delete default role", nil)
	}

	return s.repos.Role.Delete(ctx, idStr)
}

// GetRolePermissions gets all permissions for a role
func (s *Service) GetRolePermissions(ctx context.Context, roleID string) ([]gormdb.Permission, error) {
	if !isValidUUID(roleID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role ID format", fmt.Errorf("role_id: %s", roleID))
	}

	// Verify role exists
	_, err := s.repos.Role.GetByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Role not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get role", err)
	}

	return s.repos.RolePermission.GetByRoleID(ctx, roleID)
}

// AssignPermissions assigns permissions to a role
func (s *Service) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) ([]gormdb.Permission, error) {
	if !isValidUUID(roleID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role ID format", fmt.Errorf("role_id: %s", roleID))
	}

	// Verify role exists
	_, err := s.repos.Role.GetByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Role not found", err)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get role", err)
	}

	// Verify all permissions exist
	for _, permID := range permissionIDs {
		if !isValidUUID(permID) {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid permission ID format", fmt.Errorf("permission_id: %s", permID))
		}
		_, err := s.repos.Permission.GetByID(ctx, permID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.Wrap(apperror.ErrNotFound, "Permission not found", err)
			}
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get permission", err)
		}
	}

	// Replace all permissions for this role
	if err := s.repos.RolePermission.ReplacePermissions(ctx, roleID, permissionIDs); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to assign permissions", err)
	}

	// Return updated permissions
	return s.repos.RolePermission.GetByRoleID(ctx, roleID)
}

// RemovePermission removes a permission from a role
func (s *Service) RemovePermission(ctx context.Context, roleID, permissionID string) error {
	if !isValidUUID(roleID) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role ID format", fmt.Errorf("role_id: %s", roleID))
	}
	if !isValidUUID(permissionID) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid permission ID format", fmt.Errorf("permission_id: %s", permissionID))
	}

	return s.repos.RolePermission.RemovePermission(ctx, roleID, permissionID)
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
