package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"apiPos/internal/accounting"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/realtime"
	"apiPos/internal/repository"
)

// TransactionType constants
const (
	TypePurchase   = "PURCHASE"
	TypeSale       = "SALE"
	TypeReturn     = "RETURN"
	TypeAdjustment = "ADJUSTMENT"
	TypeDamage     = "DAMAGE"
	TypeTransfer   = "TRANSFER"
)

// Service handles inventory business logic
type Service struct {
	repos   *repository.Repositories
	posting *accounting.PostingService
}

// NewService creates a new inventory Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos, posting: accounting.NewPostingService(repos.DB())}
}

// CreateTransactionRequest represents the request to create an inventory transaction
type CreateTransactionRequest struct {
	ProductID     string
	StoreID       string
	Type          string
	Quantity      int32
	ReferenceType string
	ReferenceID   string
	Description   string
	CreatedBy     string
}

// AdjustStockRequest represents the request to adjust stock
type AdjustStockRequest struct {
	ProductID   string
	StoreID     string
	Direction   string // INCREASE or DECREASE
	Quantity    int32
	Reason      string
	Description string
	CreatedBy   string
}

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListResult holds the result of listing transactions
type ListResult struct {
	Transactions []gormdb.InventoryTransaction
	Pagination   PaginationResult
}

// SKUStock holds stock info for a single SKU
type SKUStock struct {
	SKUID        uuid.UUID
	SKUCode      string
	Variant      string
	Barcode      string
	Stock        int32
	MinimumStock int32
	IsLowStock   bool
}

// StockSummary holds stock summary for a product
type StockSummary struct {
	ProductID   uuid.UUID
	ProductName string
	TotalStock  int32
	SKUs        []SKUStock
}

// LowStockProduct holds low stock product info
type LowStockProduct struct {
	ProductID    uuid.UUID
	ProductName  string
	CategoryName string
	TotalStock   int32
	MinimumStock int32
	Deficit      int32
}

// LowStockResult holds low stock results
type LowStockResult struct {
	Products   []LowStockProduct
	Pagination PaginationResult
}

// CreateTransaction creates a new inventory transaction and updates stock atomically
func (s *Service) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*gormdb.InventoryTransaction, error) {
	// Validate inputs
	if !isValidUUID(req.ProductID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID", nil)
	}
	if !isValidUUID(req.StoreID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store ID", nil)
	}
	if !isValidUUID(req.CreatedBy) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID", nil)
	}

	// Validate transaction type
	validTypes := map[string]bool{
		TypePurchase:   true,
		TypeSale:       true,
		TypeReturn:     true,
		TypeAdjustment: true,
		TypeDamage:     true,
		TypeTransfer:   true,
	}
	if !validTypes[req.Type] {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, fmt.Sprintf("Invalid transaction type: %s. Valid types: PURCHASE, SALE, RETURN, ADJUSTMENT, DAMAGE, TRANSFER", req.Type), nil)
	}

	if req.Quantity <= 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Quantity must be greater than 0", nil)
	}

	productID, _ := uuid.Parse(req.ProductID)
	storeID, _ := uuid.Parse(req.StoreID)
	createdBy, _ := uuid.Parse(req.CreatedBy)

	// Determine stock delta for the transaction type.
	var delta int32
	switch req.Type {
	case TypePurchase, TypeReturn, TypeAdjustment:
		delta = req.Quantity
	case TypeSale, TypeDamage, TypeTransfer:
		delta = -req.Quantity
	}

	// Atomically lock + adjust stock and capture before/after for the ledger.
	beforeStock, afterStock, err := s.repos.Inventory.AdjustProductStock(ctx, productID, delta)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStockAtRepo) {
			return nil, apperror.Wrap(apperror.ErrInsufficientStock,
				fmt.Sprintf("Insufficient stock. Available: %d, Requested: %d", beforeStock, req.Quantity), nil)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to adjust stock", err)
	}

	// Create inventory transaction
	transaction := &gormdb.InventoryTransaction{
		ProductID:     productID,
		StoreID:       storeID,
		Type:          req.Type,
		Quantity:      req.Quantity,
		ReferenceType: req.ReferenceType,
		BeforeStock:   beforeStock,
		AfterStock:    afterStock,
		CreatedBy:     createdBy,
	}

	// Set ReferenceID if provided
	if req.ReferenceID != "" {
		refID, err := uuid.Parse(req.ReferenceID)
		if err == nil {
			transaction.ReferenceID = refID
		}
	}

	if err := s.repos.Inventory.CreateTransaction(ctx, transaction); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create inventory transaction", err)
	}

	// Accounting: post PURCHASE / ADJUSTMENT / DAMAGE journals.
	// SALE and RETURN are already covered by the SALE/REVERSAL journals.
	if req.Type == TypePurchase {
		if _, err := s.posting.PostPurchase(ctx, s.repos.DB().WithContext(ctx), transaction); err != nil {
			_ = err // best-effort: stock movement already succeeded
		}
	} else if req.Type == TypeAdjustment || req.Type == TypeDamage {
		if _, err := s.posting.PostAdjustment(ctx, s.repos.DB().WithContext(ctx), transaction); err != nil {
			_ = err
		}
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventInventoryAdjusted,
		Resource:   "inventory_transactions",
		ResourceID: transaction.ID.String(),
		Data: map[string]any{
			"product_id":  req.ProductID,
			"type":        req.Type,
			"quantity":    req.Quantity,
			"before_stock": beforeStock,
			"after_stock":  afterStock,
		},
	})

	return transaction, nil
}

// AdjustStock adjusts stock for a product (atomic)
func (s *Service) AdjustStock(ctx context.Context, req AdjustStockRequest) (*gormdb.InventoryTransaction, error) {
	// Validate inputs
	if !isValidUUID(req.ProductID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID", nil)
	}
	if !isValidUUID(req.StoreID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store ID", nil)
	}
	if !isValidUUID(req.CreatedBy) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID", nil)
	}

	if req.Quantity <= 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Quantity must be greater than 0", nil)
	}

	// Validate direction
	if req.Direction != "INCREASE" && req.Direction != "DECREASE" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Direction must be INCREASE or DECREASE", nil)
	}

	productID, _ := uuid.Parse(req.ProductID)
	storeID, _ := uuid.Parse(req.StoreID)
	createdBy, _ := uuid.Parse(req.CreatedBy)

	// Atomically lock + adjust stock and capture before/after for the ledger.
	var delta int32
	if req.Direction == "INCREASE" {
		delta = req.Quantity
	} else {
		delta = -req.Quantity
	}

	beforeStock, afterStock, err := s.repos.Inventory.AdjustProductStock(ctx, productID, delta)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStockAtRepo) {
			return nil, apperror.Wrap(apperror.ErrInsufficientStock,
				fmt.Sprintf("Insufficient stock. Available: %d, Requested: %d", beforeStock, req.Quantity), nil)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to adjust stock", err)
	}

	// Create inventory transaction
	transaction := &gormdb.InventoryTransaction{
		ProductID:     productID,
		StoreID:       storeID,
		Type:          TypeAdjustment,
		Quantity:      req.Quantity,
		ReferenceType: "MANUAL",
		BeforeStock:   beforeStock,
		AfterStock:    afterStock,
		CreatedBy:     createdBy,
	}

	if err := s.repos.Inventory.CreateTransaction(ctx, transaction); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create inventory transaction", err)
	}

	realtime.PublishFromContext(ctx, realtime.Event{
		Event:      realtime.EventInventoryAdjusted,
		Resource:   "inventory_transactions",
		ResourceID: transaction.ID.String(),
		Data: map[string]any{
			"product_id":   req.ProductID,
			"direction":    req.Direction,
			"quantity":     req.Quantity,
			"before_stock": beforeStock,
			"after_stock":  afterStock,
		},
	})

	return transaction, nil
}

// GetTransaction gets an inventory transaction by ID
func (s *Service) GetTransaction(ctx context.Context, id string) (*gormdb.InventoryTransaction, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid transaction ID", nil)
	}

	txn, err := s.repos.Inventory.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Transaction not found", err)
	}

	return txn, nil
}

// ListTransactions lists inventory transactions with filters
func (s *Service) ListTransactions(ctx context.Context, productID, storeID, txnType string, page, perPage int32) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	txns, totalCount, err := s.repos.Inventory.ListTransactions(ctx, repository.InventoryListParams{
		ProductID: productID,
		StoreID:   storeID,
		Type:      txnType,
		Page:      page,
		PerPage:   perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list transactions", err)
	}

	return &ListResult{
		Transactions: txns,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// GetStockSummary gets stock summary for a product across all SKUs
func (s *Service) GetStockSummary(ctx context.Context, productID string) (*StockSummary, error) {
	if !isValidUUID(productID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid product ID", nil)
	}

	// Get product
	var product gormdb.Product
	if err := s.repos.DB().WithContext(ctx).Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Product not found", err)
	}

	// Get SKUs
	var skus []gormdb.ProductSKU
	if err := s.repos.DB().WithContext(ctx).Where("product_id = ?", productID).Find(&skus).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get SKUs", err)
	}

	summary := &StockSummary{
		ProductID:   product.ID,
		ProductName: product.Name,
		TotalStock:  product.Stock,
		SKUs:        make([]SKUStock, 0, len(skus)),
	}

	for _, sku := range skus {
		summary.SKUs = append(summary.SKUs, SKUStock{
			SKUID:        sku.ID,
			SKUCode:      sku.SKU,
			Variant:      sku.Variant,
			Barcode:      sku.Barcode,
			Stock:        sku.Stock,
			MinimumStock: sku.MinimumStock,
			IsLowStock:   sku.Stock <= sku.MinimumStock,
		})
	}

	return summary, nil
}

// GetLowStockAlert gets products with stock below minimum
func (s *Service) GetLowStockAlert(ctx context.Context, storeID string, page, perPage int32) (*LowStockResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	var products []gormdb.Product
	var totalCount int64

	query := s.repos.DB().WithContext(ctx).Model(&gormdb.Product{}).
		Where("stock <= minimum_stock AND is_active = ?", true)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to count low stock products", err)
	}

	offset := (page - 1) * perPage
	if err := query.Offset(int(offset)).Limit(int(perPage)).Order("stock ASC").Find(&products).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get low stock products", err)
	}

	result := &LowStockResult{
		Products: make([]LowStockProduct, 0, len(products)),
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}

	for _, p := range products {
		categoryName := ""
		if p.CategoryID != uuid.Nil {
			var cat gormdb.Category
			if err := s.repos.DB().WithContext(ctx).Where("id = ?", p.CategoryID).First(&cat).Error; err == nil {
				categoryName = cat.Name
			}
		}

		result.Products = append(result.Products, LowStockProduct{
			ProductID:    p.ID,
			ProductName:  p.Name,
			CategoryName: categoryName,
			TotalStock:   p.Stock,
			MinimumStock: p.MinimumStock,
			Deficit:      p.MinimumStock - p.Stock,
		})
	}

	return result, nil
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
