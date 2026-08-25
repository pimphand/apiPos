package handler

import (
	"context"

	authv1 "apiPos/gen/auth/v1"
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

// Login authenticates a user and returns tokens
func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokenPair, u, err := h.userService.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		User: &authv1.User{
			Id:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			FullName: u.FullName,
			RoleId:   u.RoleID,
			StoreId:  u.StoreID.Int64,
			IsActive: u.IsActive,
		},
	}, nil
}

// RefreshToken refreshes an access token
func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokenPair, err := h.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
	}, nil
}

// Logout logs out a user
func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	// In a real implementation, you would revoke the refresh token
	return &authv1.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}
