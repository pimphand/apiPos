package sale

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	CustomerName     string
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

	// Calculate totals (subtotal, discount, tax, grand total).
	subtotal, discount, taxAmount, grandTotal := computeSaleTotals(req.Items, req.Discount, req.TaxRate)

	// Validate payment: total paid must be sufficient regardless of method.
	if err := validatePayment(req.PaymentMethod, req.AmountPaid, grandTotal); err != nil {
		return nil, err
	}

	// Generate invoice number
	invoiceNumber := generateInvoiceNumber()

	// Execute checkout in a database transaction
	var saleDetail SaleDetail

	err := s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// B5: a shift-capable cashier (role with cashier.* permissions) can only
		// check out while they have an OPEN shift. The sale is linked to that
		// shift so reconciliation later uses real relations, not a time window.
		shiftID, err := s.openShiftForSale(ctx, tx, cashierID, storeID)
		if err != nil {
			return err
		}

		// Resolve customer: explicit ID, or find-or-create by typed name.
		customerID, useCustomer, err := s.resolveCustomer(ctx, tx, req, storeID)
		if err != nil {
			return err
		}

		// 1. Create sale
		sale := &gormdb.Sale{
			InvoiceNumber: invoiceNumber,
			StoreID:       storeID,
			CashierID:     cashierID,
			Subtotal:      subtotal,
			Discount:      discount,
			Tax:           taxAmount,
			GrandTotal:    grandTotal,
			Status:        StatusPending,
			ShiftID:       shiftID,
		}
		if useCustomer {
			sale.CustomerID = customerID
			if err := tx.Create(sale).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to create sale", err)
			}
		} else {
			if err := tx.Omit("CustomerID").Create(sale).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to create sale", err)
			}
		}

		// 2. Create sale items and decrease stock
		var saleItems []gormdb.SaleItem
		var shiftItems []gormdb.ShiftItem // per-item OUT ledger rows (B9)
		for _, item := range req.Items {
			productID, _ := uuid.Parse(item.ProductID)

			// Get product with row locking (scoped to the current store)
			var product gormdb.Product
			if err := tx.Raw("SELECT * FROM products WHERE id = ? AND (store_id = ? OR store_id IS NULL) FOR UPDATE", productID, storeID).Scan(&product).Error; err != nil {
				return apperror.Wrap(apperror.ErrNotFound, fmt.Sprintf("Product not found: %s", item.ProductID), err)
			}
			// Scan() does not error on zero rows; detect a missing product.
			if product.ID == uuid.Nil {
				return apperror.Wrap(apperror.ErrNotFound, fmt.Sprintf("Product not found: %s", item.ProductID), nil)
			}

			beforeStock := product.Stock
			afterStock := product.Stock

			// Ledger snapshot fields for this line (B9).
			lineType := "PRODUCT"
			var lineSkuID *uuid.UUID
			lineSkuCode := ""
			lineVariant := ""

			if item.SKUID != "" && isValidUUID(item.SKUID) {
				// A specific variant was selected: the SKU stock is the source
				// of truth (the POS shows per-SKU stock). products.stock is the
				// aggregate of all SKU stocks and is kept in sync afterwards.
				skuID, _ := uuid.Parse(item.SKUID)

				// Get SKU with row locking (scoped to the current store)
				var sku gormdb.ProductSKU
				if err := tx.Raw("SELECT * FROM product_skus WHERE id = ? AND (store_id = ? OR store_id IS NULL) FOR UPDATE", skuID, storeID).Scan(&sku).Error; err != nil {
					return apperror.Wrap(apperror.ErrNotFound, fmt.Sprintf("Product SKU not found: %s", item.SKUID), err)
				}
				// Scan() does not error on zero rows; a missing SKU leaves the
				// struct zero-valued, so detect it explicitly to return a proper
				// NOT_FOUND instead of a misleading "does not belong" error.
				if sku.ID == uuid.Nil {
					return apperror.Wrap(apperror.ErrNotFound, fmt.Sprintf("Product SKU not found: %s", item.SKUID), nil)
				}

				// SKU must belong to the selected product
				if sku.ProductID != productID {
					return apperror.Wrap(apperror.ErrInvalidRequest,
						fmt.Sprintf("Product SKU %s does not belong to product %s", item.SKUID, item.ProductID), nil)
				}

				// Validate against the SKU stock
				if sku.Stock < item.Quantity {
					return apperror.Wrap(apperror.ErrInsufficientStock,
						fmt.Sprintf("Insufficient stock for SKU %s. Available: %d, Requested: %d", sku.SKU, sku.Stock, item.Quantity), nil)
				}

				// Decrease SKU stock atomically
				if err := tx.Model(&gormdb.ProductSKU{}).
					Where("id = ?", skuID).
					Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to update SKU stock", err)
				}

				// Recompute product stock as the sum of its SKU stocks so the
				// aggregate never drifts from the variants.
				var aggStock int64
				if err := tx.Model(&gormdb.ProductSKU{}).
					Where("product_id = ?", productID).
					Select("COALESCE(SUM(stock), 0)").
					Scan(&aggStock).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to recompute product stock", err)
				}
				if err := tx.Model(&gormdb.Product{}).
					Where("id = ?", productID).
					Update("stock", aggStock).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to update product stock", err)
				}
				lineType = "SKU"
				skuIDCopy := skuID
				lineSkuID = &skuIDCopy
				lineSkuCode = sku.SKU
				lineVariant = skuVariantLabel(sku)
				afterStock = int32(aggStock)
			} else {
				// No variant selected: validate against the product stock.
				if product.Stock < item.Quantity {
					return apperror.Wrap(apperror.ErrInsufficientStock,
						fmt.Sprintf("Insufficient stock for %s. Available: %d, Requested: %d", product.Name, product.Stock, item.Quantity), nil)
				}

				// Decrease stock
				if err := tx.Model(&gormdb.Product{}).
					Where("id = ?", productID).
					Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to update stock", err)
				}
				afterStock = product.Stock - item.Quantity
			}

			// Create sale item
			itemSubtotal := (item.UnitPrice * float64(item.Quantity)) - item.Discount
			if itemSubtotal < 0 {
				itemSubtotal = 0
			}

			saleItem := &gormdb.SaleItem{
				SaleID:       sale.ID,
				ProductID:    productID,
				ProductSKUID: lineSkuID, // exact SKU sold -> exact restore on refund
				Quantity:     item.Quantity,
				UnitPrice:    item.UnitPrice,
				Discount:     item.Discount,
				Subtotal:     itemSubtotal,
			}

			if err := tx.Create(saleItem).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to create sale item", err)
			}

			saleItems = append(saleItems, *saleItem)

			// Create inventory transaction
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
				return apperror.Wrap(apperror.ErrInternal, "Failed to create inventory transaction", err)
			}

			// B9: per-item OUT ledger row, written in this same transaction so
			// the shift ledger can never drift from stock movements.
			if shiftID != nil {
				shiftItems = append(shiftItems, gormdb.ShiftItem{
					ShiftID:        *shiftID,
					StoreID:        storeID,
					SaleID:         &sale.ID,
					Direction:      "OUT",
					ItemType:       lineType,
					ProductID:      productID,
					ProductSKUID:   lineSkuID,
					ProductName:    product.Name,
					SKUCode:        lineSkuCode,
					VariantLabel:   lineVariant,
					Quantity:       float64(item.Quantity),
					UnitPrice:      item.UnitPrice,
					DiscountAmount: item.Discount,
					LineAmount:     itemSubtotal,
					PaymentMethod:  req.PaymentMethod,
					ReferenceNo:    invoiceNumber,
					CreatedBy:      cashierID,
				})
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
			if err := tx.Model(&gormdb.Sale{}).
				Where("id = ?", sale.ID).
				Update("status", sale.Status).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to update sale status", err)
			}
		}

		// Persist the shift OUT ledger rows (insert-only) in the same transaction.
		if len(shiftItems) > 0 {
			if err := tx.Create(&shiftItems).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to record shift items", err)
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

// CancelSale cancels a sale and restores stock. actorID is the acting user
// (from the JWT); when they have an OPEN shift, the restored goods are
// recorded as IN ledger rows on that shift.
func (s *Service) CancelSale(ctx context.Context, id, actorID string) error {
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

	actorUUID, _ := uuid.Parse(actorID)

	// Restore stock in a transaction
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get sale items
		var items []gormdb.SaleItem
		if err := tx.Where("sale_id = ?", sale.ID).Find(&items).Error; err != nil {
			return err
		}

		if _, err := s.restoreSaleIntoShift(ctx, tx, sale, items, actorUUID, "SALE_CANCEL"); err != nil {
			return err
		}

		// Update sale status (column-only: Save() would re-write customer_id,
		// which is the zero UUID for walk-in sales and would violate the FK).
		sale.Status = StatusCancelled
		return tx.Model(&gormdb.Sale{}).Where("id = ?", sale.ID).Update("status", sale.Status).Error
	})

	return err
}

// RefundSale refunds a sale and restores stock. actorID is the acting user;
// when they have an OPEN shift the returned goods are recorded as IN ledger
// rows and (for sales paid in CASH) the shift's cash_refund is increased so
// drawer reconciliation stays correct.
func (s *Service) RefundSale(ctx context.Context, id, actorID string) error {
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

	actorUUID, _ := uuid.Parse(actorID)

	// Restore stock in a transaction
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get sale items
		var items []gormdb.SaleItem
		if err := tx.Where("sale_id = ?", sale.ID).Find(&items).Error; err != nil {
			return err
		}

		shift, err := s.restoreSaleIntoShift(ctx, tx, sale, items, actorUUID, "SALE_REFUND")
		if err != nil {
			return err
		}

		// Cash refund: money leaves the actor's drawer, so their open shift's
		// cash_refund goes up (expected cash = opening + cash sales - refunds).
		if shift != nil && s.saleWasPaidInCash(ctx, tx, sale.ID) {
			if err := tx.Model(&gormdb.CashierShift{}).
				Where("id = ? AND status = ?", shift.ID, "OPEN").
				Update("cash_refund", gorm.Expr("cash_refund + ?", sale.GrandTotal)).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to record cash refund", err)
			}
		}

		// Update sale status (column-only: Save() would re-write customer_id,
		// which is the zero UUID for walk-in sales and would violate the FK).
		sale.Status = StatusRefunded
		return tx.Model(&gormdb.Sale{}).Where("id = ?", sale.ID).Update("status", sale.Status).Error
	})

	return err
}

// saleWasPaidInCash reports whether the primary payment of a sale is CASH.
func (s *Service) saleWasPaidInCash(ctx context.Context, tx *gorm.DB, saleID uuid.UUID) bool {
	var pays []gormdb.Payment
	if err := tx.Where("sale_id = ?", saleID).Find(&pays).Error; err != nil {
		return false
	}
	for _, p := range pays {
		if p.PaymentMethod == PaymentMethodCash {
			return true
		}
	}
	return false
}

// restoreSaleIntoShift restores product stock, records RETURN inventory
// transactions, and when the acting user has an OPEN shift appends IN ledger
// rows (insert-only) for that shift inside the given transaction. It returns
// the actor's open shift (nil when none).
func (s *Service) restoreSaleIntoShift(ctx context.Context, tx *gorm.DB, sale *gormdb.Sale, items []gormdb.SaleItem, actorID uuid.UUID, referenceType string) (*gormdb.CashierShift, error) {
	actorID = effectiveActor(actorID, sale.CashierID)

	shift, err := s.openShiftOfActor(ctx, tx, actorID)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		// Load the product row (also supplies the snapshot name below).
		var product gormdb.Product
		if err := tx.Where("id = ?", item.ProductID).First(&product).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Product not found", err)
		}

		// Restore stock at the exact level it was taken from: SKU sales
		// decremented product_skus.stock (products.stock is their sum), while
		// product-level sales decremented products.stock directly.
		if item.ProductSKUID != nil {
			if err := tx.Model(&gormdb.ProductSKU{}).
				Where("id = ?", *item.ProductSKUID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return nil, apperror.Wrap(apperror.ErrInternal, "Failed to restore SKU stock", err)
			}

			// Recompute the aggregate so products.stock never drifts.
			var aggStock int64
			if err := tx.Model(&gormdb.ProductSKU{}).
				Where("product_id = ?", product.ID).
				Select("COALESCE(SUM(stock), 0)").
				Scan(&aggStock).Error; err != nil {
				return nil, apperror.Wrap(apperror.ErrInternal, "Failed to recompute product stock", err)
			}
			if err := tx.Model(&gormdb.Product{}).
				Where("id = ?", product.ID).
				Update("stock", aggStock).Error; err != nil {
				return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update product stock", err)
			}
		} else {
			if err := tx.Model(&gormdb.Product{}).
				Where("id = ?", item.ProductID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return nil, apperror.Wrap(apperror.ErrInternal, "Failed to restore stock", err)
			}
		}

		// Create inventory transaction
		inventoryTxn := &gormdb.InventoryTransaction{
			ProductID:     item.ProductID,
			StoreID:       sale.StoreID,
			Type:          "RETURN",
			Quantity:      item.Quantity,
			ReferenceType: referenceType,
			ReferenceID:   sale.ID,
			BeforeStock:   0,
			AfterStock:    0,
			CreatedBy:     actorID,
		}
		if err := tx.Create(inventoryTxn).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create inventory transaction", err)
		}

		// IN ledger row on the actor's open shift (SKU-level when known).
		if shift != nil {
			lineAmount := (item.UnitPrice * float64(item.Quantity)) - item.Discount
			if lineAmount < 0 {
				lineAmount = 0
			}
			in := &gormdb.ShiftItem{
				ShiftID:        shift.ID,
				StoreID:        sale.StoreID,
				SaleID:         &sale.ID,
				Direction:      "IN",
				ItemType:       "PRODUCT",
				ProductID:      item.ProductID,
				ProductName:    product.Name,
				Quantity:       float64(item.Quantity),
				UnitPrice:      item.UnitPrice,
				DiscountAmount: item.Discount,
				LineAmount:     lineAmount,
				ReferenceNo:    sale.InvoiceNumber,
				CreatedBy:      actorID,
			}

			if item.ProductSKUID != nil {
				// Enrich with the exact SKU (code + variant) for the ledger.
				var sku gormdb.ProductSKU
				if err := tx.Where("id = ?", *item.ProductSKUID).First(&sku).Error; err == nil {
					in.ItemType = "SKU"
					id := sku.ID
					in.ProductSKUID = &id
					in.SKUCode = sku.SKU
					in.VariantLabel = skuVariantLabel(sku)
				}
			}

			if err := tx.Create(in).Error; err != nil {
				return nil, apperror.Wrap(apperror.ErrInternal, "Failed to record shift item", err)
			}
		}
	}

	return shift, nil
}

// effectiveActor falls back to the sale's cashier when no actor is supplied.
func effectiveActor(actorID, fallback uuid.UUID) uuid.UUID {
	if actorID == uuid.Nil {
		return fallback
	}
	return actorID
}

// generateInvoiceNumber generates a unique invoice number
func generateInvoiceNumber() string {
	now := time.Now()
	return fmt.Sprintf("INV-%s-%s", now.Format("20060102"), uuid.New().String()[:8])
}

// resolveCustomer determines the customer for a sale. It returns the customer
// ID and whether it should be set on the sale. Precedence:
//  1. An explicit valid customer_id.
//  2. A customer name typed at checkout: reuse an existing customer with the
//     same name in the store (case-insensitive), or create a new one.
//  3. No customer: the sale is stored with a NULL customer_id.
func (s *Service) resolveCustomer(ctx context.Context, tx *gorm.DB, req CreateSaleRequest, storeID uuid.UUID) (uuid.UUID, bool, error) {
	// 1. Explicit customer ID takes precedence.
	if req.CustomerID != "" && isValidUUID(req.CustomerID) {
		customerID, _ := uuid.Parse(req.CustomerID)
		return customerID, true, nil
	}

	// 2. Customer typed by name: find-or-create.
	if name := strings.TrimSpace(req.CustomerName); name != "" {
		var existing gormdb.Customer
		err := tx.Where("LOWER(name) = LOWER(?)", name).
			Where("store_id = ?", storeID).
			First(&existing).Error
		if err == nil {
			return existing.ID, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, false, apperror.Wrap(apperror.ErrInternal, "Failed to find customer", err)
		}

		customer := &gormdb.Customer{
			StoreID:  storeID,
			Code:     generateCustomerCode(),
			Name:     name,
			IsActive: true,
		}
		if err := tx.Create(customer).Error; err != nil {
			return uuid.Nil, false, apperror.Wrap(apperror.ErrInternal, "Failed to create customer", err)
		}
		return customer.ID, true, nil
	}

	// 3. No customer: NULL.
	return uuid.Nil, false, nil
}

// generateCustomerCode generates a unique customer code for customers created
// on the fly during checkout.
func generateCustomerCode() string {
	return fmt.Sprintf("CUS-%s", strings.ToUpper(uuid.New().String()[:8]))
}

// openShiftForSale resolves the shift a new sale must be linked to (B5/B8).
// Shift-capable cashiers (roles holding any cashier.* permission) MUST have an
// OPEN shift; non-cashier roles (e.g. manager) are not gated and return nil.
func (s *Service) openShiftForSale(ctx context.Context, tx *gorm.DB, cashierID, storeID uuid.UUID) (*uuid.UUID, error) {
	var shiftCapable bool
	if err := tx.Raw(`SELECT EXISTS (
		SELECT 1 FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN users u ON u.role_id = rp.role_id
		WHERE u.id = ?
		  AND p.name IN ('cashier.open_shift','cashier.close_shift','cashier.view_shift')
	)`, cashierID).Scan(&shiftCapable).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to check cashier permissions", err)
	}

	if !shiftCapable {
		return nil, nil
	}

	shift, err := s.openShiftOfActor(ctx, tx, cashierID)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, apperror.Wrap(apperror.ErrShiftNotOpen, "You must open a shift before making a sale", nil)
	}
	if shift.StoreID != storeID {
		return nil, apperror.Wrap(apperror.ErrForbidden, "Open shift belongs to a different store", nil)
	}
	id := shift.ID
	return &id, nil
}

// openShiftOfActor returns the actor's current OPEN shift (nil when none).
// The row is locked FOR UPDATE so the sale and the shift state cannot race.
func (s *Service) openShiftOfActor(ctx context.Context, tx *gorm.DB, actorID uuid.UUID) (*gormdb.CashierShift, error) {
	if actorID == uuid.Nil {
		return nil, nil
	}
	var shift gormdb.CashierShift
	if err := tx.Raw(
		"SELECT * FROM cashier_shifts WHERE user_id = ? AND status = 'OPEN' ORDER BY opened_at DESC LIMIT 1 FOR UPDATE",
		actorID,
	).Scan(&shift).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to load open shift", err)
	}
	if shift.ID == uuid.Nil {
		return nil, nil
	}
	return &shift, nil
}

// skuVariantLabel builds a compact human label from a SKU's variant fields
// (e.g. "Large / Red / M") for the shift_items snapshot.
func skuVariantLabel(sku gormdb.ProductSKU) string {
	var parts []string
	for _, p := range []string{sku.Variant, sku.Color, sku.Size} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, strings.TrimSpace(p))
		}
	}
	return strings.Join(parts, " / ")
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// computeSaleTotals computes the item subtotal, discount, tax and grand total
// for a set of sale items. Item-level discounts are clamped at zero. The
// returned values are: subtotal, discount, tax, grandTotal.
func computeSaleTotals(items []SaleItemInput, discount, taxRate float64) (subtotal, discountOut, tax, grandTotal float64) {
	for _, item := range items {
		itemSubtotal := (item.UnitPrice * float64(item.Quantity)) - item.Discount
		if itemSubtotal < 0 {
			itemSubtotal = 0
		}
		subtotal += itemSubtotal
	}

	// Sale-level discount must not exceed the subtotal.
	if discount > subtotal {
		discount = subtotal
	}
	discountOut = discount

	tax = (subtotal - discount) * taxRate / 100
	grandTotal = subtotal - discount + tax
	return subtotal, discountOut, tax, grandTotal
}

// validatePayment rejects a sale when the amount paid is insufficient to cover
// the grand total, regardless of payment method.
func validatePayment(paymentMethod string, amountPaid, grandTotal float64) error {
	if paymentMethod == "" {
		return nil
	}
	if amountPaid < grandTotal {
		return apperror.Wrap(apperror.ErrPaymentInsufficient,
			fmt.Sprintf("Payment amount insufficient. Grand total: %.2f, Amount paid: %.2f", grandTotal, amountPaid), nil)
	}
	return nil
}
