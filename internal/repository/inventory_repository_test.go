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

func createTestInventoryData(t *testing.T, queries *db.Queries) (db.Store, db.User, db.Product) {
	t.Helper()
	ctx := context.Background()

	store, err := queries.CreateStore(ctx, db.CreateStoreParams{
		Name:    fmt.Sprintf("InvStore_%d", time.Now().UnixNano()),
		Address: pgtype.Text{String: "123 Inv Street", Valid: true},
		Phone:   pgtype.Text{String: "1234567890", Valid: true},
	})
	require.NoError(t, err)

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name: fmt.Sprintf("invrole_%d", time.Now().UnixNano()),
	})
	require.NoError(t, err)

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("invuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("invuser_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Inv User",
		RoleID:       role.ID,
		StoreID:      store.ID,
	})
	require.NoError(t, err)

	product, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Sku:          fmt.Sprintf("INV_SKU_%d", time.Now().UnixNano()),
		Name:         "Inventory Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	return store, user, product
}

func TestInventoryRepository_CreateInventoryTransaction(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewInventoryRepository(queries)
	ctx := context.Background()

	store, user, product := createTestInventoryData(t, queries)

	t.Run("purchase transaction", func(t *testing.T) {
		tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "PURCHASE",
			Quantity:    50,
			BeforeStock: 100,
			AfterStock:  150,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
		assert.True(t, tx.ID.Valid)
		assert.Equal(t, product.ID, tx.ProductID)
		assert.Equal(t, store.ID, tx.StoreID)
		assert.Equal(t, "PURCHASE", tx.Type)
		assert.Equal(t, int32(50), tx.Quantity)
	})

	t.Run("sale transaction", func(t *testing.T) {
		tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "SALE",
			Quantity:    10,
			BeforeStock: 150,
			AfterStock:  140,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "SALE", tx.Type)
	})

	t.Run("return transaction", func(t *testing.T) {
		tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "RETURN",
			Quantity:    5,
			BeforeStock: 140,
			AfterStock:  145,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "RETURN", tx.Type)
	})

	t.Run("adjustment transaction", func(t *testing.T) {
		tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "ADJUSTMENT",
			Quantity:    3,
			BeforeStock: 145,
			AfterStock:  148,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "ADJUSTMENT", tx.Type)
	})

	t.Run("damage transaction", func(t *testing.T) {
		tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "DAMAGE",
			Quantity:    2,
			BeforeStock: 148,
			AfterStock:  146,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "DAMAGE", tx.Type)
	})

	t.Run("transfer transaction", func(t *testing.T) {
		tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "TRANSFER",
			Quantity:    10,
			BeforeStock: 146,
			AfterStock:  136,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "TRANSFER", tx.Type)
	})
}

func TestInventoryRepository_GetInventoryTransactionByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewInventoryRepository(queries)
	ctx := context.Background()

	store, user, product := createTestInventoryData(t, queries)

	tx, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
		ProductID:   product.ID,
		StoreID:     store.ID,
		Type:        "PURCHASE",
		Quantity:    50,
		BeforeStock: 100,
		AfterStock:  150,
		CreatedBy:   user.ID,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetInventoryTransactionByID(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, tx.ID, found.ID)
		assert.Equal(t, tx.ProductID, found.ProductID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetInventoryTransactionByID(ctx, zeroUUID())
		assert.Error(t, err)
	})
}

func TestInventoryRepository_ListInventoryTransactions(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewInventoryRepository(queries)
	ctx := context.Background()

	store, user, product := createTestInventoryData(t, queries)

	for i := 0; i < 5; i++ {
		_, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "PURCHASE",
			Quantity:    10,
			BeforeStock: 100,
			AfterStock:  110,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		transactions, err := repo.ListInventoryTransactions(ctx, db.ListInventoryTransactionsParams{
			Column1: zeroUUID(),
			Column2: zeroUUID(),
			Column3: "",
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column5: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(transactions), 5)
	})

	t.Run("filter by type", func(t *testing.T) {
		transactions, err := repo.ListInventoryTransactions(ctx, db.ListInventoryTransactionsParams{
			Column1: zeroUUID(),
			Column2: zeroUUID(),
			Column3: "PURCHASE",
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column5: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(transactions), 1)
	})

	t.Run("filter by product", func(t *testing.T) {
		transactions, err := repo.ListInventoryTransactions(ctx, db.ListInventoryTransactionsParams{
			Column1: product.ID,
			Column2: zeroUUID(),
			Column3: "",
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column5: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(transactions), 1)
	})
}

func TestInventoryRepository_GetInventoryTransactionsByProduct(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewInventoryRepository(queries)
	ctx := context.Background()

	store, user, product := createTestInventoryData(t, queries)

	for i := 0; i < 5; i++ {
		_, err := repo.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ProductID:   product.ID,
			StoreID:     store.ID,
			Type:        "PURCHASE",
			Quantity:    10,
			BeforeStock: 100,
			AfterStock:  110,
			CreatedBy:   user.ID,
		})
		require.NoError(t, err)
	}

	transactions, err := repo.GetInventoryTransactionsByProduct(ctx, db.GetInventoryTransactionsByProductParams{
		ProductID: product.ID,
		Limit:     10,
		Offset:    0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(transactions), 5)

	for _, tx := range transactions {
		assert.Equal(t, product.ID, tx.ProductID)
	}
}
