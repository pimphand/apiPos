package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	"apiPos/internal/repository"
)

// ListUsers must never fall back to "all stores": without claims there is no
// tenant to scope to, so the call is rejected before any query runs.
func TestListUsersRequiresTenantScope(t *testing.T) {
	svc := NewService(&repository.Repositories{}, nil)

	t.Run("no claims", func(t *testing.T) {
		users, total, err := svc.ListUsers(context.Background(), "", "", 1, 20)
		require.Error(t, err)
		assert.ErrorIs(t, err, apperror.ErrUnauthorized)
		assert.Nil(t, users)
		assert.Zero(t, total)
	})

	t.Run("non-superadmin without store", func(t *testing.T) {
		ctx := auth.ContextWithClaims(context.Background(), &auth.Claims{
			UserID:   "11111111-1111-1111-1111-111111111111",
			RoleName: "admin",
		})
		_, _, err := svc.ListUsers(ctx, "", "", 1, 20)
		require.Error(t, err)
		assert.ErrorIs(t, err, apperror.ErrUnauthorized)
	})
}
