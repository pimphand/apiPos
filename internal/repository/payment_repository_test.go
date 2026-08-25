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

func createTestSaleForPayment(t *testing.T, queries *db.Queries) db.Sale {
	t.Helper()
	ctx := context.Background()

	store, err := queries.CreateStore(ctx, db.CreateStoreParams{
		Name:    fmt.Sprintf("PayStore_%d", time.Now().UnixNano()),
		Address: pgtype.Text{String: "123 Pay Street", Valid: true},
		Phone:   pgtype.Text{String: "1234567890", Valid: true},
	})
	require.NoError(t, err)

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name: fmt.Sprintf("payrole_%d", time.Now().UnixNano()),
	})
	require.NoError(t, err)

	cashier, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("paycashier_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("paycashier_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Pay Cashier",
		RoleID:       role.ID,
		StoreID:      store.ID,
	})
	require.NoError(t, err)

	sale, err := queries.CreateSale(ctx, db.CreateSaleParams{
		InvoiceNumber: fmt.Sprintf("PAY-INV-%d", time.Now().UnixNano()),
		StoreID:       store.ID,
		CashierID:     cashier.ID,
		Subtotal:      pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Discount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		Tax:           pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		GrandTotal:    pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		Status:        "PENDING",
	})
	require.NoError(t, err)

	return sale
}

func TestPaymentRepository_CreatePayment(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewPaymentRepository(queries)
	ctx := context.Background()

	sale := createTestSaleForPayment(t, queries)

	t.Run("cash payment", func(t *testing.T) {
		payment, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
			SaleID:        sale.ID,
			PaymentMethod: "CASH",
			Amount:        pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
			PaidAt:        time.Now(),
		})
		require.NoError(t, err)
		assert.True(t, payment.ID.Valid)
		assert.Equal(t, sale.ID, payment.SaleID)
		assert.Equal(t, "CASH", payment.PaymentMethod)
	})

	t.Run("card payment", func(t *testing.T) {
		payment, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
			SaleID:        sale.ID,
			PaymentMethod: "CARD",
			Amount:        pgtype.Numeric{Int: big.NewInt(50000), Valid: true},
			Reference:     pgtype.Text{String: "CARD-REF-123", Valid: true},
			PaidAt:        time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, "CARD", payment.PaymentMethod)
	})

	t.Run("ewallet payment", func(t *testing.T) {
		payment, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
			SaleID:        sale.ID,
			PaymentMethod: "EWALLET",
			Amount:        pgtype.Numeric{Int: big.NewInt(25000), Valid: true},
			Reference:     pgtype.Text{String: "EWALLET-TXN-456", Valid: true},
			PaidAt:        time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, "EWALLET", payment.PaymentMethod)
	})
}

func TestPaymentRepository_GetPaymentByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewPaymentRepository(queries)
	ctx := context.Background()

	sale := createTestSaleForPayment(t, queries)

	payment, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
		SaleID:        sale.ID,
		PaymentMethod: "CASH",
		Amount:        pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		PaidAt:        time.Now(),
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetPaymentByID(ctx, payment.ID)
		require.NoError(t, err)
		assert.Equal(t, payment.ID, found.ID)
		assert.Equal(t, sale.ID, found.SaleID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetPaymentByID(ctx, zeroUUID())
		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetPaymentsBySaleID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewPaymentRepository(queries)
	ctx := context.Background()

	sale := createTestSaleForPayment(t, queries)

	_, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
		SaleID:        sale.ID,
		PaymentMethod: "CASH",
		Amount:        pgtype.Numeric{Int: big.NewInt(60000), Valid: true},
		PaidAt:        time.Now(),
	})
	require.NoError(t, err)

	_, err = repo.CreatePayment(ctx, db.CreatePaymentParams{
		SaleID:        sale.ID,
		PaymentMethod: "CARD",
		Amount:        pgtype.Numeric{Int: big.NewInt(40000), Valid: true},
		PaidAt:        time.Now().Add(time.Second),
	})
	require.NoError(t, err)

	payments, err := repo.GetPaymentsBySaleID(ctx, sale.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(payments), 2)
}

func TestPaymentRepository_DeletePayment(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewPaymentRepository(queries)
	ctx := context.Background()

	sale := createTestSaleForPayment(t, queries)

	payment, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
		SaleID:        sale.ID,
		PaymentMethod: "CASH",
		Amount:        pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		PaidAt:        time.Now(),
	})
	require.NoError(t, err)

	err = repo.DeletePayment(ctx, payment.ID)
	require.NoError(t, err)

	_, err = repo.GetPaymentByID(ctx, payment.ID)
	assert.Error(t, err)
}

func TestPaymentRepository_GetTotalPaymentsBySaleID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewPaymentRepository(queries)
	ctx := context.Background()

	sale := createTestSaleForPayment(t, queries)

	t.Run("no payments", func(t *testing.T) {
		total, err := repo.GetTotalPaymentsBySaleID(ctx, sale.ID)
		require.NoError(t, err)
		assert.NotNil(t, total)
	})

	t.Run("with payments", func(t *testing.T) {
		_, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
			SaleID:        sale.ID,
			PaymentMethod: "CASH",
			Amount:        pgtype.Numeric{Int: big.NewInt(60000), Valid: true},
			PaidAt:        time.Now(),
		})
		require.NoError(t, err)

		total, err := repo.GetTotalPaymentsBySaleID(ctx, sale.ID)
		require.NoError(t, err)
		assert.NotNil(t, total)
	})
}

func TestPaymentRepository_ListPayments(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewPaymentRepository(queries)
	ctx := context.Background()

	sale := createTestSaleForPayment(t, queries)

	for i := 0; i < 5; i++ {
		_, err := repo.CreatePayment(ctx, db.CreatePaymentParams{
			SaleID:        sale.ID,
			PaymentMethod: "CASH",
			Amount:        pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
			PaidAt:        time.Now().Add(time.Duration(i) * time.Second),
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		payments, err := repo.ListPayments(ctx, db.ListPaymentsParams{
			Column1: zeroUUID(),
			Column2: "",
			Column3: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(payments), 5)
	})

	t.Run("filter by payment method", func(t *testing.T) {
		payments, err := repo.ListPayments(ctx, db.ListPaymentsParams{
			Column1: zeroUUID(),
			Column2: "CASH",
			Column3: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Column4: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(payments), 1)
	})
}
