package user

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

// Service handles user business logic
type Service struct {
	repos *repository.Repositories
	auth  *auth.AuthService
}

// NewService creates a new user Service
func NewService(repos *repository.Repositories, authService *auth.AuthService) *Service {
	return &Service{
		repos: repos,
		auth:  authService,
	}
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, username, email, password, fullName, roleIDStr, storeIDStr string) (*gormdb.User, error) {
	// Check if username already exists
	_, err := s.repos.User.GetByUsername(ctx, username)
	if err == nil {
		return nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check username", err)
	}

	// Check if email already exists
	_, err = s.repos.User.GetByEmail(ctx, email)
	if err == nil {
		return nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check email", err)
	}

	// Hash password
	hashedPassword, err := s.auth.HashPassword(password)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to hash password", err)
	}

	// Parse role_id - required
	roleUUID := parseUUID(roleIDStr)
	if roleUUID == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role_id format", fmt.Errorf("role_id: %s", roleIDStr))
	}

	// Parse store_id - optional
	storeUUID := parseUUID(storeIDStr)

	// Create user
	user := &gormdb.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		RoleID:       parseUUIDToUUID(roleIDStr),
		StoreID:      parseUUIDToUUID(storeUUID),
	}

	if err := s.repos.User.Create(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create user", err)
	}

	return user, nil
}

// GetUser gets a user by ID
func (s *Service) GetUser(ctx context.Context, id string) (*gormdb.User, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", id))
	}

	user, err := s.repos.User.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}
	return user, nil
}

// ListUsers lists users with filters
func (s *Service) ListUsers(ctx context.Context, roleIDStr string, isActiveStr string, page, perPage int32) ([]gormdb.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	users, totalCount, err := s.repos.User.List(ctx, repository.UserListParams{
		RoleID:   roleIDStr,
		IsActive: isActiveStr,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		return nil, 0, apperror.Wrap(apperror.ErrInternal, "Failed to list users", err)
	}

	return users, totalCount, nil
}

// UpdateUser updates a user
func (s *Service) UpdateUser(ctx context.Context, idStr, email, fullName, roleIDStr, storeIDStr string, isActive bool) (*gormdb.User, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	user, err := s.repos.User.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	// Update fields
	if email != "" {
		user.Email = email
	}
	if fullName != "" {
		user.FullName = fullName
	}
	if roleIDStr != "" {
		user.RoleID = parseUUIDToUUID(roleIDStr)
	}
	if storeIDStr != "" {
		user.StoreID = parseUUIDToUUID(storeIDStr)
	}
	user.IsActive = isActive

	if err := s.repos.User.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user", err)
	}

	return user, nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	_, err := s.repos.User.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrUserNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	return s.repos.User.Delete(ctx, idStr)
}

// UpdateUserRole updates a user's role
func (s *Service) UpdateUserRole(ctx context.Context, idStr, roleIDStr string) (*gormdb.User, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	user, err := s.repos.User.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	if !isValidUUID(roleIDStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role_id format", fmt.Errorf("role_id: %s", roleIDStr))
	}

	// Update user role
	user.RoleID = parseUUIDToUUID(roleIDStr)

	if err := s.repos.User.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user role", err)
	}

	return user, nil
}

// Register creates a new user with default role (Cashier)
func (s *Service) Register(ctx context.Context, username, email, password, fullName string) (*auth.TokenPair, *gormdb.User, error) {
	// Check if username already exists
	_, err := s.repos.User.GetByUsername(ctx, username)
	if err == nil {
		return nil, nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to check username", err)
	}

	// Check if email already exists
	_, err = s.repos.User.GetByEmail(ctx, email)
	if err == nil {
		return nil, nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to check email", err)
	}

	// Hash password
	hashedPassword, err := s.auth.HashPassword(password)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to hash password", err)
	}

	// Look up the cashier role
	roles, err := s.repos.Role.List(ctx)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to list roles", err)
	}

	var cashierRoleID string
	for _, r := range roles {
		if r.Name == "cashier" {
			cashierRoleID = r.ID.String()
			break
		}
	}
	if cashierRoleID == "" {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Cashier role not found", fmt.Errorf("cashier role missing"))
	}

	// Create user with default role (Cashier)
	user := &gormdb.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		RoleID:       parseUUIDToUUID(cashierRoleID),
	}

	if err := s.repos.User.Create(ctx, user); err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to create user", err)
	}

	// Generate tokens
	tokenPair, err := s.auth.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		user.RoleID.String(),
		"cashier",
		user.StoreID.String(),
	)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate tokens", err)
	}

	return tokenPair, user, nil
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, username, password string) (*auth.TokenPair, *gormdb.User, error) {
	// Get user by username
	user, err := s.repos.User.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apperror.ErrInvalidCredentials
		}
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, apperror.ErrUnauthorized
	}

	// Check password
	if !s.auth.CheckPassword(user.PasswordHash, password) {
		return nil, nil, apperror.ErrInvalidCredentials
	}

	// Generate tokens
	tokenPair, err := s.auth.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		user.RoleID.String(),
		"user",
		user.StoreID.String(),
	)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate tokens", err)
	}

	// Update last login
	_ = s.repos.User.UpdateLastLogin(ctx, user.ID.String())

	return tokenPair, user, nil
}

// RefreshToken refreshes an access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// Validate refresh token
	userIDStr, err := s.auth.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.repos.User.GetByID(ctx, userIDStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrInvalidToken
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, apperror.ErrUnauthorized
	}

	// Generate new tokens
	tokenPair, err := s.auth.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		user.RoleID.String(),
		"user",
		user.StoreID.String(),
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate tokens", err)
	}

	return tokenPair, nil
}

// parseUUID converts a string to UUID string
func parseUUID(s string) string {
	if s == "" {
		return ""
	}
	// Simple validation - just check format
	if len(s) != 36 {
		return ""
	}
	return s
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	// Simple validation - check length and format
	return len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// parseUUIDToUUID converts string to uuid.UUID
func parseUUIDToUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.UUID{}
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}
	}
	return u
}
