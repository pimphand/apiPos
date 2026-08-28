package sale

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Sale status constants
const (
	StatusPending   = "PENDING"
	StatusPaid      = "PAID"
	StatusCancelled = "CANCELLED"
	StatusRefunded  = "REFUNDED"
)

// Payment method constants
const (
	PaymentMethodCash         = "CASH"
	PaymentMethodCard         = "CARD"
	PaymentMethodBankTransfer = "BANK_TRANSFER"
	PaymentMethodEWallet      = "EWALLET"
	PaymentMethodOther        = "OTHER"
)

// Service handles sale business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new sale Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreateSaleRequest represents the request to create a sale
type CreateSaleRequest struct {
	StoreID          string
	CustomerID       string
	Items            []SaleItemInput
	Discount         float64
	TaxRate          float64
	PaymentMethod    string
	AmountPaid       float64
	PaymentReference string
	CreatedBy        string
}

// SaleItemInput represents an item in the sale
type SaleItemInput struct {
	ProductID string
	SKUID     string
	Quantity  int32
	UnitPrice float64
	Discount  float64
}

// SaleDetail holds sale with items and payments
type SaleDetail struct {
	Sale       gormdb.Sale
	Items      []gormdb.SaleItem
	Payments   []gormdb.Payment
	AmountPaid float64
	Change     float64
}

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListResult holds the result of listing sales
type ListResult struct {
	Sales      []gormdb.Sale
	Pagination PaginationResult
}

// CreateSale creates a new sale with atomic checkout
func (s *Service) CreateSale(ctx context.Context, req CreateSaleRequest) (*SaleDetail, error) {
	// Validate inputs
	if !isValidUUID(req.StoreID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store ID", nil)
	}
	if !isValidUUID(req.CreatedBy) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID", nil)
	}
	if len(req.Items) == 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Sale must have at least one item", nil)
	}

	storeID, _ := uuid.Parse(req.StoreID)
	cashierID, _ := uuid.Parse(req.CreatedBy)
	customerID := uuid.Nil
	if req.CustomerID != "" && isValidUUID(req.CustomerID) {
		customerID, _ = uuid.Parse(req.CustomerID)
	}

	// Calculate totals
	var subtotal float64
	for _, item := range req.Items {
		itemSubtotal := (item.UnitPrice * float64(item.Quantity)) - item.Discount
		if itemSubtotal < 0 {
			itemSubtotal = 0
		}
		subtotal += itemSubtotal
	}

	discount := req.Discount
	taxAmount := (subtotal - discount) * req.TaxRate / 100
	grandTotal := subtotal - discount + taxAmount

	// Validate payment
	if req.PaymentMethod == PaymentMethodCash {
		if req.AmountPaid < grandTotal {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest,
				fmt.Sprintf("Insufficient payment. Grand total: %.2f, Amount paid: %.2f", grandTotal, req.AmountPaid), nil)
		}
	}

	// Generate invoice number
	invoiceNumber := generateInvoiceNumber()

	// Execute checkout in a database transaction
	var saleDetail SaleDetail

	err := s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create sale
		sale := &gormdb.Sale{
			InvoiceNumber: invoiceNumber,
			StoreID:       storeID,
			CashierID:     cashierID,
			CustomerID:    customerID,
			Subtotal:      subtotal,
			Discount:      discount,
			Tax:           taxAmount,
			GrandTotal:    grandTotal,
			Status:        StatusPending,
		}

		if err := tx.Create(sale).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to create sale", err)
		}

		// 2. Create sale items and decrease stock
		var saleItems []gormdb.SaleItem
		for _, item := range req.Items {
			productID, _ := uuid.Parse(item.ProductID)

			// Get product with row locking
			var product gormdb.Product
			if err := tx.Raw("SELECT * FROM products WHERE id = ? FOR UPDATE", productID).Scan(&product).Error; err != nil {
				return apperror.Wrap(apperror.ErrNotFound, fmt.Sprintf("Product not found: %s", item.ProductID), err)
			}

			// Validate stock
			if product.Stock < item.Quantity {
				return apperror.Wrap(apperror.ErrInvalidRequest,
					fmt.Sprintf("Insufficient stock for %s. Available: %d, Requested: %d", product.Name, product.Stock, item.Quantity), nil)
			}

			// Decrease stock
			if err := tx.Model(&gormdb.Product{}).
				Where("id = ?", productID).
				Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to update stock", err)
			}

			// Create sale item
			itemSubtotal := (item.UnitPrice * float64(item.Quantity)) - item.Discount
			if itemSubtotal < 0 {
				itemSubtotal = 0
			}

			saleItem := &gormdb.SaleItem{
				SaleID:    sale.ID,
				ProductID: productID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Discount:  item.Discount,
				Subtotal:  itemSubtotal,
			}

			// Set SKU ID if provided
			if item.SKUID != "" && isValidUUID(item.SKUID) {
				skuID, _ := uuid.Parse(item.SKUID)
				// Update SKU stock as well
				if err := tx.Model(&gormdb.ProductSKU{}).
					Where("id = ?", skuID).
					Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
					// Non-critical, continue
				}
				_ = skuID
			}

			if err := tx.Create(saleItem).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to create sale item", err)
			}

			saleItems = append(saleItems, *saleItem)

			// Create inventory transaction
			beforeStock := product.Stock
			afterStock := product.Stock - item.Quantity
			inventoryTxn := &gormdb.InventoryTransaction{
				ProductID:     productID,
				StoreID:       storeID,
				Type:          "SALE",
				Quantity:      item.Quantity,
				ReferenceType: "SALE",
				ReferenceID:   sale.ID,
				BeforeStock:   beforeStock,
				AfterStock:    afterStock,
				CreatedBy:     cashierID,
			}
			if err := tx.Create(inventoryTxn).Error; err != nil {
				// Non-critical, continue
			}
		}

		// 3. Create payment
		var payment gormdb.Payment
		if req.PaymentMethod != "" {
			payment = gormdb.Payment{
				SaleID:        sale.ID,
				PaymentMethod: req.PaymentMethod,
				Amount:        req.AmountPaid,
				Reference:     req.PaymentReference,
				PaidAt:        time.Now(),
			}
			if err := tx.Create(&payment).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to create payment", err)
			}

			// Update sale status to PAID
			sale.Status = StatusPaid
			if err := tx.Save(sale).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to update sale status", err)
			}
		}

		// Calculate change
		change := req.AmountPaid - grandTotal
		if change < 0 {
			change = 0
		}

		saleDetail = SaleDetail{
			Sale:       *sale,
			Items:      saleItems,
			Payments:   []gormdb.Payment{payment},
			AmountPaid: req.AmountPaid,
			Change:     change,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &saleDetail, nil
}

// GetSale gets a sale by ID with items and payments
func (s *Service) GetSale(ctx context.Context, id string) (*SaleDetail, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid sale ID", nil)
	}

	sale, err := s.repos.Sale.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Sale not found", err)
	}

	return s.getSaleDetail(ctx, sale)
}

// GetSaleByInvoice gets a sale by invoice number
func (s *Service) GetSaleByInvoice(ctx context.Context, invoiceNumber string) (*SaleDetail, error) {
	if invoiceNumber == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invoice number is required", nil)
	}

	sale, err := s.repos.Sale.GetByInvoiceNumber(ctx, invoiceNumber)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Sale not found", err)
	}

	return s.getSaleDetail(ctx, sale)
}

// getSaleDetail gets sale detail with items and payments
func (s *Service) getSaleDetail(ctx context.Context, sale *gormdb.Sale) (*SaleDetail, error) {
	items, err := s.repos.Sale.GetItemsBySaleID(ctx, sale.ID.String())
	if err != nil {
		items = []gormdb.SaleItem{}
	}

	payments, err := s.repos.Payment.GetBySaleID(ctx, sale.ID.String())
	if err != nil {
		payments = []gormdb.Payment{}
	}

	totalPaid, _ := s.repos.Payment.GetTotalBySaleID(ctx, sale.ID.String())
	change := totalPaid - sale.GrandTotal
	if change < 0 {
		change = 0
	}

	return &SaleDetail{
		Sale:       *sale,
		Items:      items,
		Payments:   payments,
		AmountPaid: totalPaid,
		Change:     change,
	}, nil
}

// ListSales lists sales with filters
func (s *Service) ListSales(ctx context.Context, params repository.SaleListParams) (*ListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 20
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}

	sales, totalCount, err := s.repos.Sale.List(ctx, params)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list sales", err)
	}

	return &ListResult{
		Sales: sales,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       params.Page,
			PerPage:    params.PerPage,
		},
	}, nil
}

// CancelSale cancels a sale and restores stock
func (s *Service) CancelSale(ctx context.Context, id string) error {
	if !isValidUUID(id) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid sale ID", nil)
	}

	sale, err := s.repos.Sale.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(apperror.ErrNotFound, "Sale not found", err)
	}

	if sale.Status == StatusCancelled {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Sale is already cancelled", nil)
	}
	if sale.Status == StatusRefunded {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Cannot cancel a refunded sale", nil)
	}

	// Restore stock in a transaction
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get sale items
		var items []gormdb.SaleItem
		if err := tx.Where("sale_id = ?", sale.ID).Find(&items).Error; err != nil {
			return err
		}

		// Restore stock for each item
		for _, item := range items {
			if err := tx.Model(&gormdb.Product{}).
				Where("id = ?", item.ProductID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return err
			}

			// Create inventory transaction
			inventoryTxn := &gormdb.InventoryTransaction{
				ProductID:     item.ProductID,
				StoreID:       sale.StoreID,
				Type:          "RETURN",
				Quantity:      item.Quantity,
				ReferenceType: "SALE_CANCEL",
				ReferenceID:   sale.ID,
				BeforeStock:   0, // Will be calculated
				AfterStock:    0, // Will be calculated
				CreatedBy:     sale.CashierID,
			}
			tx.Create(inventoryTxn)
		}

		// Update sale status
		sale.Status = StatusCancelled
		return tx.Save(sale).Error
	})

	return err
}

// RefundSale refunds a sale and restores stock
func (s *Service) RefundSale(ctx context.Context, id string) error {
	if !isValidUUID(id) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid sale ID", nil)
	}

	sale, err := s.repos.Sale.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(apperror.ErrNotFound, "Sale not found", err)
	}

	if sale.Status == StatusRefunded {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Sale is already refunded", nil)
	}
	if sale.Status == StatusCancelled {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Cannot refund a cancelled sale", nil)
	}

	// Restore stock in a transaction
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get sale items
		var items []gormdb.SaleItem
		if err := tx.Where("sale_id = ?", sale.ID).Find(&items).Error; err != nil {
			return err
		}

		// Restore stock for each item
		for _, item := range items {
			if err := tx.Model(&gormdb.Product{}).
				Where("id = ?", item.ProductID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return err
			}

			// Create inventory transaction
			inventoryTxn := &gormdb.InventoryTransaction{
				ProductID:     item.ProductID,
				StoreID:       sale.StoreID,
				Type:          "RETURN",
				Quantity:      item.Quantity,
				ReferenceType: "SALE_REFUND",
				ReferenceID:   sale.ID,
				BeforeStock:   0,
				AfterStock:    0,
				CreatedBy:     sale.CashierID,
			}
			tx.Create(inventoryTxn)
		}

		// Update sale status
		sale.Status = StatusRefunded
		return tx.Save(sale).Error
	})

	return err
}

// generateInvoiceNumber generates a unique invoice number
func generateInvoiceNumber() string {
	now := time.Now()
	return fmt.Sprintf("INV-%s-%s", now.Format("20060102"), uuid.New().String()[:8])
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// Ensure fmt is used
var _ = fmt.Sprintf
