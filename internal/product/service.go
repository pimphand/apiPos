package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
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
	// Check if SKU already exists
	_, err := s.repos.Product.GetBySKU(ctx, req.SKU)
	if err == nil {
		return nil, apperror.ErrProductAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check SKU", err)
	}

	// Check if barcode already exists
	if req.Barcode != "" {
		_, err := s.repos.Product.GetByBarcode(ctx, req.Barcode)
		if err == nil {
			return nil, apperror.Wrap(apperror.ErrProductAlreadyExists, "Barcode already exists", nil)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check barcode", err)
		}
	}

	// Parse category_id
	var categoryID uuid.UUID
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

	return product, nil
}

// GetProduct gets a product by ID
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

// GetProductByBarcode looks up a product by barcode
func (s *Service) GetProductByBarcode(ctx context.Context, barcode string) (*gormdb.Product, error) {
	if barcode == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Barcode is required", nil)
	}

	product, err := s.repos.Product.GetByBarcode(ctx, barcode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get product by barcode", err)
	}

	return product, nil
}

// GetProductBySKU looks up a product by SKU
func (s *Service) GetProductBySKU(ctx context.Context, sku string) (*gormdb.Product, error) {
	if sku == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "SKU is required", nil)
	}

	product, err := s.repos.Product.GetBySKU(ctx, sku)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get product by SKU", err)
	}

	return product, nil
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

// UpdateProduct updates a product
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

	// Update fields
	if req.SKU != "" {
		product.SKU = req.SKU
	}
	if req.Barcode != "" {
		product.Barcode = req.Barcode
	}
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

	return s.repos.Product.Delete(ctx, idStr)
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

	return s.repos.Product.Activate(ctx, idStr)
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

	return s.repos.Product.Deactivate(ctx, idStr)
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
	SKU          string
	Barcode      string
	Name         string
	Description  string
	CategoryID   string
	CostPrice    float64
	SellingPrice float64
	Stock        int32
	MinimumStock int32
}

// UpdateProductRequest represents the request to update a product
type UpdateProductRequest struct {
	ID           string
	SKU          string
	Barcode      string
	Name         string
	Description  string
	CategoryID   string
	CostPrice    float64
	SellingPrice float64
	MinimumStock int32
	IsActive     bool
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
