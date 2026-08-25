package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// PermissionRepository defines the interface for permission operations
type PermissionRepository interface {
	Create(ctx context.Context, permission *gormdb.Permission) error
	GetByID(ctx context.Context, id string) (*gormdb.Permission, error)
	GetByName(ctx context.Context, name string) (*gormdb.Permission, error)
	List(ctx context.Context, params PaginationParams) ([]gormdb.Permission, int64, error)
	ListByResource(ctx context.Context, resource string, params PaginationParams) ([]gormdb.Permission, int64, error)
	Update(ctx context.Context, permission *gormdb.Permission) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// PermissionRepo implements PermissionRepository using GORM
type PermissionRepo struct {
	*GormBase
}

// NewPermissionRepository creates a new PermissionRepo
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &PermissionRepo{GormBase: NewGormBase(db)}
}

// Create creates a new permission
func (r *PermissionRepo) Create(ctx context.Context, permission *gormdb.Permission) error {
	return r.WithContext(ctx).Create(permission).Error
}

// GetByID gets a permission by ID
func (r *PermissionRepo) GetByID(ctx context.Context, id string) (*gormdb.Permission, error) {
	var permission gormdb.Permission
	err := r.WithContext(ctx).Where("id = ?", id).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetByName gets a permission by name
func (r *PermissionRepo) GetByName(ctx context.Context, name string) (*gormdb.Permission, error) {
	var permission gormdb.Permission
	err := r.WithContext(ctx).Where("name = ?", name).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// List lists all permissions with pagination
func (r *PermissionRepo) List(ctx context.Context, params PaginationParams) ([]gormdb.Permission, int64, error) {
	var permissions []gormdb.Permission
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Permission{})

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := params.normalize()
	err := query.Order("resource, action").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&permissions).Error

	return permissions, totalCount, err
}

// ListByResource lists permissions filtered by resource
func (r *PermissionRepo) ListByResource(ctx context.Context, resource string, params PaginationParams) ([]gormdb.Permission, int64, error) {
	var permissions []gormdb.Permission
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.Permission{}).Where("resource = ?", resource)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	normalized := params.normalize()
	err := query.Order("action").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&permissions).Error

	return permissions, totalCount, err
}

// Update updates a permission
func (r *PermissionRepo) Update(ctx context.Context, permission *gormdb.Permission) error {
	return r.WithContext(ctx).Save(permission).Error
}

// Delete deletes a permission
func (r *PermissionRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Permission{}).Error
}

// Count counts all permissions
func (r *PermissionRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.WithContext(ctx).Model(&gormdb.Permission{}).Count(&count).Error
	return count, err
}

// RolePermissionRepository defines the interface for role-permission operations
type RolePermissionRepository interface {
	AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	RemovePermission(ctx context.Context, roleID, permissionID string) error
	GetByRoleID(ctx context.Context, roleID string) ([]gormdb.Permission, error)
	ReplacePermissions(ctx context.Context, roleID string, permissionIDs []string) error
}

// RolePermissionRepo implements RolePermissionRepository using GORM
type RolePermissionRepo struct {
	*GormBase
}

// NewRolePermissionRepository creates a new RolePermissionRepo
func NewRolePermissionRepository(db *gorm.DB) RolePermissionRepository {
	return &RolePermissionRepo{GormBase: NewGormBase(db)}
}

// AssignPermissions assigns permissions to a role
func (r *RolePermissionRepo) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return r.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, permID := range permissionIDs {
			rp := &gormdb.RolePermission{
				RoleID:       uuidFromString(roleID),
				PermissionID: uuidFromString(permID),
			}
			// Use OnConflict to skip duplicates
			if err := tx.Clauses().Create(rp).Error; err != nil {
				// Skip duplicate entries
				var existing gormdb.RolePermission
				if err := tx.Where("role_id = ? AND permission_id = ?", roleID, permID).First(&existing).Error; err == nil {
					continue
				}
				return err
			}
		}
		return nil
	})
}

// RemovePermission removes a permission from a role
func (r *RolePermissionRepo) RemovePermission(ctx context.Context, roleID, permissionID string) error {
	return r.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&gormdb.RolePermission{}).Error
}

// GetByRoleID gets all permissions for a role
func (r *RolePermissionRepo) GetByRoleID(ctx context.Context, roleID string) ([]gormdb.Permission, error) {
	var permissions []gormdb.Permission
	err := r.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.resource, permissions.action").
		Find(&permissions).Error
	return permissions, err
}

// ReplacePermissions replaces all permissions for a role
func (r *RolePermissionRepo) ReplacePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return r.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing permissions
		if err := tx.Where("role_id = ?", roleID).Delete(&gormdb.RolePermission{}).Error; err != nil {
			return err
		}
		// Add new permissions
		for _, permID := range permissionIDs {
			rp := &gormdb.RolePermission{
				RoleID:       uuidFromString(roleID),
				PermissionID: uuidFromString(permID),
			}
			if err := tx.Create(rp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// uuidFromString helper to parse UUID string
func uuidFromString(s string) uuid.UUID {
	if s == "" {
		return uuid.UUID{}
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}
	}
	return u
}
