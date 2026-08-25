package repository

import (
	"context"

	db "apiPos/internal/database/sqlc"
)

// UserRepositoryImpl implements UserRepository.
type UserRepositoryImpl struct {
	queries *db.Queries
}

// NewUserRepository creates a new UserRepositoryImpl.
func NewUserRepository(queries *db.Queries) UserRepository {
	return &UserRepositoryImpl{queries: queries}
}

func (r *UserRepositoryImpl) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	return r.queries.CreateUser(ctx, arg)
}

func (r *UserRepositoryImpl) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *UserRepositoryImpl) GetUserByUsername(ctx context.Context, username string) (db.User, error) {
	return r.queries.GetUserByUsername(ctx, username)
}

func (r *UserRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *UserRepositoryImpl) ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.User, error) {
	return r.queries.ListUsers(ctx, arg)
}

func (r *UserRepositoryImpl) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
	return r.queries.UpdateUser(ctx, arg)
}

func (r *UserRepositoryImpl) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error {
	return r.queries.UpdateUserPassword(ctx, arg)
}

func (r *UserRepositoryImpl) UpdateUserLastLogin(ctx context.Context, id int64) error {
	return r.queries.UpdateUserLastLogin(ctx, id)
}

func (r *UserRepositoryImpl) DeleteUser(ctx context.Context, id int64) error {
	return r.queries.DeleteUser(ctx, id)
}

func (r *UserRepositoryImpl) DeactivateUser(ctx context.Context, id int64) error {
	return r.queries.DeactivateUser(ctx, id)
}

func (r *UserRepositoryImpl) CountUsers(ctx context.Context) (int64, error) {
	return r.queries.CountUsers(ctx)
}

func (r *UserRepositoryImpl) CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	return r.queries.CreateRefreshToken(ctx, arg)
}

func (r *UserRepositoryImpl) GetRefreshToken(ctx context.Context, token string) (db.RefreshToken, error) {
	return r.queries.GetRefreshToken(ctx, token)
}

func (r *UserRepositoryImpl) RevokeRefreshToken(ctx context.Context, token string) error {
	return r.queries.RevokeRefreshToken(ctx, token)
}

func (r *UserRepositoryImpl) RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error {
	return r.queries.RevokeAllUserRefreshTokens(ctx, userID)
}
