package repository

import (
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	db               *gorm.DB
	User             UserRepository
	Role             *RoleRepo
	Permission       PermissionRepository
	RolePermission   RolePermissionRepository
	Product          ProductRepository
	ProductSKU       ProductSKURepository
	Category         CategoryRepository
	Customer         CustomerRepository
	Sale             SaleRepository
	Payment          PaymentRepository
	Inventory        InventoryRepository
	Cashier          CashierRepository
	Report           ReportRepository
	ActivityLog      ActivityLogRepository
}

// DB returns the underlying GORM DB instance
func (r *Repositories) DB() *gorm.DB {
	return r.db
}

// New creates a new Repositories instance with GORM
func New(db *gorm.DB) *Repositories {
	return &Repositories{
		db:             db,
		User:           NewUserRepository(db),
		Role:           NewRoleRepository(db),
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
	}
}
