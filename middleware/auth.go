package middleware

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	"apiPos/internal/repository"
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
			return nil, status.Error(codes.Unauthenticated, "Missing metadata")
		}

		// Get authorization header
		authorization := md.Get("authorization")
		if len(authorization) == 0 {
			return nil, status.Error(codes.Unauthenticated, "Missing authorization header")
		}

		// Extract token
		token := strings.TrimPrefix(authorization[0], "Bearer ")
		if token == authorization[0] {
			return nil, status.Error(codes.Unauthenticated, "Invalid authorization header format")
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
		switch err {
		case apperror.ErrTokenExpired:
			return nil, status.Error(codes.Unauthenticated, "Token expired")
		case apperror.ErrInvalidToken:
			return nil, status.Error(codes.Unauthenticated, "Invalid token")
		default:
			return nil, status.Error(codes.Unauthenticated, "Unauthorized")
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
			return nil, status.Error(codes.Unauthenticated, "Missing claims")
		}

		// Check if user has required role
		if !hasRole(claims.RoleName, allowedRoles) {
			return nil, status.Error(codes.PermissionDenied, "Insufficient permissions")
		}

		return handler(ctx, req)
	}
}

// isPublicEndpoint checks if the endpoint is public (no auth required)
func isPublicEndpoint(method string) bool {
	publicEndpoints := []string{
		"/auth.v1.AuthService/Register",
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

// UnaryServerPermissionInterceptor returns a new unary server interceptor for
// permission-based authorization.  It looks up the user's role in the
// role_permissions table and verifies the role holds the required permission.
//
// endpointPermissions maps full gRPC method paths (e.g.
// "/customer.v1.CustomerService/CreateCustomer") to the permission string
// required to invoke them (e.g. "customers.create").  Methods not present
// in the map are NOT gated and pass through freely.
func UnaryServerPermissionInterceptor(repos *repository.Repositories, endpointPermissions map[string]string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		requiredPerm, needsCheck := endpointPermissions[info.FullMethod]
		if !needsCheck {
			return handler(ctx, req)
		}

		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Missing claims")
		}

		// superadmin bypasses all permission checks
		if claims.RoleName == "superadmin" {
			slog.Info("permission check: superadmin bypass",
				"method", info.FullMethod,
				"role", claims.RoleName,
			)
			return handler(ctx, req)
		}

		// Look up the permissions granted to this user's role from the DB.
		var permNames []string
		err := repos.DB().
			Table("permissions").
			Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
			Where("role_permissions.role_id = ?", claims.RoleID).
			Pluck("permissions.name", &permNames).Error
		if err != nil {
			slog.Error("permission check: db query failed",
				"method", info.FullMethod,
				"role_id", claims.RoleID,
				"error", err,
			)
			return nil, status.Error(codes.Internal, "Failed to check permissions")
		}

		for _, name := range permNames {
			if name == requiredPerm {
				return handler(ctx, req)
			}
		}

		slog.Warn("permission check: DENIED",
			"method", info.FullMethod,
			"required", requiredPerm,
			"role_id", claims.RoleID,
			"role_name", claims.RoleName,
			"actual_perms", permNames,
		)

		return nil, status.Error(codes.PermissionDenied, "Insufficient permissions")
	}
}
