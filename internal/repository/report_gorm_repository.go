package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// ReportRepository defines the interface for report operations
type ReportRepository interface {
	GetDailySalesReport(ctx context.Context, params ReportParams) ([]DailySalesReport, error)
	GetSalesByDateRange(ctx context.Context, params ReportParams) ([]gormdb.Sale, error)
	GetTopSellingProducts(ctx context.Context, params ReportParams) ([]TopSellingProduct, error)
	GetInventorySummaryReport(ctx context.Context) ([]InventorySummary, error)
	GetPaymentMethodSummary(ctx context.Context, params ReportParams) ([]PaymentMethodSummary, error)
}

// ReportParams holds parameters for reports
type ReportParams struct {
	StartDate string
	EndDate   string
	StoreID   string
	CashierID string
	Limit     int32
}

// DailySalesReport holds daily sales report data
type DailySalesReport struct {
	Date       string  `json:"date"`
	TotalSales int64   `json:"total_sales"`
	TotalAmount float64 `json:"total_amount"`
}

// TopSellingProduct holds top selling product data
type TopSellingProduct struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int64   `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
}

// InventorySummary holds inventory summary data
type InventorySummary struct {
	CategoryName string  `json:"category_name"`
	ProductCount int64   `json:"product_count"`
	TotalStock   int64   `json:"total_stock"`
	TotalValue   float64 `json:"total_value"`
}

// PaymentMethodSummary holds payment method summary data
type PaymentMethodSummary struct {
	PaymentMethod string  `json:"payment_method"`
	Count         int64   `json:"count"`
	TotalAmount   float64 `json:"total_amount"`
}

// ReportRepo implements ReportRepository using GORM
type ReportRepo struct {
	*GormBase
}

// NewReportRepository creates a new ReportRepo
func NewReportRepository(db *gorm.DB) ReportRepository {
	return &ReportRepo{GormBase: NewGormBase(db)}
}

// GetDailySalesReport gets daily sales report
func (r *ReportRepo) GetDailySalesReport(ctx context.Context, params ReportParams) ([]DailySalesReport, error) {
	var results []DailySalesReport

	query := r.WithContext(ctx).
		Model(&gormdb.Sale{}).
		Select("DATE(created_at) as date, COUNT(*) as total_sales, COALESCE(SUM(grand_total), 0) as total_amount").
		Where("status = ?", "PAID")

	if params.StartDate != "" {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("created_at <= ?", params.EndDate)
	}
	if params.StoreID != "" {
		query = query.Where("store_id = ?", params.StoreID)
	}

	err := query.Group("DATE(created_at)").Order("date DESC").Find(&results).Error
	return results, err
}

// GetSalesByDateRange gets sales by date range
func (r *ReportRepo) GetSalesByDateRange(ctx context.Context, params ReportParams) ([]gormdb.Sale, error) {
	var sales []gormdb.Sale

	query := r.WithContext(ctx).Model(&gormdb.Sale{}).
		Where("status = ?", "PAID")

	if params.StartDate != "" {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("created_at <= ?", params.EndDate)
	}
	if params.StoreID != "" {
		query = query.Where("store_id = ?", params.StoreID)
	}
	if params.CashierID != "" {
		query = query.Where("cashier_id = ?", params.CashierID)
	}

	err := query.Order("created_at DESC").Find(&sales).Error
	return sales, err
}

// GetTopSellingProducts gets top selling products
func (r *ReportRepo) GetTopSellingProducts(ctx context.Context, params ReportParams) ([]TopSellingProduct, error) {
	var results []TopSellingProduct

	query := r.WithContext(ctx).
		Model(&gormdb.SaleItem{}).
		Select("sale_items.product_id, products.name as product_name, SUM(sale_items.quantity) as quantity, SUM(sale_items.subtotal) as total_amount").
		Joins("JOIN products ON products.id = sale_items.product_id").
		Joins("JOIN sales ON sales.id = sale_items.sale_id").
		Where("sales.status = ?", "PAID")

	if params.StartDate != "" {
		query = query.Where("sales.created_at >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("sales.created_at <= ?", params.EndDate)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}

	err := query.Group("sale_items.product_id, products.name").
		Order("quantity DESC").
		Limit(int(limit)).
		Find(&results).Error

	return results, err
}

// GetInventorySummaryReport gets inventory summary
func (r *ReportRepo) GetInventorySummaryReport(ctx context.Context) ([]InventorySummary, error) {
	var results []InventorySummary

	err := r.WithContext(ctx).
		Model(&gormdb.Product{}).
		Select("categories.name as category_name, COUNT(products.id) as product_count, SUM(products.stock) as total_stock, SUM(products.stock * products.cost_price) as total_value").
		Joins("JOIN categories ON categories.id = products.category_id").
		Where("products.is_active = ?", true).
		Group("categories.name").
		Order("category_name").
		Find(&results).Error

	return results, err
}

// GetPaymentMethodSummary gets payment method summary
func (r *ReportRepo) GetPaymentMethodSummary(ctx context.Context, params ReportParams) ([]PaymentMethodSummary, error) {
	var results []PaymentMethodSummary

	query := r.WithContext(ctx).
		Model(&gormdb.Payment{}).
		Select("payment_method, COUNT(*) as count, COALESCE(SUM(amount), 0) as total_amount")

	if params.StartDate != "" {
		query = query.Where("paid_at >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("paid_at <= ?", params.EndDate)
	}

	err := query.Group("payment_method").Order("total_amount DESC").Find(&results).Error
	return results, err
}
