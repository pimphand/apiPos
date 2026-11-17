package handler

import (
	"context"
	"strings"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
)

// scopeStore resolves the store a request may act on from the JWT claims.
//
// requested is the store_id sent by the client (it may be empty). Non-superadmin
// callers are pinned to the store in their claims: an empty value resolves to
// that store and a different value is rejected with ErrForbidden. Superadmins
// operate across stores, so they may target another store when they send one.
//
// Handlers must always use the returned store for service calls and repository
// filters instead of the raw request field.
func scopeStore(ctx context.Context, requested string) (string, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil || claims.StoreID == "" {
		return "", apperror.ErrUnauthorized
	}

	requested = strings.TrimSpace(requested)
	switch {
	case requested == "":
		return claims.StoreID, nil
	case strings.EqualFold(requested, claims.StoreID):
		return claims.StoreID, nil
	case claims.RoleName == "superadmin":
		return requested, nil
	default:
		return "", apperror.ErrForbidden
	}
}
