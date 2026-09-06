package handler

import (
	"context"

	categoryv1 "apiPos/gen/category/v1"
	"apiPos/internal/apperror"
	"apiPos/internal/category"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/response"
)

// CategoryHandler implements the CategoryService gRPC server
type CategoryHandler struct {
	categoryv1.UnimplementedCategoryServiceServer
	service *category.Service
	logs    *ActivityLogger
}

// NewCategoryHandler creates a new category Handler
func NewCategoryHandler(service *category.Service, logs *ActivityLogger) *CategoryHandler {
	return &CategoryHandler{service: service, logs: logs}
}

// CreateCategory creates a new category
func (h *CategoryHandler) CreateCategory(ctx context.Context, req *categoryv1.CreateCategoryRequest) (*categoryv1.CategoryResponse, error) {
	c, err := h.service.CreateCategory(ctx, req.Name, req.Description)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CREATE", "categories", c.ID.String(), `{"name":"`+c.Name+`"}`)

	return &categoryv1.CategoryResponse{
		Category: toProtoCategoryFromGORM(c),
	}, nil
}

// GetCategory gets a category by ID
func (h *CategoryHandler) GetCategory(ctx context.Context, req *categoryv1.GetCategoryRequest) (*categoryv1.CategoryResponse, error) {
	c, err := h.service.GetCategory(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &categoryv1.CategoryResponse{
		Category: toProtoCategoryFromGORM(c),
	}, nil
}

// ListCategories lists all categories with pagination
func (h *CategoryHandler) ListCategories(ctx context.Context, req *categoryv1.ListCategoriesRequest) (*categoryv1.ListCategoriesResponse, error) {
	result, err := h.service.ListCategories(ctx, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoCategories := make([]*categoryv1.Category, len(result.Categories))
	for i, c := range result.Categories {
		protoCategories[i] = toProtoCategoryFromGORM(&c)
	}

	return &categoryv1.ListCategoriesResponse{
		Categories: protoCategories,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// UpdateCategory updates a category
func (h *CategoryHandler) UpdateCategory(ctx context.Context, req *categoryv1.UpdateCategoryRequest) (*categoryv1.CategoryResponse, error) {
	c, err := h.service.UpdateCategory(ctx, req.Id, req.Name, req.Description, req.IsActive)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "categories", c.ID.String(), `{"name":"`+c.Name+`"}`)

	return &categoryv1.CategoryResponse{
		Category: toProtoCategoryFromGORM(c),
	}, nil
}

// DeleteCategory deletes a category
func (h *CategoryHandler) DeleteCategory(ctx context.Context, req *categoryv1.DeleteCategoryRequest) (*categoryv1.DeleteCategoryResponse, error) {
	var meta interface{}
	if c, gerr := h.service.GetCategory(ctx, req.Id); gerr == nil {
		meta = c
	}

	err := h.service.DeleteCategory(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	if meta != nil {
		h.logs.LogJSON(ctx, "DELETE", "categories", req.Id, meta)
	} else {
		h.logs.Log(ctx, "DELETE", "categories", req.Id, `{}`)
	}

	return &categoryv1.DeleteCategoryResponse{
		Success: true,
		Message: response.MsgCategoryDeleted,
	}, nil
}

// DeactivateCategory soft-deletes a category
func (h *CategoryHandler) DeactivateCategory(ctx context.Context, req *categoryv1.DeactivateCategoryRequest) (*categoryv1.DeleteCategoryResponse, error) {
	err := h.service.DeactivateCategory(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "categories", req.Id, `{"is_active":false}`)

	return &categoryv1.DeleteCategoryResponse{
		Success: true,
		Message: response.MsgCategoryDeactivated,
	}, nil
}

// toProtoCategoryFromGORM converts a GORM Category model to proto Category
func toProtoCategoryFromGORM(c *gormdb.Category) *categoryv1.Category {
	return &categoryv1.Category{
		Id:          c.ID.String(),
		Name:        c.Name,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
