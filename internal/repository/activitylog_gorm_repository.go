package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// ActivityLogRepository defines the interface for activity log operations
type ActivityLogRepository interface {
	Create(ctx context.Context, log *gormdb.ActivityLog) error
	GetByID(ctx context.Context, id string) (*gormdb.ActivityLog, error)
	List(ctx context.Context, params ActivityLogListParams) ([]gormdb.ActivityLog, int64, error)
	ListByUser(ctx context.Context, userID string, params PaginationParams) ([]gormdb.ActivityLog, int64, error)
	ListByResource(ctx context.Context, resource, resourceID string, params PaginationParams) ([]gormdb.ActivityLog, int64, error)
}

// ActivityLogListParams holds parameters for listing activity logs
type ActivityLogListParams struct {
	UserID      string
	Action      string
	Resource    string
	StartDate   string
	EndDate     string
	Page        int32
	PerPage     int32
}

// ActivityLogRepo implements ActivityLogRepository using GORM
type ActivityLogRepo struct {
	*GormBase
}

// NewActivityLogRepository creates a new ActivityLogRepo
func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &ActivityLogRepo{GormBase: NewGormBase(db)}
}

// Create creates a new activity log
func (r *ActivityLogRepo) Create(ctx context.Context, log *gormdb.ActivityLog) error {
	return r.WithContext(ctx).Create(log).Error
}

// GetByID gets an activity log by ID
func (r *ActivityLogRepo) GetByID(ctx context.Context, id string) (*gormdb.ActivityLog, error) {
	var log gormdb.ActivityLog
	err := r.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List lists activity logs with filters and pagination
func (r *ActivityLogRepo) List(ctx context.Context, params ActivityLogListParams) ([]gormdb.ActivityLog, int64, error) {
	var logs []gormdb.ActivityLog
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.ActivityLog{})

	// Apply filters
	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}
	if params.Resource != "" {
		query = query.Where("resource = ?", params.Resource)
	}
	if params.StartDate != "" {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("created_at <= ?", params.EndDate+" 23:59:59")
	}

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	normalized := PaginationParams{Page: params.Page, PerPage: params.PerPage}.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&logs).Error

	return logs, totalCount, err
}

// ListByUser lists activity logs for a specific user
func (r *ActivityLogRepo) ListByUser(ctx context.Context, userID string, params PaginationParams) ([]gormdb.ActivityLog, int64, error) {
	var logs []gormdb.ActivityLog
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.ActivityLog{}).Where("user_id = ?", userID)

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	normalized := params.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&logs).Error

	return logs, totalCount, err
}

// ListByResource lists activity logs for a specific resource
func (r *ActivityLogRepo) ListByResource(ctx context.Context, resource, resourceID string, params PaginationParams) ([]gormdb.ActivityLog, int64, error) {
	var logs []gormdb.ActivityLog
	var totalCount int64

	query := r.WithContext(ctx).Model(&gormdb.ActivityLog{}).Where("resource = ?", resource)
	if resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	normalized := params.normalize()
	err := query.Order("created_at DESC").
		Offset(normalized.offset()).
		Limit(normalized.limit()).
		Find(&logs).Error

	return logs, totalCount, err
}
