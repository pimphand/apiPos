package settings

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// DefaultTaxRate is applied when a store has not customized its settings yet.
// It matches the historical POS default so existing behaviour is unchanged.
const DefaultTaxRate = 11.00

// UpdateParams carries the full desired settings state (upsert semantics).
type UpdateParams struct {
	TaxRate          float64
	ReceiptHeader    string
	ReceiptFooter    string
	ShowStoreName    bool
	ShowStoreContact bool
	ShowCashier      bool
	ShowCustomer     bool
	ShowItemSKU      bool
	ShowItemBarcode  bool
	ShowDiscount     bool
	ShowTax          bool
	ShowPayment      bool
	ShowChange       bool
}

// Service handles per-store settings business logic.
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new settings Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// SettingsStoreID returns the tenant scope from the authenticated user, or an
// error when the user is not bound to a store (e.g. superadmin).
func (s *Service) SettingsStoreID(ctx context.Context) (string, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return "", apperror.Wrap(apperror.ErrUnauthorized, "Authentication required", nil)
	}
	if claims.StoreID == "" {
		return "", apperror.Wrap(apperror.ErrNotFound, "Store settings not found for this account", nil)
	}
	if _, err := uuid.Parse(claims.StoreID); err != nil {
		return "", apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store on account", err)
	}
	return claims.StoreID, nil
}

// GetSettings returns the settings for the authenticated user's store,
// falling back to defaults when the store has not saved settings yet.
func (s *Service) GetSettings(ctx context.Context) (*gormdb.StoreSetting, error) {
	storeID, err := s.SettingsStoreID(ctx)
	if err != nil {
		return nil, err
	}

	setting, err := s.repos.Settings.GetByStoreID(ctx, storeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultSetting(storeID), nil
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get store settings", err)
	}
	return setting, nil
}

// UpdateSettings validates and upserts settings for the authenticated user's
// store. Only admin & manager roles may change settings.
func (s *Service) UpdateSettings(ctx context.Context, params UpdateParams) (*gormdb.StoreSetting, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return nil, apperror.Wrap(apperror.ErrUnauthorized, "Authentication required", nil)
	}
	if claims.StoreID == "" {
		return nil, apperror.Wrap(apperror.ErrForbidden, "Account is not bound to a store", nil)
	}
	if claims.RoleName != "admin" && claims.RoleName != "manager" {
		return nil, apperror.Wrap(apperror.ErrForbidden, "Only admin or manager can change store settings", nil)
	}

	if params.TaxRate < 0 || params.TaxRate > 100 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Tax rate must be between 0 and 100", nil)
	}

	storeID, err := s.SettingsStoreID(ctx)
	if err != nil {
		return nil, err
	}

	now := gormdb.StoreSetting{
		StoreID:          uuid.MustParse(storeID),
		TaxRate:          params.TaxRate,
		ReceiptHeader:    params.ReceiptHeader,
		ReceiptFooter:    params.ReceiptFooter,
		ShowStoreName:    params.ShowStoreName,
		ShowStoreContact: params.ShowStoreContact,
		ShowCashier:      params.ShowCashier,
		ShowCustomer:     params.ShowCustomer,
		ShowItemSKU:      params.ShowItemSKU,
		ShowItemBarcode:  params.ShowItemBarcode,
		ShowDiscount:     params.ShowDiscount,
		ShowTax:          params.ShowTax,
		ShowPayment:      params.ShowPayment,
		ShowChange:       params.ShowChange,
	}

	if uid, err := uuid.Parse(claims.UserID); err == nil {
		now.UpdatedBy = &uid
	}

	if err := s.repos.Settings.Upsert(ctx, &now); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to save store settings", err)
	}

	saved, err := s.repos.Settings.GetByStoreID(ctx, claims.StoreID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to save store settings", err)
	}
	return saved, nil
}

// defaultSetting returns a settings row carrying the documented defaults.
func defaultSetting(storeID string) *gormdb.StoreSetting {
	id, _ := uuid.Parse(storeID)
	return &gormdb.StoreSetting{
		StoreID:          id,
		TaxRate:          DefaultTaxRate,
		ShowStoreName:    true,
		ShowStoreContact: true,
		ShowCashier:      true,
		ShowCustomer:     true,
		ShowItemSKU:      false,
		ShowItemBarcode:  false,
		ShowDiscount:     true,
		ShowTax:          true,
		ShowPayment:      true,
		ShowChange:       true,
	}
}
