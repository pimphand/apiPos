package handler

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	productskuv1 "apiPos/gen/productsku/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/productsku"
)

// ProductSKUHandler handles product SKU gRPC requests
type ProductSKUHandler struct {
	productskuv1.UnimplementedProductServiceSKUServiceServer
	service *productsku.Service
	logs    *ActivityLogger
}

// NewProductSKUHandler creates a new product SKU handler
func NewProductSKUHandler(service *productsku.Service, logs *ActivityLogger) *ProductSKUHandler {
	return &ProductSKUHandler{
		service: service,
		logs:    logs,
	}
}

// CreateProductSKU creates a new product SKU
func (h *ProductSKUHandler) CreateProductSKU(ctx context.Context, req *productskuv1.CreateProductSKURequest) (*productskuv1.ProductSKUResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.Sku == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}

	var expiryDate *timestamppb.Timestamp
	if req.ExpiryDate != nil {
		expiryDate = req.ExpiryDate
	}

	var expiryTime *time.Time
	if expiryDate != nil {
		t := expiryDate.AsTime()
		expiryTime = &t
	}

	productSKU, err := h.service.CreateProductSKU(
		req.ProductId, req.Sku, req.Barcode, req.Variant, req.Color, req.Size, req.Unit,
		req.CostPrice, req.SellingPrice, req.Stock, req.MinimumStock, req.Weight, req.Volume,
		expiryTime, req.BatchNumber, req.Location, req.ImageUrl, req.IsActive, req.Metadata,
	)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CREATE", "product_skus", productSKU.ID.String(),
		`{"sku":"`+productSKU.SKU+`","barcode":"`+productSKU.Barcode+`"}`)

	return &productskuv1.ProductSKUResponse{
		ProductSku: toProtoProductSKU(productSKU),
	}, nil
}

// GetProductSKU gets a product SKU by ID
func (h *ProductSKUHandler) GetProductSKU(ctx context.Context, req *productskuv1.GetProductSKURequest) (*productskuv1.ProductSKUResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	productSKU, err := h.service.GetProductSKU(req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productskuv1.ProductSKUResponse{
		ProductSku: toProtoProductSKU(productSKU),
	}, nil
}

// GetProductSKUByBarcode gets a product SKU by barcode
func (h *ProductSKUHandler) GetProductSKUByBarcode(ctx context.Context, req *productskuv1.GetProductSKUByBarcodeRequest) (*productskuv1.ProductSKUResponse, error) {
	if req.Barcode == "" {
		return nil, status.Error(codes.InvalidArgument, "barcode is required")
	}

	productSKU, err := h.service.GetProductSKUByBarcode(req.Barcode)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productskuv1.ProductSKUResponse{
		ProductSku: toProtoProductSKU(productSKU),
	}, nil
}

// GetProductSKUBySKUCode gets a product SKU by SKU code
func (h *ProductSKUHandler) GetProductSKUBySKUCode(ctx context.Context, req *productskuv1.GetProductSKUBySKUCodeRequest) (*productskuv1.ProductSKUResponse, error) {
	if req.Sku == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}

	productSKU, err := h.service.GetProductSKUBySKUCode(req.Sku)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productskuv1.ProductSKUResponse{
		ProductSku: toProtoProductSKU(productSKU),
	}, nil
}

// ListProductSKUs lists product SKUs by product ID
func (h *ProductSKUHandler) ListProductSKUs(ctx context.Context, req *productskuv1.ListProductSKUsRequest) (*productskuv1.ListProductSKUsResponse, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	perPage := int(req.PerPage)
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	productSKUs, totalCount, err := h.service.ListProductSKUs(
		req.ProductId, req.Search, req.Variant, req.Color, req.Size, nil, page, perPage,
	)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productskuv1.ListProductSKUsResponse{
		ProductSkus: toProtoProductSKUs(productSKUs),
		TotalCount:  totalCount,
		Page:        int32(page),
		PerPage:     int32(perPage),
	}, nil
}

// ListAllProductSKUs lists all product SKUs
func (h *ProductSKUHandler) ListAllProductSKUs(ctx context.Context, req *productskuv1.ListProductSKUsRequest) (*productskuv1.ListProductSKUsResponse, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	perPage := int(req.PerPage)
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	productSKUs, totalCount, err := h.service.ListAllProductSKUs(
		req.Search, req.Variant, req.Color, req.Size, nil, req.ProductId, page, perPage,
	)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &productskuv1.ListProductSKUsResponse{
		ProductSkus: toProtoProductSKUs(productSKUs),
		TotalCount:  totalCount,
		Page:        int32(page),
		PerPage:     int32(perPage),
	}, nil
}

// UpdateProductSKU updates a product SKU
func (h *ProductSKUHandler) UpdateProductSKU(ctx context.Context, req *productskuv1.UpdateProductSKURequest) (*productskuv1.ProductSKUResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	var expiryDate *timestamppb.Timestamp
	if req.ExpiryDate != nil {
		expiryDate = req.ExpiryDate
	}

	var expiryTime *time.Time
	if expiryDate != nil {
		t := expiryDate.AsTime()
		expiryTime = &t
	}

	productSKU, err := h.service.UpdateProductSKU(
		req.Id, req.Sku, req.Barcode, req.Variant, req.Color, req.Size, req.Unit,
		req.CostPrice, req.SellingPrice, req.Stock, req.MinimumStock, req.Weight, req.Volume,
		expiryTime, req.BatchNumber, req.Location, req.ImageUrl, req.IsActive, req.Metadata,
	)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "product_skus", productSKU.ID.String(),
		`{"sku":"`+productSKU.SKU+`"}`)

	return &productskuv1.ProductSKUResponse{
		ProductSku: toProtoProductSKU(productSKU),
	}, nil
}

// DeleteProductSKU deletes a product SKU
func (h *ProductSKUHandler) DeleteProductSKU(ctx context.Context, req *productskuv1.DeleteProductSKURequest) (*productskuv1.DeleteProductSKUResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	var meta interface{}
	if ps, gerr := h.service.GetProductSKU(req.Id); gerr == nil {
		meta = ps
	}

	err := h.service.DeleteProductSKU(req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	if meta != nil {
		h.logs.LogJSON(ctx, "DELETE", "product_skus", req.Id, meta)
	} else {
		h.logs.Log(ctx, "DELETE", "product_skus", req.Id, `{}`)
	}

	return &productskuv1.DeleteProductSKUResponse{
		Success: true,
		Message: "Product SKU deleted successfully",
	}, nil
}

// toProtoProductSKU converts a GORM ProductSKU to proto ProductSKU
func toProtoProductSKU(ps *gormdb.ProductSKU) *productskuv1.ProductSKU {
	result := &productskuv1.ProductSKU{
		Id:           ps.ID.String(),
		ProductId:    ps.ProductID.String(),
		Sku:          ps.SKU,
		Barcode:      ps.Barcode,
		Variant:      ps.Variant,
		Color:        ps.Color,
		Size:         ps.Size,
		Unit:         ps.Unit,
		CostPrice:    ps.CostPrice,
		SellingPrice: ps.SellingPrice,
		Stock:        ps.Stock,
		MinimumStock: ps.MinimumStock,
		Weight:       ps.Weight,
		Volume:       ps.Volume,
		BatchNumber:  ps.BatchNumber,
		Location:     ps.Location,
		ImageUrl:     ps.ImageURL,
		IsActive:     ps.IsActive,
		CreatedAt:    timestamppb.New(ps.CreatedAt),
		UpdatedAt:    timestamppb.New(ps.UpdatedAt),
	}

	if ps.Metadata != nil {
		result.Metadata = make(map[string]string)
		for k, v := range ps.Metadata {
			result.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	if ps.ExpiryDate != nil {
		result.ExpiryDate = timestamppb.New(*ps.ExpiryDate)
	}

	return result
}

// toProtoProductSKUs converts a slice of GORM ProductSKU to proto ProductSKU
func toProtoProductSKUs(productSKUs []gormdb.ProductSKU) []*productskuv1.ProductSKU {
	result := make([]*productskuv1.ProductSKU, len(productSKUs))
	for i, ps := range productSKUs {
		result[i] = toProtoProductSKU(&ps)
	}
	return result
}
