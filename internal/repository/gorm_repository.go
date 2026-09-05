package repository

import (
	"context"
	"math"

	"gorm.io/gorm"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page    int32
	PerPage int32
	// StoreID optionally restricts the query to a single store (used by
	// store-scoped listers such as activity logs). Empty means no filter.
	StoreID string
}

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
	TotalPages int32
}

// offset calculates the offset for pagination
func (p PaginationParams) offset() int {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	perPage := p.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return int((page - 1) * perPage)
}

// limit returns the limit for pagination
func (p PaginationParams) limit() int {
	perPage := p.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return int(perPage)
}

// normalize normalizes pagination parameters
func (p PaginationParams) normalize() PaginationParams {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	perPage := p.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return PaginationParams{Page: page, PerPage: perPage}
}

// calcPagination calculates pagination result
func calcPagination(totalCount int64, params PaginationParams) PaginationResult {
	normalized := params.normalize()
	totalPages := int32(math.Ceil(float64(totalCount) / float64(normalized.PerPage)))
	return PaginationResult{
		TotalCount: totalCount,
		Page:       normalized.Page,
		PerPage:    normalized.PerPage,
		TotalPages: totalPages,
	}
}

// GormBase provides common GORM operations
type GormBase struct {
	db *gorm.DB
}

// NewGormBase creates a new GormBase
func NewGormBase(db *gorm.DB) *GormBase {
	return &GormBase{db: db}
}

// WithContext returns db with context
func (g *GormBase) WithContext(ctx context.Context) *gorm.DB {
	return g.db.WithContext(ctx)
}

// Paginate applies pagination to a query
func (g *GormBase) Paginate(query *gorm.DB, params PaginationParams) *gorm.DB {
	normalized := params.normalize()
	return query.Offset(normalized.offset()).Limit(normalized.limit())
}
