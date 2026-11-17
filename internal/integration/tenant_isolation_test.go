// Package integration holds cross-service regression tests that need a real
// PostgreSQL database.
//
// Run with:
//
//	TEST_DATABASE_DSN='postgres://postgres:postgres@127.0.0.1:55432/postgres?sslmode=disable' \
//	  go test ./internal/integration/...
//
// Each run creates a throwaway schema, migrates the models it needs, and drops
// the schema afterwards, so it never touches development data.
package integration

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/inventory"
	"apiPos/internal/report"
	"apiPos/internal/repository"
	"apiPos/internal/sale"
	"apiPos/internal/user"
)

// testTenant is one seeded store with the rows the isolation checks read.
type testTenant struct {
	store    gormdb.Store
	user     gormdb.User
	category gormdb.Category
	product  gormdb.Product
	sale     gormdb.Sale
	txn      gormdb.InventoryTransaction
}

func TestTenantIsolation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TEST_DATABASE_DSN is not set; skipping database-backed tenant isolation test")
	}

	db := newTestDB(t, dsn)
	repos := repository.New(db)

	role := gormdb.Role{ID: uuid.New(), Name: "cashier"}
	require.NoError(t, db.Create(&role).Error)

	tenantA := seedTenant(t, db, role, "A", 100)
	tenantB := seedTenant(t, db, role, "B", 200)

	ctx := context.Background()

	t.Run("sale", func(t *testing.T) {
		svc := sale.NewService(repos)

		detail, err := svc.GetSale(ctx, tenantA.sale.ID.String(), tenantA.store.ID.String())
		require.NoError(t, err)
		assert.Equal(t, tenantA.sale.InvoiceNumber, detail.Sale.InvoiceNumber)

		_, err = svc.GetSale(ctx, tenantB.sale.ID.String(), tenantA.store.ID.String())
		requireDenied(t, err)
		_, err = svc.GetSale(ctx, tenantA.sale.ID.String(), tenantB.store.ID.String())
		requireDenied(t, err)

		_, err = svc.GetSaleByInvoice(ctx, tenantB.sale.InvoiceNumber, tenantA.store.ID.String())
		requireDenied(t, err)

		list, err := svc.ListSales(ctx, repository.SaleListParams{
			StoreID: tenantA.store.ID.String(),
			Page:    1,
			PerPage: 20,
		})
		require.NoError(t, err)
		require.Len(t, list.Sales, 1)
		assert.Equal(t, tenantA.sale.ID, list.Sales[0].ID)

		err = svc.CancelSale(ctx, tenantB.sale.ID.String(), tenantB.user.ID.String(), tenantA.store.ID.String())
		requireDenied(t, err)

		err = svc.RefundSale(ctx, tenantB.sale.ID.String(), tenantB.user.ID.String(), tenantA.store.ID.String())
		requireDenied(t, err)

		var stored gormdb.Sale
		require.NoError(t, db.First(&stored, "id = ?", tenantB.sale.ID).Error)
		assert.Equal(t, sale.StatusPaid, stored.Status, "cross-store cancel must not change the other tenant's sale")
	})

	t.Run("inventory", func(t *testing.T) {
		svc := inventory.NewService(repos)

		txn, err := svc.GetTransaction(ctx, tenantA.txn.ID.String(), tenantA.store.ID.String())
		require.NoError(t, err)
		assert.Equal(t, tenantA.store.ID, txn.StoreID)

		_, err = svc.GetTransaction(ctx, tenantB.txn.ID.String(), tenantA.store.ID.String())
		requireDenied(t, err)

		list, err := svc.ListTransactions(ctx, "", tenantA.store.ID.String(), "", 1, 20)
		require.NoError(t, err)
		require.Len(t, list.Transactions, 1)
		assert.Equal(t, tenantA.txn.ID, list.Transactions[0].ID)

		_, err = svc.GetStockSummary(ctx, tenantB.product.ID.String(), tenantA.store.ID.String())
		requireDenied(t, err)

		low, err := svc.GetLowStockAlert(ctx, tenantA.store.ID.String(), 1, 20)
		require.NoError(t, err)
		require.Len(t, low.Products, 1, "low stock alert must only contain the caller's store")
		assert.Equal(t, tenantA.product.ID, low.Products[0].ProductID)
	})

	t.Run("report", func(t *testing.T) {
		svc := report.NewService(repos)
		start := time.Now().Add(-time.Hour).Format(time.RFC3339)
		end := time.Now().Add(time.Hour).Format(time.RFC3339)
		storeA := tenantA.store.ID.String()

		daily, err := svc.GetDailySalesReport(ctx, start, end, storeA)
		require.NoError(t, err)
		assert.InDelta(t, tenantA.sale.GrandTotal, daily.TotalAmount, 0.01)

		rangeResult, err := svc.GetSalesByDateRange(ctx, start, end, storeA, "")
		require.NoError(t, err)
		require.Len(t, rangeResult.Sales, 1)
		assert.Equal(t, tenantA.sale.InvoiceNumber, rangeResult.Sales[0].InvoiceNumber)

		byProduct, err := svc.GetSalesByProduct(ctx, start, end, storeA, 10)
		require.NoError(t, err)
		require.Len(t, byProduct, 1)
		assert.Equal(t, tenantA.product.ID.String(), byProduct[0].ProductID)

		byCategory, err := svc.GetSalesByCategory(ctx, start, end, storeA)
		require.NoError(t, err)
		require.Len(t, byCategory, 1)
		assert.Equal(t, tenantA.category.Name, byCategory[0].CategoryName)

		payments, err := svc.GetPaymentMethodSummary(ctx, start, end, storeA)
		require.NoError(t, err)
		assert.InDelta(t, tenantA.sale.GrandTotal, payments.TotalAmount, 0.01)

		stock, err := svc.GetInventorySummary(ctx, storeA)
		require.NoError(t, err)
		require.Len(t, stock.Reports, 1)
		assert.Equal(t, tenantA.category.Name, stock.Reports[0].CategoryName)
		assert.EqualValues(t, 1, stock.TotalProducts)

		low, err := svc.GetLowStockProducts(ctx, 10, storeA)
		require.NoError(t, err)
		require.Len(t, low, 1)
		assert.Equal(t, tenantA.product.ID.String(), low[0].ProductID)
	})

	t.Run("user", func(t *testing.T) {
		svc := user.NewService(repos, nil)
		storeA := tenantA.store.ID.String()

		found, err := svc.GetUser(ctx, tenantA.user.ID.String(), storeA)
		require.NoError(t, err)
		assert.Equal(t, tenantA.user.Username, found.Username)

		_, err = svc.GetUser(ctx, tenantB.user.ID.String(), storeA)
		requireDenied(t, err)

		claimsCtx := auth.ContextWithClaims(ctx, &auth.Claims{
			UserID:   tenantA.user.ID.String(),
			RoleName: "admin",
			StoreID:  storeA,
		})
		users, total, err := svc.ListUsers(claimsCtx, "", "", 1, 20)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, users, 1)
		assert.Equal(t, tenantA.user.ID, users[0].ID)

		_, err = svc.UpdateUser(ctx, tenantB.user.ID.String(), "changed@example.com", "Changed", "", storeA, true)
		requireDenied(t, err)

		err = svc.DeleteUser(ctx, tenantB.user.ID.String(), storeA)
		requireDenied(t, err)

		_, err = svc.UpdateUserRole(ctx, tenantB.user.ID.String(), role.ID.String(), storeA)
		requireDenied(t, err)

		_, err = svc.ToggleUserActive(ctx, tenantB.user.ID.String(), false, storeA)
		requireDenied(t, err)

		var stored gormdb.User
		require.NoError(t, db.First(&stored, "id = ?", tenantB.user.ID).Error)
		assert.NotEqual(t, "changed@example.com", stored.Email)
		assert.True(t, stored.IsActive, "cross-store updates must not touch the other tenant's user")
	})
}

// newTestDB creates an isolated schema for this test run and returns a
// connection scoped to it. The schema is dropped when the test finishes.
func newTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	schema := fmt.Sprintf("tenant_isolation_%d", time.Now().UnixNano())

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err, "connect to TEST_DATABASE_DSN")
	require.NoError(t, admin.Exec(`CREATE SCHEMA "`+schema+`"`).Error)

	scoped, err := gorm.Open(postgres.Open(withSearchPath(dsn, schema)), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, scoped.AutoMigrate(
		&gormdb.Store{},
		&gormdb.Role{},
		&gormdb.User{},
		&gormdb.Category{},
		&gormdb.Product{},
		&gormdb.ProductSKU{},
		&gormdb.Customer{},
		&gormdb.Sale{},
		&gormdb.SaleItem{},
		&gormdb.Payment{},
		&gormdb.InventoryTransaction{},
	))

	t.Cleanup(func() {
		if sqlDB, err := scoped.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return scoped
}

// withSearchPath points the connection at the throwaway schema, for both URL
// and keyword/value DSNs.
func withSearchPath(dsn, schema string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			query := parsed.Query()
			query.Set("search_path", schema)
			parsed.RawQuery = query.Encode()
			return parsed.String()
		}
	}
	return dsn + " search_path=" + schema
}

// seedTenant inserts one complete tenant: store, cashier, category, product
// with low stock, a PAID sale with one item and payment, and an inventory
// transaction.
func seedTenant(t *testing.T, db *gorm.DB, role gormdb.Role, suffix string, amount float64) testTenant {
	t.Helper()

	tenant := testTenant{
		store:    gormdb.Store{ID: uuid.New(), Name: "Store " + suffix, IsActive: true, IsVerified: true},
		category: gormdb.Category{ID: uuid.New(), Name: "Category " + suffix, IsActive: true},
		user: gormdb.User{
			ID:           uuid.New(),
			Username:     "cashier" + suffix,
			Email:        "cashier" + suffix + "@example.com",
			PasswordHash: "x",
			FullName:     "Cashier " + suffix,
			RoleID:       role.ID,
			IsActive:     true,
		},
	}
	tenant.category.StoreID = tenant.store.ID
	tenant.user.StoreID = tenant.store.ID

	tenant.product = gormdb.Product{
		ID:           uuid.New(),
		StoreID:      tenant.store.ID,
		SKU:          "SKU-" + suffix,
		Name:         "Product " + suffix,
		CategoryID:   tenant.category.ID,
		CostPrice:    50,
		SellingPrice: amount,
		Stock:        2,
		MinimumStock: 5,
		IsActive:     true,
	}

	tenant.sale = gormdb.Sale{
		ID:            uuid.New(),
		InvoiceNumber: "INV-" + suffix,
		StoreID:       tenant.store.ID,
		CashierID:     tenant.user.ID,
		Subtotal:      amount,
		GrandTotal:    amount,
		Status:        sale.StatusPaid,
		OrderSource:   "CASHIER",
		OrderStatus:   "COMPLETED",
	}

	item := gormdb.SaleItem{
		ID:        uuid.New(),
		SaleID:    tenant.sale.ID,
		StoreID:   tenant.store.ID,
		ProductID: tenant.product.ID,
		Quantity:  1,
		UnitPrice: amount,
		Subtotal:  amount,
	}

	payment := gormdb.Payment{
		ID:            uuid.New(),
		SaleID:        tenant.sale.ID,
		StoreID:       tenant.store.ID,
		PaymentMethod: "CASH",
		Amount:        amount,
		PaidAt:        time.Now(),
	}

	tenant.txn = gormdb.InventoryTransaction{
		ID:          uuid.New(),
		ProductID:   tenant.product.ID,
		StoreID:     tenant.store.ID,
		Type:        inventory.TypePurchase,
		Quantity:    2,
		BeforeStock: 0,
		AfterStock:  2,
		CreatedBy:   tenant.user.ID,
	}

	require.NoError(t, db.Create(&tenant.store).Error)
	require.NoError(t, db.Create(&tenant.user).Error)
	require.NoError(t, db.Create(&tenant.category).Error)
	require.NoError(t, db.Create(&tenant.product).Error)
	require.NoError(t, db.Create(&tenant.sale).Error)
	require.NoError(t, db.Create(&item).Error)
	require.NoError(t, db.Create(&payment).Error)
	require.NoError(t, db.Create(&tenant.txn).Error)

	return tenant
}

// requireDenied asserts the service hid the other tenant's row (NotFound rather
// than Forbidden, so existence is not disclosed).
func requireDenied(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, codes.NotFound, apperror.MapError(err).Code, "expected not found, got: %v", err)
}
