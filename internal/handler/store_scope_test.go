package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
)

const (
	storeA = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	storeB = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func ctxWithClaims(storeID, roleName string) context.Context {
	return auth.ContextWithClaims(context.Background(), &auth.Claims{
		UserID:   "11111111-1111-1111-1111-111111111111",
		RoleName: roleName,
		StoreID:  storeID,
	})
}

func TestScopeStoreRequiresClaims(t *testing.T) {
	_, err := scopeStore(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized)
}

func TestScopeStoreDefaultsToClaimStore(t *testing.T) {
	ctx := ctxWithClaims(storeA, "admin")

	storeID, err := scopeStore(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, storeA, storeID)
}

func TestScopeStoreAcceptsMatchingStore(t *testing.T) {
	ctx := ctxWithClaims(storeA, "admin")

	storeID, err := scopeStore(ctx, storeA)
	require.NoError(t, err)
	assert.Equal(t, storeA, storeID)
}

func TestScopeStoreRejectsOtherStore(t *testing.T) {
	ctx := ctxWithClaims(storeA, "admin")

	_, err := scopeStore(ctx, storeB)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestScopeStoreAllowsSuperadminToTargetOtherStore(t *testing.T) {
	ctx := ctxWithClaims(storeA, "superadmin")

	storeID, err := scopeStore(ctx, storeB)
	require.NoError(t, err)
	assert.Equal(t, storeB, storeID)
}
