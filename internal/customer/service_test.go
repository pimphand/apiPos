package customer

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

type mockCustomerRepo struct{}

func (m *mockCustomerRepo) Create(ctx context.Context, customer *gormdb.Customer) error {
	return nil
}
func (m *mockCustomerRepo) GetByID(ctx context.Context, id string) (*gormdb.Customer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByCode(ctx context.Context, code string) (*gormdb.Customer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) List(ctx context.Context, params repository.CustomerListParams) ([]gormdb.Customer, int64, error) {
	return nil, 0, nil
}
func (m *mockCustomerRepo) Search(ctx context.Context, params repository.CustomerSearchParams) ([]gormdb.Customer, int64, error) {
	return nil, 0, nil
}
func (m *mockCustomerRepo) Update(ctx context.Context, customer *gormdb.Customer) error {
	return nil
}
func (m *mockCustomerRepo) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockCustomerRepo) Deactivate(ctx context.Context, id string) error {
	return nil
}
func (m *mockCustomerRepo) GetPurchaseHistory(ctx context.Context, customerID string, params repository.PaginationParams) ([]gormdb.Sale, int64, error) {
	return nil, 0, nil
}

func TestCustomerService_CreateCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("empty code auto-generates", func(t *testing.T) {
		repos := &repository.Repositories{
			Customer: &mockCustomerRepo{},
		}
		svc := NewService(repos)

		customer, err := svc.CreateCustomer(ctx, CreateCustomerRequest{
			Code: "",
			Name: "Test Customer",
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, customer.Code)
		assert.True(t, strings.HasPrefix(customer.Code, "CUS-"))
	})

	t.Run("empty name", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.CreateCustomer(ctx, CreateCustomerRequest{
			Code: "C001",
			Name: "",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Customer name is required")
	})
}

func TestCustomerService_GetCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, err := svc.GetCustomer(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid customer ID")
	})
}

func TestCustomerService_ListCustomers(t *testing.T) {
	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repos := &repository.Repositories{
			Customer: &mockCustomerRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListCustomers(ctx, "", 0, 0)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		repos := &repository.Repositories{
			Customer: &mockCustomerRepo{},
		}
		svc := NewService(repos)

		result, err := svc.ListCustomers(ctx, "", 1, 200)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), result.Pagination.PerPage)
	})
}

func TestCustomerService_SearchCustomers(t *testing.T) {
	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repos := &repository.Repositories{
			Customer: &mockCustomerRepo{},
		}
		svc := NewService(repos)

		result, err := svc.SearchCustomers(ctx, "test", 0, 0)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})
}

func TestCustomerService_DeleteCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		err := svc.DeleteCustomer(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid customer ID")
	})
}

func TestCustomerService_DeactivateCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		err := svc.DeactivateCustomer(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid customer ID")
	})
}

func TestCustomerService_GetPurchaseHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		repos := &repository.Repositories{}
		svc := NewService(repos)

		_, _, err := svc.GetPurchaseHistory(ctx, "invalid", 1, 20)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid customer ID")
	})
}
