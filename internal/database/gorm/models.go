package gorm

import (
	"time"

	"github.com/google/uuid"
)

// Role represents the roles table
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Role
func (Role) TableName() string {
	return "roles"
}

// Store represents the stores table
type Store struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name       string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Address    string    `gorm:"type:text" json:"address"`
	Phone      string    `gorm:"type:varchar(20)" json:"phone"`
	IsActive   bool      `json:"is_active"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Store
func (Store) TableName() string {
	return "stores"
}

// Table represents a dine-in table for self ordering (QR per meja).
// `code` is unique per store (UNIQUE(store_id, code)); `qr_token` is globally
// unique so a QR code can never resolve to another tenant's table.
type Table struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID   uuid.UUID `gorm:"type:uuid;not null;index:idx_tables_store_id;uniqueIndex:idx_tables_store_code,priority:1" json:"store_id"`
	Code      string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_tables_store_code,priority:2" json:"code"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	QRToken   string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"` // never expose
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for Table
func (Table) TableName() string {
	return "tables"
}

// User represents the users table
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Email        string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	FullName     string     `gorm:"type:varchar(100);not null" json:"full_name"`
	RoleID       uuid.UUID  `gorm:"type:uuid;not null" json:"role_id"`
	StoreID      uuid.UUID  `gorm:"type:uuid" json:"store_id"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLogin    *time.Time `gorm:"type:timestamptz" json:"last_login"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

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
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID     uuid.UUID `gorm:"type:uuid;not null;index:idx_categories_store_id;default:00000000-0000-0000-0000-000000000001" json:"store_id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex:idx_categories_store_name;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for Category
func (Category) TableName() string {
	return "categories"
}

// Product represents the products table
type Product struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID      uuid.UUID `gorm:"type:uuid;not null;index:idx_products_store_id;default:00000000-0000-0000-0000-000000000001" json:"store_id"`
	SKU          string    `gorm:"type:varchar(50);index:idx_products_sku" json:"sku"`
	Barcode      string    `gorm:"type:varchar(50);index:idx_products_barcode" json:"barcode"`
	Name         string    `gorm:"type:varchar(200);not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	CategoryID   uuid.UUID `gorm:"type:uuid" json:"category_id"`
	CostPrice    float64   `gorm:"type:numeric(15,2);default:0" json:"cost_price"`
	SellingPrice float64   `gorm:"type:numeric(15,2);default:0" json:"selling_price"`
	Stock        int32     `gorm:"default:0" json:"stock"`
	MinimumStock int32     `gorm:"default:0" json:"minimum_stock"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store       Store        `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Category    Category     `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	ProductSKUs []ProductSKU `gorm:"foreignKey:ProductID" json:"product_skus,omitempty"`
}

// TableName returns the table name for Product
func (Product) TableName() string {
	return "products"
}

// ProductSKU represents the product_skus table
type ProductSKU struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID    uuid.UUID      `gorm:"type:uuid;not null" json:"product_id"`
	StoreID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_product_skus_store_id;default:00000000-0000-0000-0000-000000000001" json:"store_id"`
	SKU          string         `gorm:"type:varchar(100);not null" json:"sku"`
	Barcode      string         `gorm:"type:varchar(100)" json:"barcode"`
	Variant      string         `gorm:"type:varchar(255)" json:"variant"`
	Color        string         `gorm:"type:varchar(100)" json:"color"`
	Size         string         `gorm:"type:varchar(50)" json:"size"`
	Unit         string         `gorm:"type:varchar(50);default:'pcs'" json:"unit"`
	CostPrice    float64        `gorm:"type:numeric(15,2);default:0" json:"cost_price"`
	SellingPrice float64        `gorm:"type:numeric(15,2);default:0" json:"selling_price"`
	Stock        int32          `gorm:"default:0" json:"stock"`
	MinimumStock int32          `gorm:"default:0" json:"minimum_stock"`
	Weight       float64        `gorm:"type:numeric(10,2)" json:"weight"`
	Volume       float64        `gorm:"type:numeric(10,2)" json:"volume"`
	ExpiryDate   *time.Time     `gorm:"type:timestamptz" json:"expiry_date"`
	BatchNumber  string         `gorm:"type:varchar(100)" json:"batch_number"`
	Location     string         `gorm:"type:varchar(100)" json:"location"`
	ImageURL     string         `gorm:"type:varchar(500)" json:"image_url"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	Metadata     map[string]any `gorm:"type:jsonb" json:"metadata"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// TableName returns the table name for ProductSKU
func (ProductSKU) TableName() string {
	return "product_skus"
}

// Customer represents the customers table
type Customer struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID   uuid.UUID `gorm:"type:uuid;not null;index:idx_customers_store_id;default:00000000-0000-0000-0000-000000000001" json:"store_id"`
	Code      string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Email     string    `gorm:"type:varchar(100)" json:"email"`
	Phone     string    `gorm:"type:varchar(20)" json:"phone"`
	Address   string    `gorm:"type:text" json:"address"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
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
	// CashierID is NULL for self-order sales (no cashier at the table — customer
	// action). It is kept as a plain uuid.UUID and the self-order flow writes
	// NULL via tx.Omit("CashierID"), the same pattern used for CustomerID.
	CashierID     uuid.UUID  `gorm:"type:uuid" json:"cashier_id"`
	CustomerID    uuid.UUID  `gorm:"type:uuid" json:"customer_id"`
	Subtotal      float64    `gorm:"type:numeric(15,2);default:0" json:"subtotal"`
	Discount      float64    `gorm:"type:numeric(15,2);default:0" json:"discount"`
	Tax           float64    `gorm:"type:numeric(15,2);default:0" json:"tax"`
	GrandTotal    float64    `gorm:"type:numeric(15,2);default:0" json:"grand_total"`
	Status        string     `gorm:"type:varchar(20);default:PENDING" json:"status"`
	OrderSource   string     `gorm:"type:varchar(20);default:CASHIER" json:"order_source"`
	OrderStatus   string     `gorm:"type:varchar(20);default:COMPLETED" json:"order_status"`
	TableID       *uuid.UUID `gorm:"type:uuid;index:idx_sales_table_id" json:"table_id"`
	TableNo       string     `gorm:"type:varchar(20)" json:"table_no"`
	OrderToken    *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"order_token"`
	Notes         *string    `gorm:"type:text" json:"notes"`
	ShiftID       *uuid.UUID `gorm:"type:uuid;index:idx_sales_shift_id" json:"shift_id"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store    Store    `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Cashier  User     `gorm:"foreignKey:CashierID" json:"cashier,omitempty"`
	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Table    Table    `gorm:"foreignKey:TableID" json:"table,omitempty"`
}

// TableName returns the table name for Sale
func (Sale) TableName() string {
	return "sales"
}

// SaleItem represents the sale_items table
type SaleItem struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SaleID       uuid.UUID  `gorm:"type:uuid;not null" json:"sale_id"`
	StoreID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_sale_items_store_id;default:00000000-0000-0000-0000-000000000001" json:"store_id"`
	ProductID    uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	ProductSKUID *uuid.UUID `gorm:"type:uuid;column:product_sku_id" json:"product_sku_id"`
	Quantity     int32      `gorm:"not null" json:"quantity"`
	UnitPrice    float64    `gorm:"type:numeric(15,2);not null" json:"unit_price"`
	Discount     float64    `gorm:"type:numeric(15,2);default:0" json:"discount"`
	Subtotal     float64    `gorm:"type:numeric(15,2);not null" json:"subtotal"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Sale       Sale       `gorm:"foreignKey:SaleID" json:"sale,omitempty"`
	Store      Store      `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Product    Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductSKU ProductSKU `gorm:"foreignKey:ProductSKUID" json:"product_sku,omitempty"`
}

// TableName returns the table name for SaleItem
func (SaleItem) TableName() string {
	return "sale_items"
}

// Payment represents the payments table
type Payment struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SaleID        uuid.UUID `gorm:"type:uuid;not null" json:"sale_id"`
	StoreID       uuid.UUID `gorm:"type:uuid;not null;index:idx_payments_store_id;default:00000000-0000-0000-0000-000000000001" json:"store_id"`
	PaymentMethod string    `gorm:"type:varchar(20);not null" json:"payment_method"`
	Amount        float64   `gorm:"type:numeric(15,2);not null" json:"amount"`
	Reference     string    `gorm:"type:varchar(100)" json:"reference"`
	PaidAt        time.Time `gorm:"type:timestamptz;not null" json:"paid_at"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Sale  Sale  `gorm:"foreignKey:SaleID" json:"sale,omitempty"`
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for Payment
func (Payment) TableName() string {
	return "payments"
}

// InventoryTransaction represents the inventory_transactions table
type InventoryTransaction struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID     uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	StoreID       uuid.UUID `gorm:"type:uuid;not null" json:"store_id"`
	Type          string    `gorm:"type:varchar(20);not null" json:"type"`
	Quantity      int32     `gorm:"not null" json:"quantity"`
	ReferenceType string    `gorm:"type:varchar(50)" json:"reference_type"`
	ReferenceID   uuid.UUID `gorm:"type:uuid" json:"reference_id"`
	BeforeStock   int32     `gorm:"not null" json:"before_stock"`
	AfterStock    int32     `gorm:"not null" json:"after_stock"`
	// CreatedBy is NULL for customer/system actions (e.g. self-order submit /
	// expired-order cleanup). Kept as plain uuid.UUID; NULL is written via
	// tx.Omit("CreatedBy") like Sale.CashierID/CustomerID.
	CreatedBy     uuid.UUID `gorm:"type:uuid" json:"created_by"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`

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

	// Snapshot aggregates (frozen at close time)
	TotalTransactions int32   `gorm:"default:0" json:"total_transactions"`
	TotalSalesAmount  float64 `gorm:"type:numeric(15,2);default:0" json:"total_sales_amount"`
	TotalRevenue      float64 `gorm:"type:numeric(15,2);default:0" json:"total_revenue"`
	RevenueCash       float64 `gorm:"type:numeric(15,2);default:0" json:"revenue_cash"`
	TotalItemsOut     float64 `gorm:"type:numeric(12,3);default:0" json:"total_items_out"`
	TotalItemsIn      float64 `gorm:"type:numeric(12,3);default:0" json:"total_items_in"`

	// Relations
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for CashierShift
func (CashierShift) TableName() string {
	return "cashier_shifts"
}

// ShiftItem is the insert-only per-item ledger of goods moving OUT (sale) or
// IN (refund/cancel) during a cashier shift. Rows are never updated or
// deleted; they are written inside the same DB transaction as the sale or
// refund so the ledger can never drift from stock/inventory movements.
type ShiftItem struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ShiftID        uuid.UUID  `gorm:"type:uuid;not null" json:"shift_id"`
	StoreID        uuid.UUID  `gorm:"type:uuid;not null" json:"store_id"`
	SaleID         *uuid.UUID `gorm:"type:uuid" json:"sale_id"`
	Direction      string     `gorm:"type:varchar(10);not null" json:"direction"`
	ItemType       string     `gorm:"type:varchar(10);not null" json:"item_type"`
	ProductID      uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	ProductSKUID   *uuid.UUID `gorm:"type:uuid;column:product_sku_id" json:"product_sku_id"`
	ProductName    string     `gorm:"type:varchar(200);not null" json:"product_name"`
	SKUCode        string     `gorm:"type:varchar(100)" json:"sku_code"`
	VariantLabel   string     `gorm:"type:varchar(255)" json:"variant_label"`
	Quantity       float64    `gorm:"type:numeric(12,3);not null" json:"quantity"`
	UnitPrice      float64    `gorm:"type:numeric(15,2);not null" json:"unit_price"`
	DiscountAmount float64    `gorm:"type:numeric(15,2);default:0" json:"discount_amount"`
	LineAmount     float64    `gorm:"type:numeric(15,2);not null" json:"line_amount"`
	PaymentMethod  string     `gorm:"type:varchar(20)" json:"payment_method"`
	ReferenceNo    string     `gorm:"type:varchar(100)" json:"reference_no"`
	CreatedBy      uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	Shift   CashierShift `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
	Store   Store        `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Product Product      `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// TableName returns the table name for ShiftItem
func (ShiftItem) TableName() string {
	return "shift_items"
}

// RefreshToken represents the refresh_tokens table
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Token     string    `gorm:"type:varchar(255);not null" json:"token"`
	DeviceID  string    `gorm:"type:varchar(255);not null;default:''" json:"device_id"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null" json:"expires_at"`
	IsRevoked bool      `gorm:"default:false" json:"is_revoked"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

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
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid" json:"user_id"`
	StoreID     uuid.UUID `gorm:"type:uuid;index:idx_activity_logs_store_id" json:"store_id"`
	Username    string    `gorm:"type:varchar(50)" json:"username"`
	Action      string    `gorm:"type:varchar(50);not null" json:"action"`
	Resource    string    `gorm:"type:varchar(100);not null" json:"resource"`
	ResourceID  string    `gorm:"type:varchar(100)" json:"resource_id"`
	Description string    `gorm:"type:text;not null" json:"description"`
	IPAddress   string    `gorm:"type:varchar(50)" json:"ip_address"`
	UserAgent   string    `gorm:"type:text" json:"user_agent"`
	Metadata    string    `gorm:"type:jsonb" json:"metadata"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for ActivityLog
func (ActivityLog) TableName() string {
	return "activity_logs"
}

// StoreSetting holds per-store business settings (default tax rate and
// receipt/nota format). One row per store (store_id is the primary key).
//
// NOTE: no `default:` gorm tags on purpose — GORM omits zero-valued fields
// that carry a default tag from INSERT statements, so saving tax_rate=0 or
// show_x=false would silently be replaced by the DB default (11.00/true) via
// ON CONFLICT ... EXCLUDED. The DB column defaults in the migration remain as
// raw-SQL safety nets only.
type StoreSetting struct {
	StoreID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"store_id"`
	TaxRate       float64   `gorm:"type:numeric(5,2);not null" json:"tax_rate"`
	ReceiptHeader string    `gorm:"type:text;not null" json:"receipt_header"`
	ReceiptFooter string    `gorm:"type:text;not null" json:"receipt_footer"`
	// Footer text alignment on the nota: left | center | right.
	ReceiptFooterAlign string     `gorm:"type:varchar(10);not null;default:'center'" json:"receipt_footer_align"`
	ShowStoreName      bool       `gorm:"not null" json:"show_store_name"`
	ShowStoreContact   bool       `gorm:"not null" json:"show_store_contact"`
	ShowCashier        bool       `gorm:"not null" json:"show_cashier"`
	ShowCustomer       bool       `gorm:"not null" json:"show_customer"`
	ShowItemSKU        bool       `gorm:"not null" json:"show_item_sku"`
	ShowItemBarcode    bool       `gorm:"not null" json:"show_item_barcode"`
	ShowDiscount       bool       `gorm:"not null" json:"show_discount"`
	ShowTax            bool       `gorm:"not null" json:"show_tax"`
	ShowPayment        bool       `gorm:"not null" json:"show_payment"`
	ShowChange         bool       `gorm:"not null" json:"show_change"`
	ShowKitchenPrint   bool       `gorm:"not null" json:"show_kitchen_print"`
	SelfOrderEnabled   bool       `gorm:"not null" json:"self_order_enabled"`
	UpdatedBy          *uuid.UUID `gorm:"type:uuid" json:"updated_by"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

// TableName returns the table name for StoreSetting
func (StoreSetting) TableName() string {
	return "store_settings"
}
