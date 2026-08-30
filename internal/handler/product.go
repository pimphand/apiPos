package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	productv1 "apiPos/gen/product/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/product"
	"apiPos/internal/repository"
	"apiPos/internal/response"
)

// ProductHandler implements the ProductService gRPC server
type ProductHandler struct {
	productv1.UnimplementedProductServiceServer
	service *product.Service
	repos   *repository.Repositories
	logs    *ActivityLogger
}

// NewProductHandler creates a new product Handler
func NewProductHandler(service *product.Service, repos *repository.Repositories, logs *ActivityLogger) *ProductHandler {
	return &ProductHandler{service: service, repos: repos, logs: logs}
}

// CreateProduct creates a new product
func (h *ProductHandler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.ProductResponse, error) {
	p, err := h.service.CreateProduct(ctx, product.CreateProductRequest{
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		CostPrice:    req.CostPrice,
		SellingPrice: req.SellingPrice,
		Stock:        req.Stock,
		MinimumStock: req.MinimumStock,
		SKU:          req.Sku,
		Barcode:      req.Barcode,
		Variant:      req.Variant,
		Color:        req.Color,
		Size:         req.Size,
		Unit:         req.Unit,
		Weight:       req.Weight,
		Volume:       req.Volume,
		BatchNumber:  req.BatchNumber,
		Location:     req.Location,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Load SKU count and category name for the created product
	protoProduct := h.toProtoProductEnriched(p)

	h.logs.Log(ctx, "CREATE", "products", p.ID.String(),
		`{"name":"`+p.Name+`","sku":"`+req.Sku+`"}`)

	return &productv1.ProductResponse{
		Product: protoProduct,
	}, nil
}

// GetProduct gets a product by ID
func (h *ProductHandler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.ProductResponse, error) {
	p, err := h.service.GetProduct(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Load SKUs if get_sku is true
	if req.GetSku && h.repos != nil {
		skus, _, err := h.repos.ProductSKU.List(
			p.ID,
			"",  // search
			"",  // variant
			"",  // color
			"",  // size
			nil, // isActive
			1,   // page
			100, // perPage
		)
		if err == nil {
			p.ProductSKUs = skus
		}
	}

	protoProduct := h.toProtoProductEnriched(p)

	return &productv1.ProductResponse{
		Product: protoProduct,
	}, nil
}

// ListProducts lists products with filters and pagination
func (h *ProductHandler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	result, err := h.service.ListProducts(ctx, req.IsActive, req.CategoryId, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Load SKUs if get_sku is true
	if req.GetSku {
		h.service.LoadProductSKUs(ctx, result.Products)
	}

	protoProducts := h.toProtoProductsEnriched(result.Products)

	return &productv1.ListProductsResponse{
		Products:   protoProducts,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// SearchProducts searches products by name
func (h *ProductHandler) SearchProducts(ctx context.Context, req *productv1.SearchProductsRequest) (*productv1.ListProductsResponse, error) {
	result, err := h.service.SearchProducts(ctx, req.Name, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Load SKUs if get_sku is true
	if req.GetSku {
		h.service.LoadProductSKUs(ctx, result.Products)
	}

	protoProducts := h.toProtoProductsEnriched(result.Products)

	return &productv1.ListProductsResponse{
		Products:   protoProducts,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// UpdateProduct updates a product
func (h *ProductHandler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.ProductResponse, error) {
	// Convert SKU updates
	var skuUpdates []product.SKUUpdateItem
	for _, sku := range req.SkuUpdates {
		skuUpdates = append(skuUpdates, product.SKUUpdateItem{
			ID:           sku.Id,
			SKU:          sku.Sku,
			Barcode:      sku.Barcode,
			Variant:      sku.Variant,
			Color:        sku.Color,
			Size:         sku.Size,
			Unit:         sku.Unit,
			CostPrice:    sku.CostPrice,
			SellingPrice: sku.SellingPrice,
			Stock:        sku.Stock,
			MinimumStock: sku.MinimumStock,
			Weight:       sku.Weight,
			Volume:       sku.Volume,
			BatchNumber:  sku.BatchNumber,
			Location:     sku.Location,
			ImageURL:     sku.ImageUrl,
			IsActive:     sku.IsActive,
			Metadata:     sku.Metadata,
		})
		if len(skuUpdates) > 0 {
			skuUpdates[len(skuUpdates)-1].ExpiryDate = sku.ExpiryDate
		}
	}

	p, err := h.service.UpdateProduct(ctx, product.UpdateProductRequest{
		ID:           req.Id,
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		CostPrice:    req.CostPrice,
		SellingPrice: req.SellingPrice,
		MinimumStock: req.MinimumStock,
		IsActive:     req.IsActive,
		SKUUpdates:   skuUpdates,
		SKUsToDelete: req.SkuIdsToDelete,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Always load SKUs after update so response shows current state
	if h.repos != nil {
		skus, _, err := h.repos.ProductSKU.List(
			p.ID,
			"",  // search
			"",  // variant
			"",  // color
			"",  // size
			nil, // isActive
			1,   // page
			100, // perPage
		)
		if err == nil {
			p.ProductSKUs = skus
		}
	}

	protoProduct := h.toProtoProductEnriched(p)

	h.logs.Log(ctx, "UPDATE", "products", p.ID.String(),
		`{"name":"`+p.Name+`"}`)

	return &productv1.ProductResponse{
		Product: protoProduct,
	}, nil
}

// DeleteProduct deletes a product
func (h *ProductHandler) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductResponse, error) {
	var meta interface{}
	if p, gerr := h.service.GetProduct(ctx, req.Id); gerr == nil {
		meta = p
	}

	err := h.service.DeleteProduct(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	if meta != nil {
		h.logs.LogJSON(ctx, "DELETE", "products", req.Id, meta)
	} else {
		h.logs.Log(ctx, "DELETE", "products", req.Id, `{}`)
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

	h.logs.Log(ctx, "UPDATE", "products", req.Id, `{"is_active":true}`)

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

	h.logs.Log(ctx, "UPDATE", "products", req.Id, `{"is_active":false}`)

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

	// Load SKUs if get_sku is true
	if req.GetSku {
		h.service.LoadProductSKUs(ctx, result.Products)
	}

	protoProducts := h.toProtoProductsEnriched(result.Products)

	return &productv1.ListProductsResponse{
		Products:   protoProducts,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// toProtoProductsEnriched converts a list of products to proto with SKU counts and category names
func (h *ProductHandler) toProtoProductsEnriched(products []gormdb.Product) []*productv1.Product {
	if len(products) == 0 {
		return []*productv1.Product{}
	}

	// Batch collect product IDs for SKU count
	productIDs := make([]uuid.UUID, 0, len(products))
	categoryIDSet := make(map[string]struct{})
	for _, p := range products {
		productIDs = append(productIDs, p.ID)
		if p.CategoryID != uuid.Nil {
			categoryIDSet[p.CategoryID.String()] = struct{}{}
		}
	}

	// Batch load SKU counts
	skuCountMap := make(map[string]int32)
	if h.repos != nil {
		counts, err := h.repos.ProductSKU.CountByProductIDs(productIDs)
		if err != nil {
			log.Printf("Warning: failed to batch load SKU counts: %v", err)
		} else {
			skuCountMap = counts
		}
	}

	// Batch load category names
	categoryNameMap := make(map[string]string)
	if h.repos != nil && len(categoryIDSet) > 0 {
		catIDs := make([]string, 0, len(categoryIDSet))
		for id := range categoryIDSet {
			catIDs = append(catIDs, id)
		}
		cats, err := h.repos.Category.GetByIDs(context.Background(), catIDs)
		if err != nil {
			log.Printf("Warning: failed to batch load category names: %v", err)
		} else {
			for _, c := range cats {
				categoryNameMap[c.ID.String()] = c.Name
			}
		}
	}

	// Convert products
	protoProducts := make([]*productv1.Product, len(products))
	for i, p := range products {
		protoProduct := toProtoProductFromGORM(&p)
		protoProduct.SkuCount = skuCountMap[p.ID.String()]
		if name, ok := categoryNameMap[p.CategoryID.String()]; ok {
			protoProduct.CategoryName = name
		}
		protoProducts[i] = protoProduct
	}

	return protoProducts
}

// toProtoProductEnriched converts a single product to proto with SKU count and category name
func (h *ProductHandler) toProtoProductEnriched(p *gormdb.Product) *productv1.Product {
	protoProduct := toProtoProductFromGORM(p)

	// Load SKU count
	if h.repos != nil {
		counts, err := h.repos.ProductSKU.CountByProductIDs([]uuid.UUID{p.ID})
		if err == nil {
			protoProduct.SkuCount = counts[p.ID.String()]
		}
		// If SKUs are already loaded (e.g. from GetProduct), use that count
		if len(p.ProductSKUs) > 0 {
			protoProduct.SkuCount = int32(len(p.ProductSKUs))
		}
	}

	// Load category name
	if h.repos != nil && p.CategoryID != uuid.Nil {
		cats, err := h.repos.Category.GetByIDs(context.Background(), []string{p.CategoryID.String()})
		if err == nil && len(cats) > 0 {
			protoProduct.CategoryName = cats[0].Name
		}
	}

	return protoProduct
}

// toProtoProductFromGORM converts a GORM Product model to proto Product
func toProtoProductFromGORM(p *gormdb.Product) *productv1.Product {
	protoProduct := &productv1.Product{
		Id:           p.ID.String(),
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

	// Convert SKUs if loaded
	if len(p.ProductSKUs) > 0 {
		protoSKUs := make([]*productv1.ProductSKU, len(p.ProductSKUs))
		for i, sku := range p.ProductSKUs {
			protoSKUs[i] = toProtoProductSKUFromGORM(&sku)
		}
		protoProduct.Skus = protoSKUs
		protoProduct.SkuCount = int32(len(p.ProductSKUs))
	}

	return protoProduct
}

// toProtoProductSKUFromGORM converts a GORM ProductSKU model to proto ProductSKU
func toProtoProductSKUFromGORM(sku *gormdb.ProductSKU) *productv1.ProductSKU {
	protoSKU := &productv1.ProductSKU{
		Id:           sku.ID.String(),
		ProductId:    sku.ProductID.String(),
		Sku:          sku.SKU,
		Barcode:      sku.Barcode,
		Variant:      sku.Variant,
		Color:        sku.Color,
		Size:         sku.Size,
		Unit:         sku.Unit,
		CostPrice:    sku.CostPrice,
		SellingPrice: sku.SellingPrice,
		Stock:        sku.Stock,
		MinimumStock: sku.MinimumStock,
		Weight:       sku.Weight,
		Volume:       sku.Volume,
		BatchNumber:  sku.BatchNumber,
		Location:     sku.Location,
		ImageUrl:     sku.ImageURL,
		IsActive:     sku.IsActive,
		CreatedAt:    sku.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    sku.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if sku.ExpiryDate != nil {
		protoSKU.ExpiryDate = timestamppb.New(*sku.ExpiryDate)
	}

	if sku.Metadata != nil {
		protoSKU.Metadata = make(map[string]string)
		for k, v := range sku.Metadata {
			protoSKU.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	return protoSKU
}
