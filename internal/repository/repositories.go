package repository

import (
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	User             UserRepository
	Role             *RoleRepo
	Permission       PermissionRepository
	RolePermission   RolePermissionRepository
	Product          ProductRepository
	Category         CategoryRepository
	Customer         CustomerRepository
	Sale             SaleRepository
	Payment          PaymentRepository
	Inventory        InventoryRepository
	Cashier          CashierRepository
	Report           ReportRepository
	ActivityLog      ActivityLogRepository
}

// New creates a new Repositories instance with GORM
func New(db *gorm.DB) *Repositories {
	return &Repositories{
		User:           NewUserRepository(db),
		Role:           NewRoleRepository(db),
		Permission:     NewPermissionRepository(db),
		RolePermission: NewRolePermissionRepository(db),
		Product:        NewProductRepository(db),
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
