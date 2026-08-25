package repository

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "apiPos/internal/database/sqlc"
)

func TestProductRepository_CreateProduct(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	// Create a category first
	category, err := queries.CreateCategory(ctx, db.CreateCategoryParams{
		Name:        "Test Category",
		Description: pgtype.Text{String: "Test category description", Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		product, err := repo.CreateProduct(ctx, db.CreateProductParams{
			Sku:          "SKU001",
			Barcode:      pgtype.Text{String: "123456789", Valid: true},
			Name:         "Test Product",
			Description:  pgtype.Text{String: "Test product description", Valid: true},
			CategoryID:   pgtype.Int8{Int64: category.ID, Valid: true},
			CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
			SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
			Stock:        100,
			MinimumStock: 10,
		})

		require.NoError(t, err)
		assert.NotZero(t, product.ID)
		assert.Equal(t, "SKU001", product.Sku)
		assert.Equal(t, "Test Product", product.Name)
		assert.Equal(t, int32(100), product.Stock)
		assert.True(t, product.IsActive)
	})

	t.Run("duplicate sku", func(t *testing.T) {
		_, err := repo.CreateProduct(ctx, db.CreateProductParams{
			Sku:          "SKU001",
			Name:         "Duplicate SKU Product",
			CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
			SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
			Stock:        50,
		})
		assert.Error(t, err)
	})
}

func TestProductRepository_GetProductByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	// Create product
	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "GET001",
		Name:         "Get Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
		assert.Equal(t, "GET001", found.Sku)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetProductByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestProductRepository_GetProductBySKU(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "SKU123",
		Name:         "SKU Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetProductBySKU(ctx, "SKU123")
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetProductBySKU(ctx, "NONEXISTENT")
		assert.Error(t, err)
	})
}

func TestProductRepository_GetProductByBarcode(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "BAR001",
		Barcode:      pgtype.Text{String: "1234567890", Valid: true},
		Name:         "Barcode Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetProductByBarcode(ctx, pgtype.Text{String: "1234567890", Valid: true})
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetProductByBarcode(ctx, pgtype.Text{String: "0000000000", Valid: true})
		assert.Error(t, err)
	})
}

func TestProductRepository_UpdateProduct(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "UPD001",
		Name:         "Update Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		updated, err := repo.UpdateProduct(ctx, db.UpdateProductParams{
			ID:           product.ID,
			Sku:          "UPD001-UPDATED",
			Name:         "Updated Product",
			CostPrice:    pgtype.Numeric{Int: big.NewInt(6000), Valid: true},
			SellingPrice: pgtype.Numeric{Int: big.NewInt(12000), Valid: true},
			MinimumStock: 15,
		})
		require.NoError(t, err)
		assert.Equal(t, "UPD001-UPDATED", updated.Sku)
		assert.Equal(t, "Updated Product", updated.Name)
	})
}

func TestProductRepository_UpdateProductStock(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "STK001",
		Name:         "Stock Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("increase stock", func(t *testing.T) {
		err := repo.UpdateProductStock(ctx, db.UpdateProductStockParams{
			ID:    product.ID,
			Stock: 150,
		})
		require.NoError(t, err)

		updated, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(150), updated.Stock)
	})
}

func TestProductRepository_DecreaseProductStock(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "DEC001",
		Name:         "Decrease Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("decrease stock", func(t *testing.T) {
		err := repo.DecreaseProductStock(ctx, db.DecreaseProductStockParams{
			ID:    product.ID,
			Stock: 30,
		})
		require.NoError(t, err)

		updated, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(70), updated.Stock)
	})

	t.	Run("insufficient stock", func(t *testing.T) {
		err := repo.DecreaseProductStock(ctx, db.DecreaseProductStockParams{
			ID:    product.ID,
			Stock: 200,
		})
		// Note: The SQL query uses WHERE stock >= $2, so it won't update any rows
		// but won't return an error. We need to check the stock wasn't decreased.
		assert.NoError(t, err)

		// Verify stock wasn't changed
		updated, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(70), updated.Stock)
	})
}

func TestProductRepository_IncreaseProductStock(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "INC001",
		Name:         "Increase Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        50,
	})
	require.NoError(t, err)

	t.Run("increase stock", func(t *testing.T) {
		err := repo.IncreaseProductStock(ctx, db.IncreaseProductStockParams{
			ID:    product.ID,
			Stock: 25,
		})
		require.NoError(t, err)

		updated, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(75), updated.Stock)
	})
}

func TestProductRepository_ActivateDeactivateProduct(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "ACT001",
		Name:         "Activate Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	t.Run("deactivate", func(t *testing.T) {
		err := repo.DeactivateProduct(ctx, product.ID)
		require.NoError(t, err)

		updated, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("activate", func(t *testing.T) {
		err := repo.ActivateProduct(ctx, product.ID)
		require.NoError(t, err)

		updated, err := repo.GetProductByID(ctx, product.ID)
		require.NoError(t, err)
		assert.True(t, updated.IsActive)
	})
}

func TestProductRepository_DeleteProduct(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	product, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "DEL001",
		Name:         "Delete Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        100,
	})
	require.NoError(t, err)

	err = repo.DeleteProduct(ctx, product.ID)
	require.NoError(t, err)

	_, err = repo.GetProductByID(ctx, product.ID)
	assert.Error(t, err)
}

func TestProductRepository_ListProducts(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	// Create multiple products
	for i := 0; i < 5; i++ {
		_, err := repo.CreateProduct(ctx, db.CreateProductParams{
			Sku:          fmt.Sprintf("LIST%03d", i),
			Name:         fmt.Sprintf("Product %d", i),
			CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
			SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
			Stock:        100,
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		products, err := repo.ListProducts(ctx, db.ListProductsParams{
			Column1: false,
			Column2: 0,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(products), 5)
	})

	t.Run("with pagination", func(t *testing.T) {
		products, err := repo.ListProducts(ctx, db.ListProductsParams{
			Column1: false,
			Column2: 0,
			Limit:   2,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(products), 2)
	})
}

func TestProductRepository_GetLowStockProducts(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	// Create a low stock product
	_, err := repo.CreateProduct(ctx, db.CreateProductParams{
		Sku:          "LOW001",
		Name:         "Low Stock Product",
		CostPrice:    pgtype.Numeric{Int: big.NewInt(5000), Valid: true},
		SellingPrice: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		Stock:        5,
		MinimumStock: 10,
	})
	require.NoError(t, err)

	products, err := repo.GetLowStockProducts(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(products), 1)

	// Verify the low stock product is in the list
	found := false
	for _, p := range products {
		if p.Sku == "LOW001" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestProductRepository_CountProducts(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewProductRepository(queries)
	ctx := context.Background()

	count, err := repo.CountProducts(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
