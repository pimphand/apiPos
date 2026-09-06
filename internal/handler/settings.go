package handler

import (
	"context"

	settingsv1 "apiPos/gen/settings/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/settings"
)

// SettingsHandler implements the SettingsService gRPC server
type SettingsHandler struct {
	settingsv1.UnimplementedSettingsServiceServer
	service *settings.Service
	logs    *ActivityLogger
}

// NewSettingsHandler creates a new settings Handler
func NewSettingsHandler(service *settings.Service, logs *ActivityLogger) *SettingsHandler {
	return &SettingsHandler{service: service, logs: logs}
}

// GetSettings gets the current store's settings
func (h *SettingsHandler) GetSettings(ctx context.Context, _ *settingsv1.GetSettingsRequest) (*settingsv1.SettingsResponse, error) {
	s, err := h.service.GetSettings(ctx)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Store info (name/address/phone) is auxiliary for the page and receipt;
	// include it best-effort so the store name shows up on first load without
	// requiring a save first.
	info, infoErr := h.service.StoreInfo(ctx)
	resp := &settingsv1.SettingsResponse{Settings: toProtoStoreSettings(s)}
	if infoErr == nil && info != nil {
		resp.StoreInfo = toProtoStoreInfo(info)
	}
	return resp, nil
}

// UpdateSettings saves the current store's settings
func (h *SettingsHandler) UpdateSettings(ctx context.Context, req *settingsv1.UpdateSettingsRequest) (*settingsv1.SettingsResponse, error) {
	res, err := h.service.UpdateSettings(ctx, settings.UpdateParams{
		TaxRate:            req.TaxRate,
		ReceiptHeader:      req.ReceiptHeader,
		ReceiptFooter:      req.ReceiptFooter,
		ReceiptFooterAlign: req.ReceiptFooterAlign,
		ShowStoreName:      req.ShowStoreName,
		ShowStoreContact:   req.ShowStoreContact,
		ShowCashier:        req.ShowCashier,
		ShowCustomer:       req.ShowCustomer,
		ShowItemSKU:        req.ShowItemSku,
		ShowItemBarcode:    req.ShowItemBarcode,
		ShowDiscount:       req.ShowDiscount,
		ShowTax:            req.ShowTax,
		ShowPayment:        req.ShowPayment,        ShowChange:         req.ShowChange,
        ShowKitchenPrint:   req.ShowKitchenPrint,
    })
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.LogJSON(ctx, "UPDATE", "settings", res.Settings.StoreID.String(), map[string]any{
		"tax_rate":       res.Settings.TaxRate,
		"receipt_header": res.Settings.ReceiptHeader,
		"receipt_footer": res.Settings.ReceiptFooter,
	})

	resp := &settingsv1.SettingsResponse{Settings: toProtoStoreSettings(res.Settings)}
	if res.StoreInfo != nil {
		resp.StoreInfo = toProtoStoreInfo(res.StoreInfo)
	}
	return resp, nil
}

// UpdateStoreInfo saves the store's address & phone
func (h *SettingsHandler) UpdateStoreInfo(ctx context.Context, req *settingsv1.UpdateStoreInfoRequest) (*settingsv1.SettingsResponse, error) {
	store, err := h.service.UpdateStoreInfo(ctx, req.Address, req.Phone)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.LogJSON(ctx, "UPDATE", "settings", store.ID.String(), map[string]any{
		"store_address": store.Address,
		"store_phone":   store.Phone,
	})

	return &settingsv1.SettingsResponse{StoreInfo: toProtoStoreInfo(store)}, nil
}

// toProtoStoreInfo converts a GORM Store model to proto StoreInfo
func toProtoStoreInfo(s *gormdb.Store) *settingsv1.StoreInfo {
	return &settingsv1.StoreInfo{
		Name:    s.Name,
		Address: s.Address,
		Phone:   s.Phone,
	}
}

// toProtoStoreSettings converts a GORM StoreSetting model to proto StoreSettings
func toProtoStoreSettings(s *gormdb.StoreSetting) *settingsv1.StoreSettings {
	footerAlign := s.ReceiptFooterAlign
	if footerAlign == "" {
		footerAlign = "center"
	}
	updatedBy := ""
	if s.UpdatedBy != nil {
		updatedBy = s.UpdatedBy.String()
	}
	updatedAt := ""
	if !s.UpdatedAt.IsZero() {
		updatedAt = s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return &settingsv1.StoreSettings{
		StoreId:            s.StoreID.String(),
		TaxRate:            s.TaxRate,
		ReceiptHeader:      s.ReceiptHeader,
		ReceiptFooter:      s.ReceiptFooter,
		ReceiptFooterAlign: footerAlign,
		ShowStoreName:      s.ShowStoreName,
		ShowStoreContact:   s.ShowStoreContact,
		ShowCashier:        s.ShowCashier,
		ShowCustomer:       s.ShowCustomer,
		ShowItemSku:        s.ShowItemSKU,
		ShowItemBarcode:    s.ShowItemBarcode,
		ShowDiscount:       s.ShowDiscount,
		ShowTax:            s.ShowTax,
		ShowPayment:        s.ShowPayment,        ShowChange:         s.ShowChange,
        ShowKitchenPrint:   s.ShowKitchenPrint,
        UpdatedBy:          updatedBy,
		UpdatedAt:          updatedAt,
	}
}
