package handler

import (
	"context"

	selforderv1 "apiPos/gen/selforder/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/selforder"
)

// SelfOrderHandler implements the public SelfOrderService gRPC server.
// All methods are unauthenticated — the service resolves the tenant from the
// table token / order token.
type SelfOrderHandler struct {
	selforderv1.UnimplementedSelfOrderServiceServer
	service *selforder.Service
	logs    *ActivityLogger
}

// NewSelfOrderHandler creates a new self-order Handler
func NewSelfOrderHandler(service *selforder.Service, logs *ActivityLogger) *SelfOrderHandler {
	return &SelfOrderHandler{service: service, logs: logs}
}

// ResolveTable resolves a table token into table + store info.
func (h *SelfOrderHandler) ResolveTable(ctx context.Context, req *selforderv1.ResolveTableRequest) (*selforderv1.TableInfoResponse, error) {
	info, err := h.service.ResolveTable(ctx, req.TableToken)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return &selforderv1.TableInfoResponse{
		Table: &selforderv1.TableInfo{
			Id:               info.Table.ID.String(),
			StoreId:          info.Table.StoreID.String(),
			StoreName:        info.Store.Name,
			Code:             info.Table.Code,
			Name:             info.Table.Name,
			SelfOrderEnabled: info.Setting != nil && info.Setting.SelfOrderEnabled,
		},
	}, nil
}

// GetPublicMenu returns the safe dine-in menu for the table's store.
func (h *SelfOrderHandler) GetPublicMenu(ctx context.Context, req *selforderv1.GetPublicMenuRequest) (*selforderv1.GetPublicMenuResponse, error) {
	info, categories, products, err := h.service.GetPublicMenu(ctx, req.TableToken)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	resp := &selforderv1.GetPublicMenuResponse{
		StoreName: info.Store.Name,
		TableCode: info.Table.Code,
		TableName: info.Table.Name,
	}
	for _, c := range categories {
		resp.Categories = append(resp.Categories, &selforderv1.PublicCategory{
			Id:   c.ID.String(),
			Name: c.Name,
		})
	}
	for _, p := range products {
		proto := &selforderv1.PublicProduct{
			Id:           p.Product.ID.String(),
			Name:         p.Product.Name,
			Description:  p.Product.Description,
			CategoryId:   p.Product.CategoryID.String(),
			CategoryName: p.CategoryName,
		}
		for _, sku := range p.Variants {
			if proto.ImageUrl == "" {
				proto.ImageUrl = sku.ImageURL
			}
			proto.Variants = append(proto.Variants, &selforderv1.PublicVariant{
				SkuId:        sku.ID.String(),
				SkuCode:      sku.SKU,
				Variant:      sku.Variant,
				Color:        sku.Color,
				Size:         sku.Size,
				SellingPrice: sku.SellingPrice,
				Available:    sku.IsActive && sku.Stock > 0,
			})
		}
		resp.Products = append(resp.Products, proto)
	}
	return resp, nil
}

// CreatePublicOrder submits a self order (PENDING, stock locked).
func (h *SelfOrderHandler) CreatePublicOrder(ctx context.Context, req *selforderv1.CreatePublicOrderRequest) (*selforderv1.OrderResponse, error) {
	items := make([]selforder.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, selforder.OrderItem{
			ProductID:    it.ProductId,
			ProductSKUID: it.ProductSkuId,
			Quantity:     it.Quantity,
		})
	}
	sale, err := h.service.CreatePublicOrder(ctx, req.TableToken, items, req.CustomerName, req.Notes)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Audit trail with no user (customer action, docs §6.2).
	h.logs.LogWithUser(ctx, "", sale.StoreID.String(), "", "CREATE", "sales", sale.ID.String(),
		`{"invoice_number":"`+sale.InvoiceNumber+`","order_source":"SELF_ORDER"}`)

	return &selforderv1.OrderResponse{Order: toProtoOrder(sale)}, nil
}

// PayPublicOrder pays a submitted order (→ PAID) and triggers realtime events.
func (h *SelfOrderHandler) PayPublicOrder(ctx context.Context, req *selforderv1.PayPublicOrderRequest) (*selforderv1.OrderResponse, error) {
	sale, err := h.service.PayPublicOrder(ctx, req.OrderToken, req.PaymentMethod, req.Reference)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.LogWithUser(ctx, "", sale.StoreID.String(), "", "PAY", "sales", sale.ID.String(),
		`{"invoice_number":"`+sale.InvoiceNumber+`","payment_method":"`+req.PaymentMethod+`"}`)

	return &selforderv1.OrderResponse{Order: toProtoOrder(sale)}, nil
}

// GetOrderStatus returns the customer's order status.
func (h *SelfOrderHandler) GetOrderStatus(ctx context.Context, req *selforderv1.GetOrderStatusRequest) (*selforderv1.OrderResponse, error) {
	sale, err := h.service.GetOrderStatus(ctx, req.OrderToken)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return &selforderv1.OrderResponse{Order: toProtoOrder(sale)}, nil
}

// toProtoOrder converts a GORM Sale to the public Order proto.
func toProtoOrder(s *gormdb.Sale) *selforderv1.Order {
	order := &selforderv1.Order{
		Id:           s.ID.String(),
		InvoiceNumber: s.InvoiceNumber,
		StoreId:      s.StoreID.String(),
		TableId:      "",
		TableNo:      s.TableNo,
		Status:       s.Status,
		OrderStatus:  s.OrderStatus,
		OrderSource:  s.OrderSource,
		CustomerName: "",
		Subtotal:     s.Subtotal,
		Discount:     s.Discount,
		Tax:          s.Tax,
		GrandTotal:   s.GrandTotal,
		OrderToken:   "",
		CreatedAt:    s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if s.TableID != nil {
		order.TableId = s.TableID.String()
	}
	if s.OrderToken != nil {
		order.OrderToken = s.OrderToken.String()
	}
	return order
}