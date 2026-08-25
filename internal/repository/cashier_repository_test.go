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

func createTestCashierData(t *testing.T, queries *db.Queries) (db.Store, db.User) {
	t.Helper()
	ctx := context.Background()

	store, err := queries.CreateStore(ctx, db.CreateStoreParams{
		Name:    fmt.Sprintf("CashStore_%d", time.Now().UnixNano()),
		Address: pgtype.Text{String: "123 Cash Street", Valid: true},
		Phone:   pgtype.Text{String: "1234567890", Valid: true},
	})
	require.NoError(t, err)

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name: fmt.Sprintf("cashrole_%d", time.Now().UnixNano()),
	})
	require.NoError(t, err)

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("cashuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("cashuser_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Cash User",
		RoleID:       role.ID,
		StoreID:      pgtype.Int8{Int64: store.ID, Valid: true},
	})
	require.NoError(t, err)

	return store, user
}

func TestCashierRepository_OpenCashierShift(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	t.Run("success", func(t *testing.T) {
		shift, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
			UserID:      user.ID,
			StoreID:     store.ID,
			OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		})
		require.NoError(t, err)
		assert.NotZero(t, shift.ID)
		assert.Equal(t, user.ID, shift.UserID)
		assert.Equal(t, store.ID, shift.StoreID)
		assert.Equal(t, "OPEN", shift.Status)
	})
}

func TestCashierRepository_GetCurrentShift(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	// Open a shift
	shift, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
		UserID:      user.ID,
		StoreID:     store.ID,
		OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		current, err := repo.GetCurrentShift(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, current.ID)
		assert.Equal(t, "OPEN", current.Status)
	})

	t.Run("no open shift", func(t *testing.T) {
		// Create another user with no shift
		role, err := queries.CreateRole(ctx, db.CreateRoleParams{
			Name: fmt.Sprintf("noshiftrole_%d", time.Now().UnixNano()),
		})
		require.NoError(t, err)

		user2, err := queries.CreateUser(ctx, db.CreateUserParams{
			Username:     fmt.Sprintf("noshiftuser_%d", time.Now().UnixNano()),
			Email:        fmt.Sprintf("noshiftuser_%d@example.com", time.Now().UnixNano()),
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "No Shift User",
			RoleID:       role.ID,
			StoreID:      pgtype.Int8{Int64: store.ID, Valid: true},
		})
		require.NoError(t, err)

		_, err = repo.GetCurrentShift(ctx, user2.ID)
		assert.Error(t, err)
	})
}

func TestCashierRepository_GetShiftByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	shift, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
		UserID:      user.ID,
		StoreID:     store.ID,
		OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetShiftByID(ctx, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, found.ID)
		assert.Equal(t, user.ID, found.UserID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetShiftByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestCashierRepository_CloseCashierShift(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	shift, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
		UserID:      user.ID,
		StoreID:     store.ID,
		OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		closed, err := repo.CloseCashierShift(ctx, db.CloseCashierShiftParams{
			ID:             shift.ID,
			ClosingCash:    pgtype.Numeric{Int: big.NewInt(500000), Valid: true},
			CashSales:      pgtype.Numeric{Int: big.NewInt(400000), Valid: true},
			CashRefund:     pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			ExpectedCash:   pgtype.Numeric{Int: big.NewInt(500000), Valid: true},
			ActualCash:     pgtype.Numeric{Int: big.NewInt(498000), Valid: true},
			CashDifference: pgtype.Numeric{Int: big.NewInt(-2000), Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, "CLOSED", closed.Status)
		assert.NotNil(t, closed.ClosedAt)
	})

	t.Run("cannot close already closed shift", func(t *testing.T) {
		_, err := repo.CloseCashierShift(ctx, db.CloseCashierShiftParams{
			ID:             shift.ID,
			ClosingCash:    pgtype.Numeric{Int: big.NewInt(500000), Valid: true},
			CashSales:      pgtype.Numeric{Int: big.NewInt(400000), Valid: true},
			CashRefund:     pgtype.Numeric{Int: big.NewInt(0), Valid: true},
			ExpectedCash:   pgtype.Numeric{Int: big.NewInt(500000), Valid: true},
			ActualCash:     pgtype.Numeric{Int: big.NewInt(500000), Valid: true},
			CashDifference: pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		})
		assert.Error(t, err)
	})
}

func TestCashierRepository_UpdateShiftCashSales(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	shift, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
		UserID:      user.ID,
		StoreID:     store.ID,
		OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
	})
	require.NoError(t, err)

	t.Run("add cash sales", func(t *testing.T) {
		err := repo.UpdateShiftCashSales(ctx, db.UpdateShiftCashSalesParams{
			ID:        shift.ID,
			CashSales: pgtype.Numeric{Int: big.NewInt(50000), Valid: true},
		})
		require.NoError(t, err)

		// Verify the shift was updated
		found, err := repo.GetShiftByID(ctx, shift.ID)
		require.NoError(t, err)
		assert.NotNil(t, found.CashSales)
	})
}

func TestCashierRepository_UpdateShiftCashRefund(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	shift, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
		UserID:      user.ID,
		StoreID:     store.ID,
		OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
	})
	require.NoError(t, err)

	t.Run("add cash refund", func(t *testing.T) {
		err := repo.UpdateShiftCashRefund(ctx, db.UpdateShiftCashRefundParams{
			ID:         shift.ID,
			CashRefund: pgtype.Numeric{Int: big.NewInt(10000), Valid: true},
		})
		require.NoError(t, err)

		// Verify the shift was updated
		found, err := repo.GetShiftByID(ctx, shift.ID)
		require.NoError(t, err)
		assert.NotNil(t, found.CashRefund)
	})
}

func TestCashierRepository_ListShifts(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	// Create multiple shifts
	for i := 0; i < 3; i++ {
		_, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
			UserID:      user.ID,
			StoreID:     store.ID,
			OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		shifts, err := repo.ListShifts(ctx, db.ListShiftsParams{
			Column1: 0,
			Column2: 0,
			Column3: "",
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(shifts), 3)
	})

	t.Run("filter by status", func(t *testing.T) {
		shifts, err := repo.ListShifts(ctx, db.ListShiftsParams{
			Column1: 0,
			Column2: 0,
			Column3: "OPEN",
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(shifts), 1)
		for _, s := range shifts {
			assert.Equal(t, "OPEN", s.Status)
		}
	})

	t.Run("filter by user", func(t *testing.T) {
		shifts, err := repo.ListShifts(ctx, db.ListShiftsParams{
			Column1: user.ID,
			Column2: 0,
			Column3: "",
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(shifts), 1)
	})

	t.Run("with pagination", func(t *testing.T) {
		shifts, err := repo.ListShifts(ctx, db.ListShiftsParams{
			Column1: 0,
			Column2: 0,
			Column3: "",
			Limit:   2,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(shifts), 2)
	})
}

func TestCashierRepository_CountOpenShifts(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCashierRepository(queries)
	ctx := context.Background()

	store, user := createTestCashierData(t, queries)

	// Open a shift
	_, err := repo.OpenCashierShift(ctx, db.OpenCashierShiftParams{
		UserID:      user.ID,
		StoreID:     store.ID,
		OpeningCash: pgtype.Numeric{Int: big.NewInt(100000), Valid: true},
	})
	require.NoError(t, err)

	count, err := repo.CountOpenShifts(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1))
}
