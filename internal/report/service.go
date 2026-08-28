package report

import (
	"context"

	"github.com/google/uuid"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Service handles report business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new report Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// DailySalesReport holds daily sales report data
type DailySalesReport struct {
	Date        string
	TotalSales  int64
	TotalAmount float64
}

// DailySalesResult holds daily sales report result
type DailySalesResult struct {
	Reports    []DailySalesReport
	TotalSales float64
	TotalAmount float64
}

// SaleReport holds sale report data
type SaleReport struct {
	ID            string
	InvoiceNumber string
	StoreName     string
	CashierName   string
	CustomerName  string
	Subtotal      float64
	Discount      float64
	Tax           float64
	GrandTotal    float64
	Status        string
	CreatedAt     string
}

// SalesByDateResult holds sales by date result
type SalesByDateResult struct {
	Sales       []SaleReport
	TotalCount  int64
	TotalAmount float64
}

// CashierSalesReport holds cashier sales report
type CashierSalesReport struct {
	CashierID        string
	CashierName      string
	TotalTransactions int64
	TotalAmount      float64
}

// ProductSalesReport holds product sales report
type ProductSalesReport struct {
	ProductID    string
	ProductName  string
	QuantitySold int64
	TotalAmount  float64
}

// CategorySalesReport holds category sales report
type CategorySalesReport struct {
	CategoryID   string
	CategoryName string
	ProductCount int64
	QuantitySold int64
	TotalAmount  float64
}

// PaymentMethodReport holds payment method report
type PaymentMethodReport struct {
	PaymentMethod string
	Count         int64
	TotalAmount   float64
}

// PaymentMethodResult holds payment method summary result
type PaymentMethodResult struct {
	Reports     []PaymentMethodReport
	TotalAmount float64
}

// InventoryReport holds inventory summary report
type InventoryReport struct {
	CategoryName string
	ProductCount int64
	TotalStock   int64
	TotalValue   float64
}

// InventoryResult holds inventory summary result
type InventoryResult struct {
	Reports      []InventoryReport
	TotalProducts int64
	TotalStock   int64
	TotalValue   float64
}

// LowStockReport holds low stock report
type LowStockReport struct {
	ProductID    string
	ProductName  string
	CategoryName string
	CurrentStock int32
	MinimumStock int32
	Deficit      int32
}

// GetDailySalesReport gets daily sales report
func (s *Service) GetDailySalesReport(ctx context.Context, startDate, endDate, storeID string) (*DailySalesResult, error) {
	reports, err := s.repos.Report.GetDailySalesReport(ctx, repository.ReportParams{
		StartDate: startDate,
		EndDate:   endDate,
		StoreID:   storeID,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get daily sales report", err)
	}

	var totalSales, totalAmount float64
	for _, r := range reports {
		totalSales += float64(r.TotalSales)
		totalAmount += r.TotalAmount
	}

	result := &DailySalesResult{
		Reports: make([]DailySalesReport, 0, len(reports)),
	}

	for _, r := range reports {
		result.Reports = append(result.Reports, DailySalesReport{
			Date:        r.Date,
			TotalSales:  r.TotalSales,
			TotalAmount: r.TotalAmount,
		})
	}

	result.TotalSales = totalSales
	result.TotalAmount = totalAmount

	return result, nil
}

// GetSalesByDateRange gets sales by date range
func (s *Service) GetSalesByDateRange(ctx context.Context, startDate, endDate, storeID, cashierID string) (*SalesByDateResult, error) {
	sales, err := s.repos.Report.GetSalesByDateRange(ctx, repository.ReportParams{
		StartDate: startDate,
		EndDate:   endDate,
		StoreID:   storeID,
		CashierID: cashierID,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get sales by date range", err)
	}

	result := &SalesByDateResult{
		Sales: make([]SaleReport, 0, len(sales)),
	}

	for _, sale := range sales {
		// Load store name
		storeName := ""
		var store gormdb.Store
		if err := s.repos.DB().Where("id = ?", sale.StoreID).First(&store).Error; err == nil {
			storeName = store.Name
		}

		// Load cashier name
		cashierName := ""
		var cashier gormdb.User
		if err := s.repos.DB().Where("id = ?", sale.CashierID).First(&cashier).Error; err == nil {
			cashierName = cashier.FullName
		}

		// Load customer name
		customerName := ""
		if sale.CustomerID != uuid.Nil {
			var customer gormdb.Customer
			if err := s.repos.DB().Where("id = ?", sale.CustomerID).First(&customer).Error; err == nil {
				customerName = customer.Name
			}
		}

		result.Sales = append(result.Sales, SaleReport{
			ID:            sale.ID.String(),
			InvoiceNumber: sale.InvoiceNumber,
			StoreName:     storeName,
			CashierName:   cashierName,
			CustomerName:  customerName,
			Subtotal:      sale.Subtotal,
			Discount:      sale.Discount,
			Tax:           sale.Tax,
			GrandTotal:    sale.GrandTotal,
			Status:        sale.Status,
			CreatedAt:     sale.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})

		result.TotalCount++
		result.TotalAmount += sale.GrandTotal
	}

	return result, nil
}

// GetSalesByCashier gets sales grouped by cashier
func (s *Service) GetSalesByCashier(ctx context.Context, startDate, endDate, storeID string) ([]CashierSalesReport, error) {
	var results []struct {
		CashierID        string
		CashierName      string
		TotalTransactions int64
		TotalAmount      float64
	}

	query := s.repos.DB().WithContext(ctx).
		Model(&gormdb.Sale{}).
		Select("sales.cashier_id, users.full_name as cashier_name, COUNT(*) as total_transactions, COALESCE(SUM(sales.grand_total), 0) as total_amount").
		Joins("JOIN users ON users.id = sales.cashier_id").
		Where("sales.status = ?", "PAID")

	if startDate != "" {
		query = query.Where("sales.created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("sales.created_at <= ?", endDate)
	}
	if storeID != "" {
		query = query.Where("sales.store_id = ?", storeID)
	}

	if err := query.Group("sales.cashier_id, users.full_name").Order("total_amount DESC").Find(&results).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get sales by cashier", err)
	}

	reports := make([]CashierSalesReport, 0, len(results))
	for _, r := range results {
		reports = append(reports, CashierSalesReport{
			CashierID:        r.CashierID,
			CashierName:      r.CashierName,
			TotalTransactions: r.TotalTransactions,
			TotalAmount:      r.TotalAmount,
		})
	}

	return reports, nil
}

// GetSalesByProduct gets sales grouped by product
func (s *Service) GetSalesByProduct(ctx context.Context, startDate, endDate string, limit int32) ([]ProductSalesReport, error) {
	if limit <= 0 {
		limit = 10
	}

	reports, err := s.repos.Report.GetTopSellingProducts(ctx, repository.ReportParams{
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get sales by product", err)
	}

	result := make([]ProductSalesReport, 0, len(reports))
	for _, r := range reports {
		result = append(result, ProductSalesReport{
			ProductID:    r.ProductID,
			ProductName:  r.ProductName,
			QuantitySold: r.Quantity,
			TotalAmount:  r.TotalAmount,
		})
	}

	return result, nil
}

// GetSalesByCategory gets sales grouped by category
func (s *Service) GetSalesByCategory(ctx context.Context, startDate, endDate string) ([]CategorySalesReport, error) {
	var results []struct {
		CategoryID   string
		CategoryName string
		ProductCount int64
		QuantitySold int64
		TotalAmount  float64
	}

	query := s.repos.DB().WithContext(ctx).
		Model(&gormdb.SaleItem{}).
		Select("categories.id as category_id, categories.name as category_name, COUNT(DISTINCT sale_items.product_id) as product_count, SUM(sale_items.quantity) as quantity_sold, SUM(sale_items.subtotal) as total_amount").
		Joins("JOIN products ON products.id = sale_items.product_id").
		Joins("JOIN categories ON categories.id = products.category_id").
		Joins("JOIN sales ON sales.id = sale_items.sale_id").
		Where("sales.status = ?", "PAID")

	if startDate != "" {
		query = query.Where("sales.created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("sales.created_at <= ?", endDate)
	}

	if err := query.Group("categories.id, categories.name").Order("total_amount DESC").Find(&results).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get sales by category", err)
	}

	reports := make([]CategorySalesReport, 0, len(results))
	for _, r := range results {
		reports = append(reports, CategorySalesReport{
			CategoryID:   r.CategoryID,
			CategoryName: r.CategoryName,
			ProductCount: r.ProductCount,
			QuantitySold: r.QuantitySold,
			TotalAmount:  r.TotalAmount,
		})
	}

	return reports, nil
}

// GetPaymentMethodSummary gets payment method summary
func (s *Service) GetPaymentMethodSummary(ctx context.Context, startDate, endDate string) (*PaymentMethodResult, error) {
	reports, err := s.repos.Report.GetPaymentMethodSummary(ctx, repository.ReportParams{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get payment method summary", err)
	}

	var totalAmount float64
	result := &PaymentMethodResult{
		Reports: make([]PaymentMethodReport, 0, len(reports)),
	}

	for _, r := range reports {
		result.Reports = append(result.Reports, PaymentMethodReport{
			PaymentMethod: r.PaymentMethod,
			Count:         r.Count,
			TotalAmount:   r.TotalAmount,
		})
		totalAmount += r.TotalAmount
	}

	result.TotalAmount = totalAmount
	return result, nil
}

// GetInventorySummary gets inventory summary
func (s *Service) GetInventorySummary(ctx context.Context) (*InventoryResult, error) {
	reports, err := s.repos.Report.GetInventorySummaryReport(ctx)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get inventory summary", err)
	}

	var totalProducts, totalStock int64
	var totalValue float64

	result := &InventoryResult{
		Reports: make([]InventoryReport, 0, len(reports)),
	}

	for _, r := range reports {
		result.Reports = append(result.Reports, InventoryReport{
			CategoryName: r.CategoryName,
			ProductCount: r.ProductCount,
			TotalStock:   r.TotalStock,
			TotalValue:   r.TotalValue,
		})
		totalProducts += r.ProductCount
		totalStock += r.TotalStock
		totalValue += r.TotalValue
	}

	result.TotalProducts = totalProducts
	result.TotalStock = totalStock
	result.TotalValue = totalValue

	return result, nil
}

// GetLowStockProducts gets low stock products
func (s *Service) GetLowStockProducts(ctx context.Context, limit int32) ([]LowStockReport, error) {
	if limit <= 0 {
		limit = 20
	}

	var products []gormdb.Product
	query := s.repos.DB().WithContext(ctx).
		Model(&gormdb.Product{}).
		Where("stock <= minimum_stock AND is_active = ?", true).
		Order("stock ASC").
		Limit(int(limit))

	if err := query.Find(&products).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get low stock products", err)
	}

	reports := make([]LowStockReport, 0, len(products))
	for _, p := range products {
		categoryName := ""
		if p.CategoryID != uuid.Nil {
			var cat gormdb.Category
			if err := s.repos.DB().Where("id = ?", p.CategoryID).First(&cat).Error; err == nil {
				categoryName = cat.Name
			}
		}

		reports = append(reports, LowStockReport{
			ProductID:    p.ID.String(),
			ProductName:  p.Name,
			CategoryName: categoryName,
			CurrentStock: p.Stock,
			MinimumStock: p.MinimumStock,
			Deficit:      p.MinimumStock - p.Stock,
		})
	}

	return reports, nil
}
