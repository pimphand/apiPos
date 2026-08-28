package server

import (
	"context"
	"encoding/json"
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

	activitylogv1 "apiPos/gen/activitylog/v1"
	authv1 "apiPos/gen/auth/v1"
	categoryv1 "apiPos/gen/category/v1"
	inventoryv1 "apiPos/gen/inventory/v1"
	permissionv1 "apiPos/gen/permission/v1"
	productv1 "apiPos/gen/product/v1"
	productskuv1 "apiPos/gen/productsku/v1"
	rolev1 "apiPos/gen/role/v1"
	userv1 "apiPos/gen/user/v1"
	"apiPos/internal/activitylog"
	"apiPos/internal/auth"
	"apiPos/internal/category"
	"apiPos/internal/config"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/handler"
	"apiPos/internal/inventory"
	"apiPos/internal/permission"
	"apiPos/internal/product"
	"apiPos/internal/productsku"
	"apiPos/internal/repository"
	"apiPos/internal/role"
	"apiPos/internal/user"
	"apiPos/middleware"
)

// Server holds the gRPC server and its dependencies
type Server struct {
	grpcServer          *grpc.Server
	config              *config.Config
	db                  *gormdb.DB
	repositories        *repository.Repositories
	authService         *auth.AuthService
	userService         *user.Service
	categoryService     *category.Service
	productService      *product.Service
	productSKUService   *productsku.Service
	inventoryService    *inventory.Service
	roleService         *role.Service
	permissionService   *permission.Service
	activityLogService  *activitylog.Service
}

// New creates a new gRPC server
func New(cfg *config.Config, db *gormdb.DB) *Server {
	repos := repository.New(db.GetDB())
	authService := auth.NewAuthService(cfg)
	userService := user.NewService(repos, authService)
	categoryService := category.NewService(repos)
	productService := product.NewService(repos)
	productSKUService := productsku.NewService(repos.ProductSKU)
	inventoryService := inventory.NewService(repos)
	roleService := role.NewService(repos)
	permissionService := permission.NewService(repos)
	activityLogService := activitylog.NewService(repos)

	return &Server{
		config:             cfg,
		db:                 db,
		repositories:       repos,
		authService:        authService,
		userService:        userService,
		categoryService:    categoryService,
		productService:     productService,
		productSKUService:  productSKUService,
		inventoryService:   inventoryService,
		roleService:        roleService,
		permissionService:  permissionService,
		activityLogService: activityLogService,
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

	// Register CategoryService
	if err := categoryv1.RegisterCategoryServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register category gateway: %w", err)
	}

	// Register ProductService
	if err := productv1.RegisterProductServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register product gateway: %w", err)
	}

	// Register ProductSKUService
	if err := productskuv1.RegisterProductServiceSKUServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register product SKU gateway: %w", err)
	}

	// Register RoleService
	if err := rolev1.RegisterRoleServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register role gateway: %w", err)
	}

	// Register PermissionService
	if err := permissionv1.RegisterPermissionServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register permission gateway: %w", err)
	}

	// Register InventoryService
	if err := inventoryv1.RegisterInventoryServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register inventory gateway: %w", err)
	}

	// Register ActivityLogService
	if err := activitylogv1.RegisterActivityLogServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register activity log gateway: %w", err)
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

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err == nil {
		spec["host"] = r.Host
		data, _ = json.Marshal(spec)
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
	healthServer.SetServingStatus("category.v1.CategoryService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("product.v1.ProductService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("productsku.v1.ProductServiceSKUService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("inventory.v1.InventoryService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("role.v1.RoleService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("permission.v1.PermissionService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("activitylog.v1.ActivityLogService", healthpb.HealthCheckResponse_SERVING)

	authHandler := handler.NewAuthHandler(s.userService)
	authv1.RegisterAuthServiceServer(s.grpcServer, authHandler)

	userHandler := handler.NewUserHandler(s.userService)
	userv1.RegisterUserServiceServer(s.grpcServer, userHandler)

	categoryHandler := handler.NewCategoryHandler(s.categoryService)
	categoryv1.RegisterCategoryServiceServer(s.grpcServer, categoryHandler)

	productHandler := handler.NewProductHandler(s.productService, s.repositories)
	productv1.RegisterProductServiceServer(s.grpcServer, productHandler)

	productSKUHandler := handler.NewProductSKUHandler(s.productSKUService)
	productskuv1.RegisterProductServiceSKUServiceServer(s.grpcServer, productSKUHandler)

	inventoryHandler := handler.NewInventoryHandler(s.inventoryService)
	inventoryv1.RegisterInventoryServiceServer(s.grpcServer, inventoryHandler)

	roleHandler := handler.NewRoleHandler(s.roleService)
	rolev1.RegisterRoleServiceServer(s.grpcServer, roleHandler)

	permissionHandler := handler.NewPermissionHandler(s.permissionService)
	permissionv1.RegisterPermissionServiceServer(s.grpcServer, permissionHandler)

	activityLogHandler := handler.NewActivityLogHandler(s.activityLogService)
	activitylogv1.RegisterActivityLogServiceServer(s.grpcServer, activityLogHandler)

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
    .token-status { font-size: 12px; padding: 4px 8px; border-radius: 4px; margin-left: 8px; }
    .token-valid { background: #4caf50; color: white; }
    .token-expired { background: #f44336; color: white; }
    .token-none { background: #9e9e9e; color: white; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="/">Swagger UI</a>
    <a href="/swagger.json">Raw Spec</a>
    <span id="token-status" class="token-status token-none">No Token</span>
    <span class="nav-right"></span>
    <a href="http://localhost:8081" target="_blank" class="adminer">🗄️ Adminer</a>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
  <script>
    // Token Management with localStorage
    var TokenManager = {
      ACCESS_TOKEN_KEY: 'pos_access_token',
      REFRESH_TOKEN_KEY: 'pos_refresh_token',
      TOKEN_EXPIRY_KEY: 'pos_token_expiry',

      setTokens: function(accessToken, refreshToken, expiresIn) {
        localStorage.setItem(this.ACCESS_TOKEN_KEY, accessToken);
        if (refreshToken) {
          localStorage.setItem(this.REFRESH_TOKEN_KEY, refreshToken);
        }
        if (expiresIn) {
          var expiry = new Date().getTime() + (expiresIn * 1000);
          localStorage.setItem(this.TOKEN_EXPIRY_KEY, expiry.toString());
        }
        this.updateStatus();
      },

      getAccessToken: function() {
        return localStorage.getItem(this.ACCESS_TOKEN_KEY);
      },

      getRefreshToken: function() {
        return localStorage.getItem(this.REFRESH_TOKEN_KEY);
      },

      isExpired: function() {
        var expiry = localStorage.getItem(this.TOKEN_EXPIRY_KEY);
        if (!expiry) return true;
        return new Date().getTime() > parseInt(expiry);
      },

      clearTokens: function() {
        localStorage.removeItem(this.ACCESS_TOKEN_KEY);
        localStorage.removeItem(this.REFRESH_TOKEN_KEY);
        localStorage.removeItem(this.TOKEN_EXPIRY_KEY);
        this.updateStatus();
      },

      updateStatus: function() {
        var statusEl = document.getElementById('token-status');
        var token = this.getAccessToken();
        if (!token) {
          statusEl.textContent = 'No Token';
          statusEl.className = 'token-status token-none';
        } else if (this.isExpired()) {
          statusEl.textContent = 'Token Expired';
          statusEl.className = 'token-status token-expired';
        } else {
          statusEl.textContent = 'Token Active';
          statusEl.className = 'token-status token-valid';
        }
      }
    };

    // Initialize Swagger UI with token support
    var ui = SwaggerUIBundle({
      url: "/swagger.json",
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: "StandaloneLayout",
      onComplete: function() {
        // Load saved token into Authorization header
        var savedToken = TokenManager.getAccessToken();
        if (savedToken && !TokenManager.isExpired()) {
          ui.preauthorizeApiKey('BearerAuth', 'Bearer ' + savedToken);
        }
        TokenManager.updateStatus();
      },
      requestInterceptor: function(req) {
        if (req.url === '/swagger.json') return req;
        var apiHost = window.location.hostname + ':' + window.location.port;
        if (req.url.startsWith('/')) {
          req.url = window.location.protocol + '//' + apiHost + req.url;
        }

        // Auto-add token to requests
        var token = TokenManager.getAccessToken();
        if (token && !TokenManager.isExpired() && req.headers) {
          req.headers['Authorization'] = 'Bearer ' + token;
        }

        return req;
      },
      responseInterceptor: function(res) {
        // Try to extract tokens from login response
        try {
          if (res.url && res.url.includes('/auth.v1.AuthService/Login')) {
            var body = JSON.parse(res.text);
            if (body.accessToken) {
              TokenManager.setTokens(body.accessToken, body.refreshToken, body.expiresIn || 900);
              console.log('Tokens saved to localStorage');
            }
          }
        } catch(e) {}
        return res;
      }
    });

    // Check token status on load
    TokenManager.updateStatus();

    // Refresh token status every 30 seconds
    setInterval(function() {
      TokenManager.updateStatus();
    }, 30000);
  </script>
</body>
</html>`
}
