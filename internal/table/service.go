// Package table implements per-store management of dine-in tables used by
// self ordering (docs/pos-self-order.md). Every operation is tenant-scoped to
// the authenticated user's store (enforced by the GORM store_scope callbacks
// plus an explicit store check).
package table

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Service handles dine-in table business logic.
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new table Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// storeScope returns the authenticated user's store id ("" when superadmin).
func (s *Service) storeScope(ctx context.Context) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return uuid.Nil, apperror.Wrap(apperror.ErrUnauthorized, "Authentication required", nil)
	}
	if claims.StoreID == "" {
		return uuid.Nil, apperror.Wrap(apperror.ErrForbidden, "Account is not bound to a store", nil)
	}
	id, err := uuid.Parse(claims.StoreID)
	if err != nil {
		return uuid.Nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid store on account", err)
	}
	return id, nil
}

// CreateTable creates a new table for the store with a fresh QR token.
func (s *Service) CreateTable(ctx context.Context, code, name string) (*gormdb.Table, error) {
	storeID, err := s.storeScope(ctx)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Table code is required", nil)
	}
	if len(code) > 20 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Table code is too long (max 20)", nil)
	}

	table := &gormdb.Table{
		StoreID:  storeID,
		Code:     code,
		Name:     strings.TrimSpace(name),
		QRToken:  uuid.NewString(),
		IsActive: true,
	}
	if err := s.repos.Table.Create(ctx, table); err != nil {
		return nil, apperror.Wrap(apperror.ErrAlreadyExists, "Kode meja sudah digunakan", err)
	}
	return table, nil
}

// ListTables lists the store's tables with pagination.
func (s *Service) ListTables(ctx context.Context, isActive string, page, perPage int32) ([]gormdb.Table, int64, error) {
	if _, err := s.storeScope(ctx); err != nil {
		return nil, 0, err
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

	tables, total, err := s.repos.Table.List(ctx, repository.PaginationParams{Page: page, PerPage: perPage})
	if err != nil {
		return nil, 0, apperror.Wrap(apperror.ErrInternal, "Failed to list tables", err)
	}

	// Post-filter on is_active (repo list is store-scoped already).
	if isActive == "true" || isActive == "false" {
		want := isActive == "true"
		filtered := tables[:0]
		for _, t := range tables {
			if t.IsActive == want {
				filtered = append(filtered, t)
			}
		}
		return filtered, int64(len(filtered)), nil
	}
	return tables, total, nil
}

// GetTable returns one table (store-scoped).
func (s *Service) GetTable(ctx context.Context, id string) (*gormdb.Table, error) {
	storeID, err := s.storeScope(ctx)
	if err != nil {
		return nil, err
	}
	tableID, err := uuid.Parse(id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid table ID", nil)
	}
	table, err := s.repos.Table.GetByID(ctx, tableID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Wrap(apperror.ErrNotFound, "Table not found", nil)
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to get table", err)
	}
	if table.StoreID != storeID {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Table not found", nil)
	}
	return table, nil
}

// UpdateTable updates a table's code/name/active state.
func (s *Service) UpdateTable(ctx context.Context, id, code, name string, isActive bool) (*gormdb.Table, error) {
	table, err := s.GetTable(ctx, id)
	if err != nil {
		return nil, err
	}
	if c := strings.TrimSpace(strings.ToUpper(code)); c != "" {
		if len(c) > 20 {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Table code is too long (max 20)", nil)
		}
		table.Code = c
	}
	if strings.TrimSpace(name) != "" {
		table.Name = strings.TrimSpace(name)
	}
	if err := s.repos.Table.Update(ctx, table); err != nil {
		return nil, apperror.Wrap(apperror.ErrAlreadyExists, "Kode meja sudah digunakan", err)
	}
	// Apply the active flag separately (Save() would persist the loaded value).
	if table.IsActive != isActive {
		table.IsActive = isActive
		if err := s.repos.DB().WithContext(ctx).Model(&gormdb.Table{}).
			Where("id = ?", table.ID).Update("is_active", isActive).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update table", err)
		}
	}
	return table, nil
}

// DeleteTable deletes a table. Sales referencing it keep their table_no
// snapshot (FK is ON DELETE SET NULL).
func (s *Service) DeleteTable(ctx context.Context, id string) error {
	table, err := s.GetTable(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repos.Table.Delete(ctx, table.ID); err != nil {
		return apperror.Wrap(apperror.ErrInternal, "Failed to delete table", err)
	}
	return nil
}

// RotateTableToken invalidates the current QR by issuing a new token.
func (s *Service) RotateTableToken(ctx context.Context, id string) (*gormdb.Table, error) {
	table, err := s.GetTable(ctx, id)
	if err != nil {
		return nil, err
	}
	table.QRToken = uuid.NewString()
	if err := s.repos.DB().WithContext(ctx).Model(&gormdb.Table{}).
		Where("id = ?", table.ID).Update("qr_token", table.QRToken).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to rotate table token", err)
	}
	return table, nil
}