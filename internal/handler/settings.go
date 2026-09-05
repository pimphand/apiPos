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
	return &settingsv1.SettingsResponse{Settings: toProtoStoreSettings(s)}, nil
}

// UpdateSettings saves the current store's settings
func (h *SettingsHandler) UpdateSettings(ctx context.Context, req *settingsv1.UpdateSettingsRequest) (*settingsv1.SettingsResponse, error) {
	s, err := h.service.UpdateSettings(ctx, settings.UpdateParams{
		TaxRate:          req.TaxRate,
		ReceiptHeader:    req.ReceiptHeader,
		ReceiptFooter:    req.ReceiptFooter,
		ShowStoreName:    req.ShowStoreName,
		ShowStoreContact: req.ShowStoreContact,
		ShowCashier:      req.ShowCashier,
		ShowCustomer:     req.ShowCustomer,
		ShowItemSKU:      req.ShowItemSku,
		ShowItemBarcode:  req.ShowItemBarcode,
		ShowDiscount:     req.ShowDiscount,
		ShowTax:          req.ShowTax,
		ShowPayment:      req.ShowPayment,
		ShowChange:       req.ShowChange,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.LogJSON(ctx, "UPDATE", "settings", s.StoreID.String(), map[string]any{
		"tax_rate":       s.TaxRate,
		"receipt_header": s.ReceiptHeader,
		"receipt_footer": s.ReceiptFooter,
	})

	return &settingsv1.SettingsResponse{Settings: toProtoStoreSettings(s)}, nil
}

// toProtoStoreSettings converts a GORM StoreSetting model to proto StoreSettings
func toProtoStoreSettings(s *gormdb.StoreSetting) *settingsv1.StoreSettings {
	updatedBy := ""
	if s.UpdatedBy != nil {
		updatedBy = s.UpdatedBy.String()
	}
	updatedAt := ""
	if !s.UpdatedAt.IsZero() {
		updatedAt = s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return &settingsv1.StoreSettings{
		StoreId:          s.StoreID.String(),
		TaxRate:          s.TaxRate,
		ReceiptHeader:    s.ReceiptHeader,
		ReceiptFooter:    s.ReceiptFooter,
		ShowStoreName:    s.ShowStoreName,
		ShowStoreContact: s.ShowStoreContact,
		ShowCashier:      s.ShowCashier,
		ShowCustomer:     s.ShowCustomer,
		ShowItemSku:      s.ShowItemSKU,
		ShowItemBarcode:  s.ShowItemBarcode,
		ShowDiscount:     s.ShowDiscount,
		ShowTax:          s.ShowTax,
		ShowPayment:      s.ShowPayment,
		ShowChange:       s.ShowChange,
		UpdatedBy:        updatedBy,
		UpdatedAt:        updatedAt,
	}
}
