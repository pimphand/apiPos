package handler

import (
	"context"

	productv1 "apiPos/gen/product/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/product"
	"apiPos/internal/response"
)

// ProductHandler implements the ProductService gRPC server
type ProductHandler struct {
	productv1.UnimplementedProductServiceServer
	service *product.Service
}

// NewProductHandler creates a new product Handler
func NewProductHandler(service *product.Service) *ProductHandler {
	return &ProductHandler{service: service}
}

// CreateProduct creates a new product
func (h *ProductHandler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.ProductResponse, error) {
	p, err := h.service.CreateProduct(ctx, product.CreateProductRequest{
		SKU:          req.Sku,
		Barcode:      req.Barcode,
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		CostPrice:    req.CostPrice,
		SellingPrice: req.SellingPrice,
		Stock:        req.Stock,
		MinimumStock: req.MinimumStock,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.ProductResponse{
		Product: toProtoProductFromGORM(p),
	}, nil
}

// GetProduct gets a product by ID
func (h *ProductHandler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.ProductResponse, error) {
	p, err := h.service.GetProduct(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.ProductResponse{
		Product: toProtoProductFromGORM(p),
	}, nil
}

// GetProductByBarcode looks up a product by barcode
func (h *ProductHandler) GetProductByBarcode(ctx context.Context, req *productv1.GetProductByBarcodeRequest) (*productv1.ProductResponse, error) {
	p, err := h.service.GetProductByBarcode(ctx, req.Barcode)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.ProductResponse{
		Product: toProtoProductFromGORM(p),
	}, nil
}

// GetProductBySKU looks up a product by SKU
func (h *ProductHandler) GetProductBySKU(ctx context.Context, req *productv1.GetProductBySKURequest) (*productv1.ProductResponse, error) {
	p, err := h.service.GetProductBySKU(ctx, req.Sku)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.ProductResponse{
		Product: toProtoProductFromGORM(p),
	}, nil
}

// ListProducts lists products with filters and pagination
func (h *ProductHandler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	result, err := h.service.ListProducts(ctx, req.IsActive, req.CategoryId, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoProducts := make([]*productv1.Product, len(result.Products))
	for i, p := range result.Products {
		protoProducts[i] = toProtoProductFromGORM(&p)
	}

	return &productv1.ListProductsResponse{
		Products:    protoProducts,
		TotalCount:  result.Pagination.TotalCount,
		Page:        result.Pagination.Page,
		PerPage:     result.Pagination.PerPage,
	}, nil
}

// SearchProducts searches products by name
func (h *ProductHandler) SearchProducts(ctx context.Context, req *productv1.SearchProductsRequest) (*productv1.ListProductsResponse, error) {
	result, err := h.service.SearchProducts(ctx, req.Name, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoProducts := make([]*productv1.Product, len(result.Products))
	for i, p := range result.Products {
		protoProducts[i] = toProtoProductFromGORM(&p)
	}

	return &productv1.ListProductsResponse{
		Products:    protoProducts,
		TotalCount:  result.Pagination.TotalCount,
		Page:        result.Pagination.Page,
		PerPage:     result.Pagination.PerPage,
	}, nil
}

// UpdateProduct updates a product
func (h *ProductHandler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.ProductResponse, error) {
	p, err := h.service.UpdateProduct(ctx, product.UpdateProductRequest{
		ID:           req.Id,
		SKU:          req.Sku,
		Barcode:      req.Barcode,
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		CostPrice:    req.CostPrice,
		SellingPrice: req.SellingPrice,
		MinimumStock: req.MinimumStock,
		IsActive:     req.IsActive,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.ProductResponse{
		Product: toProtoProductFromGORM(p),
	}, nil
}

// DeleteProduct deletes a product
func (h *ProductHandler) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductResponse, error) {
	err := h.service.DeleteProduct(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.DeleteProductResponse{
		Success: true,
		Message: response.MsgProductDeleted,
	}, nil
}

// ActivateProduct activates a product
func (h *ProductHandler) ActivateProduct(ctx context.Context, req *productv1.ActivateProductRequest) (*productv1.DeleteProductResponse, error) {
	err := h.service.ActivateProduct(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.DeleteProductResponse{
		Success: true,
		Message: response.MsgProductActivated,
	}, nil
}

// DeactivateProduct deactivates a product
func (h *ProductHandler) DeactivateProduct(ctx context.Context, req *productv1.DeactivateProductRequest) (*productv1.DeleteProductResponse, error) {
	err := h.service.DeactivateProduct(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productv1.DeleteProductResponse{
		Success: true,
		Message: response.MsgProductDeactivated,
	}, nil
}

// GetLowStockProducts lists products with low stock
func (h *ProductHandler) GetLowStockProducts(ctx context.Context, req *productv1.LowStockProductsRequest) (*productv1.ListProductsResponse, error) {
	result, err := h.service.GetLowStockProducts(ctx, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoProducts := make([]*productv1.Product, len(result.Products))
	for i, p := range result.Products {
		protoProducts[i] = toProtoProductFromGORM(&p)
	}

	return &productv1.ListProductsResponse{
		Products:    protoProducts,
		TotalCount:  result.Pagination.TotalCount,
		Page:        result.Pagination.Page,
		PerPage:     result.Pagination.PerPage,
	}, nil
}

// toProtoProductFromGORM converts a GORM Product model to proto Product
func toProtoProductFromGORM(p *gormdb.Product) *productv1.Product {
	return &productv1.Product{
		Id:           p.ID.String(),
		Sku:          p.SKU,
		Barcode:      p.Barcode,
		Name:         p.Name,
		Description:  p.Description,
		CategoryId:   p.CategoryID.String(),
		CostPrice:    p.CostPrice,
		SellingPrice: p.SellingPrice,
		Stock:        p.Stock,
		MinimumStock: p.MinimumStock,
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
