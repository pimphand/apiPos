package repository

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "apiPos/internal/database/sqlc"
)

// Helper to create prerequisite data for sales tests
func createTestStore(t *testing.T, queries *db.Queries) db.Store {
	t.Helper()
	ctx := context.Background()
	store, err := queries.CreateStore(ctx, db.CreateStoreParams{
		Name: fmt.Sprintf("TestStore_%d", time.Now().UnixNano()),
		Address: pgtype.Text{String: "123 Test Street", Valid: true},
		Phone: pgtype.Text{String: "1234567890", Valid: true},
	})
	require.NoError(t, err)
	return store
}

func createTestUserForSale(t *testing.T, queries *db.Queries, storeID int64) db.User {
	t.Helper()
	ctx := context.Background()
	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name: fmt.Sprintf("sale_role_%d", time.Now().UnixNano()),
	})
	require.NoError(t, err)

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("saleuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("saleuser_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Sale User",
		RoleID:       role.ID,
		StoreID:      pgtype.Int8{Int64: storeID, Valid: true},
	})
	require.NoError(t, err)
	return user
}

func createTestProductForSale(t *testing.T, queries *db.Queries) db.Product {
	t.Helper()
	ctx := context.Background()
	product, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Sku:          fmt.Sprintf("SALE_SKU_%d", time.Now().UnixNano()),
		Name:         "Sale Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)
	return product
}

func TestSaleRepository_CreateSale(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	t.Run("success", func(t *testing.T) {
		sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
			InvoiceNumber: fmt.Sprintf("INV-%d", time.Now().UnixNano()),
			StoreID:       store.ID,
			CashierID:     cashier.ID,
			Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Discount:      pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
			Tax:           pgtype.Numeric{Int: big.NewInt(9500), Valid: true},
			GrandTotal:    pgtype.Numeric{Int: big.NewInt(104500), Valid: true},
			Status:        "PENDING",
		})

		require.NoError(t, err)
		assert.NotZero(t, sale.ID)
		assert.NotEmpty(t, sale.InvoiceNumber)
		assert.Equal(t, store.ID, sale.StoreID)
		assert.Equal(t, cashier.ID, sale.CashierID)
		assert.Equal(t, "PENDING", sale.Status)
	})

	t.Run("duplicate invoice number", func(t *testing.T) {
		invoiceNum := fmt.Sprintf("DUP-INV-%d", time.Now().UnixNano())
		_, err := repo.CreateSale(ctx, db.CreateSaleParams{
			InvoiceNumber: invoiceNum,
			StoreID:       store.ID,
			CashierID:     cashier.ID,
			Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Status:        "PENDING",
		})
		require.NoError(t, err)

		_, err = repo.CreateSale(ctx, db.CreateSaleParams{
			InvoiceNumber: invoiceNum,
			StoreID:       store.ID,
			CashierID:     cashier.ID,
			Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Status:        "PENDING",
		})
		assert.Error(t, err)
	})
}

func TestSaleRepository_GetSaleByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: fmt.Sprintf("GET-INV-%d", time.Now().UnixNano()),
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PAID",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetSaleByID(ctx, sale.ID)
		require.NoError(t, err)
		assert.Equal(t, sale.ID, found.ID)
		assert.Equal(t, sale.InvoiceNumber, found.InvoiceNumber)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetSaleByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestSaleRepository_GetSaleByInvoiceNumber(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	invoiceNum := fmt.Sprintf("INV-SEARCH-%d", time.Now().UnixNano())
	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: invoiceNum,
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PAID",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetSaleByInvoiceNumber(ctx, invoiceNum)
		require.NoError(t, err)
		assert.Equal(t, sale.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetSaleByInvoiceNumber(ctx, "NONEXISTENT")
		assert.Error(t, err)
	})
}

func TestSaleRepository_UpdateSaleStatus(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: fmt.Sprintf("STATUS-INV-%d", time.Now().UnixNano()),
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PENDING",
	})
	require.NoError(t, err)

	t.Run("update to paid", func(t *testing.T) {
		err := repo.UpdateSaleStatus(ctx, db.UpdateSaleStatusParams{
			ID:     sale.ID,
			Status: "PAID",
		})
		require.NoError(t, err)

		updated, err := repo.GetSaleByID(ctx, sale.ID)
		require.NoError(t, err)
		assert.Equal(t, "PAID", updated.Status)
	})

	t.Run("update to cancelled", func(t *testing.T) {
		err := repo.UpdateSaleStatus(ctx, db.UpdateSaleStatusParams{
			ID:     sale.ID,
			Status: "CANCELLED",
		})
		require.NoError(t, err)

		updated, err := repo.GetSaleByID(ctx, sale.ID)
		require.NoError(t, err)
		assert.Equal(t, "CANCELLED", updated.Status)
	})
}

func TestSaleRepository_UpdateSale(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: fmt.Sprintf("UPDATE-INV-%d", time.Now().UnixNano()),
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PENDING",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		updated, err := repo.UpdateSale(ctx, db.UpdateSaleParams{
			ID:         sale.ID,
			Subtotal:   pgtype.Numeric{Int: big.NewInt(50000), Valid: true},
			Discount:   pgtype.Numeric{Int: big.NewInt(2500), Valid: true},
			Tax:        pgtype.Numeric{Int: big.NewInt(4750), Valid: true},
			GrandTotal: pgtype.Numeric{Int: big.NewInt(52250), Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, sale.ID, updated.ID)
	})
}

func TestSaleRepository_DeleteSale(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: fmt.Sprintf("DELETE-INV-%d", time.Now().UnixNano()),
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PENDING",
	})
	require.NoError(t, err)

	err = repo.DeleteSale(ctx, sale.ID)
	require.NoError(t, err)

	_, err = repo.GetSaleByID(ctx, sale.ID)
	assert.Error(t, err)
}

func TestSaleRepository_CreateSaleItem(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)
	product := createTestProductForSale(t, queries)
	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
			InvoiceNumber: fmt.Sprintf("ITEM-INV-%d", time.Now().UnixNano()),
			StoreID:       store.ID,
			CashierID:     cashier.ID,
			Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Status:        "PENDING",
		})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		item, err := repo.CreateSaleItem(ctx, db.CreateSaleItemParams{
			SaleID:    sale.ID,
			ProductID: product.ID,
			Quantity:  2,
			UnitPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
			Discount:  pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			Subtotal:  pgtype.Numeric{Int: big.NewInt(20000), Valid: true},
		})
		require.NoError(t, err)
		assert.NotZero(t, item.ID)
		assert.Equal(t, sale.ID, item.SaleID)
		assert.Equal(t, product.ID, item.ProductID)
		assert.Equal(t, int32(2), item.Quantity)
	})

	t.Run("get sale items", func(t *testing.T) {
		items, err := repo.GetSaleItemsBySaleID(ctx, sale.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 1)
	})
}

func TestSaleRepository_DeleteSaleItemsBySaleID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)
	product := createTestProductForSale(t, queries)

	sale, err := repo.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: fmt.Sprintf("DELITEM-INV-%d", time.Now().UnixNano()),
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PENDING",
	})
	require.NoError(t, err)

	// Create sale item
	_, err = repo.CreateSaleItem(ctx, db.CreateSaleItemParams{
		SaleID:    sale.ID,
		ProductID: product.ID,
		Quantity:  1,
		UnitPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Discount:  pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Subtotal:  pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
	})
	require.NoError(t, err)

	// Delete sale items
	err = repo.DeleteSaleItemsBySaleID(ctx, sale.ID)
	require.NoError(t, err)

	// Verify items are deleted
	items, err := repo.GetSaleItemsBySaleID(ctx, sale.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestSaleRepository_ListSales(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	store := createTestStore(t, queries)
	cashier := createTestUserForSale(t, queries, store.ID)

	// Create multiple sales
	for i := 0; i < 5; i++ {
		_, err := repo.CreateSale(ctx, db.CreateSaleParams{
			InvoiceNumber: fmt.Sprintf("LIST-INV-%d-%d", time.Now().UnixNano(), i),
			StoreID:       store.ID,
			CashierID:     cashier.ID,
			Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			Status:        "PAID",
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		sales, err := repo.ListSales(ctx, db.ListSalesParams{
			Column1: 0,
			Column2: 0,
			Column3: "",
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column5: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(sales), 5)
	})

	t.Run("filter by status", func(t *testing.T) {
		sales, err := repo.ListSales(ctx, db.ListSalesParams{
			Column1: 0,
			Column2: 0,
			Column3: "PAID",
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column5: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(sales), 1)
		for _, s := range sales {
			assert.Equal(t, "PAID", s.Status)
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		sales, err := repo.ListSales(ctx, db.ListSalesParams{
			Column1: 0,
			Column2: 0,
			Column3: "",
			Column4: time.Time{},
			Column5: time.Time{},
			Limit:   2,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(sales), 2)
	})
}

func TestSaleRepository_CountSales(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewSaleRepository(queries)
	ctx := context.Background()

	count, err := repo.CountSales(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
