package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "apiPos/internal/database/sqlc"
)

func TestCategoryRepository_CreateCategory(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		category, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
			Name:        "Electronics",
			Description: pgtype.Text{String: "Electronic devices", Valid: true},
		})
		require.NoError(t, err)
		assert.NotZero(t, category.ID)
		assert.Equal(t, "Electronics", category.Name)
		assert.True(t, category.IsActive)
	})

	t.Run("without description", func(t *testing.T) {
		category, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
			Name: "Food",
		})
		require.NoError(t, err)
		assert.NotZero(t, category.ID)
		assert.Equal(t, "Food", category.Name)
	})
}

func TestCategoryRepository_GetCategoryByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	category, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
		Name:        "GetCategory",
		Description: pgtype.Text{String: "Get category", Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetCategoryByID(ctx, category.ID)
		require.NoError(t, err)
		assert.Equal(t, category.ID, found.ID)
		assert.Equal(t, "GetCategory", found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCategoryByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestCategoryRepository_UpdateCategory(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	category, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
		Name:        "UpdateCategory",
		Description: pgtype.Text{String: "Original", Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		updated, err := repo.UpdateCategory(ctx, db.UpdateCategoryParams{
			ID:          category.ID,
			Name:        "UpdatedCategory",
			Description: pgtype.Text{String: "Updated", Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, "UpdatedCategory", updated.Name)
		assert.Equal(t, "Updated", updated.Description.String)
	})
}

func TestCategoryRepository_DeleteCategory(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	category, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
		Name: "DeleteCategory",
	})
	require.NoError(t, err)

	err = repo.DeleteCategory(ctx, category.ID)
	require.NoError(t, err)

	_, err = repo.GetCategoryByID(ctx, category.ID)
	assert.Error(t, err)
}

func TestCategoryRepository_DeactivateCategory(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	category, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
		Name: "DeactivateCategory",
	})
	require.NoError(t, err)

	err = repo.DeactivateCategory(ctx, category.ID)
	require.NoError(t, err)

	found, err := repo.GetCategoryByID(ctx, category.ID)
	require.NoError(t, err)
	assert.False(t, found.IsActive)
}

func TestCategoryRepository_ListCategories(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	// Create multiple categories
	for i := 0; i < 5; i++ {
		_, err := repo.CreateCategory(ctx, db.CreateCategoryParams{
			Name: fmt.Sprintf("Category %d", i),
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		categories, err := repo.ListCategories(ctx, false)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(categories), 5)
	})

	t.Run("list active only", func(t *testing.T) {
		// Deactivate one category
		categories, err := repo.ListCategories(ctx, false)
		require.NoError(t, err)
		if len(categories) > 0 {
			err := repo.DeactivateCategory(ctx, categories[0].ID)
			require.NoError(t, err)
		}

		activeCategories, err := repo.ListCategories(ctx, true)
		require.NoError(t, err)
		for _, c := range activeCategories {
			assert.True(t, c.IsActive)
		}
	})
}

func TestCategoryRepository_CountCategories(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewCategoryRepository(queries)
	ctx := context.Background()

	count, err := repo.CountCategories(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
