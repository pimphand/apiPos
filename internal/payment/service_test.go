package payment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// mockPaymentRepo implements a minimal mock for payment repository
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

func TestPaymentService_CreatePayment(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid sale id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreatePayment(ctx, CreatePaymentRequest{
			SaleID:        "invalid",
			PaymentMethod: PaymentMethodCash,
			Amount:        10000,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sale ID")
	})

	t.Run("zero amount", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreatePayment(ctx, CreatePaymentRequest{
			SaleID:        "550e8400-e29b-41d4-a716-446655440000",
			PaymentMethod: PaymentMethodCash,
			Amount:        0,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Amount must be greater than 0")
	})

	t.Run("invalid payment method", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreatePayment(ctx, CreatePaymentRequest{
			SaleID:        "550e8400-e29b-41d4-a716-446655440000",
			PaymentMethod: "INVALID",
			Amount:        10000,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid payment method")
	})
}

func TestPaymentService_GetPayment(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetPayment(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid payment ID")
	})
}

func TestPaymentService_GetPaymentsBySale(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid sale id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetPaymentsBySale(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sale ID")
	})
}

func TestPaymentService_ListPayments(t *testing.T) {
	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repos := &repository.Repositories{
			Payment: &mockPaymentRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListPayments(ctx, "", "", 0, 0)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		repos := &repository.Repositories{
			Payment: &mockPaymentRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListPayments(ctx, "", "", 1, 200)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), result.Pagination.PerPage)
	})
}

func TestPaymentService_DeletePayment(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		err := svc.DeletePayment(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid payment ID")
	})
}

func TestPaymentService_GetPaymentSummary(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid sale id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetPaymentSummary(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sale ID")
	})
}
