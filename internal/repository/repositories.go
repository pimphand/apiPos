package repository

import (
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	db             *gorm.DB
	User           UserRepository
	Role           *RoleRepo
	Store          StoreRepository
	Settings       SettingsRepository
	Permission     PermissionRepository
	RolePermission RolePermissionRepository
	Product        ProductRepository
	ProductSKU     ProductSKURepository
	Category       CategoryRepository
	Customer       CustomerRepository
	Sale           SaleRepository
	Payment        PaymentRepository
	Inventory      InventoryRepository
	Cashier        CashierRepository
	Report         ReportRepository
	ActivityLog    ActivityLogRepository
	RefreshToken   RefreshTokenRepository
}

// DB returns the underlying GORM DB instance
func (r *Repositories) DB() *gorm.DB {
	return r.db
}

// New creates a new Repositories instance with GORM
func New(db *gorm.DB) *Repositories {
	// Install automatic store_id scoping for tenant tables.
	registerStoreScopeCallbacks(db)

	return &Repositories{
		db:             db,
		User:           NewUserRepository(db),
		Role:           NewRoleRepository(db),
		Store:          NewStoreRepository(db),
		Settings:       NewSettingsRepository(db),
		Permission:     NewPermissionRepository(db),
		RolePermission: NewRolePermissionRepository(db),
		Product:        NewProductRepository(db),
		ProductSKU:     NewProductSKURepository(db),
		Category:       NewCategoryRepository(db),
		Customer:       NewCustomerRepository(db),
		Sale:           NewSaleRepository(db),
		Payment:        NewPaymentRepository(db),
		Inventory:      NewInventoryRepository(db),
		Cashier:        NewCashierRepository(db),
		Report:         NewReportRepository(db),
		ActivityLog:    NewActivityLogRepository(db),
		RefreshToken:   NewRefreshTokenRepository(db),
	}
}
