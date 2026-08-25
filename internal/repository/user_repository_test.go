package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "apiPos/internal/database/sqlc"
)

func TestUserRepository_CreateUser(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("testadmin_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Admin role", Valid: true},
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		user, err := repo.CreateUser(ctx, db.CreateUserParams{
			Username:     fmt.Sprintf("testuser_%d", time.Now().UnixNano()),
			Email:        fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Test User",
			RoleID:       role.ID,
		})
		require.NoError(t, err)
		assert.True(t, user.ID.Valid)
		assert.NotEmpty(t, user.Username)
		assert.Equal(t, "Test User", user.FullName)
		assert.Equal(t, role.ID, user.RoleID)
		assert.True(t, user.IsActive)
	})

	t.Run("duplicate username", func(t *testing.T) {
		username := fmt.Sprintf("dupuser_%d", time.Now().UnixNano())
		_, err := repo.CreateUser(ctx, db.CreateUserParams{
			Username:     username,
			Email:        fmt.Sprintf("dup1_%d@example.com", time.Now().UnixNano()),
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Dup User 1",
			RoleID:       role.ID,
		})
		require.NoError(t, err)

		_, err = repo.CreateUser(ctx, db.CreateUserParams{
			Username:     username,
			Email:        fmt.Sprintf("dup2_%d@example.com", time.Now().UnixNano()),
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Dup User 2",
			RoleID:       role.ID,
		})
		assert.Error(t, err)
	})

	t.Run("duplicate email", func(t *testing.T) {
		email := fmt.Sprintf("dupemail_%d@example.com", time.Now().UnixNano())
		_, err := repo.CreateUser(ctx, db.CreateUserParams{
			Username:     fmt.Sprintf("unique1_%d", time.Now().UnixNano()),
			Email:        email,
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Unique 1",
			RoleID:       role.ID,
		})
		require.NoError(t, err)

		_, err = repo.CreateUser(ctx, db.CreateUserParams{
			Username:     fmt.Sprintf("unique2_%d", time.Now().UnixNano()),
			Email:        email,
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Unique 2",
			RoleID:       role.ID,
		})
		assert.Error(t, err)
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("mgr_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Manager role", Valid: true},
	})
	require.NoError(t, err)

	user, err := repo.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("getbyid_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("getbyid_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Get By ID User",
		RoleID:       role.ID,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Username, found.Username)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetUserByID(ctx, zeroUUID())
		assert.Error(t, err)
	})
}

func TestUserRepository_GetUserByUsername(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("cash_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Cashier role", Valid: true},
	})
	require.NoError(t, err)

	username := fmt.Sprintf("findme_%d", time.Now().UnixNano())
	user, err := repo.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        fmt.Sprintf("findme_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Find Me User",
		RoleID:       role.ID,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		found, err := repo.GetUserByUsername(ctx, username)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetUserByUsername(ctx, "nonexistent")
		assert.Error(t, err)
	})
}

func TestUserRepository_UpdateUser(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("updaterole_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Update role", Valid: true},
	})
	require.NoError(t, err)

	user, err := repo.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("updateme_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("update_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Update Me",
		RoleID:       role.ID,
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		updated, err := repo.UpdateUser(ctx, db.UpdateUserParams{
			ID:       user.ID,
			FullName: "Updated Name",
			Email:    fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
			RoleID:   role.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.FullName)
	})
}

func TestUserRepository_DeactivateUser(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("deactrole_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Deactivate role", Valid: true},
	})
	require.NoError(t, err)

	user, err := repo.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("deactivate_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("deactivate_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Deactivate User",
		RoleID:       role.ID,
	})
	require.NoError(t, err)

	err = repo.DeactivateUser(ctx, user.ID)
	require.NoError(t, err)

	found, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, found.IsActive)
}

func TestUserRepository_DeleteUser(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("deleterole_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Delete role", Valid: true},
	})
	require.NoError(t, err)

	user, err := repo.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("delete_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("delete_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Delete User",
		RoleID:       role.ID,
	})
	require.NoError(t, err)

	err = repo.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	_, err = repo.GetUserByID(ctx, user.ID)
	assert.Error(t, err)
}

func TestUserRepository_ListUsers(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("listrole_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "List role", Valid: true},
	})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := repo.CreateUser(ctx, db.CreateUserParams{
			Username:     fmt.Sprintf("listuser_%d_%d", time.Now().UnixNano(), i),
			Email:        fmt.Sprintf("listuser_%d_%d@example.com", time.Now().UnixNano(), i),
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     fmt.Sprintf("User %d", i),
			RoleID:       role.ID,
		})
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		users, err := repo.ListUsers(ctx, db.ListUsersParams{
			Column1: zeroUUID(),
			Column2: false,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 5)
	})

	t.Run("with pagination", func(t *testing.T) {
		users, err := repo.ListUsers(ctx, db.ListUsersParams{
			Column1: zeroUUID(),
			Column2: false,
			Limit:   2,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(users), 2)
	})
}

func TestUserRepository_RefreshTokens(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	role, err := queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        fmt.Sprintf("tokenrole_%d", time.Now().UnixNano()),
		Description: pgtype.Text{String: "Token role", Valid: true},
	})
	require.NoError(t, err)

	user, err := repo.CreateUser(ctx, db.CreateUserParams{
		Username:     fmt.Sprintf("tokenuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("token_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Token User",
		RoleID:       role.ID,
	})
	require.NoError(t, err)

	t.Run("create and get refresh token", func(t *testing.T) {
		tokenStr := fmt.Sprintf("test-refresh-token-%d", time.Now().UnixNano())
		token, err := repo.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
			UserID:    user.ID,
			Token:     tokenStr,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)
		assert.True(t, token.ID.Valid)

		found, err := repo.GetRefreshToken(ctx, tokenStr)
		require.NoError(t, err)
		assert.Equal(t, token.ID, found.ID)
	})

	t.Run("revoke refresh token", func(t *testing.T) {
		tokenStr := fmt.Sprintf("revoke-token-%d", time.Now().UnixNano())
		_, err := repo.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
			UserID:    user.ID,
			Token:     tokenStr,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		err = repo.RevokeRefreshToken(ctx, tokenStr)
		require.NoError(t, err)

		_, err = repo.GetRefreshToken(ctx, tokenStr)
		assert.Error(t, err)
	})
}

func TestUserRepository_CountUsers(t *testing.T) {
	queries := setupTestDB(t)
	repo := NewUserRepository(queries)
	ctx := context.Background()

	count, err := repo.CountUsers(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
