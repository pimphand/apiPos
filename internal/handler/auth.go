package handler

import (
	"context"
	"log/slog"

	authv1 "apiPos/gen/auth/v1"
	"apiPos/internal/apperror"
	"apiPos/internal/response"
	"apiPos/internal/user"
)

// AuthHandler implements the AuthService gRPC server
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	userService *user.Service
	logs        *ActivityLogger
}

// NewAuthHandler creates a new auth Handler
func NewAuthHandler(userService *user.Service, logs *ActivityLogger) *AuthHandler {
	return &AuthHandler{userService: userService, logs: logs}
}

// Register creates a new user account
func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	tokenPair, u, err := h.userService.Register(ctx, req.Username, req.Email, req.Password, req.FullName, req.DeviceId, req.StoreName)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.LogWithUser(ctx, u.ID.String(), u.StoreID.String(), u.Username, "CREATE", "users", u.ID.String(),
		`{"username":"`+u.Username+`","email":"`+u.Email+`"}`)

	userProto := &authv1.User{
		Id:       u.ID.String(),
		Username: u.Username,
		Email:    u.Email,
		FullName: u.FullName,
		RoleId:   u.RoleID.String(),
		RoleName: registerRoleName(u.Role.Name),
		StoreId:  u.StoreID.String(),
		IsActive: u.IsActive,
	}
	userProto.Permissions = h.rolePermissions(ctx, u.RoleID.String())

	return &authv1.RegisterResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		User:         userProto,
	}, nil
}

// registerRoleName returns the role name for a newly registered user.
// Registration always assigns the admin role, so fall back if not populated.
func registerRoleName(name string) string {
	if name == "" {
		return "admin"
	}
	return name
}

// Login authenticates a user and returns tokens
func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokenPair, u, err := h.userService.Login(ctx, req.Username, req.Password, req.DeviceId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.LogWithUser(ctx, u.ID.String(), u.StoreID.String(), u.Username, "LOGIN", "auth", u.ID.String(),
		`{"username":"`+u.Username+`"}`)

	userProto := &authv1.User{
		Id:       u.ID.String(),
		Username: u.Username,
		Email:    u.Email,
		FullName: u.FullName,
		RoleId:   u.RoleID.String(),
		RoleName: u.Role.Name,
		StoreId:  u.StoreID.String(),
		IsActive: u.IsActive,
	}
	userProto.Permissions = h.rolePermissions(ctx, u.RoleID.String())

	return &authv1.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		User:         userProto,
	}, nil
}

// RefreshToken refreshes an access token
func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokenPair, err := h.userService.RefreshToken(ctx, req.RefreshToken, req.DeviceId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Parse the freshly minted access token to learn the user's role so the
	// response can carry the permission names (no separate permissions request
	// needed on the client). Best-effort: if the token can't be decoded, return
	// an empty permission list rather than failing the refresh.
	var permissions []string
	if claims, err := h.userService.DecodeAccessToken(tokenPair.AccessToken); err == nil {
		permissions = h.rolePermissions(ctx, claims.RoleID)
	} else {
		slog.Warn("auth refresh: failed to decode access token for permissions", "error", err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		Permissions:  permissions,
	}, nil
}

// rolePermissions returns the permission names for a role, tolerating lookup
// failures (returns nil so the auth response still succeeds).
func (h *AuthHandler) rolePermissions(ctx context.Context, roleID string) []string {
	perms, err := h.userService.RolePermissionNames(ctx, roleID)
	if err != nil {
		slog.Warn("auth: failed to load role permissions", "role_id", roleID, "error", err)
		return nil
	}
	return perms
}

// Logout logs out a user by revoking the presented refresh token
func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := h.userService.Logout(ctx, req.RefreshToken, req.DeviceId); err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "LOGOUT", "auth", "", `{}`)

	return &authv1.LogoutResponse{
		Success: true,
		Message: response.MsgLogoutSuccess,
	}, nil
}
