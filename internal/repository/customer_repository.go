package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	db "apiPos/internal/database/sqlc"
)

// CustomerRepositoryImpl implements CustomerRepository.
type CustomerRepositoryImpl struct {
	queries *db.Queries
}

// NewCustomerRepository creates a new CustomerRepositoryImpl.
func NewCustomerRepository(queries *db.Queries) CustomerRepository {
	return &CustomerRepositoryImpl{queries: queries}
}

func (r *CustomerRepositoryImpl) CreateCustomer(ctx context.Context, arg db.CreateCustomerParams) (db.Customer, error) {
	return r.queries.CreateCustomer(ctx, arg)
}

func (r *CustomerRepositoryImpl) GetCustomerByID(ctx context.Context, id pgtype.UUID) (db.Customer, error) {
	return r.queries.GetCustomerByID(ctx, id)
}

func (r *CustomerRepositoryImpl) GetCustomerByCode(ctx context.Context, code string) (db.Customer, error) {
	return r.queries.GetCustomerByCode(ctx, code)
}

func (r *CustomerRepositoryImpl) ListCustomers(ctx context.Context, arg db.ListCustomersParams) ([]db.Customer, error) {
	return r.queries.ListCustomers(ctx, arg)
}

func (r *CustomerRepositoryImpl) SearchCustomers(ctx context.Context, arg db.SearchCustomersParams) ([]db.Customer, error) {
	return r.queries.SearchCustomers(ctx, arg)
}

func (r *CustomerRepositoryImpl) UpdateCustomer(ctx context.Context, arg db.UpdateCustomerParams) (db.Customer, error) {
	return r.queries.UpdateCustomer(ctx, arg)
}

func (r *CustomerRepositoryImpl) DeleteCustomer(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteCustomer(ctx, id)
}

func (r *CustomerRepositoryImpl) DeactivateCustomer(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeactivateCustomer(ctx, id)
}

func (r *CustomerRepositoryImpl) CountCustomers(ctx context.Context) (int64, error) {
	return r.queries.CountCustomers(ctx)
}

func (r *CustomerRepositoryImpl) GetCustomerPurchaseHistory(ctx context.Context, arg db.GetCustomerPurchaseHistoryParams) ([]db.GetCustomerPurchaseHistoryRow, error) {
	return r.queries.GetCustomerPurchaseHistory(ctx, arg)
}
