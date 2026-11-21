package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"apiPos/internal/apperror"
	"apiPos/internal/config"
	gormdb "apiPos/internal/database/gorm"
)

type sessionUser struct{ account gormdb.User }

func (u *sessionUser) GetByID(context.Context, string) (*gormdb.User, error) {
	return &u.account, nil
}

func TestAuthenticateRequiresCurrentAccountAndRole(t *testing.T) {
	roleID, storeID := uuid.New(), uuid.New()
	users := &sessionUser{account: gormdb.User{
		ID: uuid.New(), RoleID: roleID, StoreID: storeID, IsActive: true,
		Role: gormdb.Role{ID: roleID, Name: "cashier"},
	}}
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-only-key-for-session-validation", AccessTTL: time.Minute}}
	service := NewAuthService(cfg, users)
	token, _, err := service.GenerateAccessToken(users.account.ID.String(), "employee", "employee@example.test", roleID.String(), "cashier", storeID.String())
	require.NoError(t, err)
	_, err = service.Authenticate(t.Context(), token)
	require.NoError(t, err)

	users.account.IsActive = false
	_, err = service.Authenticate(t.Context(), token)
	require.ErrorIs(t, err, apperror.ErrInvalidToken)

	users.account.IsActive = true
	users.account.Role.Name = "manager"
	_, err = service.Authenticate(t.Context(), token)
	require.ErrorIs(t, err, apperror.ErrInvalidToken)

	users.account.Role.Name = "cashier"
	users.account.StoreID = uuid.New()
	_, err = service.Authenticate(t.Context(), token)
	require.ErrorIs(t, err, apperror.ErrInvalidToken)
}
