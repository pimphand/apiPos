package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	activitylogv1 "apiPos/gen/activitylog/v1"
	accountingv1 "apiPos/gen/accounting/v1"
	authv1 "apiPos/gen/auth/v1"
	cashierpb "apiPos/gen/cashier/v1"
	categoryv1 "apiPos/gen/category/v1"
	customerv1 "apiPos/gen/customer/v1"
	inventoryv1 "apiPos/gen/inventory/v1"
	paymentv1 "apiPos/gen/payment/v1"
	permissionv1 "apiPos/gen/permission/v1"
	productv1 "apiPos/gen/product/v1"
	productskuv1 "apiPos/gen/productsku/v1"
	reportv1 "apiPos/gen/report/v1"
	rolev1 "apiPos/gen/role/v1"
	salev1 "apiPos/gen/sale/v1"
	selforderv1 "apiPos/gen/selforder/v1"
	settingsv1 "apiPos/gen/settings/v1"
	storev1 "apiPos/gen/store/v1"
	tablev1 "apiPos/gen/table/v1"
	userv1 "apiPos/gen/user/v1"
	"apiPos/internal/accounting"
	accountexport "apiPos/internal/accounting/export"
	accountringreport "apiPos/internal/accounting/report"
	"apiPos/internal/activitylog"
	"apiPos/internal/auth"
	"apiPos/internal/cashier"
	"apiPos/internal/category"
	"apiPos/internal/config"
	"apiPos/internal/customer"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/handler"
	"apiPos/internal/inventory"
	"apiPos/internal/payment"
	"apiPos/internal/permission"
	"apiPos/internal/product"
	"apiPos/internal/productsku"
	"apiPos/internal/realtime"
	"apiPos/internal/report"
	"apiPos/internal/repository"
	"apiPos/internal/role"
	"apiPos/internal/sale"
	"apiPos/internal/selforder"
	"apiPos/internal/settings"
	"apiPos/internal/store"
	"apiPos/internal/table"
	"apiPos/internal/user"
	"apiPos/middleware"
)

// Server holds the gRPC server and its dependencies
type Server struct {
	grpcServer         *grpc.Server
	config             *config.Config
	db                 *gormdb.DB
	repositories       *repository.Repositories
	authService        *auth.AuthService
	userService        *user.Service
	categoryService    *category.Service
	productService     *product.Service
	productSKUService  *productsku.Service
	inventoryService   *inventory.Service
	customerService    *customer.Service
	saleService        *sale.Service
	paymentService     *payment.Service
	cashierService     *cashier.Service
	reportService      *report.Service
	accountingReport   *accountringreport.Service
	accountingAdmin    *accounting.AdminService
	accountingExport   *accountexport.Service
	accountingPosting  *accounting.PostingService
	roleService        *role.Service
	permissionService  *permission.Service
	activityLogService *activitylog.Service
	storeService       *store.Service
	settingsService    *settings.Service
	selfOrderService   *selforder.Service
	tableService       *table.Service
	realtimeHub        *realtime.Hub
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
	customerService := customer.NewService(repos)
	saleService := sale.NewService(repos)
	paymentService := payment.NewService(repos)
	cashierService := cashier.NewService(repos)
	reportService := report.NewService(repos)
	accountingReport := accountringreport.NewService(repos)
	accountingAdmin := accounting.NewAdminService(repos)
	accountingExport := accountexport.NewService(accountingReport)
	accountingPosting := accounting.NewPostingService(repos.DB())
	roleService := role.NewService(repos)
	permissionService := permission.NewService(repos)
	activityLogService := activitylog.NewService(repos)
	storeService := store.NewService(repos)
	settingsService := settings.NewService(repos)
	selfOrderService := selforder.NewService(repos)
	tableService := table.NewService(repos)

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
		customerService:    customerService,
		saleService:        saleService,
		paymentService:     paymentService,
		cashierService:     cashierService,
		reportService:      reportService,
		accountingReport:   accountingReport,
		accountingAdmin:    accountingAdmin,
		accountingExport:   accountingExport,
		accountingPosting:  accountingPosting,
		roleService:        roleService,
		permissionService:  permissionService,
		activityLogService: activityLogService,
		storeService:       storeService,
		settingsService:    settingsService,
		selfOrderService:   selfOrderService,
		tableService:       tableService,
		realtimeHub:        realtime.Default(),
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
			middleware.UnaryServerRateLimitInterceptor(),
			middleware.UnaryServerAuthInterceptor(s.authService),
			middleware.UnaryServerPermissionInterceptor(s.repositories, endpointPermissions()),
		),
	)

	s.registerServices()
	reflection.Register(s.grpcServer)

	// Periodic cleanup: cancel self orders stuck in PENDING past the payment
	// timeout and restore their stock (docs/pos-self-order.md §6.2).
	go s.runExpiredOrderCleanup()

	log.Printf("gRPC server starting on :%d", s.config.GRPC.Port)
	return s.grpcServer.Serve(lis)
}

// runExpiredOrderCleanup periodically cancels expired PENDING self orders.
func (s *Server) runExpiredOrderCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		n, err := s.selfOrderService.CancelExpiredPending(ctx)
		if err != nil {
			log.Printf("selforder cleanup: %v", err)
		} else if n > 0 {
			log.Printf("selforder cleanup: cancelled %d expired order(s)", n)
		}
		cancel()
	}
}

// startHTTP starts the HTTP/REST server with grpc-gateway
func (s *Server) startHTTP() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The REST gateway must serialize proto messages using the original proto
	// field names (snake_case), matching the .proto definitions, the swagger
	// docs, and the frontend types. protojson's default json_name (camelCase)
	// output silently breaks every client that reads snake_case fields.
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
		}),
	)
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

	// Register CustomerService
	if err := customerv1.RegisterCustomerServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register customer gateway: %w", err)
	}

	// Register SaleService
	if err := salev1.RegisterSaleServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register sale gateway: %w", err)
	}

	// Register PaymentService
	if err := paymentv1.RegisterPaymentServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register payment gateway: %w", err)
	}

	// Register CashierService
	if err := cashierpb.RegisterCashierServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register cashier gateway: %w", err)
	}

	// Register ReportService
	if err := reportv1.RegisterReportServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register report gateway: %w", err)
	}

	// Register ActivityLogService
	if err := activitylogv1.RegisterActivityLogServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register activity log gateway: %w", err)
	}

	// Register StoreService (superadmin)
	if err := storev1.RegisterStoreServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register store gateway: %w", err)
	}

	// Register SettingsService (tenant store settings)
	if err := settingsv1.RegisterSettingsServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register settings gateway: %w", err)
	}

	// Register SelfOrderService (public, unauthenticated)
	if err := selforderv1.RegisterSelfOrderServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register self-order gateway: %w", err)
	}

	// Register TableService (store-scoped admin CRUD)
	if err := tablev1.RegisterTableServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register table gateway: %w", err)
	}

	// Register AccountingService
	if err := accountingv1.RegisterAccountingServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", s.config.GRPC.Port), opts); err != nil {
		return fmt.Errorf("failed to register accounting gateway: %w", err)
	}

	importHandler := handler.NewProductImportHandler(s.productService, s.categoryService, s.repositories, s.authService, s.config)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", s.handleSwaggerUI)
	httpMux.HandleFunc("/swagger.json", s.handleSwaggerJSON)
	httpMux.HandleFunc("/health", s.handleHealth)
	httpMux.HandleFunc("/ws", realtime.Handler(s.realtimeHub, s.authService))
	httpMux.HandleFunc("/api/v1/products/import", importHandler.ServeHTTP)
	httpMux.HandleFunc("/api/v1/products/import-sample", importHandler.ServeSampleCSV)
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
	healthServer.SetServingStatus("customer.v1.CustomerService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("sale.v1.SaleService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("payment.v1.PaymentService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("cashier.v1.CashierService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("report.v1.ReportService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("accounting.v1.AccountingService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("role.v1.RoleService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("permission.v1.PermissionService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("activitylog.v1.ActivityLogService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("store.v1.StoreService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("settings.v1.SettingsService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("selforder.v1.SelfOrderService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("table.v1.TableService", healthpb.HealthCheckResponse_SERVING)

	logs := handler.NewActivityLogger(s.activityLogService)

	authHandler := handler.NewAuthHandler(s.userService, logs)
	authv1.RegisterAuthServiceServer(s.grpcServer, authHandler)

	userHandler := handler.NewUserHandler(s.userService, logs)
	userv1.RegisterUserServiceServer(s.grpcServer, userHandler)

	categoryHandler := handler.NewCategoryHandler(s.categoryService, logs)
	categoryv1.RegisterCategoryServiceServer(s.grpcServer, categoryHandler)

	productHandler := handler.NewProductHandler(s.productService, s.repositories, logs)
	productv1.RegisterProductServiceServer(s.grpcServer, productHandler)

	productSKUHandler := handler.NewProductSKUHandler(s.productSKUService, logs)
	productskuv1.RegisterProductServiceSKUServiceServer(s.grpcServer, productSKUHandler)

	inventoryHandler := handler.NewInventoryHandler(s.inventoryService, logs)
	inventoryv1.RegisterInventoryServiceServer(s.grpcServer, inventoryHandler)

	customerHandler := handler.NewCustomerHandler(s.customerService, logs)
	customerv1.RegisterCustomerServiceServer(s.grpcServer, customerHandler)

	saleHandler := handler.NewSaleHandler(s.saleService, s.repositories, logs)
	salev1.RegisterSaleServiceServer(s.grpcServer, saleHandler)

	paymentHandler := handler.NewPaymentHandler(s.paymentService, logs)
	paymentv1.RegisterPaymentServiceServer(s.grpcServer, paymentHandler)

	cashierHandler := handler.NewCashierHandler(s.cashierService, s.repositories, logs)
	cashierpb.RegisterCashierServiceServer(s.grpcServer, cashierHandler)

	reportHandler := handler.NewReportHandler(s.reportService)
	reportv1.RegisterReportServiceServer(s.grpcServer, reportHandler)

	roleHandler := handler.NewRoleHandler(s.roleService, logs)
	rolev1.RegisterRoleServiceServer(s.grpcServer, roleHandler)

	permissionHandler := handler.NewPermissionHandler(s.permissionService, logs)
	permissionv1.RegisterPermissionServiceServer(s.grpcServer, permissionHandler)

	activityLogHandler := handler.NewActivityLogHandler(s.activityLogService)
	activitylogv1.RegisterActivityLogServiceServer(s.grpcServer, activityLogHandler)

	storeHandler := handler.NewStoreHandler(s.storeService, logs)
	storev1.RegisterStoreServiceServer(s.grpcServer, storeHandler)

	settingsHandler := handler.NewSettingsHandler(s.settingsService, logs)
	settingsv1.RegisterSettingsServiceServer(s.grpcServer, settingsHandler)

	selfOrderHandler := handler.NewSelfOrderHandler(s.selfOrderService, logs)
	selforderv1.RegisterSelfOrderServiceServer(s.grpcServer, selfOrderHandler)

	tableHandler := handler.NewTableHandler(s.tableService, logs)
	tablev1.RegisterTableServiceServer(s.grpcServer, tableHandler)

	accountingHandler := handler.NewAccountingHandler(s.accountingReport, s.accountingAdmin, s.accountingExport, s.accountingPosting, s.repositories)
	accountingv1.RegisterAccountingServiceServer(s.grpcServer, accountingHandler)

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

// endpointPermissions maps gRPC method paths to the permission required to
// invoke them.  Methods not present in this map are NOT permission-gated
// (only the JWT auth interceptor runs).
func endpointPermissions() map[string]string {
	return map[string]string{
		// ── Customer ──────────────────────────────────────────────
		"/customer.v1.CustomerService/CreateCustomer":     "customers.create",
		"/customer.v1.CustomerService/GetCustomer":        "customers.read",
		"/customer.v1.CustomerService/ListCustomers":      "customers.read",
		"/customer.v1.CustomerService/SearchCustomers":    "customers.read",
		"/customer.v1.CustomerService/UpdateCustomer":     "customers.update",
		"/customer.v1.CustomerService/DeleteCustomer":     "customers.delete",
		"/customer.v1.CustomerService/DeactivateCustomer": "customers.update",
		"/customer.v1.CustomerService/GetPurchaseHistory": "customers.read",

		// ── Product ───────────────────────────────────────────────
		"/product.v1.ProductService/CreateProduct":       "products.create",
		"/product.v1.ProductService/GetProduct":          "products.read",
		"/product.v1.ProductService/ListProducts":        "products.read",
		"/product.v1.ProductService/SearchProducts":      "products.read",
		"/product.v1.ProductService/UpdateProduct":       "products.update",
		"/product.v1.ProductService/DeleteProduct":       "products.delete",
		"/product.v1.ProductService/ActivateProduct":     "products.update",
		"/product.v1.ProductService/DeactivateProduct":   "products.update",
		"/product.v1.ProductService/GetLowStockProducts": "products.read",

		// ── ProductSKU ────────────────────────────────────────────
		"/productsku.v1.ProductServiceSKUService/CreateProductSKU":       "products.create",
		"/productsku.v1.ProductServiceSKUService/GetProductSKU":          "products.read",
		"/productsku.v1.ProductServiceSKUService/GetProductSKUByBarcode": "products.read",
		"/productsku.v1.ProductServiceSKUService/GetProductSKUBySKUCode": "products.read",
		"/productsku.v1.ProductServiceSKUService/ListProductSKUs":        "products.read",
		"/productsku.v1.ProductServiceSKUService/ListAllProductSKUs":     "products.read",
		"/productsku.v1.ProductServiceSKUService/UpdateProductSKU":       "products.update",
		"/productsku.v1.ProductServiceSKUService/DeleteProductSKU":       "products.delete",

		// ── Category ──────────────────────────────────────────────
		"/category.v1.CategoryService/CreateCategory":     "categories.create",
		"/category.v1.CategoryService/GetCategory":        "categories.read",
		"/category.v1.CategoryService/ListCategories":     "categories.read",
		"/category.v1.CategoryService/UpdateCategory":     "categories.update",
		"/category.v1.CategoryService/DeleteCategory":     "categories.delete",
		"/category.v1.CategoryService/DeactivateCategory": "categories.update",

		// ── Inventory ─────────────────────────────────────────────
		"/inventory.v1.InventoryService/CreateTransaction": "inventory.adjust",
		"/inventory.v1.InventoryService/GetTransaction":    "inventory.read",
		"/inventory.v1.InventoryService/ListTransactions":  "inventory.read",
		"/inventory.v1.InventoryService/AdjustStock":       "inventory.adjust",
		"/inventory.v1.InventoryService/GetStockSummary":   "inventory.read",
		"/inventory.v1.InventoryService/GetLowStockAlert":  "inventory.read",

		// ── Sale ──────────────────────────────────────────────────
		"/sale.v1.SaleService/CreateSale":       "sales.create",
		"/sale.v1.SaleService/GetSale":          "sales.read",
		"/sale.v1.SaleService/GetSaleByInvoice": "sales.read",
		"/sale.v1.SaleService/ListSales":        "sales.read",
		"/sale.v1.SaleService/CancelSale":       "sales.cancel",
		"/sale.v1.SaleService/RefundSale":       "sales.refund",

		// ── Payment ───────────────────────────────────────────────
		"/payment.v1.PaymentService/CreatePayment":     "payments.create",
		"/payment.v1.PaymentService/GetPayment":        "payments.read",
		"/payment.v1.PaymentService/GetPaymentsBySale": "payments.read",
		"/payment.v1.PaymentService/ListPayments":      "payments.read",
		"/payment.v1.PaymentService/DeletePayment":     "payments.delete",
		"/payment.v1.PaymentService/GetPaymentSummary": "payments.read",

		// ── Cashier ───────────────────────────────────────────────
		"/cashier.v1.CashierService/OpenShift":       "cashier.open_shift",
		"/cashier.v1.CashierService/CloseShift":      "cashier.close_shift",
		"/cashier.v1.CashierService/GetCurrentShift": "cashier.view_shift",
		"/cashier.v1.CashierService/GetShift":        "cashier.view_shift",
		"/cashier.v1.CashierService/ListShifts":      "cashier.view_shift",
		"/cashier.v1.CashierService/GetShiftSummary": "cashier.view_shift",
		"/cashier.v1.CashierService/ListShiftItems":  "cashier.view_shift",

		// ── Report ────────────────────────────────────────────────
		"/report.v1.ReportService/GetDailySalesReport":     "reports.read",
		"/report.v1.ReportService/GetSalesByCashier":       "reports.read",
		"/report.v1.ReportService/GetSalesByProduct":       "reports.read",
		"/report.v1.ReportService/GetSalesByCategory":      "reports.read",
		"/report.v1.ReportService/GetSalesByDateRange":     "reports.read",
		"/report.v1.ReportService/GetPaymentMethodSummary": "reports.read",
		"/report.v1.ReportService/GetInventorySummary":     "reports.read",
		"/report.v1.ReportService/GetLowStockProducts":     "reports.read",

		// ── Accounting ────────────────────────────────────────────
		"/accounting.v1.AccountingService/ListAccounts":         "accounting.read",
		"/accounting.v1.AccountingService/GetAccount":           "accounting.read",
		"/accounting.v1.AccountingService/CreateAccount":      "accounting.post",
		"/accounting.v1.AccountingService/UpdateAccount":       "accounting.post",
		"/accounting.v1.AccountingService/ListJournals":         "accounting.read",
		"/accounting.v1.AccountingService/GetJournal":           "accounting.read",
		"/accounting.v1.AccountingService/CreateManualJournal": "accounting.post",
		"/accounting.v1.AccountingService/PostJournal":         "accounting.post",
		"/accounting.v1.AccountingService/ReverseJournal":      "accounting.post",
		"/accounting.v1.AccountingService/ListPeriods":         "accounting.read",
		"/accounting.v1.AccountingService/CreatePeriod":       "accounting.close",
		"/accounting.v1.AccountingService/ClosePeriod":         "accounting.close",
		"/accounting.v1.AccountingService/GetTrialBalance":     "accounting.read",
		"/accounting.v1.AccountingService/GetGeneralLedger":    "accounting.read",
		"/accounting.v1.AccountingService/GetGeneralJournal":   "accounting.read",
		"/accounting.v1.AccountingService/GetIncomeStatement":  "accounting.read",
		"/accounting.v1.AccountingService/GetBalanceSheet":     "accounting.read",
		"/accounting.v1.AccountingService/GetChangesInEquity":  "accounting.read",
		"/accounting.v1.AccountingService/GetCashFlow":         "accounting.read",
		"/accounting.v1.AccountingService/ValidateBooks":       "accounting.read",
		"/accounting.v1.AccountingService/ExportStatement":     "accounting.read",

		// ── Role management ──────────────────────────────────────
		// NOTE: GetRole, ListRoles, GetRolePermissions are intentionally NOT
		// permission-gated. The frontend needs them to bootstrap the permission
		// system (fetchAndAttachPermissions). They are read-only and already
		// protected by JWT auth.
		"/role.v1.RoleService/CreateRole":        "roles.create",
		"/role.v1.RoleService/UpdateRole":        "roles.update",
		"/role.v1.RoleService/DeleteRole":        "roles.delete",
		"/role.v1.RoleService/AssignPermissions": "roles.assign",
		"/role.v1.RoleService/RemovePermission":  "roles.assign",

		// ── Permission management ─────────────────────────────────
		// NOTE: ListPermissions is intentionally NOT permission-gated. The frontend
		// needs it to show the permissions management UI. It is read-only and
		// already protected by JWT auth.
		"/permission.v1.PermissionService/CreatePermission":          "permissions.create",
		"/permission.v1.PermissionService/GetPermission":             "permissions.read",
		"/permission.v1.PermissionService/ListPermissionsByResource": "permissions.read",
		"/permission.v1.PermissionService/UpdatePermission":          "permissions.update",
		"/permission.v1.PermissionService/DeletePermission":          "permissions.delete",

		// ── User management ──────────────────────────────────────
		// NOTE: GetUser is intentionally NOT permission-gated. The frontend
		// needs it to refresh the user profile on session restore (bootstrap).
		// It is read-only and already protected by JWT auth.
		"/user.v1.UserService/CreateUser":     "users.create",
		"/user.v1.UserService/ListUsers":      "users.read",
		"/user.v1.UserService/UpdateUser":     "users.update",
		"/user.v1.UserService/DeleteUser":     "users.delete",
		"/user.v1.UserService/UpdateUserRole": "users.update",

		// ── Store (superadmin only) ──────────────────────────────
		"/store.v1.StoreService/CreateStore": "roles.create",
		"/store.v1.StoreService/GetStore":    "roles.read",
		"/store.v1.StoreService/ListStores":  "roles.read",
		"/store.v1.StoreService/UpdateStore": "roles.update",
		"/store.v1.StoreService/DeleteStore": "roles.delete",
		"/store.v1.StoreService/VerifyStore": "roles.update",

		// ── Activity logs ─────────────────────────────────────────
		"/activitylog.v1.ActivityLogService/ListActivityLogs":          "activity_logs.read",
		"/activitylog.v1.ActivityLogService/GetActivityLog":            "activity_logs.read",
		"/activitylog.v1.ActivityLogService/GetActivityLogsByResource": "activity_logs.read",
		"/activitylog.v1.ActivityLogService/GetActivityLogsByUser":     "activity_logs.read",

		// ── Tables (self ordering) ─────────────────────────────────
		"/table.v1.TableService/CreateTable":        "tables.create",
		"/table.v1.TableService/ListTables":         "tables.read",
		"/table.v1.TableService/GetTable":           "tables.read",
		"/table.v1.TableService/UpdateTable":        "tables.update",
		"/table.v1.TableService/DeleteTable":        "tables.delete",
		"/table.v1.TableService/RotateTableToken":   "tables.update",
	}
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
