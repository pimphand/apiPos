package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	"apiPos/internal/database/sqlc"
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

// parseUUID safely parses a string to pgtype.UUID
func parseUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	var u pgtype.UUID
	err := u.Scan(s)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

// uuidToString converts pgtype.UUID to string
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, username, email, password, fullName, roleIDStr, storeIDStr string) (*db.User, error) {
	// Check if username already exists
	_, err := s.repos.User.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check username", err)
	}

	// Check if email already exists
	_, err = s.repos.User.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check email", err)
	}

	// Hash password
	hashedPassword, err := s.auth.HashPassword(password)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to hash password", err)
	}

	// Parse role_id - required
	roleUUID := parseUUID(roleIDStr)
	if !roleUUID.Valid {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role_id format", fmt.Errorf("role_id: %s", roleIDStr))
	}

	// Parse store_id - optional
	storeUUID := parseUUID(storeIDStr)

	// Create user
	user, err := s.repos.User.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		RoleID:       roleUUID,
		StoreID:      storeUUID,
	})
	if err != nil {
		// Let MapError handle PostgreSQL-specific errors (FK violation, etc.)
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create user", err)
	}

	return &user, nil
}

// GetUser gets a user by ID
func (s *Service) GetUser(ctx context.Context, id string) (*db.User, error) {
	userUUID := parseUUID(id)
	if !userUUID.Valid {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", id))
	}

	user, err := s.repos.User.GetUserByID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}
	return &user, nil
}

// ListUsers lists users with filters
func (s *Service) ListUsers(ctx context.Context, roleIDStr string, isActive bool, page, perPage int32) ([]db.User, int64, error) {
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	roleUUID := parseUUID(roleIDStr)

	users, err := s.repos.User.ListUsers(ctx, db.ListUsersParams{
		Column1: roleUUID,
		Column2: isActive,
		Limit:   perPage,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, apperror.Wrap(apperror.ErrInternal, "Failed to list users", err)
	}

	// Get total count
	count, err := s.repos.User.CountUsers(ctx)
	if err != nil {
		return nil, 0, apperror.Wrap(apperror.ErrInternal, "Failed to count users", err)
	}

	return users, count, nil
}

// UpdateUser updates a user
func (s *Service) UpdateUser(ctx context.Context, idStr, email, fullName, roleIDStr, storeIDStr string, isActive bool) (*db.User, error) {
	idUUID := parseUUID(idStr)
	if !idUUID.Valid {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	_, err := s.repos.User.GetUserByID(ctx, idUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	roleUUID := parseUUID(roleIDStr)
	storeUUID := parseUUID(storeIDStr)

	// Update user
	user, err := s.repos.User.UpdateUser(ctx, db.UpdateUserParams{
		ID:       idUUID,
		FullName: fullName,
		Email:    email,
		RoleID:   roleUUID,
		StoreID:  storeUUID,
		IsActive: isActive,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user", err)
	}

	return &user, nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, idStr string) error {
	idUUID := parseUUID(idStr)
	if !idUUID.Valid {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	_, err := s.repos.User.GetUserByID(ctx, idUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.ErrUserNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	return s.repos.User.DeleteUser(ctx, idUUID)
}

// UpdateUserRole updates a user's role
func (s *Service) UpdateUserRole(ctx context.Context, idStr, roleIDStr string) (*db.User, error) {
	idUUID := parseUUID(idStr)
	if !idUUID.Valid {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	user, err := s.repos.User.GetUserByID(ctx, idUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	roleUUID := parseUUID(roleIDStr)
	if !roleUUID.Valid {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role_id format", fmt.Errorf("role_id: %s", roleIDStr))
	}

	// Update user role
	updatedUser, err := s.repos.User.UpdateUser(ctx, db.UpdateUserParams{
		ID:       idUUID,
		FullName: user.FullName,
		Email:    user.Email,
		RoleID:   roleUUID,
		StoreID:  user.StoreID,
		IsActive: user.IsActive,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user role", err)
	}

	return &updatedUser, nil
}

// Register creates a new user with default role (Cashier)
func (s *Service) Register(ctx context.Context, username, email, password, fullName string) (*auth.TokenPair, *db.User, error) {
	// Check if username already exists
	_, err := s.repos.User.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to check username", err)
	}

	// Check if email already exists
	_, err = s.repos.User.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, nil, apperror.ErrUserAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to check email", err)
	}

	// Hash password
	hashedPassword, err := s.auth.HashPassword(password)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to hash password", err)
	}

	// Look up the cashier role
	roles, err := s.repos.User.ListRoles(ctx)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to list roles", err)
	}

	var cashierRoleID pgtype.UUID
	for _, r := range roles {
		if r.Name == "cashier" {
			cashierRoleID = r.ID
			break
		}
	}
	if !cashierRoleID.Valid {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Cashier role not found", fmt.Errorf("cashier role missing"))
	}

	// Create user with default role (Cashier)
	user, err := s.repos.User.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		RoleID:       cashierRoleID,
	})
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to create user", err)
	}

	// Generate tokens
	storeIDStr := uuidToString(user.StoreID)
	tokenPair, err := s.auth.GenerateTokenPair(
		uuidToString(user.ID),
		user.Username,
		user.Email,
		uuidToString(user.RoleID),
		"cashier",
		storeIDStr,
	)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate tokens", err)
	}

	return tokenPair, &user, nil
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, username, password string) (*auth.TokenPair, *db.User, error) {
	// Get user by username
	user, err := s.repos.User.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	storeIDStr := uuidToString(user.StoreID)
	tokenPair, err := s.auth.GenerateTokenPair(
		uuidToString(user.ID),
		user.Username,
		user.Email,
		uuidToString(user.RoleID),
		"user",
		storeIDStr,
	)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate tokens", err)
	}

	// Update last login
	_ = s.repos.User.UpdateUserLastLogin(ctx, user.ID)

	return tokenPair, &user, nil
}

// RefreshToken refreshes an access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// Validate refresh token
	userIDStr, err := s.auth.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	userID := parseUUID(userIDStr)

	// Get user
	user, err := s.repos.User.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrInvalidToken
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, apperror.ErrUnauthorized
	}

	// Generate new tokens
	storeIDStr := uuidToString(user.StoreID)
	tokenPair, err := s.auth.GenerateTokenPair(
		uuidToString(user.ID),
		user.Username,
		user.Email,
		uuidToString(user.RoleID),
		"user",
		storeIDStr,
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate tokens", err)
	}

	return tokenPair, nil
}
