package handler

import (
	"context"

	"github.com/google/uuid"

	activitylogv1 "apiPos/gen/activitylog/v1"
	"apiPos/internal/activitylog"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
)

// ActivityLogHandler implements the ActivityLogService gRPC server
type ActivityLogHandler struct {
	activitylogv1.UnimplementedActivityLogServiceServer
	service *activitylog.Service
}

// NewActivityLogHandler creates a new activity log Handler
func NewActivityLogHandler(service *activitylog.Service) *ActivityLogHandler {
	return &ActivityLogHandler{service: service}
}

// ListActivityLogs lists activity logs with filters
func (h *ActivityLogHandler) ListActivityLogs(ctx context.Context, req *activitylogv1.ListActivityLogsRequest) (*activitylogv1.ListActivityLogsResponse, error) {
	logs, total, err := h.service.ListLogs(ctx, req.UserId, req.Action, req.Resource, req.StartDate, req.EndDate, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoLogs := make([]*activitylogv1.ActivityLog, len(logs))
	for i, l := range logs {
		protoLogs[i] = toProtoActivityLogFromGORM(&l)
	}

	page := req.Page
	perPage := req.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return &activitylogv1.ListActivityLogsResponse{
		Logs:       protoLogs,
		TotalCount: total,
	}, nil
}

// GetActivityLog gets an activity log by ID
func (h *ActivityLogHandler) GetActivityLog(ctx context.Context, req *activitylogv1.GetActivityLogRequest) (*activitylogv1.ActivityLogResponse, error) {
	l, err := h.service.GetLog(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &activitylogv1.ActivityLogResponse{
		Log: toProtoActivityLogFromGORM(l),
	}, nil
}

// GetActivityLogsByUser gets activity logs for a specific user
func (h *ActivityLogHandler) GetActivityLogsByUser(ctx context.Context, req *activitylogv1.GetActivityLogsByUserRequest) (*activitylogv1.ListActivityLogsResponse, error) {
	logs, total, err := h.service.ListLogsByUser(ctx, req.UserId, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoLogs := make([]*activitylogv1.ActivityLog, len(logs))
	for i, l := range logs {
		protoLogs[i] = toProtoActivityLogFromGORM(&l)
	}

	page := req.Page
	perPage := req.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return &activitylogv1.ListActivityLogsResponse{
		Logs:       protoLogs,
		TotalCount: total,
	}, nil
}

// GetActivityLogsByResource gets activity logs for a specific resource
func (h *ActivityLogHandler) GetActivityLogsByResource(ctx context.Context, req *activitylogv1.GetActivityLogsByResourceRequest) (*activitylogv1.ListActivityLogsResponse, error) {
	logs, total, err := h.service.ListLogsByResource(ctx, req.Resource, req.ResourceId, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoLogs := make([]*activitylogv1.ActivityLog, len(logs))
	for i, l := range logs {
		protoLogs[i] = toProtoActivityLogFromGORM(&l)
	}

	page := req.Page
	perPage := req.PerPage
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return &activitylogv1.ListActivityLogsResponse{
		Logs:       protoLogs,
		TotalCount: total,
	}, nil
}

// toProtoActivityLogFromGORM converts a GORM ActivityLog model to proto ActivityLog
func toProtoActivityLogFromGORM(l *gormdb.ActivityLog) *activitylogv1.ActivityLog {
	storeName := ""
	if l.Store.ID != uuid.Nil {
		storeName = l.Store.Name
	}
	return &activitylogv1.ActivityLog{
		Id:          l.ID.String(),
		UserId:      l.UserID.String(),
		Username:    l.Username,
		Action:      l.Action,
		Resource:    l.Resource,
		ResourceId:  l.ResourceID,
		Description: l.Description,
		IpAddress:   l.IPAddress,
		UserAgent:   l.UserAgent,
		Metadata:    l.Metadata,
		CreatedAt:   l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		StoreId:     l.StoreID.String(),
		StoreName:   storeName,
	}
}
