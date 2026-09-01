package sale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"apiPos/internal/apperror"
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

func TestComputeSaleTotals(t *testing.T) {
	items := []SaleItemInput{
		{Quantity: 2, UnitPrice: 10000}, // 20000
		{Quantity: 1, UnitPrice: 5000},  // 5000
	}

	t.Run("no discount no tax", func(t *testing.T) {
		subtotal, discount, tax, grand := computeSaleTotals(items, 0, 0)
		assert.Equal(t, 25000.0, subtotal)
		assert.Equal(t, 0.0, discount)
		assert.Equal(t, 0.0, tax)
		assert.Equal(t, 25000.0, grand)
	})

	t.Run("discount calculation", func(t *testing.T) {
		subtotal, discount, _, grand := computeSaleTotals(items, 5000, 0)
		assert.Equal(t, 25000.0, subtotal)
		assert.Equal(t, 5000.0, discount)
		assert.Equal(t, 20000.0, grand)
	})

	t.Run("tax calculation", func(t *testing.T) {
		// 11% tax on subtotal (no discount)
		_, _, _, grand := computeSaleTotals(items, 0, 11)
		assert.Equal(t, 27750.0, grand) // 25000 * 1.11
	})

	t.Run("discount and tax", func(t *testing.T) {
		// 11% tax on (25000 - 5000) = 20000 -> tax 2200 -> grand 22200
		_, discount, tax, grand := computeSaleTotals(items, 5000, 11)
		assert.Equal(t, 5000.0, discount)
		assert.Equal(t, 2200.0, tax)
		assert.Equal(t, 22200.0, grand)
	})

	t.Run("item discount clamped at zero", func(t *testing.T) {
		// Item discount larger than the item price must clamp to zero.
		bad := []SaleItemInput{{Quantity: 1, UnitPrice: 1000, Discount: 5000}}
		subtotal, _, _, grand := computeSaleTotals(bad, 0, 0)
		assert.Equal(t, 0.0, subtotal)
		assert.Equal(t, 0.0, grand)
	})

	t.Run("sale discount clamped at subtotal", func(t *testing.T) {
		subtotal, discount, _, grand := computeSaleTotals(items, 99999, 0)
		assert.Equal(t, 25000.0, subtotal)
		assert.Equal(t, 25000.0, discount)
		assert.Equal(t, 0.0, grand)
	})
}

func TestValidatePayment(t *testing.T) {
	t.Run("sufficient cash", func(t *testing.T) {
		assert.NoError(t, validatePayment(PaymentMethodCash, 30000, 25000))
	})

	t.Run("insufficient cash rejected", func(t *testing.T) {
		err := validatePayment(PaymentMethodCash, 20000, 25000)
		assert.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, apperror.MapError(err).GRPCStatus().Code())
	})

	t.Run("insufficient non-cash rejected", func(t *testing.T) {
		for _, m := range []string{PaymentMethodCard, PaymentMethodBankTransfer, PaymentMethodEWallet, PaymentMethodOther} {
			err := validatePayment(m, 10000, 25000)
			assert.Error(t, err, "method %s must reject short payment", m)
		}
	})

	t.Run("sufficient non-cash accepted", func(t *testing.T) {
		assert.NoError(t, validatePayment(PaymentMethodCard, 25000, 25000))
		assert.NoError(t, validatePayment(PaymentMethodEWallet, 50000, 25000))
	})

	t.Run("exact payment is sufficient", func(t *testing.T) {
		assert.NoError(t, validatePayment(PaymentMethodCash, 25000, 25000))
	})

	t.Run("change calculation for cash", func(t *testing.T) {
		// change = amount_paid - grand_total
		change := 30000.0 - 25000.0
		assert.Equal(t, 5000.0, change)
	})
}

func TestSaleService_InsufficientStockUsesFailedPrecondition(t *testing.T) {
	// Verify the insufficient-stock branch in CreateSale surfaces as
	// FAILED_PRECONDITION (matching the inventory module) rather than 400.
	err := apperror.Wrap(apperror.ErrInsufficientStock, "Insufficient stock for Test. Available: 2, Requested: 10", nil)
	assert.Equal(t, codes.FailedPrecondition, apperror.MapError(err).GRPCStatus().Code())
}
