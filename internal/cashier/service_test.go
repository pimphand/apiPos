package cashier

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

type mockCashierRepo struct{}

func (m *mockCashierRepo) OpenShift(ctx context.Context, shift *gormdb.CashierShift) error {
	return nil
}
func (m *mockCashierRepo) CloseShift(ctx context.Context, params repository.CloseShiftParams) (bool, error) {
	return true, nil
}
func (m *mockCashierRepo) GetCurrentShift(ctx context.Context, userID string) (*gormdb.CashierShift, error) {
	return nil, nil
}
func (m *mockCashierRepo) GetByID(ctx context.Context, id string) (*gormdb.CashierShift, error) {
	return nil, nil
}
func (m *mockCashierRepo) List(ctx context.Context, params repository.CashierListParams) ([]gormdb.CashierShift, int64, error) {
	return nil, 0, nil
}
func (m *mockCashierRepo) UpdateCashSales(ctx context.Context, id string, amount float64) error {
	return nil
}
func (m *mockCashierRepo) UpdateCashRefund(ctx context.Context, id string, amount float64) error {
	return nil
}
func (m *mockCashierRepo) CountOpenShifts(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockCashierRepo) ListShiftItems(ctx context.Context, params repository.ShiftItemListParams) ([]gormdb.ShiftItem, int64, error) {
	return nil, 0, nil
}

type mockPaymentRepo struct{}

func (m *mockPaymentRepo) Create(ctx context.Context, payment *gormdb.Payment) error {
	return nil
}
func (m *mockPaymentRepo) GetByID(ctx context.Context, id string) (*gormdb.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) GetBySaleID(ctx context.Context, saleID string) ([]gormdb.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) List(ctx context.Context, params repository.PaymentListParams) ([]gormdb.Payment, int64, error) {
	return nil, 0, nil
}
func (m *mockPaymentRepo) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockPaymentRepo) GetTotalBySaleID(ctx context.Context, saleID string) (float64, error) {
	return 0, nil
}

func TestCashierService_OpenShift(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid store id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.OpenShift(ctx, OpenShiftRequest{
			StoreID:     "invalid",
			OpeningCash: 100,
			UserID:      "550e8400-e29b-41d4-a716-446655440000",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid store ID")
	})

	t.Run("invalid user id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.OpenShift(ctx, OpenShiftRequest{
			StoreID:     "550e8400-e29b-41d4-a716-446655440000",
			OpeningCash: 100,
			UserID:      "invalid",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid user ID")
	})

	t.Run("negative opening cash", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.OpenShift(ctx, OpenShiftRequest{
			StoreID:     "550e8400-e29b-41d4-a716-446655440000",
			OpeningCash: -100,
			UserID:      "550e8400-e29b-41d4-a716-446655440000",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

func TestCashierService_CloseShift(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid shift id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CloseShift(ctx, CloseShiftRequest{
			ShiftID:     "invalid",
			ClosingCash: 500,
			ActualCash:  500,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid shift ID")
	})
}

func TestCashierService_GetCurrentShift(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid user id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetCurrentShift(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid user ID")
	})
}

func TestCashierService_GetShift(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetShift(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid shift ID")
	})
}

func TestCashierService_ListShifts(t *testing.T) {
	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repos := &repository.Repositories{
			Cashier: &mockCashierRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListShifts(ctx, "", "", "", 0, 0)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		repos := &repository.Repositories{
			Cashier: &mockCashierRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListShifts(ctx, "", "", "", 1, 200)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), result.Pagination.PerPage)
	})
}

func TestCashierService_GetShiftSummary(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetShiftSummary(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid shift ID")
	})
}

func TestValidateSaleMovementConsistency(t *testing.T) {
	t.Run("consistent rows produce no issues", func(t *testing.T) {
		rows := []saleMovementRow{
			{Invoice: "INV-1", LedgerOutQty: 2, InventoryQty: 2},
			{Invoice: "INV-2", LedgerOutQty: 5, InventoryQty: 5},
		}
		issues := validateSaleMovementConsistency(rows)
		assert.Empty(t, issues)
	})

	t.Run("mismatch is reported with invoice and quantities", func(t *testing.T) {
		rows := []saleMovementRow{
			{Invoice: "INV-A", LedgerOutQty: 3, InventoryQty: 3},
			{Invoice: "INV-B", LedgerOutQty: 1, InventoryQty: 4},
		}
		issues := validateSaleMovementConsistency(rows)
		assert.Len(t, issues, 1)
		assert.Contains(t, issues[0], "INV-B")
		assert.Contains(t, issues[0], "ledger OUT 1.000 != inventory SALE 4.000")
	})

	t.Run("missing ledger rows are a mismatch too", func(t *testing.T) {
		rows := []saleMovementRow{
			{Invoice: "INV-C", LedgerOutQty: 0, InventoryQty: 2},
		}
		issues := validateSaleMovementConsistency(rows)
		assert.Len(t, issues, 1)
		assert.Contains(t, issues[0], "INV-C")
	})

	t.Run("empty input is fine", func(t *testing.T) {
		assert.Empty(t, validateSaleMovementConsistency(nil))
	})
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, isValidUUID(uuid.New().String()))
	assert.False(t, isValidUUID("invalid"))
	assert.False(t, isValidUUID(""))
}

var _ = time.Now
