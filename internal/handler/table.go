package handler

import (
	"context"

	tablev1 "apiPos/gen/table/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/table"
)

// TableHandler implements the TableService gRPC server (store-scoped admin CRUD).
type TableHandler struct {
	tablev1.UnimplementedTableServiceServer
	service *table.Service
	logs    *ActivityLogger
}

// NewTableHandler creates a new table Handler
func NewTableHandler(service *table.Service, logs *ActivityLogger) *TableHandler {
	return &TableHandler{service: service, logs: logs}
}

// CreateTable creates a new table for the store.
func (h *TableHandler) CreateTable(ctx context.Context, req *tablev1.CreateTableRequest) (*tablev1.TableResponse, error) {
	t, err := h.service.CreateTable(ctx, req.Code, req.Name)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	h.logs.Log(ctx, "CREATE", "tables", t.ID.String(), `{"code":"`+t.Code+`"}`)
	return &tablev1.TableResponse{Table: toProtoTable(t)}, nil
}

// ListTables lists the store's tables.
func (h *TableHandler) ListTables(ctx context.Context, req *tablev1.ListTablesRequest) (*tablev1.ListTablesResponse, error) {
	tables, total, err := h.service.ListTables(ctx, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	out := make([]*tablev1.Table, 0, len(tables))
	for i := range tables {
		out = append(out, toProtoTable(&tables[i]))
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	perPage := req.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	return &tablev1.ListTablesResponse{
		Tables:     out,
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
	}, nil
}

// GetTable returns one table.
func (h *TableHandler) GetTable(ctx context.Context, req *tablev1.GetTableRequest) (*tablev1.TableResponse, error) {
	t, err := h.service.GetTable(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return &tablev1.TableResponse{Table: toProtoTable(t)}, nil
}

// UpdateTable updates a table's code/name/active state.
func (h *TableHandler) UpdateTable(ctx context.Context, req *tablev1.UpdateTableRequest) (*tablev1.TableResponse, error) {
	t, err := h.service.UpdateTable(ctx, req.Id, req.Code, req.Name, req.IsActive)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	h.logs.Log(ctx, "UPDATE", "tables", t.ID.String(), `{"code":"`+t.Code+`"}`)
	return &tablev1.TableResponse{Table: toProtoTable(t)}, nil
}

// DeleteTable deletes a table.
func (h *TableHandler) DeleteTable(ctx context.Context, req *tablev1.DeleteTableRequest) (*tablev1.DeleteTableResponse, error) {
	t, err := h.service.GetTable(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	if err := h.service.DeleteTable(ctx, req.Id); err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	h.logs.Log(ctx, "DELETE", "tables", req.Id, `{"code":"`+t.Code+`"}`)
	return &tablev1.DeleteTableResponse{Success: true, Message: "Table deleted"}, nil
}

// RotateTableToken issues a new QR token (old QR codes stop working).
func (h *TableHandler) RotateTableToken(ctx context.Context, req *tablev1.RotateTableTokenRequest) (*tablev1.TableResponse, error) {
	t, err := h.service.RotateTableToken(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	h.logs.Log(ctx, "UPDATE", "tables", t.ID.String(), `{"action":"rotate-token","code":"`+t.Code+`"}`)
	return &tablev1.TableResponse{Table: toProtoTable(t)}, nil
}

// toProtoTable converts a GORM Table to proto. The raw qr_token is never
// exposed; only the QR URL (relative path for rendering/printing the code).
func toProtoTable(t *gormdb.Table) *tablev1.Table {
	return &tablev1.Table{
		Id:        t.ID.String(),
		StoreId:   t.StoreID.String(),
		Code:      t.Code,
		Name:      t.Name,
		QrUrl:     "/o?table=" + t.QRToken,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}