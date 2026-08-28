package productsku

import (
	"time"

	"github.com/google/uuid"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/apperror"
	"apiPos/internal/repository"
)

// Service handles product SKU business logic
type Service struct {
	productSKURepo repository.ProductSKURepository
}

// NewService creates a new product SKU service
func NewService(productSKURepo repository.ProductSKURepository) *Service {
	return &Service{
		productSKURepo: productSKURepo,
	}
}

// CreateProductSKU creates a new product SKU
func (s *Service) CreateProductSKU(productID, sku, barcode, variant, color, size, unit string, costPrice, sellingPrice float64, stock, minimumStock int32, weight, volume float64, expiryDate *time.Time, batchNumber, location, imageURL string, isActive bool, metadata map[string]string) (*gormdb.ProductSKU, error) {
	productSKU := &gormdb.ProductSKU{
		ProductID:    uuid.MustParse(productID),
		SKU:          sku,
		Barcode:      barcode,
		Variant:      variant,
		Color:        color,
		Size:         size,
		Unit:         unit,
		CostPrice:    costPrice,
		SellingPrice: sellingPrice,
		Stock:        stock,
		MinimumStock: minimumStock,
		Weight:       weight,
		Volume:       volume,
		ExpiryDate:   expiryDate,
		BatchNumber:  batchNumber,
		Location:     location,
		ImageURL:     imageURL,
		IsActive:     isActive,
	}

	if metadata != nil {
		productSKU.Metadata = make(map[string]any)
		for k, v := range metadata {
			productSKU.Metadata[k] = v
		}
	}

	if err := s.productSKURepo.Create(productSKU); err != nil {
		return nil, err
	}

	return productSKU, nil
}

// GetProductSKU gets a product SKU by ID
func (s *Service) GetProductSKU(id string) (*gormdb.ProductSKU, error) {
	productSKUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return s.productSKURepo.GetByID(productSKUID)
}

// GetProductSKUByBarcode gets a product SKU by barcode
func (s *Service) GetProductSKUByBarcode(barcode string) (*gormdb.ProductSKU, error) {
	if barcode == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Barcode is required", nil)
	}
	return s.productSKURepo.GetByBarcode(barcode)
}

// GetProductSKUBySKUCode gets a product SKU by SKU code
func (s *Service) GetProductSKUBySKUCode(sku string) (*gormdb.ProductSKU, error) {
	if sku == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "SKU is required", nil)
	}
	return s.productSKURepo.GetBySKU(sku)
}

// ListProductSKUs lists product SKUs by product ID
func (s *Service) ListProductSKUs(productID string, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	productUUID, err := uuid.Parse(productID)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.productSKURepo.List(productUUID, search, variant, color, size, isActive, page, perPage)
}

// ListAllProductSKUs lists all product SKUs
func (s *Service) ListAllProductSKUs(search, variant, color, size string, isActive *bool, productID string, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	var productUUID uuid.UUID
	if productID != "" {
		productUUID, _ = uuid.Parse(productID)
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.productSKURepo.ListAll(search, variant, color, size, isActive, productUUID, page, perPage)
}

// UpdateProductSKU updates a product SKU
func (s *Service) UpdateProductSKU(id, sku, barcode, variant, color, size, unit string, costPrice, sellingPrice float64, stock, minimumStock int32, weight, volume float64, expiryDate *time.Time, batchNumber, location, imageURL string, isActive bool, metadata map[string]string) (*gormdb.ProductSKU, error) {
	productSKUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	productSKU, err := s.productSKURepo.GetByID(productSKUID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if sku != "" {
		productSKU.SKU = sku
	}
	if barcode != "" {
		productSKU.Barcode = barcode
	}
	if variant != "" {
		productSKU.Variant = variant
	}
	if color != "" {
		productSKU.Color = color
	}
	if size != "" {
		productSKU.Size = size
	}
	if unit != "" {
		productSKU.Unit = unit
	}
	if costPrice > 0 {
		productSKU.CostPrice = costPrice
	}
	if sellingPrice > 0 {
		productSKU.SellingPrice = sellingPrice
	}
	if stock >= 0 {
		productSKU.Stock = stock
	}
	if minimumStock >= 0 {
		productSKU.MinimumStock = minimumStock
	}
	if weight > 0 {
		productSKU.Weight = weight
	}
	if volume > 0 {
		productSKU.Volume = volume
	}
	if expiryDate != nil {
		productSKU.ExpiryDate = expiryDate
	}
	if batchNumber != "" {
		productSKU.BatchNumber = batchNumber
	}
	if location != "" {
		productSKU.Location = location
	}
	if imageURL != "" {
		productSKU.ImageURL = imageURL
	}
	productSKU.IsActive = isActive
	if metadata != nil {
		productSKU.Metadata = make(map[string]any)
		for k, v := range metadata {
			productSKU.Metadata[k] = v
		}
	}

	if err := s.productSKURepo.Update(productSKU); err != nil {
		return nil, err
	}

	return productSKU, nil
}

// DeleteProductSKU deletes a product SKU
func (s *Service) DeleteProductSKU(id string) error {
	productSKUID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	return s.productSKURepo.Delete(productSKUID)
}
