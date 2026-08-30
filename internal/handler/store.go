package handler

import (
	"context"

	storev1 "apiPos/gen/store/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/store"
)

// StoreHandler implements the StoreService gRPC server (superadmin only).
type StoreHandler struct {
	storev1.UnimplementedStoreServiceServer
	service *store.Service
	logs    *ActivityLogger
}

// NewStoreHandler creates a new store Handler
func NewStoreHandler(service *store.Service, logs *ActivityLogger) *StoreHandler {
	return &StoreHandler{service: service, logs: logs}
}

// GetStore gets a store by ID
func (h *StoreHandler) GetStore(ctx context.Context, req *storev1.GetStoreRequest) (*storev1.StoreResponse, error) {
	s, err := h.service.GetStore(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return &storev1.StoreResponse{
		Store: toProtoStoreFromGORM(s),
	}, nil
}

// ListStores lists all stores with optional filters and pagination
func (h *StoreHandler) ListStores(ctx context.Context, req *storev1.ListStoresRequest) (*storev1.ListStoresResponse, error) {
	result, err := h.service.ListStores(ctx, req.IsVerified, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoStores := make([]*storev1.Store, len(result.Stores))
	for i, s := range result.Stores {
		protoStores[i] = toProtoStoreFromGORM(&s)
	}

	return &storev1.ListStoresResponse{
		Stores:     protoStores,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// VerifyStore verifies/approves a pending store
func (h *StoreHandler) VerifyStore(ctx context.Context, req *storev1.VerifyStoreRequest) (*storev1.StoreResponse, error) {
	s, err := h.service.VerifyStore(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "stores", s.ID.String(), `{"is_verified":true,"is_active":true}`)

	return &storev1.StoreResponse{
		Store: toProtoStoreFromGORM(s),
	}, nil
}

// toProtoStoreFromGORM converts a GORM Store model to proto Store
func toProtoStoreFromGORM(s *gormdb.Store) *storev1.Store {
	return &storev1.Store{
		Id:         s.ID.String(),
		Name:       s.Name,
		Address:    s.Address,
		Phone:      s.Phone,
		IsActive:   boolStr(s.IsActive),
		IsVerified: boolStr(s.IsVerified),
		CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
