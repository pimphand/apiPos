package handler

import (
	"context"

	reportv1 "apiPos/gen/report/v1"
	"apiPos/internal/apperror"
	"apiPos/internal/report"
)

// ReportHandler implements the ReportService gRPC server
type ReportHandler struct {
	reportv1.UnimplementedReportServiceServer
	service *report.Service
}

// NewReportHandler creates a new report Handler
func NewReportHandler(service *report.Service) *ReportHandler {
	return &ReportHandler{service: service}
}

// GetDailySalesReport gets daily sales report
func (h *ReportHandler) GetDailySalesReport(ctx context.Context, req *reportv1.DailySalesReportRequest) (*reportv1.DailySalesReportResponse, error) {
	result, err := h.service.GetDailySalesReport(ctx, req.StartDate, req.EndDate, req.StoreId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	reports := make([]*reportv1.DailySalesReport, 0, len(result.Reports))
	for _, r := range result.Reports {
		reports = append(reports, &reportv1.DailySalesReport{
			Date:        r.Date,
			TotalSales:  r.TotalSales,
			TotalAmount: r.TotalAmount,
		})
	}

	return &reportv1.DailySalesReportResponse{
		Reports:     reports,
		TotalSales:  result.TotalSales,
		TotalAmount: result.TotalAmount,
	}, nil
}

// GetSalesByDateRange gets sales by date range
func (h *ReportHandler) GetSalesByDateRange(ctx context.Context, req *reportv1.SalesByDateRangeRequest) (*reportv1.SalesByDateRangeResponse, error) {
	result, err := h.service.GetSalesByDateRange(ctx, req.StartDate, req.EndDate, req.StoreId, req.CashierId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	sales := make([]*reportv1.SaleReport, 0, len(result.Sales))
	for _, s := range result.Sales {
		sales = append(sales, &reportv1.SaleReport{
			Id:            s.ID,
			InvoiceNumber: s.InvoiceNumber,
			StoreName:     s.StoreName,
			CashierName:   s.CashierName,
			CustomerName:  s.CustomerName,
			Subtotal:      s.Subtotal,
			Discount:      s.Discount,
			Tax:           s.Tax,
			GrandTotal:    s.GrandTotal,
			Status:        s.Status,
			CreatedAt:     s.CreatedAt,
		})
	}

	return &reportv1.SalesByDateRangeResponse{
		Sales:       sales,
		TotalCount:  result.TotalCount,
		TotalAmount: result.TotalAmount,
	}, nil
}

// GetSalesByCashier gets sales by cashier
func (h *ReportHandler) GetSalesByCashier(ctx context.Context, req *reportv1.SalesByCashierRequest) (*reportv1.SalesByCashierResponse, error) {
	reports, err := h.service.GetSalesByCashier(ctx, req.StartDate, req.EndDate, req.StoreId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	result := make([]*reportv1.CashierSalesReport, 0, len(reports))
	for _, r := range reports {
		result = append(result, &reportv1.CashierSalesReport{
			CashierId:         r.CashierID,
			CashierName:       r.CashierName,
			TotalTransactions: r.TotalTransactions,
			TotalAmount:       r.TotalAmount,
		})
	}

	return &reportv1.SalesByCashierResponse{
		Reports: result,
	}, nil
}

// GetSalesByProduct gets sales by product
func (h *ReportHandler) GetSalesByProduct(ctx context.Context, req *reportv1.SalesByProductRequest) (*reportv1.SalesByProductResponse, error) {
	reports, err := h.service.GetSalesByProduct(ctx, req.StartDate, req.EndDate, req.Limit)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	result := make([]*reportv1.ProductSalesReport, 0, len(reports))
	for _, r := range reports {
		result = append(result, &reportv1.ProductSalesReport{
			ProductId:    r.ProductID,
			ProductName:  r.ProductName,
			QuantitySold: r.QuantitySold,
			TotalAmount:  r.TotalAmount,
		})
	}

	return &reportv1.SalesByProductResponse{
		Reports: result,
	}, nil
}

// GetSalesByCategory gets sales by category
func (h *ReportHandler) GetSalesByCategory(ctx context.Context, req *reportv1.SalesByCategoryRequest) (*reportv1.SalesByCategoryResponse, error) {
	reports, err := h.service.GetSalesByCategory(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	result := make([]*reportv1.CategorySalesReport, 0, len(reports))
	for _, r := range reports {
		result = append(result, &reportv1.CategorySalesReport{
			CategoryId:   r.CategoryID,
			CategoryName: r.CategoryName,
			ProductCount: r.ProductCount,
			QuantitySold: r.QuantitySold,
			TotalAmount:  r.TotalAmount,
		})
	}

	return &reportv1.SalesByCategoryResponse{
		Reports: result,
	}, nil
}

// GetPaymentMethodSummary gets payment method summary
func (h *ReportHandler) GetPaymentMethodSummary(ctx context.Context, req *reportv1.PaymentMethodSummaryRequest) (*reportv1.PaymentMethodSummaryResponse, error) {
	result, err := h.service.GetPaymentMethodSummary(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	reports := make([]*reportv1.PaymentMethodReport, 0, len(result.Reports))
	for _, r := range result.Reports {
		reports = append(reports, &reportv1.PaymentMethodReport{
			PaymentMethod: r.PaymentMethod,
			Count:         r.Count,
			TotalAmount:   r.TotalAmount,
		})
	}

	return &reportv1.PaymentMethodSummaryResponse{
		Reports:     reports,
		TotalAmount: result.TotalAmount,
	}, nil
}

// GetInventorySummary gets inventory summary
func (h *ReportHandler) GetInventorySummary(ctx context.Context, req *reportv1.InventorySummaryRequest) (*reportv1.InventorySummaryResponse, error) {
	result, err := h.service.GetInventorySummary(ctx)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	reports := make([]*reportv1.InventoryReport, 0, len(result.Reports))
	for _, r := range result.Reports {
		reports = append(reports, &reportv1.InventoryReport{
			CategoryName: r.CategoryName,
			ProductCount: r.ProductCount,
			TotalStock:   r.TotalStock,
			TotalValue:   r.TotalValue,
		})
	}

	return &reportv1.InventorySummaryResponse{
		Reports:       reports,
		TotalProducts: result.TotalProducts,
		TotalStock:    result.TotalStock,
		TotalValue:    result.TotalValue,
	}, nil
}

// GetLowStockProducts gets low stock products
func (h *ReportHandler) GetLowStockProducts(ctx context.Context, req *reportv1.LowStockProductsRequest) (*reportv1.LowStockProductsResponse, error) {
	reports, err := h.service.GetLowStockProducts(ctx, req.Limit)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	result := make([]*reportv1.LowStockReport, 0, len(reports))
	for _, r := range reports {
		result = append(result, &reportv1.LowStockReport{
			ProductId:    r.ProductID,
			ProductName:  r.ProductName,
			CategoryName: r.CategoryName,
			CurrentStock: r.CurrentStock,
			MinimumStock: r.MinimumStock,
			Deficit:      r.Deficit,
		})
	}

	return &reportv1.LowStockProductsResponse{
		Products: result,
	}, nil
}
