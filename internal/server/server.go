package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	repos := repository.New(db.Queries())
	authService := auth.NewAuthService(cfg)
	userService := user.NewService(repos, authService)

	return &Server{
		config:       cfg,
		db:           db,
		repositories: repos,
		authService:  authService,
		userService:  userService,
	}
}

// Start starts both gRPC and HTTP (REST) servers
func (s *Server) Start() error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.startGRPC(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Start HTTP/REST server (grpc-gateway)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.startHTTP(); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for first error
	err := <-errChan
	return err
}

// startGRPC starts the gRPC server
func (s *Server) startGRPC() error {
	addr := fmt.Sprintf(":%d", s.config.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryServerRecoveryInterceptor(),
			middleware.UnaryServerLoggingInterceptor(),
			middleware.UnaryServerRequestIDInterceptor(),
			middleware.UnaryServerAuthInterceptor(s.authService),
		),
	)

	s.registerServices()
	reflection.Register(s.grpcServer)

	log.Printf("gRPC server starting on :%d", s.config.GRPC.Port)
	return s.grpcServer.Serve(lis)
}

// startHTTP starts the HTTP/REST server with grpc-gateway
func (s *Server) startHTTP() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Register AuthService
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register auth gateway: %w", err)
	}

	// Register UserService
	if err := userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register user gateway: %w", err)
	}

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", s.handleSwaggerUI)
	httpMux.HandleFunc("/swagger.json", s.handleSwaggerJSON)
	httpMux.HandleFunc("/health", s.handleHealth)
	httpMux.Handle("/api/", enableCORS(mux))

	addr := fmt.Sprintf(":%d", s.config.HTTP.Port)
	log.Printf("HTTP/REST server starting on %s", addr)
	return http.ListenAndServe(addr, httpMux)
}

func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(getSwaggerUIHTML()))
}

func (s *Server) handleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	data, err := readFile("docs/swagger-enhanced.json")
	if err != nil {
		data, err = readFile("gen/api.swagger.json")
		if err != nil {
			http.Error(w, `{"error": "swagger file not found"}`, http.StatusInternalServerError)
			return
		}
	}
	w.Write(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"status":"ok","grpc_port":%d,"http_port":%d}`, s.config.GRPC.Port, s.config.HTTP.Port)))
}

func (s *Server) Stop() {
	log.Println("Shutting down servers...")
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	log.Println("Servers stopped")
}

func (s *Server) registerServices() {
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s.grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.AuthService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("user.v1.UserService", healthpb.HealthCheckResponse_SERVING)

	authHandler := handler.NewAuthHandler(s.userService)
	authv1.RegisterAuthServiceServer(s.grpcServer, authHandler)

	userHandler := handler.NewUserHandler(s.userService)
	userv1.RegisterUserServiceServer(s.grpcServer, userHandler)

	log.Println("All services registered")
}

func enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func readFile(path string) ([]byte, error) {
	f, err := http.Dir(".").Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, 0, 1024*1024)
	tmp := make([]byte, 1024)
	for {
		n, err := f.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

func getSwaggerUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>POS API Documentation</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css" />
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .topbar { display: none; }
    .nav { background: #fff; padding: 10px 20px; border-bottom: 1px solid #e0e0e0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
    .nav { display: flex; align-items: center; gap: 16px; }
    .nav a { color: #1976d2; text-decoration: none; font-weight: 500; }
    .nav a:hover { text-decoration: underline; }
    .nav a.adminer { background: #00695c; color: #fff; padding: 6px 14px; border-radius: 6px; font-weight: 600; }
    .nav a.adminer:hover { background: #004d40; text-decoration: none; }
    .nav-right { margin-left: auto; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="/">Swagger UI</a>
    <a href="/swagger.json">Raw Spec</a>
    <span class="nav-right"></span>
    <a href="http://localhost:8081" target="_blank" class="adminer">🗄️ Adminer</a>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger.json",
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: "StandaloneLayout",
      requestInterceptor: function(req) {
        // Force requests to go to the correct API host
        var apiHost = window.location.hostname + ':8080';
        if (req.url.startsWith('/')) {
          req.url = 'http://' + apiHost + req.url;
        }
        return req;
      }
    })
  </script>
</body>
</html>`
}
