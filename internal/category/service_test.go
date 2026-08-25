package category

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// mockCategoryRepo is a mock for CategoryRepository
type mockCategoryRepo struct {
	createFunc     func(ctx context.Context, category *gormdb.Category) error
	getByIDFunc    func(ctx context.Context, id string) (*gormdb.Category, error)
	listFunc       func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error)
	updateFunc     func(ctx context.Context, category *gormdb.Category) error
	deleteFunc     func(ctx context.Context, id string) error
	deactivateFunc func(ctx context.Context, id string) error
}

func (m *mockCategoryRepo) Create(ctx context.Context, category *gormdb.Category) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, category)
	}
	return nil
}

func (m *mockCategoryRepo) GetByID(ctx context.Context, id string) (*gormdb.Category, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockCategoryRepo) List(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, params)
	}
	return []gormdb.Category{}, 0, nil
}

func (m *mockCategoryRepo) Update(ctx context.Context, category *gormdb.Category) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, category)
	}
	return nil
}

func (m *mockCategoryRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockCategoryRepo) Deactivate(ctx context.Context, id string) error {
	if m.deactivateFunc != nil {
		return m.deactivateFunc(ctx, id)
	}
	return nil
}

func testCategory() *gormdb.Category {
	return &gormdb.Category{
		ID:          uuid.New(),
		Name:        "Electronics",
		Description: "Electronic devices",
		IsActive:    true,
	}
}

func TestCategoryService_CreateCategory(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return []gormdb.Category{}, 0, nil
			},
			createFunc: func(ctx context.Context, category *gormdb.Category) error {
				category.ID = uuid.New()
				return nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		cat, err := svc.CreateCategory(ctx, "Electronics", "Electronic devices")
		require.NoError(t, err)
		assert.Equal(t, "Electronics", cat.Name)
		assert.Equal(t, "Electronic devices", cat.Description)
	})

	t.Run("duplicate name", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return []gormdb.Category{
					{ID: uuid.New(), Name: "Electronics", Description: "Description"},
				}, 1, nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		_, err := svc.CreateCategory(ctx, "Electronics", "Description")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestCategoryService_GetCategory(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return testCategory(), nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		cat, err := svc.GetCategory(ctx, validID)
		require.NoError(t, err)
		assert.Equal(t, "Electronics", cat.Name)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		_, err := svc.GetCategory(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid id format", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{}
		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		_, err := svc.GetCategory(ctx, "invalid-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid")
	})
}

func TestCategoryService_ListCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("success with pagination", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return []gormdb.Category{
					{ID: uuid.New(), Name: "Category 1"},
					{ID: uuid.New(), Name: "Category 2"},
				}, 2, nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		result, err := svc.ListCategories(ctx, "", 1, 10)
		require.NoError(t, err)
		assert.Len(t, result.Categories, 2)
		assert.Equal(t, int64(2), result.Pagination.TotalCount)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(10), result.Pagination.PerPage)
	})

	t.Run("empty list", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return []gormdb.Category{}, 0, nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		result, err := svc.ListCategories(ctx, "", 1, 10)
		require.NoError(t, err)
		assert.Empty(t, result.Categories)
		assert.Equal(t, int64(0), result.Pagination.TotalCount)
	})

	t.Run("default pagination", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return []gormdb.Category{}, 0, nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		result, err := svc.ListCategories(ctx, "", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(20), result.Pagination.PerPage)
	})
}

func TestCategoryService_UpdateCategory(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return testCategory(), nil
			},
			updateFunc: func(ctx context.Context, category *gormdb.Category) error {
				return nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		cat, err := svc.UpdateCategory(ctx, validID, "New Name", "New Description", true)
		require.NoError(t, err)
		assert.Equal(t, "New Name", cat.Name)
		assert.Equal(t, "New Description", cat.Description)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		_, err := svc.UpdateCategory(ctx, validID, "Name", "Desc", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCategoryService_DeleteCategory(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return testCategory(), nil
			},
			deleteFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		err := svc.DeleteCategory(ctx, validID)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		err := svc.DeleteCategory(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCategoryService_DeactivateCategory(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return testCategory(), nil
			},
			deactivateFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		err := svc.DeactivateCategory(ctx, validID)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Category, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		err := svc.DeactivateCategory(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCategoryService_CountCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return []gormdb.Category{}, 10, nil
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		result, err := svc.ListCategories(ctx, "", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result.Pagination.TotalCount)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo := &mockCategoryRepo{
			listFunc: func(ctx context.Context, params repository.CategoryListParams) ([]gormdb.Category, int64, error) {
				return nil, 0, fmt.Errorf("database error")
			},
		}

		repos := &repository.Repositories{Category: mockRepo}
		svc := NewService(repos)

		_, err := svc.ListCategories(ctx, "", 1, 10)
		assert.Error(t, err)
	})
}
