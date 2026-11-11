package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/accounting"
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

// RolePermissionNames returns the permission names granted to a role. Used by
// auth responses (login/register/refresh) so the client gets its permissions
// embedded in the auth payload instead of calling a separate endpoint.
func (s *Service) RolePermissionNames(ctx context.Context, roleID string) ([]string, error) {
	if roleID == "" {
		return nil, nil
	}
	perms, err := s.repos.RolePermission.GetByRoleID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(perms))
	for i, p := range perms {
		names[i] = p.Name
	}
	return names, nil
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

// ListUsers lists users with filters. Non-superadmin callers are restricted to
// their own store's users; superadmins (system-wide) see all stores.
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
		StoreID:  storeScopeFromClaims(ctx),
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		return nil, 0, apperror.Wrap(apperror.ErrInternal, "Failed to list users", err)
	}

	return users, totalCount, nil
}

// storeScopeFromClaims returns the store_id a non-superadmin caller is
// restricted to, or "" for superadmins (who can see all stores) and for
// contexts without claims.
func storeScopeFromClaims(ctx context.Context) string {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return ""
	}
	if claims.RoleName == "superadmin" {
		return ""
	}
	return claims.StoreID
}

// UpdateUser updates a user — uses targeted Updates to avoid overwriting
// fields that weren't loaded (e.g. Store relation) which would cause
// foreign-key violations.
func (s *Service) UpdateUser(ctx context.Context, idStr, email, fullName, roleIDStr, storeIDStr string, isActive bool) (*gormdb.User, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if user exists
	if _, err := s.repos.User.GetByID(ctx, idStr); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	// Build a map of fields to update so we only touch columns that were
	// explicitly provided, avoiding zero-value overwrites (especially StoreID
	// which is not preloaded by GetByID).
	updates := map[string]interface{}{"is_active": isActive}
	if email != "" {
		updates["email"] = email
	}
	if fullName != "" {
		updates["full_name"] = fullName
	}
	if roleIDStr != "" {
		updates["role_id"] = parseUUIDToUUID(roleIDStr)
	}
	if storeIDStr != "" {
		updates["store_id"] = parseUUIDToUUID(storeIDStr)
	}

	if err := s.repos.User.UpdateFields(ctx, idStr, updates); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user", err)
	}

	// Re-fetch so returned user has fresh data
	return s.repos.User.GetByID(ctx, idStr)
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

// ToggleUserActive toggles a user's active status.
func (s *Service) ToggleUserActive(ctx context.Context, idStr string, isActive bool) (*gormdb.User, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	if _, err := s.repos.User.GetByID(ctx, idStr); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	if err := s.repos.User.UpdateFields(ctx, idStr, map[string]interface{}{
		"is_active": isActive,
	}); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user", err)
	}

	return s.repos.User.GetByID(ctx, idStr)
}

// UpdateUserRole updates a user's role using targeted updates.
func (s *Service) UpdateUserRole(ctx context.Context, idStr, roleIDStr string) (*gormdb.User, error) {
	if !isValidUUID(idStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("id: %s", idStr))
	}

	if !isValidUUID(roleIDStr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid role_id format", fmt.Errorf("role_id: %s", roleIDStr))
	}

	// Check if user exists
	if _, err := s.repos.User.GetByID(ctx, idStr); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get user", err)
	}

	// Use targeted update to avoid foreign-key issues with Store relation
	if err := s.repos.User.UpdateFields(ctx, idStr, map[string]interface{}{
		"role_id": parseUUIDToUUID(roleIDStr),
	}); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update user role", err)
	}

	return s.repos.User.GetByID(ctx, idStr)
}

// Register creates a new user with default role (Cashier)
func (s *Service) Register(ctx context.Context, username, email, password, fullName, deviceID, storeName string) (*auth.TokenPair, *gormdb.User, error) {
	// Store name is required for SaaS onboarding
	if strings.TrimSpace(storeName) == "" {
		return nil, nil, apperror.Wrap(apperror.ErrInvalidRequest, "Store name is required", fmt.Errorf("store name required"))
	}
	storeName = strings.TrimSpace(storeName)
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

	// Look up the admin role that the registering user is assigned
	roles, err := s.repos.Role.List(ctx)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to list roles", err)
	}

	var adminRoleID string
	for _, r := range roles {
		if r.Name == "admin" {
			adminRoleID = r.ID.String()
			break
		}
	}
	if adminRoleID == "" {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Admin role not found", fmt.Errorf("admin role missing"))
	}

	// Registration auto-creates a new store for the user and assigns the user
	// the admin role for that store. The store name is provided by the user
	// during signup (required for SaaS onboarding).
	store := &gormdb.Store{
		Name:       storeName,
		IsActive:   false, // pending until verified by a superadmin
		IsVerified: false,
	}
	if err := s.repos.Store.Create(ctx, store); err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to create store", err)
	}

	// Seed default COA for the new store (non-fatal: auto-posting will still
	// fail loudly on first transaction if the seed is missing).
	if err := accounting.EnsureCOAForStore(ctx, s.repos.DB(), store.ID); err != nil {
		_ = err // best-effort; GetAccountID will surface a clear error later
	}

	// Create user with the admin role and store_id set to the new store
	user := &gormdb.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		RoleID:       parseUUIDToUUID(adminRoleID),
		StoreID:      store.ID,
		IsActive:     true,
	}

	if err := s.repos.User.Create(ctx, user); err != nil {
		return nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to create user", err)
	}

	// Registration assigns the admin role for the auto-created store.
	tokenPair, err := s.issueTokenPair(ctx, user, deviceID, "admin")
	if err != nil {
		return nil, nil, err
	}

	return tokenPair, user, nil
}

// DecodeAccessToken validates a JWT access token and returns its claims.
// Used by auth responses to attach role permissions to refreshed sessions.
func (s *Service) DecodeAccessToken(tokenString string) (*auth.Claims, error) {
	return s.auth.ValidateToken(tokenString)
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, username, password, deviceID string) (*auth.TokenPair, *gormdb.User, error) {
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

	// Issue a device-bound token pair
	tokenPair, err := s.issueTokenPair(ctx, user, deviceID, user.Role.Name)
	if err != nil {
		return nil, nil, err
	}

	// Update last login
	_ = s.repos.User.UpdateLastLogin(ctx, user.ID.String())

	return tokenPair, user, nil
}

// issueTokenPair generates an access token and a device-bound refresh token,
// persisting only the refresh token hash. The raw refresh token is returned to
// the client. Called from Login/Register only (RefreshToken reuses the
// presented token — see RefreshToken).
func (s *Service) issueTokenPair(ctx context.Context, user *gormdb.User, deviceID, roleName string) (*auth.TokenPair, error) {
	accessToken, expiresAt, err := s.auth.GenerateAccessToken(
		user.ID.String(),
		user.Username,
		user.Email,
		user.RoleID.String(),
		roleName,
		user.StoreID.String(),
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate access token", err)
	}

	// Reuse the existing active refresh token for this device when the caller
	// is refreshing the session (so refresh is idempotent and concurrent
	// refreshes don't create a new token each time). Login/register go through
	// this path too but with no existing token they create a fresh one.
	// We return the *raw* token only on creation; on refresh we must return
	// the same raw token, so the caller must supply it via refresh (see
	// RefreshToken). issueTokenPair is only called from Login/Register (fresh
	// token) and RefreshToken (which now reuses the presented token).
	rawRefresh, hash, refreshExpiresAt, err := s.auth.GenerateRefreshToken()
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate refresh token", err)
	}

	// Persist only the hash, bound to the device so a token used from any other
	// device is rejected.
	rt := &gormdb.RefreshToken{
		UserID:    user.ID,
		Token:     hash,
		DeviceID:  deviceID,
		ExpiresAt: refreshExpiresAt,
		IsRevoked: false,
	}
	if err := s.repos.RefreshToken.Create(ctx, rt); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to store refresh token", err)
	}

	return &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshToken exchanges a valid device-bound refresh token for a new access
// token. The refresh token itself is NOT rotated (it stays valid until
// explicit logout or expiry), so concurrent refreshes — e.g. a page guard and
// a 401 interceptor firing at the same time after rapid navigation — are
// idempotent and cannot race into a logout.
func (s *Service) RefreshToken(ctx context.Context, refreshToken, deviceID string) (*auth.TokenPair, error) {
	tokenHash := s.auth.HashToken(refreshToken)

	// Look up the active token for this hash + device. A device mismatch,
	// revoked token, or expired token all resolve to NOT_FOUND => invalid.
	rt, err := s.repos.RefreshToken.GetActiveByTokenHash(ctx, tokenHash, deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrInvalidToken
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to validate refresh token", err)
	}

	// Get user
	user, err := s.repos.User.GetByID(ctx, rt.UserID.String())
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

	// Issue a new access token but reuse the SAME refresh token (no rotation,
	// no DB writes for the refresh token). Returning the presented raw token
	// keeps the client trivially in sync.
	accessToken, expiresAt, err := s.auth.GenerateAccessToken(
		user.ID.String(),
		user.Username,
		user.Email,
		user.RoleID.String(),
		user.Role.Name,
		user.StoreID.String(),
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to generate access token", err)
	}

	return &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout revokes the presented refresh token (optionally scoped to a device).
func (s *Service) Logout(ctx context.Context, refreshToken, deviceID string) error {
	if refreshToken == "" {
		return nil
	}
	tokenHash := s.auth.HashToken(refreshToken)
	return s.repos.RefreshToken.RevokeByTokenHash(ctx, tokenHash, deviceID)
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
