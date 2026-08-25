package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerLoggingInterceptor returns a new unary server interceptor for logging
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Get status code
		st, _ := status.FromError(err)

		// Log request
		slog.Info("gRPC request",
			"method", info.FullMethod,
			"duration", duration.String(),
			"status", st.Code().String(),
			"error", err,
		)

		return resp, err
	}
}

// UnaryServerRecoveryInterceptor returns a new unary server interceptor for recovery
func UnaryServerRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					"method", info.FullMethod,
					"error", r,
				)
				err = grpc.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// UnaryServerRequestIDInterceptor returns a new unary server interceptor for request ID
func UnaryServerRequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Generate or extract request ID
		// For simplicity, we'll just pass through the context
		return handler(ctx, req)
	}
}
