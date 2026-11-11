package accounting

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
)

// EnsureCOAForStore creates the default chart of accounts for a store if missing.
// Idempotent: existing accounts (by code) are left untouched.
func EnsureCOAForStore(ctx context.Context, db *gorm.DB, storeID uuid.UUID) error {
	for _, acc := range DefaultCOA {
		var count int64
		if err := db.WithContext(ctx).Model(&gormdb.ChartOfAccount{}).
			Where("store_id = ? AND code = ?", storeID, acc.Code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		rec := gormdb.ChartOfAccount{
			StoreID: storeID, Code: acc.Code, Name: acc.Name,
			AccountType: acc.AccountType, NormalBalance: acc.NormalBalance,
			IsCashEquiv: acc.IsCashEquiv, IsActive: true,
		}
		if err := db.WithContext(ctx).Create(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

// EnsureCOAForAllStores backfills COA for every store (used by migration backfill
// and admin tooling). Idempotent.
func EnsureCOAForAllStores(ctx context.Context, db *gorm.DB) error {
	var stores []gormdb.Store
	if err := db.WithContext(ctx).Find(&stores).Error; err != nil {
		return err
	}
	for _, s := range stores {
		if err := EnsureCOAForStore(ctx, db, s.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetAccountID resolves the account id for a (store, code) pair.
func GetAccountID(ctx context.Context, db *gorm.DB, storeID uuid.UUID, code string) (uuid.UUID, error) {
	var acc gormdb.ChartOfAccount
	if err := db.WithContext(ctx).Where("store_id = ? AND code = ?", storeID, code).First(&acc).Error; err != nil {
		return uuid.Nil, apperror.Wrap(apperror.ErrInvalidRequest, "account not found for code "+code, err)
	}
	return acc.ID, nil
}
