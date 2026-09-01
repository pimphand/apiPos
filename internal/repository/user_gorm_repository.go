package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// UserRepository defines the interface for user operations
type UserRepository interface {
	Create(ctx context.Context, user *gormdb.User) error
	GetByID(ctx context.Context, id string) (*gormdb.User, error)
	GetByUsername(ctx context.Context, username string) (*gormdb.User, error)
	GetByEmail(ctx context.Context, email string) (*gormdb.User, error)
	List(ctx context.Context, params UserListParams) ([]gormdb.User, int64, error)
	Update(ctx context.Context, user *gormdb.User) error
	UpdateFields(ctx context.Context, id string, fields map[string]interface{}) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	UpdateLastLogin(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// UserListParams holds parameters for listing users
type UserListParams struct {
	RoleID    string
	IsActive  string // "" = all, "true" = active only, "false" = inactive only
	Page      int32
	PerPage   int32
}

// UserRepo implements UserRepository using GORM
type UserRepo struct {
	*GormBase
}

// NewUserRepository creates a new UserRepo
func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepo{GormBase: NewGormBase(db)}
}

// Create creates a new user
func (r *UserRepo) Create(ctx context.Context, user *gormdb.User) error {
	return r.WithContext(ctx).Create(user).Error
}

// GetByID gets a user by ID
func (r *UserRepo) GetByID(ctx context.Context, id string) (*gormdb.User, error) {
	var user gormdb.User
	err := r.WithContext(ctx).Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername gets a user by username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*gormdb.User, error) {
	var user gormdb.User
	err := r.WithContext(ctx).Preload("Role").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail gets a user by email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*gormdb.User, error) {
	var user gormdb.User
	err := r.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List lists users with filters and pagination
func (r *UserRepo) List(ctx context.Context, params UserListParams) ([]gormdb.User, int64, error) {
	var users []gormdb.User
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.User{})

	// Apply filters
	if params.RoleID != "" {
		query = query.Where("role_id = ?", params.RoleID)
	}

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
	err := query.Preload("Role").
		Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&users).Error

	return users, totalCount, err
}

// Update updates a user (full save — use UpdateFields for targeted updates)
func (r *UserRepo) Update(ctx context.Context, user *gormdb.User) error {
	return r.WithContext(ctx).Save(user).Error
}

// UpdateFields updates specific columns on a user by ID.
func (r *UserRepo) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.WithContext(ctx).Model(&gormdb.User{}).Where("id = ?", id).Updates(fields).Error
}

// UpdatePassword updates a user's password
func (r *UserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return r.WithContext(ctx).
		Model(&gormdb.User{}).
		Where("id = ?", id).
		Update("password_hash", passwordHash).Error
}

// UpdateLastLogin updates a user's last login timestamp
func (r *UserRepo) UpdateLastLogin(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.User{}).
		Where("id = ?", id).
		Update("last_login", gorm.Expr("NOW()")).Error
}

// Delete deletes a user
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.User{}).Error
}

// Deactivate soft-deletes a user
func (r *UserRepo) Deactivate(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.User{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// Count counts all users
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.WithContext(ctx).Model(&gormdb.User{}).Count(&count).Error
	return count, err
}

// RoleRepo implements role operations using GORM
type RoleRepo struct {
	*GormBase
}

// NewRoleRepository creates a new RoleRepo
func NewRoleRepository(db *gorm.DB) *RoleRepo {
	return &RoleRepo{GormBase: NewGormBase(db)}
}

// List lists all roles
func (r *RoleRepo) List(ctx context.Context) ([]gormdb.Role, error) {
	var roles []gormdb.Role
	err := r.WithContext(ctx).Order("name").Find(&roles).Error
	return roles, err
}

// GetByID gets a role by ID
func (r *RoleRepo) GetByID(ctx context.Context, id string) (*gormdb.Role, error) {
	var role gormdb.Role
	err := r.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Create creates a new role
func (r *RoleRepo) Create(ctx context.Context, role *gormdb.Role) error {
	return r.WithContext(ctx).Create(role).Error
}

// Update updates a role
func (r *RoleRepo) Update(ctx context.Context, role *gormdb.Role) error {
	return r.WithContext(ctx).Save(role).Error
}

// Delete deletes a role
func (r *RoleRepo) Delete(ctx context.Context, id string) error {
	return r.WithContext(ctx).Where("id = ?", id).Delete(&gormdb.Role{}).Error
}
