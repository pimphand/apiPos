package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "apiPos/internal/database/sqlc"
)

func TestCustomerRepository_CreateCustomer(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		customer, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
			Code:    "CUST001",
			Name:    "John Doe",
			Email:   pgtype.Text{String: "john@example.com", Valid: true},
			Phone:   pgtype.Text{String: "1234567890", Valid: true},
			Address: pgtype.Text{String: "123 Main St", Valid: true},
		})
		require.NoError(t, err)
		assert.NotZero(t, customer.ID)
		assert.Equal(t, "CUST001", customer.Code)
		assert.Equal(t, "John Doe", customer.Name)
		assert.True(t, customer.IsActive)
	})

	t.Run("duplicate code", func(t *testing.T) {
		_, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
			Code: "CUST001",
			Name: "Duplicate Customer",
		})
		assert.Error(t, err)
	})
}

func TestCustomerRepository_GetCustomerByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	customer, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "GET001",
		Name: "Get Customer",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetCustomerByID(ctx, customer.ID)
		require.NoError(t, err)
		assert.Equal(t, customer.ID, found.ID)
		assert.Equal(t, "GET001", found.Code)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCustomerByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestCustomerRepository_GetCustomerByCode(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	customer, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "CODE001",
		Name: "Code Customer",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetCustomerByCode(ctx, "CODE001")
		require.NoError(t, err)
		assert.Equal(t, customer.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCustomerByCode(ctx, "NONEXISTENT")
		assert.Error(t, err)
	})
}

func TestCustomerRepository_UpdateCustomer(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	customer, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "UPD001",
		Name: "Update Customer",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		updated, err := repo.UpdateCustomer(ctx, db.UpdateCustomerParams{
			ID:    customer.ID,
			Name:  "Updated Customer",
			Email: pgtype.Text{String: "updated@example.com", Valid: true},
			Phone: pgtype.Text{String: "0987654321", Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Customer", updated.Name)
		assert.Equal(t, "updated@example.com", updated.Email.String)
	})
}

func TestCustomerRepository_DeleteCustomer(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	customer, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "DEL001",
		Name: "Delete Customer",
	})
	require.NoError(t, err)

	err = repo.DeleteCustomer(ctx, customer.ID)
	require.NoError(t, err)

	_, err = repo.GetCustomerByID(ctx, customer.ID)
	assert.Error(t, err)
}

func TestCustomerRepository_DeactivateCustomer(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	customer, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "DEC001",
		Name: "Deactivate Customer",
	})
	require.NoError(t, err)

	err = repo.DeactivateCustomer(ctx, customer.ID)
	require.NoError(t, err)

	found, err := repo.GetCustomerByID(ctx, customer.ID)
	require.NoError(t, err)
	assert.False(t, found.IsActive)
}

func TestCustomerRepository_ListCustomers(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	// Create multiple customers
	for i := 0; i < 5; i++ {
		_, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
			Code: fmt.Sprintf("LIST%03d", i),
			Name: fmt.Sprintf("Customer %d", i),
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		customers, err := repo.ListCustomers(ctx, db.ListCustomersParams{
			Column1: false,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(customers), 5)
	})
}

func TestCustomerRepository_SearchCustomers(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	// Create test customers
	_, err := repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "SRC001",
		Name: "Alice Smith",
		Phone: pgtype.Text{String: "1112223333", Valid: true},
	})
	require.NoError(t, err)

	_, err = repo.CreateCustomer(ctx, db.CreateCustomerParams{
		Code: "SRC002",
		Name: "Bob Johnson",
		Phone: pgtype.Text{String: "4445556666", Valid: true},
	})
	require.NoError(t, err)

	t.Run("search by name", func(t *testing.T) {
		customers, err := repo.SearchCustomers(ctx, db.SearchCustomersParams{
			Column1: pgtype.Text{String: "Alice", Valid: true},
			Column2: false,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(customers), 1)
	})

	t.Run("search by phone", func(t *testing.T) {
		customers, err := repo.SearchCustomers(ctx, db.SearchCustomersParams{
			Column1: pgtype.Text{String: "444", Valid: true},
			Column2: false,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(customers), 1)
	})
}

func TestCustomerRepository_CountCustomers(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCustomerRepository(queries)
	ctx := context.Background()

	count, err := repo.CountCustomers(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
