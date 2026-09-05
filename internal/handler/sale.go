package handler

import (
	"context"

	salev1 "apiPos/gen/sale/v1"
	"github.com/google/uuid"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
	"apiPos/internal/sale"
)

// SaleHandler implements the SaleService gRPC server
type SaleHandler struct {
	salev1.UnimplementedSaleServiceServer
	service *sale.Service
	repos   *repository.Repositories
	logs    *ActivityLogger
}

// NewSaleHandler creates a new sale Handler
func NewSaleHandler(service *sale.Service, repos *repository.Repositories, logs *ActivityLogger) *SaleHandler {
	return &SaleHandler{service: service, repos: repos, logs: logs}
}

// CreateSale creates a new sale (checkout)
func (h *SaleHandler) CreateSale(ctx context.Context, req *salev1.CreateSaleRequest) (*salev1.SaleResponse, error) {
	// Get user from context
	userID := getUserIDFromContext(ctx)

	// Convert items
	var items []sale.SaleItemInput
	for _, item := range req.Items {
		items = append(items, sale.SaleItemInput{
			ProductID: item.ProductId,
			SKUID:     item.SkuId,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Discount:  item.Discount,
		})
	}

	detail, err := h.service.CreateSale(ctx, sale.CreateSaleRequest{
		StoreID:          req.StoreId,
		CustomerID:       req.CustomerId,
		CustomerName:     req.CustomerName,
		Items:            items,
		Discount:         req.Discount,
		TaxRate:          req.TaxRate,
		PaymentMethod:    req.PaymentMethod,
		AmountPaid:       req.AmountPaid,
		PaymentReference: req.PaymentReference,
		CreatedBy:        userID,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CREATE", "sales", detail.Sale.ID.String(),
		`{"invoice_number":"`+detail.Sale.InvoiceNumber+`","grand_total":`+float64Str(detail.Sale.GrandTotal)+`}`)

	return &salev1.SaleResponse{
		Sale: toProtoSale(&detail.Sale, h.repos),
	}, nil
}

// GetSale gets a sale by ID with items
func (h *SaleHandler) GetSale(ctx context.Context, req *salev1.GetSaleRequest) (*salev1.SaleDetailResponse, error) {
	detail, err := h.service.GetSale(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return toProtoSaleDetail(detail, h.repos), nil
}

// GetSaleByInvoice gets a sale by invoice number
func (h *SaleHandler) GetSaleByInvoice(ctx context.Context, req *salev1.GetSaleByInvoiceRequest) (*salev1.SaleDetailResponse, error) {
	detail, err := h.service.GetSaleByInvoice(ctx, req.InvoiceNumber)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return toProtoSaleDetail(detail, h.repos), nil
}

// ListSales lists sales with filters
func (h *SaleHandler) ListSales(ctx context.Context, req *salev1.ListSalesRequest) (*salev1.ListSalesResponse, error) {
	result, err := h.service.ListSales(ctx, repository.SaleListParams{
		StoreID:    req.StoreId,
		CashierID:  req.CashierId,
		CustomerID: req.CustomerId,
		Status:     req.Status,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		Page:       req.Page,
		PerPage:    req.PerPage,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	sales := make([]*salev1.Sale, 0, len(result.Sales))
	for _, s := range result.Sales {
		sales = append(sales, toProtoSale(&s, h.repos))
	}

	return &salev1.ListSalesResponse{
		Sales:      sales,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// CancelSale cancels a sale
func (h *SaleHandler) CancelSale(ctx context.Context, req *salev1.CancelSaleRequest) (*salev1.SaleResponse, error) {
	userID := getUserIDFromContext(ctx)
	err := h.service.CancelSale(ctx, req.Id, userID)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	detail, err := h.service.GetSale(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CANCEL", "sales", req.Id,
		`{"invoice_number":"`+detail.Sale.InvoiceNumber+`"}`)

	return &salev1.SaleResponse{
		Sale: toProtoSale(&detail.Sale, h.repos),
	}, nil
}

// RefundSale refunds a sale
func (h *SaleHandler) RefundSale(ctx context.Context, req *salev1.RefundSaleRequest) (*salev1.SaleResponse, error) {
	userID := getUserIDFromContext(ctx)
	err := h.service.RefundSale(ctx, req.Id, userID)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	detail, err := h.service.GetSale(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "REFUND", "sales", req.Id,
		`{"invoice_number":"`+detail.Sale.InvoiceNumber+`"}`)

	return &salev1.SaleResponse{
		Sale: toProtoSale(&detail.Sale, h.repos),
	}, nil
}

// toProtoSale converts a GORM Sale to proto Sale
func toProtoSale(s *gormdb.Sale, repos *repository.Repositories) *salev1.Sale {
	proto := &salev1.Sale{
		Id:            s.ID.String(),
		InvoiceNumber: s.InvoiceNumber,
		StoreId:       s.StoreID.String(),
		CashierId:     s.CashierID.String(),
		CustomerId:    customerIDStr(s.CustomerID),
		Subtotal:      s.Subtotal,
		Discount:      s.Discount,
		Tax:           s.Tax,
		GrandTotal:    s.GrandTotal,
		Status:        s.Status,
		CreatedAt:     s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Load store name
	if repos != nil {
		var store gormdb.Store
		if err := repos.DB().Where("id = ?", s.StoreID).First(&store).Error; err == nil {
			proto.StoreName = store.Name
		}

		// Load cashier name
		var cashier gormdb.User
		if err := repos.DB().Where("id = ?", s.CashierID).First(&cashier).Error; err == nil {
			proto.CashierName = cashier.FullName
		}

		// Load customer name
		if s.CustomerID != uuid.Nil {
			var customer gormdb.Customer
			if err := repos.DB().Where("id = ?", s.CustomerID).First(&customer).Error; err == nil {
				proto.CustomerName = customer.Name
			}
		}

		// Count items
		var itemCount int64
		repos.DB().Model(&gormdb.SaleItem{}).Where("sale_id = ?", s.ID).Count(&itemCount)
		proto.ItemCount = int32(itemCount)
	}

	return proto
}

// customerIDStr returns the customer ID string, or empty when the sale has no
// customer (NULL in the database maps to the zero UUID).
func customerIDStr(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

// toProtoSaleDetail converts a SaleDetail to proto SaleDetailResponse
func toProtoSaleDetail(detail *sale.SaleDetail, repos *repository.Repositories) *salev1.SaleDetailResponse {
	protoSale := toProtoSale(&detail.Sale, repos)

	// Convert items
	items := make([]*salev1.SaleItem, 0, len(detail.Items))
	for _, item := range detail.Items {
		protoItem := &salev1.SaleItem{
			Id:        item.ID.String(),
			SaleId:    item.SaleID.String(),
			ProductId: item.ProductID.String(),
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Discount:  item.Discount,
			Subtotal:  item.Subtotal,
			CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Load product name
		if repos != nil {
			var product gormdb.Product
			if err := repos.DB().Where("id = ?", item.ProductID).First(&product).Error; err == nil {
				protoItem.ProductName = product.Name
			}
		}

		items = append(items, protoItem)
	}

	// Convert payments
	payments := make([]*salev1.Payment, 0, len(detail.Payments))
	for _, p := range detail.Payments {
		payments = append(payments, &salev1.Payment{
			Id:            p.ID.String(),
			SaleId:        p.SaleID.String(),
			PaymentMethod: p.PaymentMethod,
			Amount:        p.Amount,
			Reference:     p.Reference,
			PaidAt:        p.PaidAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &salev1.SaleDetailResponse{
		Sale:       protoSale,
		Items:      items,
		Payments:   payments,
		AmountPaid: detail.AmountPaid,
		Change:     detail.Change,
	}
}
