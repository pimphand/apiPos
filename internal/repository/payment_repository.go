package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "apiPos/internal/database/sqlc"
)

// PaymentRepositoryImpl implements PaymentRepository.
type PaymentRepositoryImpl struct {
	queries *db.Queries
}

// NewPaymentRepository creates a new PaymentRepositoryImpl.
func NewPaymentRepository(queries *db.Queries) PaymentRepository {
	return &PaymentRepositoryImpl{queries: queries}
}

func (r *PaymentRepositoryImpl) CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.Payment, error) {
	return r.queries.CreatePayment(ctx, arg)
}

func (r *PaymentRepositoryImpl) GetPaymentByID(ctx context.Context, id pgtype.UUID) (db.Payment, error) {
	return r.queries.GetPaymentByID(ctx, id)
}

func (r *PaymentRepositoryImpl) GetPaymentsBySaleID(ctx context.Context, saleID pgtype.UUID) ([]db.Payment, error) {
	return r.queries.GetPaymentsBySaleID(ctx, saleID)
}

func (r *PaymentRepositoryImpl) ListPayments(ctx context.Context, arg db.ListPaymentsParams) ([]db.Payment, error) {
	return r.queries.ListPayments(ctx, arg)
}

func (r *PaymentRepositoryImpl) DeletePayment(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeletePayment(ctx, id)
}

func (r *PaymentRepositoryImpl) GetTotalPaymentsBySaleID(ctx context.Context, saleID pgtype.UUID) (interface{}, error) {
	return r.queries.GetTotalPaymentsBySaleID(ctx, saleID)
}
