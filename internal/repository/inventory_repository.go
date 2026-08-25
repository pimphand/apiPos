package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "apiPos/internal/database/sqlc"
)

// InventoryRepositoryImpl implements InventoryRepository.
type InventoryRepositoryImpl struct {
	queries *db.Queries
}

// NewInventoryRepository creates a new InventoryRepositoryImpl.
func NewInventoryRepository(queries *db.Queries) InventoryRepository {
	return &InventoryRepositoryImpl{queries: queries}
}

func (r *InventoryRepositoryImpl) CreateInventoryTransaction(ctx context.Context, arg db.CreateInventoryTransactionParams) (db.InventoryTransaction, error) {
	return r.queries.CreateInventoryTransaction(ctx, arg)
}

func (r *InventoryRepositoryImpl) GetInventoryTransactionByID(ctx context.Context, id pgtype.UUID) (db.InventoryTransaction, error) {
	return r.queries.GetInventoryTransactionByID(ctx, id)
}

func (r *InventoryRepositoryImpl) ListInventoryTransactions(ctx context.Context, arg db.ListInventoryTransactionsParams) ([]db.InventoryTransaction, error) {
	return r.queries.ListInventoryTransactions(ctx, arg)
}

func (r *InventoryRepositoryImpl) GetInventoryTransactionsByProduct(ctx context.Context, arg db.GetInventoryTransactionsByProductParams) ([]db.InventoryTransaction, error) {
	return r.queries.GetInventoryTransactionsByProduct(ctx, arg)
}
