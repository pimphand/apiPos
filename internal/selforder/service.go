// Package selforder implements the public dine-in self-ordering flow
// (docs/pos-self-order.md). All operations are resolved server-side from a
// table token; the client can never choose a tenant.
package selforder

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
	"apiPos/internal/realtime"
	"apiPos/internal/repository"
)

// Statuses copied from the sale package (kept local to avoid a cycle).
const (
	StatusPending   = "PENDING"
	StatusPaid      = "PAID"
	StatusCancelled = "CANCELLED"
)

// Payment methods allowed for self ordering (CASH is deliberately excluded —
// there is no cashier at the table).
var allowedPaymentMethods = map[string]bool{
	"CARD":          true,
	"BANK_TRANSFER": true,
	"EWALLET":       true,
	"OTHER":         true,
}

// SelfOrderPaymentTimeout is how long a submitted (PENDING) self order may
// wait before the cleanup job cancels it and restores stock.
const SelfOrderPaymentTimeout = 15 * time.Minute

// Service handles the public self-order business logic.
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new selforder Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// TableInfo is the resolved table + store for the public flow.
type TableInfo struct {
	Table  *gormdb.Table
	Store  *gormdb.Store
	Setting *gormdb.StoreSetting
}

// ResolveTable looks up a table by its globally-unique QR token and verifies
// the store + table are accepting self orders.
func (s *Service) ResolveTable(ctx context.Context, tableToken string) (*TableInfo, error) {
	token := strings.TrimSpace(tableToken)
	if token == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Table token is required", nil)
	}

	table, err := s.repos.Table.GetByQRToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "QR meja tidak valid", nil)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to resolve table", err)
	}

	if err := s.validateTableActive(ctx, table); err != nil {
		return nil, err
	}
	return s.tableInfo(ctx, table)
}

// validateTableActive ensures the table and its store are operational.
func (s *Service) validateTableActive(ctx context.Context, table *gormdb.Table) error {
	var store gormdb.Store
	if err := s.repos.DB().WithContext(ctx).Where("id = ?", table.StoreID).First(&store).Error; err != nil {
		return apperror.Wrap(apperror.ErrNotFound, "QR meja tidak valid atau layanan nonaktif", err)
	}
	if !store.IsActive || !store.IsVerified {
		return apperror.Wrap(apperror.ErrFailedPrecondition, "QR meja tidak valid atau layanan nonaktif", nil)
	}

	setting, err := s.repos.Settings.GetByStoreID(ctx, store.ID.String())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.Wrap(apperror.ErrInternal, "Failed to load store settings", err)
	}
	if setting == nil {
		// A store without settings rows defaults to disabled.
		return apperror.Wrap(apperror.ErrFailedPrecondition, "QR meja tidak valid atau layanan nonaktif", nil)
	}
	if !setting.SelfOrderEnabled {
		return apperror.Wrap(apperror.ErrFailedPrecondition, "QR meja tidak valid atau layanan nonaktif", nil)
	}
	if !table.IsActive {
		return apperror.Wrap(apperror.ErrFailedPrecondition, "QR meja tidak valid atau layanan nonaktif", nil)
	}
	return nil
}

// tableInfo loads the store + settings for a resolved table.
func (s *Service) tableInfo(ctx context.Context, table *gormdb.Table) (*TableInfo, error) {
	var store gormdb.Store
	if err := s.repos.DB().WithContext(ctx).Where("id = ?", table.StoreID).First(&store).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Store not found", err)
	}
	setting, _ := s.repos.Settings.GetByStoreID(ctx, store.ID.String())
	return &TableInfo{Table: table, Store: &store, Setting: setting}, nil
}

// PublicProduct is a menu product with safe fields only.
type PublicProduct struct {
	Product      gormdb.Product
	CategoryName string
	Variants     []gormdb.ProductSKU
}

// GetPublicMenu returns the dine-in menu for the store of the table token.
func (s *Service) GetPublicMenu(ctx context.Context, tableToken string) (*TableInfo, []gormdb.Category, []PublicProduct, error) {
	info, err := s.ResolveTable(ctx, tableToken)
	if err != nil {
		return nil, nil, nil, err
	}

	storeID := info.Table.StoreID

	var categories []gormdb.Category
	if err := s.repos.DB().WithContext(ctx).
		Where("store_id = ? AND is_active = ?", storeID, true).
		Order("name ASC").Find(&categories).Error; err != nil {
		return nil, nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to load menu", err)
	}

	var products []gormdb.Product
	if err := s.repos.DB().WithContext(ctx).
		Where("store_id = ? AND is_active = ?", storeID, true).
		Order("name ASC").Find(&products).Error; err != nil {
		return nil, nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to load menu", err)
	}

	catName := map[string]string{}
	for _, c := range categories {
		catName[c.ID.String()] = c.Name
	}

	// Load SKUs for the store's active products in one query.
	var skus []gormdb.ProductSKU
	if len(products) > 0 {
		productIDs := make([]string, 0, len(products))
		for _, p := range products {
			productIDs = append(productIDs, p.ID.String())
		}
		if err := s.repos.DB().WithContext(ctx).
			Where("store_id = ? AND is_active = ? AND product_id IN ?", storeID, true, productIDs).
			Find(&skus).Error; err != nil {
			return nil, nil, nil, apperror.Wrap(apperror.ErrInternal, "Failed to load menu", err)
		}
	}

	skusByProduct := map[string][]gormdb.ProductSKU{}
	for _, sku := range skus {
		skusByProduct[sku.ProductID.String()] = append(skusByProduct[sku.ProductID.String()], sku)
	}

	menu := make([]PublicProduct, 0, len(products))
	for _, p := range products {
		menu = append(menu, PublicProduct{
			Product:      p,
			CategoryName: catName[p.CategoryID.String()],
			Variants:     skusByProduct[p.ID.String()],
		})
	}

	return info, categories, menu, nil
}

// CreatePublicOrder validates the cart and creates a PENDING self-order sale,
// locking stock in one DB transaction. No realtime event is published here.
func (s *Service) CreatePublicOrder(ctx context.Context, tableToken string, items []OrderItem, customerName, notes string) (*gormdb.Sale, error) {
	info, err := s.ResolveTable(ctx, tableToken)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Order must have at least one item", nil)
	}

	storeID := info.Table.StoreID
	taxRate := 11.0
	if info.Setting != nil {
		taxRate = info.Setting.TaxRate
	}
	if taxRate < 0 {
		taxRate = 0
	}

	token := uuid.New()
	invoice := generateInvoiceNumber()

	var saved *gormdb.Sale
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Resolve optional customer by typed name (find-or-create in store).
		var customerID uuid.UUID
		useCustomer := false
		if name := strings.TrimSpace(customerName); name != "" {
			customer, ok, cerr := s.resolveCustomer(ctx, tx, storeID, name)
			if cerr != nil {
				return cerr
			}
			if ok {
				customerID = customer
				useCustomer = true
			}
		}

		var subtotal float64
		var saleItems []gormdb.SaleItem
		for _, item := range items {
			var unitPrice float64
			productID, perr := uuid.Parse(item.ProductID)
			if perr != nil || item.Quantity <= 0 {
				return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid order item", nil)
			}

			var product gormdb.Product
			if err := tx.Raw("SELECT * FROM products WHERE id = ? AND (store_id = ? OR store_id IS NULL) FOR UPDATE", productID, storeID).Scan(&product).Error; err != nil {
				return apperror.Wrap(apperror.ErrNotFound, "Product not found", err)
			}
			if product.ID == uuid.Nil {
				return apperror.Wrap(apperror.ErrNotFound, "Product tidak tersedia", nil)
			}
			if !product.IsActive {
				return apperror.Wrap(apperror.ErrFailedPrecondition, "Product tidak tersedia", nil)
			}

			skuID, err := uuid.Parse(item.ProductSKUID)
			if err != nil {
				skuID = uuid.Nil
			}

			var beforeStock, afterStock int32
			var skuIDRef *uuid.UUID
			if skuID != uuid.Nil {
				var sku gormdb.ProductSKU
				if err := tx.Raw("SELECT * FROM product_skus WHERE id = ? AND (store_id = ? OR store_id IS NULL) FOR UPDATE", skuID, storeID).Scan(&sku).Error; err != nil {
					return apperror.Wrap(apperror.ErrNotFound, "Product tidak tersedia", err)
				}
				if sku.ID == uuid.Nil {
					return apperror.Wrap(apperror.ErrNotFound, "Product tidak tersedia", nil)
				}
				if sku.ProductID != productID {
					return apperror.Wrap(apperror.ErrInvalidRequest, "SKU tidak cocok dengan produk", nil)
				}
				if !sku.IsActive {
					return apperror.Wrap(apperror.ErrFailedPrecondition, "Product tidak tersedia", nil)
				}
				if sku.Stock < item.Quantity {
					return apperror.Wrap(apperror.ErrInsufficientStock, fmt.Sprintf("Stok tidak cukup untuk %s", sku.SKU), nil)
				}
				beforeStock = sku.Stock
				afterStock = sku.Stock - item.Quantity
				if err := tx.Model(&gormdb.ProductSKU{}).Where("id = ?", skuID).Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to update stock", err)
				}
				// Keep products.stock = SUM(product_skus.stock).
				var agg int64
				if err := tx.Model(&gormdb.ProductSKU{}).Where("product_id = ?", productID).Select("COALESCE(SUM(stock), 0)").Scan(&agg).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to recompute stock", err)
				}
				if err := tx.Model(&gormdb.Product{}).Where("id = ?", productID).Update("stock", agg).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to update product stock", err)
				}
				id := skuID
				skuIDRef = &id
				unitPrice = sku.SellingPrice
				subtotal += sku.SellingPrice * float64(item.Quantity)
			} else {
				if product.Stock < item.Quantity {
					return apperror.Wrap(apperror.ErrInsufficientStock, fmt.Sprintf("Stok tidak cukup untuk %s", product.Name), nil)
				}
				beforeStock = product.Stock
				afterStock = product.Stock - item.Quantity
				if err := tx.Model(&gormdb.Product{}).Where("id = ?", productID).Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to update stock", err)
				}
				unitPrice = product.SellingPrice
				subtotal += product.SellingPrice * float64(item.Quantity)
			}

			saleItem := &gormdb.SaleItem{
				SaleID:       uuid.Nil, // assigned after sale creation
				StoreID:      storeID,
				ProductID:    productID,
				ProductSKUID: skuIDRef,
				Quantity:     item.Quantity,
				UnitPrice:    unitPrice,
				Discount:     0,
				Subtotal:     unitPrice * float64(item.Quantity),
			}
			saleItems = append(saleItems, *saleItem)

			invTxn := &gormdb.InventoryTransaction{
				ProductID:     productID,
				StoreID:       storeID,
				Type:          "SALE",
				Quantity:      item.Quantity,
				ReferenceType: "SELF_ORDER",
				BeforeStock:   beforeStock,
				AfterStock:    afterStock,
			}
			if err := tx.Omit("CreatedBy").Create(invTxn).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to record inventory", err)
			}
		}

		tax, grandTotal := computeOrderTotals(subtotal, taxRate)
		sale := &gormdb.Sale{
			InvoiceNumber: invoice,
			StoreID:       storeID,
			CustomerID:    customerID,
			Subtotal:      subtotal,
			Discount:      0,
			Tax:           tax,
			GrandTotal:    grandTotal,
			Status:        StatusPending,
			OrderSource:   "SELF_ORDER",
			OrderStatus:   "ORDERED",
			TableID:       &info.Table.ID,
			TableNo:       info.Table.Code,
			OrderToken:    &token,
		}
		if n := strings.TrimSpace(notes); n != "" {
			sale.Notes = &n
		}

		// NULL cashier_id (customer action, no cashier at the table); when no
		// customer was resolved, NULL customer_id too (same Omit pattern as
		// the cashier checkout in sale/service.go).
		create := tx.Omit("CashierID")
		if !useCustomer {
			create = create.Omit("CustomerID")
		}
		if err := create.Create(sale).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to create order", err)
		}

		for i := range saleItems {
			saleItems[i].SaleID = sale.ID
			if err := tx.Create(&saleItems[i]).Error; err != nil {
				return apperror.Wrap(apperror.ErrInternal, "Failed to create order items", err)
			}
		}

		// Customer row (if any) must reference this sale's store — already
		// created before; the sale row references it via CustomerID.
		saved = sale
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

// resolveCustomer finds an active customer by name in the store, or creates
// one. Mirrors the cashier checkout behaviour.
func (s *Service) resolveCustomer(ctx context.Context, tx *gorm.DB, storeID uuid.UUID, name string) (uuid.UUID, bool, error) {
	var existing gormdb.Customer
	err := tx.Where("LOWER(name) = LOWER(?)", name).Where("store_id = ?", storeID).First(&existing).Error
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

// PayPublicOrder marks a submitted order as PAID and records the payment in
// one transaction, then publishes the realtime events (sale.created with
// order_source=SELF_ORDER + status=PAID, and inventory.adjusted so POS stock
// views update in real time).
func (s *Service) PayPublicOrder(ctx context.Context, orderToken, paymentMethod, reference string) (*gormdb.Sale, error) {
	ot := strings.TrimSpace(orderToken)
	if ot == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Order token is required", nil)
	}
	if err := validatePaymentMethod(paymentMethod); err != nil {
		return nil, err
	}

	sale, err := s.saleByOrderToken(ctx, ot)
	if err != nil {
		return nil, err
	}
	if sale.OrderSource != "SELF_ORDER" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Order is not a self order", nil)
	}
	if sale.Status != StatusPending {
		return nil, apperror.Wrap(apperror.ErrInvalidSaleStatus, "Order is not awaiting payment", nil)
	}
	if time.Since(sale.CreatedAt) > SelfOrderPaymentTimeout {
		return nil, apperror.Wrap(apperror.ErrFailedPrecondition, "Order has expired, please submit again", nil)
	}

	paidAt := time.Now()
	err = s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		payment := &gormdb.Payment{
			SaleID:        sale.ID,
			StoreID:       sale.StoreID,
			PaymentMethod: paymentMethod,
			Amount:        sale.GrandTotal,
			Reference:     reference,
			PaidAt:        paidAt,
		}
		if err := tx.Create(payment).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to record payment", err)
		}
		if err := tx.Model(&gormdb.Sale{}).Where("id = ?", sale.ID).Update("status", StatusPaid).Error; err != nil {
			return apperror.Wrap(apperror.ErrInternal, "Failed to update order status", err)
		}
		sale.Status = StatusPaid
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Public endpoints carry no JWT claims, so PublishFromContext would leave
	// StoreID empty — set it explicitly (docs/pos-self-order.md §6.2).
	realtime.Publish(realtime.Event{
		Event:      realtime.EventSaleCreated,
		Resource:   "sales",
		ResourceID: sale.ID.String(),
		StoreID:    sale.StoreID.String(),
		Data: map[string]any{
			"order_source":  "SELF_ORDER",
			"status":        StatusPaid,
			"invoice_number": sale.InvoiceNumber,
			"table_no":      sale.TableNo,
		},
	})
	realtime.Publish(realtime.Event{
		Event:      realtime.EventInventoryAdjusted,
		Resource:   "sales",
		ResourceID: sale.ID.String(),
		StoreID:    sale.StoreID.String(),
	})

	return sale, nil
}

// GetOrderStatus returns the current status of the customer's order.
func (s *Service) GetOrderStatus(ctx context.Context, orderToken string) (*gormdb.Sale, error) {
	ot := strings.TrimSpace(orderToken)
	if ot == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Order token is required", nil)
	}
	sale, err := s.saleByOrderToken(ctx, ot)
	if err != nil {
		return nil, err
	}
	return sale, nil
}

// saleByOrderToken loads a sale by its unique order token.
func (s *Service) saleByOrderToken(ctx context.Context, token string) (*gormdb.Sale, error) {
	var sale gormdb.Sale
	if err := s.repos.DB().WithContext(ctx).Where("order_token = ?", token).First(&sale).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Order tidak ditemukan", nil)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to load order", err)
	}
	return &sale, nil
}

// OrderItem is a validated cart line (server-side pricing).
type OrderItem struct {
	ProductID   string
	ProductSKUID string
	Quantity    int32
}

// CancelExpiredPending cancels self orders stuck in PENDING past the payment
// timeout and restores their stock. Runs per sale in its own transaction.
// Returns the number of cancelled orders.
func (s *Service) CancelExpiredPending(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-SelfOrderPaymentTimeout)

	var expired []gormdb.Sale
	if err := s.repos.DB().WithContext(ctx).
		Where("order_source = ? AND status = ? AND created_at < ?", "SELF_ORDER", StatusPending, cutoff).
		Find(&expired).Error; err != nil {
		return 0, apperror.Wrap(apperror.ErrInternal, "Failed to find expired orders", err)
	}

	cancelled := 0
	for i := range expired {
		sale := &expired[i]
		err := s.repos.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var items []gormdb.SaleItem
			if err := tx.Where("sale_id = ?", sale.ID).Find(&items).Error; err != nil {
				return err
			}
			for _, item := range items {
				// Restore stock at the exact level it was taken from.
				if item.ProductSKUID != nil {
					if err := tx.Model(&gormdb.ProductSKU{}).
						Where("id = ?", *item.ProductSKUID).
						Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
						return apperror.Wrap(apperror.ErrInternal, "Failed to restore stock", err)
					}
					var agg int64
					if err := tx.Model(&gormdb.ProductSKU{}).
						Where("product_id = ?", item.ProductID).
						Select("COALESCE(SUM(stock), 0)").Scan(&agg).Error; err != nil {
						return apperror.Wrap(apperror.ErrInternal, "Failed to recompute stock", err)
					}
					if err := tx.Model(&gormdb.Product{}).Where("id = ?", item.ProductID).Update("stock", agg).Error; err != nil {
						return apperror.Wrap(apperror.ErrInternal, "Failed to update product stock", err)
					}
				} else {
					if err := tx.Model(&gormdb.Product{}).
						Where("id = ?", item.ProductID).
						Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
						return apperror.Wrap(apperror.ErrInternal, "Failed to restore stock", err)
					}
				}
				invTxn := &gormdb.InventoryTransaction{
					ProductID:     item.ProductID,
					StoreID:       sale.StoreID,
					Type:          "RETURN",
					Quantity:      item.Quantity,
					ReferenceType: "SELF_ORDER_EXPIRED",
					ReferenceID:   sale.ID,
				}
				if err := tx.Omit("CreatedBy").Create(invTxn).Error; err != nil {
					return apperror.Wrap(apperror.ErrInternal, "Failed to record inventory", err)
				}
			}
			return tx.Model(&gormdb.Sale{}).Where("id = ?", sale.ID).Update("status", StatusCancelled).Error
		})
		if err != nil {
			return cancelled, err
		}
		cancelled++
	}
	return cancelled, nil
}

// validatePaymentMethod rejects CASH (no cashier at the table) and unknown
// methods for self ordering.
func validatePaymentMethod(method string) error {
	if !allowedPaymentMethods[method] {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Cash payment is not supported for self ordering", nil)
	}
	return nil
}

// computeOrderTotals applies the store tax rate to the item subtotal.
// Self orders have no discounts, so grand total is simply subtotal + tax.
func computeOrderTotals(subtotal, taxRate float64) (tax, grandTotal float64) {
	if taxRate < 0 {
		taxRate = 0
	}
	tax = subtotal * taxRate / 100
	return tax, subtotal + tax
}

// generateInvoiceNumber generates a unique invoice number.
func generateInvoiceNumber() string {
	return fmt.Sprintf("INV-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
}

// generateCustomerCode generates a unique customer code.
func generateCustomerCode() string {
	return fmt.Sprintf("CUS-%s", strings.ToUpper(uuid.New().String()[:8]))
}