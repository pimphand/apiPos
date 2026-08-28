package sale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// mockSaleRepo implements a minimal mock for sale repository
type mockSaleRepo struct{}

func (m *mockSaleRepo) Create(ctx context.Context, sale *gormdb.Sale) error {
	return nil
}

func (m *mockSaleRepo) GetByID(ctx context.Context, id string) (*gormdb.Sale, error) {
	return nil, nil
}

func (m *mockSaleRepo) GetByInvoiceNumber(ctx context.Context, invoiceNumber string) (*gormdb.Sale, error) {
	return nil, nil
}

func (m *mockSaleRepo) List(ctx context.Context, params repository.SaleListParams) ([]gormdb.Sale, int64, error) {
	return nil, 0, nil
}

func (m *mockSaleRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return nil
}

func (m *mockSaleRepo) Update(ctx context.Context, sale *gormdb.Sale) error {
	return nil
}

func (m *mockSaleRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockSaleRepo) Count(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockSaleRepo) CreateItem(ctx context.Context, item *gormdb.SaleItem) error {
	return nil
}

func (m *mockSaleRepo) GetItemsBySaleID(ctx context.Context, saleID string) ([]gormdb.SaleItem, error) {
	return nil, nil
}

func (m *mockSaleRepo) DeleteItemsBySaleID(ctx context.Context, saleID string) error {
	return nil
}

func TestSaleService_CreateSale(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid store id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreateSale(ctx, CreateSaleRequest{
			StoreID: "invalid",
			Items: []SaleItemInput{
				{ProductID: "test", Quantity: 1, UnitPrice: 10000},
			},
			CreatedBy: "test",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid store ID")
	})

	t.Run("empty items", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreateSale(ctx, CreateSaleRequest{
			StoreID:    "550e8400-e29b-41d4-a716-446655440000",
			Items:      []SaleItemInput{},
			CreatedBy:  "550e8400-e29b-41d4-a716-446655440000",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one item")
	})
}

func TestSaleService_GetSale(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetSale(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sale ID")
	})
}

func TestSaleService_GetSaleByInvoice(t *testing.T) {
	ctx := context.Background()

	t.Run("empty invoice", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetSaleByInvoice(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invoice number is required")
	})
}

func TestSaleService_ListSales(t *testing.T) {
	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repos := &repository.Repositories{
			Sale: &mockSaleRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListSales(ctx, repository.SaleListParams{
			Page:    0,
			PerPage: 0,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		repos := &repository.Repositories{
			Sale: &mockSaleRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListSales(ctx, repository.SaleListParams{
			Page:    1,
			PerPage: 200,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(100), result.Pagination.PerPage)
	})
}

func TestSaleService_CancelSale(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		err := svc.CancelSale(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sale ID")
	})
}

func TestSaleService_RefundSale(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		err := svc.RefundSale(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sale ID")
	})
}

func TestGenerateInvoiceNumber(t *testing.T) {
	invoice1 := generateInvoiceNumber()
	invoice2 := generateInvoiceNumber()

	assert.NotEmpty(t, invoice1)
	assert.NotEmpty(t, invoice2)
	assert.NotEqual(t, invoice1, invoice2)
	assert.Contains(t, invoice1, "INV-")
}
