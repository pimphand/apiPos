package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"apiPos/internal/auth"
	"apiPos/internal/database/sqlc"
	"apiPos/internal/repository"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidRole       = errors.New("invalid role")
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
func (s *Service) CreateUser(ctx context.Context, username, email, password, fullName string, roleID int64, storeID int64) (*db.User, error) {
	// Check if username already exists
	_, err := s.repos.User.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}

	// Check if email already exists
	_, err = s.repos.User.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// Hash password
	hashedPassword, err := s.auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.repos.User.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		RoleID:       roleID,
		StoreID:      pgtype.Int8{Int64: storeID, Valid: storeID > 0},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUser gets a user by ID
func (s *Service) GetUser(ctx context.Context, id int64) (*db.User, error) {
	user, err := s.repos.User.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// ListUsers lists users with filters
func (s *Service) ListUsers(ctx context.Context, roleID int64, isActive bool, page, perPage int32) ([]db.User, int64, error) {
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	users, err := s.repos.User.ListUsers(ctx, db.ListUsersParams{
		Column1: roleID,
		Column2: isActive,
		Limit:   perPage,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Get total count
	count, err := s.repos.User.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	return users, count, nil
}

// UpdateUser updates a user
func (s *Service) UpdateUser(ctx context.Context, id int64, email, fullName string, roleID int64, storeID int64, isActive bool) (*db.User, error) {
	// Check if user exists
	_, err := s.repos.User.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update user
	user, err := s.repos.User.UpdateUser(ctx, db.UpdateUserParams{
		ID:       id,
		FullName: fullName,
		Email:    email,
		RoleID:   roleID,
		StoreID:  pgtype.Int8{Int64: storeID, Valid: storeID > 0},
		IsActive: isActive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	// Check if user exists
	_, err := s.repos.User.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	return s.repos.User.DeleteUser(ctx, id)
}

// UpdateUserRole updates a user's role
func (s *Service) UpdateUserRole(ctx context.Context, id int64, roleID int64) (*db.User, error) {
	// Check if user exists
	user, err := s.repos.User.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update user role
	updatedUser, err := s.repos.User.UpdateUser(ctx, db.UpdateUserParams{
		ID:       id,
		FullName: user.FullName,
		Email:    user.Email,
		RoleID:   roleID,
		StoreID:  user.StoreID,
		IsActive: user.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	return &updatedUser, nil
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, username, password string) (*auth.TokenPair, *db.User, error) {
	// Get user by username
	user, err := s.repos.User.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, auth.ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, auth.ErrUnauthorized
	}

	// Check password
	if !s.auth.CheckPassword(user.PasswordHash, password) {
		return nil, nil, auth.ErrInvalidCredentials
	}

	// Get role name (simplified - in real app, you'd fetch the role)
	roleName := "user" // Default role name

	// Generate tokens
	tokenPair, err := s.auth.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Email,
		user.RoleID,
		roleName,
		user.StoreID.Int64,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login
	_ = s.repos.User.UpdateUserLastLogin(ctx, user.ID)

	return tokenPair, &user, nil
}

// RefreshToken refreshes an access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// Validate refresh token
	userID, err := s.auth.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.repos.User.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, auth.ErrUnauthorized
	}

	// Get role name (simplified)
	roleName := "user"

	// Generate new tokens
	tokenPair, err := s.auth.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Email,
		user.RoleID,
		roleName,
		user.StoreID.Int64,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}
