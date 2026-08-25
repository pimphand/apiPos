package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"apiPos/internal/auth"
)

// UnaryServerAuthInterceptor returns a new unary server interceptor for authentication
func UnaryServerAuthInterceptor(authService *auth.AuthService) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for public endpoints
		if isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization header
		authorization := md.Get("authorization")
		if len(authorization) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token
		token := strings.TrimPrefix(authorization[0], "Bearer ")
		if token == authorization[0] {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			switch err {
			case auth.ErrTokenExpired:
				return nil, status.Errorf(codes.Unauthenticated, "token expired")
			case auth.ErrInvalidToken:
				return nil, status.Errorf(codes.Unauthenticated, "invalid token")
			default:
				return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
			}
		}

		// Add claims to context
		ctx = auth.ContextWithClaims(ctx, claims)

		return handler(ctx, req)
	}
}

// UnaryServerRoleInterceptor returns a new unary server interceptor for role-based authorization
func UnaryServerRoleInterceptor(allowedRoles ...string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Get claims from context
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing claims")
		}

		// Check if user has required role
		if !hasRole(claims.RoleName, allowedRoles) {
			return nil, status.Errorf(codes.PermissionDenied, "insufficient permissions")
		}

		return handler(ctx, req)
	}
}

// isPublicEndpoint checks if the endpoint is public (no auth required)
func isPublicEndpoint(method string) bool {
	publicEndpoints := []string{
		"/auth.v1.AuthService/Login",
		"/auth.v1.AuthService/RefreshToken",
		"/grpc.health.v1.Health/Check",
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}

	for _, endpoint := range publicEndpoints {
		if method == endpoint {
			return true
		}
	}
	return false
}

// hasRole checks if the user has one of the allowed roles
func hasRole(userRole string, allowedRoles []string) bool {
	if len(allowedRoles) == 0 {
		return true
	}

	for _, role := range allowedRoles {
		if userRole == role {
			return true
		}
	}
	return false
}
