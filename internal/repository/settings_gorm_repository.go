package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	gormdb "apiPos/internal/database/gorm"
)

// SettingsRepository defines the interface for per-store settings operations
type SettingsRepository interface {
	GetByStoreID(ctx context.Context, storeID string) (*gormdb.StoreSetting, error)
	// Upsert inserts the settings row or updates it when a row for the same
	// store already exists (store_id is the primary key).
	Upsert(ctx context.Context, setting *gormdb.StoreSetting) error
}

// SettingsRepo implements SettingsRepository using GORM
type SettingsRepo struct {
	*GormBase
}

// NewSettingsRepository creates a new SettingsRepo
func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &SettingsRepo{GormBase: NewGormBase(db)}
}

// GetByStoreID gets the settings row for a store
func (r *SettingsRepo) GetByStoreID(ctx context.Context, storeID string) (*gormdb.StoreSetting, error) {
	var setting gormdb.StoreSetting
	err := r.WithContext(ctx).Where("store_id = ?", storeID).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// Upsert inserts or updates the settings row for the store
func (r *SettingsRepo) Upsert(ctx context.Context, setting *gormdb.StoreSetting) error {
	cols := []string{
		"tax_rate",
		"receipt_header",
		"receipt_footer",
		"receipt_footer_align",
		"show_store_name",
		"show_store_contact",
		"show_cashier",
		"show_customer",
		"show_item_sku",
		"show_item_barcode",
		"show_discount",
		"show_tax",      "show_payment",
      "show_change",
      "show_kitchen_print",
      "updated_by",
		"updated_at",
	}
	return r.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "store_id"}},
		DoUpdates: clause.AssignmentColumns(cols),
	}).Create(setting).Error
}
