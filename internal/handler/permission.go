package handler

import (
	"context"

	permissionv1 "apiPos/gen/permission/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/permission"
	"apiPos/internal/response"
)

// PermissionHandler implements the PermissionService gRPC server
type PermissionHandler struct {
	permissionv1.UnimplementedPermissionServiceServer
	service *permission.Service
	logs    *ActivityLogger
}

// NewPermissionHandler creates a new permission Handler
func NewPermissionHandler(service *permission.Service, logs *ActivityLogger) *PermissionHandler {
	return &PermissionHandler{service: service, logs: logs}
}

// GetMyPermissions gets the permissions of the currently authenticated user
func (h *PermissionHandler) GetMyPermissions(ctx context.Context, req *permissionv1.GetMyPermissionsRequest) (*permissionv1.GetMyPermissionsResponse, error) {
	permissions, err := h.service.GetMyPermissions(ctx)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoPermissions := make([]*permissionv1.Permission, len(permissions))
	for i, p := range permissions {
		protoPermissions[i] = toProtoPermFromGORM(&p)
	}

	return &permissionv1.GetMyPermissionsResponse{
		Permissions: protoPermissions,
		TotalCount:  int64(len(protoPermissions)),
	}, nil
}

// GetPermission gets a permission by ID
func (h *PermissionHandler) GetPermission(ctx context.Context, req *permissionv1.GetPermissionRequest) (*permissionv1.PermissionResponse, error) {
	p, err := h.service.GetPermission(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &permissionv1.PermissionResponse{
		Permission: toProtoPermFromGORM(p),
	}, nil
}

// ListPermissions lists all permissions
func (h *PermissionHandler) ListPermissions(ctx context.Context, req *permissionv1.ListPermissionsRequest) (*permissionv1.ListPermissionsResponse, error) {
	permissions, total, err := h.service.ListPermissions(ctx, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoPermissions := make([]*permissionv1.Permission, len(permissions))
	for i, p := range permissions {
		protoPermissions[i] = toProtoPermFromGORM(&p)
	}

	page := req.Page
	perPage := req.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return &permissionv1.ListPermissionsResponse{
		Permissions: protoPermissions,
		TotalCount:  total,
	}, nil
}

// ListPermissionsByResource lists permissions filtered by resource
func (h *PermissionHandler) ListPermissionsByResource(ctx context.Context, req *permissionv1.ListPermissionsByResourceRequest) (*permissionv1.ListPermissionsResponse, error) {
	permissions, total, err := h.service.ListPermissionsByResource(ctx, req.Resource, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoPermissions := make([]*permissionv1.Permission, len(permissions))
	for i, p := range permissions {
		protoPermissions[i] = toProtoPermFromGORM(&p)
	}

	page := req.Page
	perPage := req.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return &permissionv1.ListPermissionsResponse{
		Permissions: protoPermissions,
		TotalCount:  total,
	}, nil
}

// CreatePermission creates a new permission
func (h *PermissionHandler) CreatePermission(ctx context.Context, req *permissionv1.CreatePermissionRequest) (*permissionv1.PermissionResponse, error) {
	p, err := h.service.CreatePermission(ctx, req.Name, req.Resource, req.Action, req.Description)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CREATE", "permissions", p.ID.String(),
		`{"name":"`+p.Name+`","resource":"`+p.Resource+`"}`)

	return &permissionv1.PermissionResponse{
		Permission: toProtoPermFromGORM(p),
	}, nil
}

// UpdatePermission updates a permission
func (h *PermissionHandler) UpdatePermission(ctx context.Context, req *permissionv1.UpdatePermissionRequest) (*permissionv1.PermissionResponse, error) {
	p, err := h.service.UpdatePermission(ctx, req.Id, req.Name, req.Resource, req.Action, req.Description)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "permissions", p.ID.String(), `{"name":"`+p.Name+`"}`)

	return &permissionv1.PermissionResponse{
		Permission: toProtoPermFromGORM(p),
	}, nil
}

// DeletePermission deletes a permission
func (h *PermissionHandler) DeletePermission(ctx context.Context, req *permissionv1.DeletePermissionRequest) (*permissionv1.DeletePermissionResponse, error) {
	var meta interface{}
	if p, gerr := h.service.GetPermission(ctx, req.Id); gerr == nil {
		meta = p
	}

	err := h.service.DeletePermission(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	if meta != nil {
		h.logs.LogJSON(ctx, "DELETE", "permissions", req.Id, meta)
	} else {
		h.logs.Log(ctx, "DELETE", "permissions", req.Id, `{}`)
	}

	return &permissionv1.DeletePermissionResponse{
		Success: true,
		Message: response.MsgPermissionDeleted,
	}, nil
}

// toProtoPermFromGORM converts a GORM Permission model to proto Permission
func toProtoPermFromGORM(p *gormdb.Permission) *permissionv1.Permission {
	return &permissionv1.Permission{
		Id:          p.ID.String(),
		Name:        p.Name,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
