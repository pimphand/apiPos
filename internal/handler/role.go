package handler

import (
	"context"

	rolev1 "apiPos/gen/role/v1"
	"apiPos/internal/apperror"
	"apiPos/internal/response"
	"apiPos/internal/role"

	gormdb "apiPos/internal/database/gorm"
)

// RoleHandler implements the RoleService gRPC server
type RoleHandler struct {
	rolev1.UnimplementedRoleServiceServer
	service *role.Service
	logs    *ActivityLogger
}

// NewRoleHandler creates a new role Handler
func NewRoleHandler(service *role.Service, logs *ActivityLogger) *RoleHandler {
	return &RoleHandler{service: service, logs: logs}
}

// CreateRole creates a new role
func (h *RoleHandler) CreateRole(ctx context.Context, req *rolev1.CreateRoleRequest) (*rolev1.RoleResponse, error) {
	r, err := h.service.CreateRole(ctx, req.Name, req.Description)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CREATE", "roles", r.ID.String(), `{"name":"`+r.Name+`"}`)

	return &rolev1.RoleResponse{
		Role: toProtoRoleFromGORM(r),
	}, nil
}

// GetRole gets a role by ID
func (h *RoleHandler) GetRole(ctx context.Context, req *rolev1.GetRoleRequest) (*rolev1.RoleResponse, error) {
	r, err := h.service.GetRole(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &rolev1.RoleResponse{
		Role: toProtoRoleFromGORM(r),
	}, nil
}

// ListRoles lists all roles
func (h *RoleHandler) ListRoles(ctx context.Context, req *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
	roles, total, err := h.service.ListRoles(ctx, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoRoles := make([]*rolev1.Role, len(roles))
	for i, r := range roles {
		protoRoles[i] = toProtoRoleFromGORM(&r)
	}

	page := req.Page
	perPage := req.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return &rolev1.ListRolesResponse{
		Roles:      protoRoles,
		TotalCount: total,
	}, nil
}

// UpdateRole updates a role
func (h *RoleHandler) UpdateRole(ctx context.Context, req *rolev1.UpdateRoleRequest) (*rolev1.RoleResponse, error) {
	r, err := h.service.UpdateRole(ctx, req.Id, req.Name, req.Description)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "roles", r.ID.String(), `{"name":"`+r.Name+`"}`)

	return &rolev1.RoleResponse{
		Role: toProtoRoleFromGORM(r),
	}, nil
}

// DeleteRole deletes a role
func (h *RoleHandler) DeleteRole(ctx context.Context, req *rolev1.DeleteRoleRequest) (*rolev1.DeleteRoleResponse, error) {
	var meta interface{}
	if r, gerr := h.service.GetRole(ctx, req.Id); gerr == nil {
		meta = r
	}

	err := h.service.DeleteRole(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	if meta != nil {
		h.logs.LogJSON(ctx, "DELETE", "roles", req.Id, meta)
	} else {
		h.logs.Log(ctx, "DELETE", "roles", req.Id, `{}`)
	}

	return &rolev1.DeleteRoleResponse{
		Success: true,
		Message: response.MsgRoleDeleted,
	}, nil
}

// GetRolePermissions gets all permissions for a role
func (h *RoleHandler) GetRolePermissions(ctx context.Context, req *rolev1.GetRolePermissionsRequest) (*rolev1.GetRolePermissionsResponse, error) {
	permissions, err := h.service.GetRolePermissions(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoPermissions := make([]*rolev1.Permission, len(permissions))
	for i, p := range permissions {
		protoPermissions[i] = toProtoPermissionFromGORM(&p)
	}

	return &rolev1.GetRolePermissionsResponse{
		Permissions: protoPermissions,
	}, nil
}

// AssignPermissions assigns permissions to a role
func (h *RoleHandler) AssignPermissions(ctx context.Context, req *rolev1.AssignPermissionsRequest) (*rolev1.AssignPermissionsResponse, error) {
	permissions, err := h.service.AssignPermissions(ctx, req.Id, req.PermissionIds)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoPermissions := make([]*rolev1.Permission, len(permissions))
	for i, p := range permissions {
		protoPermissions[i] = toProtoPermissionFromGORM(&p)
	}

	h.logs.Log(ctx, "UPDATE", "roles", req.Id,
		`{"action":"assign_permissions","count":`+itoa(len(req.PermissionIds))+`}`)

	return &rolev1.AssignPermissionsResponse{
		Permissions: protoPermissions,
		Message:     "Permissions assigned successfully",
	}, nil
}

// RemovePermission removes a permission from a role
func (h *RoleHandler) RemovePermission(ctx context.Context, req *rolev1.RemovePermissionRequest) (*rolev1.RemovePermissionResponse, error) {
	err := h.service.RemovePermission(ctx, req.RoleId, req.PermissionId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "roles", req.RoleId,
		`{"action":"remove_permission","permission_id":"`+req.PermissionId+`"}`)

	return &rolev1.RemovePermissionResponse{
		Success: true,
		Message: "Permission removed successfully",
	}, nil
}

// toProtoRoleFromGORM converts a GORM Role model to proto Role
func toProtoRoleFromGORM(r *gormdb.Role) *rolev1.Role {
	return &rolev1.Role{
		Id:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// toProtoPermissionFromGORM converts a GORM Permission model to proto Permission
func toProtoPermissionFromGORM(p *gormdb.Permission) *rolev1.Permission {
	return &rolev1.Permission{
		Id:          p.ID.String(),
		Name:        p.Name,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
	}
}
