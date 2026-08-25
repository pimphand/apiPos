package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"apiPos/internal/database/sqlc"
)

var (
	testPool    *pgxpool.Pool
	testQueries *db.Queries
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to start postgres container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to get connection string: %v", err)
	}

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Failed to parse pool config: %v", err)
	}

	testPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}

	// Run migrations
	if err := runMigrations(ctx, testPool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create queries instance
	testQueries = db.New(testPool)

	// Run tests
	code := m.Run()

	// Cleanup
	testPool.Close()
	if err := pgContainer.Terminate(ctx); err != nil {
		log.Printf("Failed to terminate container: %v", err)
	}

	os.Exit(code)
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Read and execute migration file
	migrationSQL, err := os.ReadFile("../../db/migration/000001_init_schema.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = pool.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// setupTestDB creates a fresh database connection for each test
func setupTestDB(t *testing.T) *db.Queries {
	t.Helper()
	return testQueries
}
