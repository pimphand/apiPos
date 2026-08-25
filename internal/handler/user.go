package handler

import (
	"context"

	userv1 "apiPos/gen/user/v1"
	db "apiPos/internal/database/sqlc"
	"apiPos/internal/apperror"
	"apiPos/internal/response"
	"apiPos/internal/user"
)

// UserHandler implements the UserService gRPC server
type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	service *user.Service
}

// NewUserHandler creates a new user Handler
func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{service: service}
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.UserResponse, error) {
	u, err := h.service.CreateUser(ctx, req.Username, req.Email, req.Password, req.FullName, req.RoleId, req.StoreId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &userv1.UserResponse{
		User: toProtoUser(u, ""),
	}, nil
}

// GetUser gets a user by ID
func (h *UserHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.UserResponse, error) {
	u, err := h.service.GetUser(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &userv1.UserResponse{
		User: toProtoUser(u, ""),
	}, nil
}

// ListUsers lists users with filters
func (h *UserHandler) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	users, total, err := h.service.ListUsers(ctx, req.RoleId, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoUsers := make([]*userv1.User, len(users))
	for i, u := range users {
		protoUsers[i] = toProtoUser(&u, "")
	}

	return &userv1.ListUsersResponse{
		Users:      protoUsers,
		TotalCount: total,
	}, nil
}

// UpdateUser updates a user
func (h *UserHandler) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UserResponse, error) {
	u, err := h.service.UpdateUser(ctx, req.Id, req.Email, req.FullName, req.RoleId, req.StoreId, req.IsActive)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &userv1.UserResponse{
		User: toProtoUser(u, ""),
	}, nil
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	err := h.service.DeleteUser(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &userv1.DeleteUserResponse{
		Success: true,
		Message: response.MsgUserDeleted,
	}, nil
}

// UpdateUserRole updates a user's role
func (h *UserHandler) UpdateUserRole(ctx context.Context, req *userv1.UpdateUserRoleRequest) (*userv1.UserResponse, error) {
	u, err := h.service.UpdateUserRole(ctx, req.Id, req.RoleId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &userv1.UserResponse{
		User: toProtoUser(u, ""),
	}, nil
}

// toProtoUser converts a db.User to proto User
func toProtoUser(u *db.User, roleName string) *userv1.User {
	return &userv1.User{
		Id:       uuidToString(u.ID),
		Username: u.Username,
		Email:    u.Email,
		FullName: u.FullName,
		RoleId:   uuidToString(u.RoleID),
		RoleName: roleName,
		StoreId:  uuidToString(u.StoreID),
		IsActive: u.IsActive,
	}
}
