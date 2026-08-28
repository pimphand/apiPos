package inventory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// mockInventoryRepo implements a minimal mock for inventory repository
type mockInventoryRepo struct {
	createFunc func(ctx context.Context, txn *gormdb.InventoryTransaction) error
}

func (m *mockInventoryRepo) CreateTransaction(ctx context.Context, txn *gormdb.InventoryTransaction) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, txn)
	}
	return nil
}

func (m *mockInventoryRepo) GetTransactionByID(ctx context.Context, id string) (*gormdb.InventoryTransaction, error) {
	return nil, nil
}

func (m *mockInventoryRepo) ListTransactions(ctx context.Context, params repository.InventoryListParams) ([]gormdb.InventoryTransaction, int64, error) {
	return nil, 0, nil
}

func (m *mockInventoryRepo) ListTransactionsByProduct(ctx context.Context, productID string, params repository.PaginationParams) ([]gormdb.InventoryTransaction, int64, error) {
	return nil, 0, nil
}

// mockProductRepo implements a minimal mock for product repository
type mockProductRepo struct {
	createFunc func(ctx context.Context, product *gormdb.Product) error
}

func (m *mockProductRepo) Create(ctx context.Context, product *gormdb.Product) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, product)
	}
	return nil
}

func (m *mockProductRepo) GetByID(ctx context.Context, id string) (*gormdb.Product, error) {
	return &gormdb.Product{
		ID:    uuid.New(),
		Name:  "Test Product",
		Stock: 100,
	}, nil
}

func (m *mockProductRepo) List(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error) {
	return nil, 0, nil
}

func (m *mockProductRepo) SearchByName(ctx context.Context, params repository.ProductSearchParams) ([]gormdb.Product, int64, error) {
	return nil, 0, nil
}

func (m *mockProductRepo) Update(ctx context.Context, product *gormdb.Product) error {
	return nil
}

func (m *mockProductRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockProductRepo) Activate(ctx context.Context, id string) error {
	return nil
}

func (m *mockProductRepo) Deactivate(ctx context.Context, id string) error {
	return nil
}

func (m *mockProductRepo) GetLowStock(ctx context.Context, params repository.PaginationParams) ([]gormdb.Product, int64, error) {
	return nil, 0, nil
}

func TestInventoryService_CreateTransaction(t *testing.T) {
	ctx := context.Background()
	validProductID := uuid.New().String()
	validStoreID := uuid.New().String()
	validUserID := uuid.New().String()

	t.Run("invalid product id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreateTransaction(ctx, CreateTransactionRequest{
			ProductID: "invalid",
			StoreID:   validStoreID,
			Type:      TypePurchase,
			Quantity:  10,
			CreatedBy: validUserID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid product ID")
	})

	t.Run("invalid transaction type", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreateTransaction(ctx, CreateTransactionRequest{
			ProductID: validProductID,
			StoreID:   validStoreID,
			Type:      "INVALID",
			Quantity:  10,
			CreatedBy: validUserID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid transaction type")
	})

	t.Run("zero quantity", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreateTransaction(ctx, CreateTransactionRequest{
			ProductID: validProductID,
			StoreID:   validStoreID,
			Type:      TypePurchase,
			Quantity:  0,
			CreatedBy: validUserID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be greater than 0")
	})
}

func TestInventoryService_AdjustStock(t *testing.T) {
	ctx := context.Background()
	validProductID := uuid.New().String()
	validStoreID := uuid.New().String()
	validUserID := uuid.New().String()

	t.Run("invalid direction", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.AdjustStock(ctx, AdjustStockRequest{
			ProductID: validProductID,
			StoreID:   validStoreID,
			Direction: "INVALID",
			Quantity:  10,
			CreatedBy: validUserID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Direction must be INCREASE or DECREASE")
	})

	t.Run("zero quantity", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.AdjustStock(ctx, AdjustStockRequest{
			ProductID: validProductID,
			StoreID:   validStoreID,
			Direction: "INCREASE",
			Quantity:  0,
			CreatedBy: validUserID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be greater than 0")
	})
}

func TestInventoryService_GetTransaction(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetTransaction(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid transaction ID")
	})
}

func TestInventoryService_ListTransactions(t *testing.T) {
	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repos := &repository.Repositories{
			Inventory: &mockInventoryRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListTransactions(ctx, "", "", "", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		repos := &repository.Repositories{
			Inventory: &mockInventoryRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListTransactions(ctx, "", "", "", 1, 200)
		require.NoError(t, err)
		assert.Equal(t, int32(100), result.Pagination.PerPage)
	})
}

func TestInventoryService_GetStockSummary(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid product id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetStockSummary(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid product ID")
	})
}
