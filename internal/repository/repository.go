package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "apiPos/internal/database/sqlc"
)

// Repositories holds all repository instances.
type Repositories struct {
	User       UserRepository
	Product    ProductRepository
	Category   CategoryRepository
	Customer   CustomerRepository
	Sale       SaleRepository
	Payment    PaymentRepository
	Inventory  InventoryRepository
	Cashier    CashierRepository
	Report     ReportRepository
}

// New creates a new Repositories instance.
func New(queries *db.Queries) *Repositories {
	return &Repositories{
		User:      NewUserRepository(queries),
		Product:   NewProductRepository(queries),
		Category:  NewCategoryRepository(queries),
		Customer:  NewCustomerRepository(queries),
		Sale:      NewSaleRepository(queries),
		Payment:   NewPaymentRepository(queries),
		Inventory: NewInventoryRepository(queries),
		Cashier:   NewCashierRepository(queries),
		Report:    NewReportRepository(queries),
	}
}

// UserRepository defines the interface for user operations.
type UserRepository interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByID(ctx context.Context, id int64) (db.User, error)
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.User, error)
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error)
	UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error
	UpdateUserLastLogin(ctx context.Context, id int64) error
	DeleteUser(ctx context.Context, id int64) error
	DeactivateUser(ctx context.Context, id int64) error
	CountUsers(ctx context.Context) (int64, error)
	CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error)
	GetRefreshToken(ctx context.Context, token string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error
}

// ProductRepository defines the interface for product operations.
type ProductRepository interface {
	CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error)
	GetProductByID(ctx context.Context, id int64) (db.Product, error)
	GetProductBySKU(ctx context.Context, sku string) (db.Product, error)
	GetProductByBarcode(ctx context.Context, barcode pgtype.Text) (db.Product, error)
	ListProducts(ctx context.Context, arg db.ListProductsParams) ([]db.Product, error)
	SearchProductsByName(ctx context.Context, arg db.SearchProductsByNameParams) ([]db.Product, error)
	UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.Product, error)
	UpdateProductStock(ctx context.Context, arg db.UpdateProductStockParams) error
	DecreaseProductStock(ctx context.Context, arg db.DecreaseProductStockParams) error
	IncreaseProductStock(ctx context.Context, arg db.IncreaseProductStockParams) error
	ActivateProduct(ctx context.Context, id int64) error
	DeactivateProduct(ctx context.Context, id int64) error
	DeleteProduct(ctx context.Context, id int64) error
	CountProducts(ctx context.Context) (int64, error)
	GetLowStockProducts(ctx context.Context) ([]db.Product, error)
	GetProductStockForUpdate(ctx context.Context, id int64) (db.GetProductStockForUpdateRow, error)
}

// CategoryRepository defines the interface for category operations.
type CategoryRepository interface {
	CreateCategory(ctx context.Context, arg db.CreateCategoryParams) (db.Category, error)
	GetCategoryByID(ctx context.Context, id int64) (db.Category, error)
	ListCategories(ctx context.Context, isActive bool) ([]db.Category, error)
	UpdateCategory(ctx context.Context, arg db.UpdateCategoryParams) (db.Category, error)
	DeleteCategory(ctx context.Context, id int64) error
	DeactivateCategory(ctx context.Context, id int64) error
	CountCategories(ctx context.Context) (int64, error)
}

// CustomerRepository defines the interface for customer operations.
type CustomerRepository interface {
	CreateCustomer(ctx context.Context, arg db.CreateCustomerParams) (db.Customer, error)
	GetCustomerByID(ctx context.Context, id int64) (db.Customer, error)
	GetCustomerByCode(ctx context.Context, code string) (db.Customer, error)
	ListCustomers(ctx context.Context, arg db.ListCustomersParams) ([]db.Customer, error)
	SearchCustomers(ctx context.Context, arg db.SearchCustomersParams) ([]db.Customer, error)
	UpdateCustomer(ctx context.Context, arg db.UpdateCustomerParams) (db.Customer, error)
	DeleteCustomer(ctx context.Context, id int64) error
	DeactivateCustomer(ctx context.Context, id int64) error
	CountCustomers(ctx context.Context) (int64, error)
	GetCustomerPurchaseHistory(ctx context.Context, arg db.GetCustomerPurchaseHistoryParams) ([]db.GetCustomerPurchaseHistoryRow, error)
}

// SaleRepository defines the interface for sale operations.
type SaleRepository interface {
	CreateSale(ctx context.Context, arg db.CreateSaleParams) (db.Sale, error)
	GetSaleByID(ctx context.Context, id int64) (db.Sale, error)
	GetSaleByInvoiceNumber(ctx context.Context, invoiceNumber string) (db.Sale, error)
	ListSales(ctx context.Context, arg db.ListSalesParams) ([]db.Sale, error)
	UpdateSaleStatus(ctx context.Context, arg db.UpdateSaleStatusParams) error
	UpdateSale(ctx context.Context, arg db.UpdateSaleParams) (db.Sale, error)
	DeleteSale(ctx context.Context, id int64) error
	CountSales(ctx context.Context) (int64, error)
	CreateSaleItem(ctx context.Context, arg db.CreateSaleItemParams) (db.SaleItem, error)
	GetSaleItemsBySaleID(ctx context.Context, saleID int64) ([]db.SaleItem, error)
	DeleteSaleItemsBySaleID(ctx context.Context, saleID int64) error
}

// PaymentRepository defines the interface for payment operations.
type PaymentRepository interface {
	CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.Payment, error)
	GetPaymentByID(ctx context.Context, id int64) (db.Payment, error)
	GetPaymentsBySaleID(ctx context.Context, saleID int64) ([]db.Payment, error)
	ListPayments(ctx context.Context, arg db.ListPaymentsParams) ([]db.Payment, error)
	DeletePayment(ctx context.Context, id int64) error
	GetTotalPaymentsBySaleID(ctx context.Context, saleID int64) (interface{}, error)
}

// InventoryRepository defines the interface for inventory operations.
type InventoryRepository interface {
	CreateInventoryTransaction(ctx context.Context, arg db.CreateInventoryTransactionParams) (db.InventoryTransaction, error)
	GetInventoryTransactionByID(ctx context.Context, id int64) (db.InventoryTransaction, error)
	ListInventoryTransactions(ctx context.Context, arg db.ListInventoryTransactionsParams) ([]db.InventoryTransaction, error)
	GetInventoryTransactionsByProduct(ctx context.Context, arg db.GetInventoryTransactionsByProductParams) ([]db.InventoryTransaction, error)
}

// CashierRepository defines the interface for cashier shift operations.
type CashierRepository interface {
	OpenCashierShift(ctx context.Context, arg db.OpenCashierShiftParams) (db.CashierShift, error)
	CloseCashierShift(ctx context.Context, arg db.CloseCashierShiftParams) (db.CashierShift, error)
	GetCurrentShift(ctx context.Context, userID int64) (db.CashierShift, error)
	GetShiftByID(ctx context.Context, id int64) (db.CashierShift, error)
	ListShifts(ctx context.Context, arg db.ListShiftsParams) ([]db.CashierShift, error)
	UpdateShiftCashSales(ctx context.Context, arg db.UpdateShiftCashSalesParams) error
	UpdateShiftCashRefund(ctx context.Context, arg db.UpdateShiftCashRefundParams) error
	CountOpenShifts(ctx context.Context) (int64, error)
}

// ReportRepository defines the interface for report operations.
type ReportRepository interface {
	GetDailySalesReport(ctx context.Context, arg db.GetDailySalesReportParams) ([]db.GetDailySalesReportRow, error)
	GetSalesByDateRange(ctx context.Context, arg db.GetSalesByDateRangeParams) ([]db.Sale, error)
	GetTopSellingProducts(ctx context.Context, arg db.GetTopSellingProductsParams) ([]db.GetTopSellingProductsRow, error)
	GetInventorySummaryReport(ctx context.Context) ([]db.GetInventorySummaryReportRow, error)
	GetPaymentMethodSummary(ctx context.Context, arg db.GetPaymentMethodSummaryParams) ([]db.GetPaymentMethodSummaryRow, error)
}
