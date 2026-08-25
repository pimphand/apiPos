package gorm

import (
	"time"

	"github.com/google/uuid"
)

// Role represents the roles table
type Role struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Role
func (Role) TableName() string {
	return "roles"
}

// Store represents the stores table
type Store struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Address   string     `gorm:"type:text" json:"address"`
	Phone     string     `gorm:"type:varchar(20)" json:"phone"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Store
func (Store) TableName() string {
	return "stores"
}

// User represents the users table
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Email        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	FullName     string         `gorm:"type:varchar(100);not null" json:"full_name"`
	RoleID       uuid.UUID      `gorm:"type:uuid;not null" json:"role_id"`
	StoreID      uuid.UUID      `gorm:"type:uuid" json:"store_id"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	LastLogin    *time.Time     `gorm:"type:timestamptz" json:"last_login"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Role  Role  `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// Category represents the categories table
type Category struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Category
func (Category) TableName() string {
	return "categories"
}

// Product represents the products table
type Product struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SKU          string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"sku"`
	Barcode      string     `gorm:"type:varchar(100);uniqueIndex" json:"barcode"`
	Name         string     `gorm:"type:varchar(200);not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	CategoryID   uuid.UUID  `gorm:"type:uuid" json:"category_id"`
	CostPrice    float64    `gorm:"type:numeric(15,2);default:0" json:"cost_price"`
	SellingPrice float64    `gorm:"type:numeric(15,2);default:0" json:"selling_price"`
	Stock        int32      `gorm:"default:0" json:"stock"`
	MinimumStock int32      `gorm:"default:0" json:"minimum_stock"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Category Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

// TableName returns the table name for Product
func (Product) TableName() string {
	return "products"
}

// Customer represents the customers table
type Customer struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	Email     string     `gorm:"type:varchar(100)" json:"email"`
	Phone     string     `gorm:"type:varchar(20)" json:"phone"`
	Address   string     `gorm:"type:text" json:"address"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Customer
func (Customer) TableName() string {
	return "customers"
}

// Sale represents the sales table
type Sale struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InvoiceNumber string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"invoice_number"`
	StoreID       uuid.UUID  `gorm:"type:uuid;not null" json:"store_id"`
	CashierID     uuid.UUID  `gorm:"type:uuid;not null" json:"cashier_id"`
	CustomerID    uuid.UUID  `gorm:"type:uuid" json:"customer_id"`
	Subtotal      float64    `gorm:"type:numeric(15,2);default:0" json:"subtotal"`
	Discount      float64    `gorm:"type:numeric(15,2);default:0" json:"discount"`
	Tax           float64    `gorm:"type:numeric(15,2);default:0" json:"tax"`
	GrandTotal    float64    `gorm:"type:numeric(15,2);default:0" json:"grand_total"`
	Status        string     `gorm:"type:varchar(20);default:PENDING" json:"status"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store    Store    `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Cashier  User     `gorm:"foreignKey:CashierID" json:"cashier,omitempty"`
	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

// TableName returns the table name for Sale
func (Sale) TableName() string {
	return "sales"
}

// SaleItem represents the sale_items table
type SaleItem struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SaleID    uuid.UUID  `gorm:"type:uuid;not null" json:"sale_id"`
	ProductID uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	Quantity  int32      `gorm:"not null" json:"quantity"`
	UnitPrice float64    `gorm:"type:numeric(15,2);not null" json:"unit_price"`
	Discount  float64    `gorm:"type:numeric(15,2);default:0" json:"discount"`
	Subtotal  float64    `gorm:"type:numeric(15,2);not null" json:"subtotal"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Sale    Sale    `gorm:"foreignKey:SaleID" json:"sale,omitempty"`
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// TableName returns the table name for SaleItem
func (SaleItem) TableName() string {
	return "sale_items"
}

// Payment represents the payments table
type Payment struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SaleID        uuid.UUID  `gorm:"type:uuid;not null" json:"sale_id"`
	PaymentMethod string     `gorm:"type:varchar(20);not null" json:"payment_method"`
	Amount        float64    `gorm:"type:numeric(15,2);not null" json:"amount"`
	Reference     string     `gorm:"type:varchar(100)" json:"reference"`
	PaidAt        time.Time  `gorm:"type:timestamptz;not null" json:"paid_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Sale Sale `gorm:"foreignKey:SaleID" json:"sale,omitempty"`
}

// TableName returns the table name for Payment
func (Payment) TableName() string {
	return "payments"
}

// InventoryTransaction represents the inventory_transactions table
type InventoryTransaction struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID     uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	StoreID       uuid.UUID  `gorm:"type:uuid;not null" json:"store_id"`
	Type          string     `gorm:"type:varchar(20);not null" json:"type"`
	Quantity      int32      `gorm:"not null" json:"quantity"`
	ReferenceType string     `gorm:"type:varchar(50)" json:"reference_type"`
	ReferenceID   uuid.UUID  `gorm:"type:uuid" json:"reference_id"`
	BeforeStock   int32      `gorm:"not null" json:"before_stock"`
	AfterStock    int32      `gorm:"not null" json:"after_stock"`
	CreatedBy     uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Store   Store   `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for InventoryTransaction
func (InventoryTransaction) TableName() string {
	return "inventory_transactions"
}

// CashierShift represents the cashier_shifts table
type CashierShift struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	StoreID        uuid.UUID  `gorm:"type:uuid;not null" json:"store_id"`
	OpeningCash    float64    `gorm:"type:numeric(15,2);default:0" json:"opening_cash"`
	ClosingCash    float64    `gorm:"type:numeric(15,2);default:0" json:"closing_cash"`
	CashSales      float64    `gorm:"type:numeric(15,2);default:0" json:"cash_sales"`
	CashRefund     float64    `gorm:"type:numeric(15,2);default:0" json:"cash_refund"`
	ExpectedCash   float64    `gorm:"type:numeric(15,2);default:0" json:"expected_cash"`
	ActualCash     float64    `gorm:"type:numeric(15,2);default:0" json:"actual_cash"`
	CashDifference float64    `gorm:"type:numeric(15,2);default:0" json:"cash_difference"`
	Status         string     `gorm:"type:varchar(20);default:OPEN" json:"status"`
	OpenedAt       time.Time  `gorm:"type:timestamptz;not null" json:"opened_at"`
	ClosedAt       *time.Time `gorm:"type:timestamptz" json:"closed_at"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for CashierShift
func (CashierShift) TableName() string {
	return "cashier_shifts"
}

// RefreshToken represents the refresh_tokens table
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Token     string     `gorm:"type:varchar(255);not null" json:"token"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null" json:"expires_at"`
	IsRevoked bool       `gorm:"default:false" json:"is_revoked"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// Permission represents the permissions table
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Resource    string    `gorm:"type:varchar(100);not null" json:"resource"`
	Action      string    `gorm:"type:varchar(50);not null" json:"action"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Permission
func (Permission) TableName() string {
	return "permissions"
}

// RolePermission represents the role_permissions junction table
type RolePermission struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RoleID       uuid.UUID `gorm:"type:uuid;not null" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;not null" json:"permission_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Role       Role       `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Permission Permission `gorm:"foreignKey:PermissionID" json:"permission,omitempty"`
}

// TableName returns the table name for RolePermission
func (RolePermission) TableName() string {
	return "role_permissions"
}

// ActivityLog represents the activity_logs table
type ActivityLog struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	Username    string     `gorm:"type:varchar(50)" json:"username"`
	Action      string     `gorm:"type:varchar(50);not null" json:"action"`
	Resource    string     `gorm:"type:varchar(100);not null" json:"resource"`
	ResourceID  string     `gorm:"type:varchar(100)" json:"resource_id"`
	Description string     `gorm:"type:text;not null" json:"description"`
	IPAddress   string     `gorm:"type:varchar(50)" json:"ip_address"`
	UserAgent   string     `gorm:"type:text" json:"user_agent"`
	Metadata    string     `gorm:"type:jsonb" json:"metadata"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for ActivityLog
func (ActivityLog) TableName() string {
	return "activity_logs"
}

