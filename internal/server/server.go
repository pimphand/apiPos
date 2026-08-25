package server

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

authv1 "apiPos/gen/auth/v1"
userv1 "apiPos/gen/user/v1"
	"apiPos/internal/auth"
	"apiPos/internal/config"
	"apiPos/internal/database"
	"apiPos/internal/handler"
	"apiPos/internal/repository"
	"apiPos/internal/user"
	"apiPos/middleware"
)

// Server holds the gRPC server and its dependencies
type Server struct {
	grpcServer   *grpc.Server
	config       *config.Config
	db           database.Service
	repositories *repository.Repositories
	authService  *auth.AuthService
	userService  *user.Service
}

// New creates a new gRPC server
func New(cfg *config.Config, db database.Service) *Server {
	// Create repositories
	repos := repository.New(db.Queries())

	// Create auth service
	authService := auth.NewAuthService(cfg)

	// Create user service
	userService := user.NewService(repos, authService)

	return &Server{
		config:       cfg,
		db:           db,
		repositories: repos,
		authService:  authService,
		userService:  userService,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Create gRPC server with interceptors
	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryServerRecoveryInterceptor(),
			middleware.UnaryServerLoggingInterceptor(),
			middleware.UnaryServerRequestIDInterceptor(),
			middleware.UnaryServerAuthInterceptor(s.authService),
		),
	)

	// Register services
	s.registerServices()

	// Enable server reflection for grpcurl
	reflection.Register(s.grpcServer)

	log.Printf("gRPC server starting on %s", addr)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	log.Println("Shutting down gRPC server...")
	s.grpcServer.GracefulStop()
	log.Println("gRPC server stopped")
}

// registerServices registers all gRPC services
func (s *Server) registerServices() {
	// Register gRPC health service (standard health check)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s.grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.AuthService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("user.v1.UserService", healthpb.HealthCheckResponse_SERVING)

	// Register auth service
	authHandler := handler.NewAuthHandler(s.userService)
	authv1.RegisterAuthServiceServer(s.grpcServer, authHandler)

	// Register user service
	userHandler := handler.NewUserHandler(s.userService)
	userv1.RegisterUserServiceServer(s.grpcServer, userHandler)

	log.Println("All services registered")
}
