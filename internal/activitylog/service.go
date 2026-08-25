package activitylog

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Service handles activity log business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new activity log Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// LogParams holds parameters for creating an activity log
type LogParams struct {
	UserID     string
	Username   string
	Action     string
	Resource   string
	ResourceID string
	IPAddress  string
	UserAgent  string
	Metadata   string
}

// CreateLog creates a new activity log entry
func (s *Service) CreateLog(ctx context.Context, params LogParams) (*gormdb.ActivityLog, error) {
	log := &gormdb.ActivityLog{
		Action:     params.Action,
		Resource:   params.Resource,
		ResourceID: params.ResourceID,
		Description: generateDescription(params),
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		Metadata:   params.Metadata,
	}

	if params.UserID != "" {
		log.UserID = parseUUID(params.UserID)
	}
	if params.Username != "" {
		log.Username = params.Username
	}

	if err := s.repos.ActivityLog.Create(ctx, log); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create activity log", err)
	}

	return log, nil
}

// GetLog gets an activity log by ID
func (s *Service) GetLog(ctx context.Context, id string) (*gormdb.ActivityLog, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid log ID format", fmt.Errorf("id: %s", id))
	}

	log, err := s.repos.ActivityLog.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Activity log not found", err)
	}

	return log, nil
}

// ListLogs lists activity logs with filters and pagination
func (s *Service) ListLogs(ctx context.Context, userID, action, resource, startDate, endDate string, page, perPage int32) ([]gormdb.ActivityLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.repos.ActivityLog.List(ctx, repository.ActivityLogListParams{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      page,
		PerPage:   perPage,
	})
}

// ListLogsByUser lists activity logs for a specific user
func (s *Service) ListLogsByUser(ctx context.Context, userID string, page, perPage int32) ([]gormdb.ActivityLog, int64, error) {
	if !isValidUUID(userID) {
		return nil, 0, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid user ID format", fmt.Errorf("user_id: %s", userID))
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.repos.ActivityLog.ListByUser(ctx, userID, repository.PaginationParams{
		Page:    page,
		PerPage: perPage,
	})
}

// ListLogsByResource lists activity logs for a specific resource
func (s *Service) ListLogsByResource(ctx context.Context, resource, resourceID string, page, perPage int32) ([]gormdb.ActivityLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return s.repos.ActivityLog.ListByResource(ctx, resource, resourceID, repository.PaginationParams{
		Page:    page,
		PerPage: perPage,
	})
}

// generateDescription generates a human-readable description for the activity
func generateDescription(params LogParams) string {
	switch params.Action {
	case "LOGIN":
		return fmt.Sprintf("User '%s' logged in", params.Username)
	case "LOGOUT":
		return fmt.Sprintf("User '%s' logged out", params.Username)
	case "CREATE":
		return fmt.Sprintf("User '%s' created %s", params.Username, params.Resource)
	case "UPDATE":
		return fmt.Sprintf("User '%s' updated %s", params.Username, params.Resource)
	case "DELETE":
		return fmt.Sprintf("User '%s' deleted %s", params.Username, params.Resource)
	case "VIEW":
		return fmt.Sprintf("User '%s' viewed %s", params.Username, params.Resource)
	default:
		return fmt.Sprintf("User '%s' performed %s on %s", params.Username, params.Action, params.Resource)
	}
}

// parseUUID converts string to uuid.UUID
func parseUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.UUID{}
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}
	}
	return u
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
