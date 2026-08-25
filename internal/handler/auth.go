package handler

import (
	"context"

	authv1 "apiPos/gen/auth/v1"
	"apiPos/internal/apperror"
	"apiPos/internal/response"
	"apiPos/internal/user"
)

// AuthHandler implements the AuthService gRPC server
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	userService *user.Service
}

// NewAuthHandler creates a new auth Handler
func NewAuthHandler(userService *user.Service) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// Register creates a new user account
func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	tokenPair, u, err := h.userService.Register(ctx, req.Username, req.Email, req.Password, req.FullName)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &authv1.RegisterResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		User: &authv1.User{
			Id:       u.ID.String(),
			Username: u.Username,
			Email:    u.Email,
			FullName: u.FullName,
			RoleId:   u.RoleID.String(),
			StoreId:  u.StoreID.String(),
			IsActive: u.IsActive,
		},
	}, nil
}

// Login authenticates a user and returns tokens
func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokenPair, u, err := h.userService.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &authv1.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		User: &authv1.User{
			Id:       u.ID.String(),
			Username: u.Username,
			Email:    u.Email,
			FullName: u.FullName,
			RoleId:   u.RoleID.String(),
			StoreId:  u.StoreID.String(),
			IsActive: u.IsActive,
		},
	}, nil
}

// RefreshToken refreshes an access token
func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokenPair, err := h.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
	}, nil
}

// Logout logs out a user
func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	return &authv1.LogoutResponse{
		Success: true,
		Message: response.MsgLogoutSuccess,
	}, nil
}
