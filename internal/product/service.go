package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/realtime"
	"apiPos/internal/repository"
	"apiPos/pkg/timeutil"
)

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListProductsResult holds the result of listing products
type ListProductsResult struct {
	Products   []gormdb.Product
	Pagination PaginationResult
}

// Service handles product business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new product Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreateProduct creates a new product
func (s *Service) CreateProduct(ctx context.Context, req CreateProductRequest) (*gormdb.Product, error) {
	// Parse category_id
	var categoryID uuid.UUID
	var err error
	if req.CategoryID != "" {
		categoryID, err = uuid.Parse(req.CategoryID)
		if err != nil {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid category_id format", fmt.Errorf("category_id: %s", req.CategoryID))
		}
	}

	// Create product
	product := &gormdb.Product{
		SKU:          req.SKU,
		Barcode:      req.Barcode,
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   categoryID,
		CostPrice:    req.CostPrice,
		SellingPrice: req.SellingPrice,
		Stock:        req.Stock,
		MinimumStock: req.MinimumStock,
	}

	if err := s.repos.Product.Create(ctx, product); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create product", err)
	}

	// Create default SKU if SKU is provided
	if req.SKU != "" {
		defaultSKU := &gormdb.ProductSKU{
			ProductID:    product.ID,
			SKU:          req.SKU,
			Barcode:      req.Barcode,
			Variant:      req.Variant,
			Color:        req.Color,
			Size:         req.Size,
			Unit:         req.Unit,
			CostPrice:    req.CostPrice,
			SellingPrice: req.SellingPrice,
			Stock:        req.Stock,
			MinimumStock: req.MinimumStock,
			Weight:       req.Weight,
			Volume:       req.Volume,
			BatchNumber:  req.BatchNumber,
			Location:     req.Location,
			IsActive:     true,
		}

		if req.Unit != "" {
			defaultSKU.Unit = req.Unit
		} else {
			defaultSKU.Unit = "pcs"
		}

		if err := s.repos.ProductSKU.Create(ctx, defaultSKU); err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create default SKU", err)
		}
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventProductCreated,
		Resource:   "products",
		ResourceID: product.ID.String(),
	})

	return product, nil
}

// GetProduct gets a product by ID (without SKUs by default for performance)
func (s *Service) GetProduct(ctx context.Context, id string) (*gormdb.Product, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID format", fmt.Errorf("id: %s", id))
	}

	product, err := s.repos.Product.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get product", err)
	}

	return product, nil
}

// GetProductByBarcode looks up a product by barcode (via product_skus)
func (s *Service) GetProductByBarcode(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
	if barcode == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Barcode is required", nil)
	}

	productSKU, err := s.repos.ProductSKU.GetByBarcode(ctx, barcode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get product by barcode", err)
	}

	return productSKU, nil
}

// GetProductBySKU looks up a product by SKU (via product_skus)
func (s *Service) GetProductBySKU(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
	if sku == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "SKU is required", nil)
	}

	productSKU, err := s.repos.ProductSKU.GetBySKU(ctx, sku)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get product by SKU", err)
	}

	return productSKU, nil
}

// ListProducts lists products with filters and pagination
func (s *Service) ListProducts(ctx context.Context, isActiveStr string, categoryIDStr string, page, perPage int32) (*ListProductsResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	products, totalCount, err := s.repos.Product.List(ctx, repository.ProductListParams{
		IsActive:   isActiveStr,
		CategoryID: categoryIDStr,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list products", err)
	}

	return &ListProductsResult{
		Products: products,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// LoadProductSKUs loads SKUs for multiple products in batch
func (s *Service) LoadProductSKUs(ctx context.Context, products []gormdb.Product) {
	if len(products) == 0 {
		return
	}

	productIDs := make([]uuid.UUID, len(products))
	for i, p := range products {
		productIDs[i] = p.ID
	}

	// Batch load all SKUs for these products
	for i := range products {
		skus, _, err := s.repos.ProductSKU.List(
			ctx,
			products[i].ID,
			"",  // search
			"",  // variant
			"",  // color
			"",  // size
			nil, // isActive
			1,   // page
			100, // perPage
		)
		if err == nil && len(skus) > 0 {
			products[i].ProductSKUs = skus
		}
	}
}

// SearchProducts searches products by name
func (s *Service) SearchProducts(ctx context.Context, name string, isActiveStr string, page, perPage int32) (*ListProductsResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	products, totalCount, err := s.repos.Product.SearchByName(ctx, repository.ProductSearchParams{
		Name:     name,
		IsActive: isActiveStr,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to search products", err)
	}

	return &ListProductsResult{
		Products: products,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// UpdateProduct updates a product and its SKUs
func (s *Service) UpdateProduct(ctx context.Context, req UpdateProductRequest) (*gormdb.Product, error) {
	if !isValidUUID(req.ID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID format", fmt.Errorf("id: %s", req.ID))
	}

	// Check if product exists
	product, err := s.repos.Product.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get product", err)
	}

	// Update product fields
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.CategoryID != "" {
		product.CategoryID, _ = uuid.Parse(req.CategoryID)
	}
	if req.CostPrice > 0 {
		product.CostPrice = req.CostPrice
	}
	if req.SellingPrice > 0 {
		product.SellingPrice = req.SellingPrice
	}
	if req.MinimumStock > 0 {
		product.MinimumStock = req.MinimumStock
	}
	product.IsActive = req.IsActive

	if err := s.repos.Product.Update(ctx, product); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update product", err)
	}

	// Delete SKUs
	for _, skuID := range req.SKUsToDelete {
		if isValidUUID(skuID) {
			skuUUID, _ := uuid.Parse(skuID)
			_ = s.repos.ProductSKU.Delete(ctx, skuUUID)
		}
	}

	// Create or update SKUs
	for _, skuItem := range req.SKUUpdates {
		if skuItem.ID != "" && isValidUUID(skuItem.ID) {
			// Update existing SKU
			skuUUID, _ := uuid.Parse(skuItem.ID)
			existingSKU, err := s.repos.ProductSKU.GetByID(ctx, skuUUID)
			if err != nil {
				continue // skip if not found
			}

			if skuItem.SKU != "" {
				existingSKU.SKU = skuItem.SKU
			}
			if skuItem.Barcode != "" {
				existingSKU.Barcode = skuItem.Barcode
			}
			if skuItem.Variant != "" {
				existingSKU.Variant = skuItem.Variant
			}
			if skuItem.Color != "" {
				existingSKU.Color = skuItem.Color
			}
			if skuItem.Size != "" {
				existingSKU.Size = skuItem.Size
			}
			if skuItem.Unit != "" {
				existingSKU.Unit = skuItem.Unit
			}
			if skuItem.CostPrice > 0 {
				existingSKU.CostPrice = skuItem.CostPrice
			}
			if skuItem.SellingPrice > 0 {
				existingSKU.SellingPrice = skuItem.SellingPrice
			}
			if skuItem.Stock > 0 {
				existingSKU.Stock = skuItem.Stock
			}
			if skuItem.MinimumStock > 0 {
				existingSKU.MinimumStock = skuItem.MinimumStock
			}
			if skuItem.Weight > 0 {
				existingSKU.Weight = skuItem.Weight
			}
			if skuItem.Volume > 0 {
				existingSKU.Volume = skuItem.Volume
			}
			if skuItem.BatchNumber != "" {
				existingSKU.BatchNumber = skuItem.BatchNumber
			}
			if skuItem.Location != "" {
				existingSKU.Location = skuItem.Location
			}
			if skuItem.ImageURL != "" {
				existingSKU.ImageURL = skuItem.ImageURL
			}
			existingSKU.IsActive = skuItem.IsActive

			if skuItem.Metadata != nil {
				existingSKU.Metadata = make(map[string]any)
				for k, v := range skuItem.Metadata {
					existingSKU.Metadata[k] = v
				}
			}

			if skuItem.ExpiryDate != "" {
				if t, err := timeutil.ParseFlexible(skuItem.ExpiryDate); err == nil {
					existingSKU.ExpiryDate = &t
				}
			}

			if err := s.repos.ProductSKU.Update(ctx, existingSKU); err != nil {
				continue
			}
		} else {
			// Create new SKU
			unit := skuItem.Unit
			if unit == "" {
				unit = "pcs"
			}

			newSKU := &gormdb.ProductSKU{
				ProductID:    product.ID,
				SKU:          skuItem.SKU,
				Barcode:      skuItem.Barcode,
				Variant:      skuItem.Variant,
				Color:        skuItem.Color,
				Size:         skuItem.Size,
				Unit:         unit,
				CostPrice:    skuItem.CostPrice,
				SellingPrice: skuItem.SellingPrice,
				Stock:        skuItem.Stock,
				MinimumStock: skuItem.MinimumStock,
				Weight:       skuItem.Weight,
				Volume:       skuItem.Volume,
				BatchNumber:  skuItem.BatchNumber,
				Location:     skuItem.Location,
				ImageURL:     skuItem.ImageURL,
				IsActive:     true,
			}

			if skuItem.Metadata != nil {
				newSKU.Metadata = make(map[string]any)
				for k, v := range skuItem.Metadata {
					newSKU.Metadata[k] = v
				}
			}

			if skuItem.ExpiryDate != "" {
				if t, err := timeutil.ParseFlexible(skuItem.ExpiryDate); err == nil {
					newSKU.ExpiryDate = &t
				}
			}

			if err := s.repos.ProductSKU.Create(ctx, newSKU); err != nil {
				continue
			}
		}
	}

	// Keep the aggregate product stock in sync with its SKUs after any
	// create/update/delete above.
	if db := s.repos.DB(); db != nil {
		var aggStock int64
		if err := db.WithContext(ctx).
			Model(&gormdb.ProductSKU{}).
			Where("product_id = ?", product.ID).
			Select("COALESCE(SUM(stock), 0)").
			Scan(&aggStock).Error; err == nil {
			_ = db.WithContext(ctx).
				Model(&gormdb.Product{}).
				Where("id = ?", product.ID).
				Update("stock", aggStock)
		}
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventProductUpdated,
		Resource:   "products",
		ResourceID: product.ID.String(),
	})

	return product, nil
}

// DeleteProduct deletes a product
func (s *Service) DeleteProduct(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if product exists
	_, err := s.repos.Product.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrProductNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get product", err)
	}

	if err := s.repos.Product.Delete(ctx, idStr); err != nil {
		return err
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventProductDeleted,
		Resource:   "products",
		ResourceID: idStr,
	})
	return nil
}

// ActivateProduct activates a product
func (s *Service) ActivateProduct(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if product exists
	_, err := s.repos.Product.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrProductNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get product", err)
	}

	if err := s.repos.Product.Activate(ctx, idStr); err != nil {
		return err
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventProductUpdated,
		Resource:   "products",
		ResourceID: idStr,
	})
	return nil
}

// DeactivateProduct deactivates a product
func (s *Service) DeactivateProduct(ctx context.Context, idStr string) error {
	if !isValidUUID(idStr) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID format", fmt.Errorf("id: %s", idStr))
	}

	// Check if product exists
	_, err := s.repos.Product.GetByID(ctx, idStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrProductNotFound
		}
		return apperror.Wrap(apperror.ErrInternal, "Failed to get product", err)
	}

	if err := s.repos.Product.Deactivate(ctx, idStr); err != nil {
		return err
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventProductUpdated,
		Resource:   "products",
		ResourceID: idStr,
	})
	return nil
}

// GetLowStockProducts lists products with low stock
func (s *Service) GetLowStockProducts(ctx context.Context, page, perPage int32) (*ListProductsResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	products, totalCount, err := s.repos.Product.GetLowStock(ctx, repository.PaginationParams{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get low stock products", err)
	}

	return &ListProductsResult{
		Products: products,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	Name         string
	Description  string
	CategoryID   string
	CostPrice    float64
	SellingPrice float64
	Stock        int32
	MinimumStock int32

	// Default SKU fields
	SKU         string
	Barcode     string
	Variant     string
	Color       string
	Size        string
	Unit        string
	Weight      float64
	Volume      float64
	BatchNumber string
	Location    string
}

// UpdateProductRequest represents the request to update a product
type UpdateProductRequest struct {
	ID           string
	Name         string
	Description  string
	CategoryID   string
	CostPrice    float64
	SellingPrice float64
	MinimumStock int32
	IsActive     bool

	// SKU updates
	SKUUpdates   []SKUUpdateItem
	SKUsToDelete []string
}

// SKUUpdateItem represents an SKU to create or update
type SKUUpdateItem struct {
	ID           string
	SKU          string
	Barcode      string
	Variant      string
	Color        string
	Size         string
	Unit         string
	CostPrice    float64
	SellingPrice float64
	Stock        int32
	MinimumStock int32
	Weight       float64
	Volume       float64
	ExpiryDate   string
	BatchNumber  string
	Location     string
	ImageURL     string
	IsActive     bool
	Metadata     map[string]string
}

// CreateProductFromImportParams carries fields for a simplified CSV/Excel import.
type CreateProductFromImportParams struct {
	StoreID      uuid.UUID
	UserID       uuid.UUID
	Name         string
	CategoryID   string
	CostPrice    float64
	SellingPrice float64
	Stock        int
	MinimumStock int
	SKU          string
	Barcode      string
	Unit         string
	Variant      string
}

// CreateProductFromImport creates a product + default SKU from an import row.
func (s *Service) CreateProductSKUForImport(ctx context.Context, productID uuid.UUID, p CreateProductFromImportParams) (*gormdb.ProductSKU, error) {
	unit := p.Unit
	if unit == "" {
		unit = "pcs"
	}
	sku := &gormdb.ProductSKU{
		ProductID:    productID,
		StoreID:      p.StoreID,
		SKU:          p.SKU,
		Barcode:      p.Barcode,
		Variant:      p.Variant,
		Unit:         unit,
		CostPrice:    p.CostPrice,
		SellingPrice: p.SellingPrice,
		Stock:        int32(p.Stock),
		MinimumStock: int32(p.MinimumStock),
		IsActive:     true,
	}
	if err := s.repos.ProductSKU.Create(ctx, sku); err != nil {
		return nil, err
	}
	return sku, nil
}

// CreateProductFromImport creates a product + SKU from an import row.
// If a product with the same name (case-insensitive) already exists in the
// store, the SKU is added to that existing product instead of creating a
// duplicate product.
func (s *Service) CreateProductFromImport(ctx context.Context, p CreateProductFromImportParams) (*gormdb.Product, error) {
	var catID uuid.UUID
	if p.CategoryID != "" {
		catID, _ = uuid.Parse(p.CategoryID)
	}

	// Find existing product by name in this store (case-insensitive).
	var existing *gormdb.Product
	if p.Name != "" {
		products, _, err := s.repos.Product.SearchByName(ctx, repository.ProductSearchParams{
			Name:     p.Name,
			IsActive: "",
			Page:     1,
			PerPage: 1,
		})
		if err == nil && len(products) > 0 && products[0].Name != "" {
			// Simple case-insensitive match on the first result.
			if strings.EqualFold(products[0].Name, p.Name) && products[0].StoreID == p.StoreID {
				existing = &products[0]
			}
		}
	}

	if existing != nil {
		// Reuse existing product, just add the SKU.
		if _, err := s.CreateProductSKUForImport(ctx, existing.ID, p); err != nil {
			return existing, err
		}
		return existing, nil
	}

	prod := &gormdb.Product{
		StoreID:      p.StoreID,
		Name:         p.Name,
		CategoryID:   catID,
		CostPrice:    p.CostPrice,
		SellingPrice: p.SellingPrice,
		Stock:        int32(p.Stock),
		MinimumStock: int32(p.MinimumStock),
		IsActive:     true,
	}
	if err := s.repos.Product.Create(ctx, prod); err != nil {
		return nil, err
	}

	if _, err := s.CreateProductSKUForImport(ctx, prod.ID, p); err != nil {
		return prod, err
	}
	return prod, nil
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
