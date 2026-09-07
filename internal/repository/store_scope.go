package repository

import (
	"context"

	"gorm.io/gorm"

	"apiPos/internal/auth"
)

// tenantTables is the set of tenant-scoped tables that MUST be isolated by
// store_id. Any table not listed here (e.g. stores, roles, permissions,
// role_permissions, refresh_tokens) is global/system-level and never scoped.
var tenantTables = map[string]bool{
	"users":                  true,
	"categories":             true,
	"products":               true,
	"product_skus":           true,
	"customers":              true,
	"sales":                  true,
	"sale_items":             true,
	"payments":               true,
	"inventory_transactions": true,
	"cashier_shifts":         true,
	"activity_logs":          true,
	"tables":                 true,
}

// defaultStoreID is used as a fallback tenant scope when a non-superadmin
// token does not carry a store_id. All pre-existing rows were backfilled to
// this store, so this prevents lookups from leaking across tenants.
const DefaultStoreID = "00000000-0000-0000-0000-000000000001"

// defaultStoreID exists for backwards compatibility with existing callers that
// reference the unexported symbol. Prefer DefaultStoreID in new code.
const defaultStoreID = DefaultStoreID

// storeIDFromContext returns the tenant scope for the authenticated user, or
// "" for system-wide access (superadmin, or contexts without claims such as
// the public login endpoint).
func storeIDFromContext(ctx context.Context) string {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return ""
	}
	if claims.RoleName == "superadmin" {
		return ""
	}
	if claims.StoreID == "" {
		return defaultStoreID
	}
	return claims.StoreID
}

// isTenantTable reports whether the statement's target model is a tenant table.
func isTenantTable(db *gorm.DB) bool {
	if db.Statement.Schema == nil {
		dest := db.Statement.Dest
		if dest == nil {
			dest = db.Statement.Model
		}
		if dest != nil {
			_ = db.Statement.Parse(dest)
		}
	}
	return db.Statement.Schema != nil && tenantTables[db.Statement.Schema.Table]
}

// assignStoreIDOnCreate sets the store_id column on tenant records being
// created, derived from the authenticated user's tenant scope.
func assignStoreIDOnCreate(db *gorm.DB) {
	storeID := storeIDFromContext(db.Statement.Context)
	if storeID == "" || !isTenantTable(db) {
		return
	}
	db.Statement.SetColumn("store_id", storeID, true)
}

// scopeQuery injects a store_id filter into SELECT queries against tenant tables.
// Uses COALESCE so that rows with NULL store_id are still visible within the
// default tenant scope.
func scopeQuery(db *gorm.DB) {
	storeID := storeIDFromContext(db.Statement.Context)
	if storeID == "" || !isTenantTable(db) {
		return
	}
	tbl := db.Statement.Schema.Table
	db.Where(tbl+".store_id = ? OR "+tbl+".store_id IS NULL", storeID)
}

// scopeUpdate injects a store_id filter into UPDATE queries against tenant tables.
func scopeUpdate(db *gorm.DB) {
	storeID := storeIDFromContext(db.Statement.Context)
	if storeID == "" || !isTenantTable(db) {
		return
	}
	tbl := db.Statement.Schema.Table
	db.Where(tbl+".store_id = ? OR "+tbl+".store_id IS NULL", storeID)
}

// scopeDelete injects a store_id filter into DELETE queries against tenant tables.
func scopeDelete(db *gorm.DB) {
	storeID := storeIDFromContext(db.Statement.Context)
	if storeID == "" || !isTenantTable(db) {
		return
	}
	tbl := db.Statement.Schema.Table
	db.Where(tbl+".store_id = ? OR "+tbl+".store_id IS NULL", storeID)
}

// registerStoreScopeCallbacks installs the automatic store_id scoping and
// store_id assignment callbacks on the shared GORM connection.
func registerStoreScopeCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").
		Register("store_scope:assign_store_id", assignStoreIDOnCreate)
	db.Callback().Query().Before("gorm:query").
		Register("store_scope:scope_query", scopeQuery)
	db.Callback().Update().Before("gorm:update").
		Register("store_scope:scope_update", scopeUpdate)
	db.Callback().Delete().Before("gorm:delete").
		Register("store_scope:scope_delete", scopeDelete)
}
