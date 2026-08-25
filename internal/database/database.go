package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"apiPos/internal/config"
	db "apiPos/internal/database/sqlc"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	Health() map[string]string

	// Close terminates the database connection.
	Close() error

	// Pool returns the underlying connection pool.
	Pool() *pgxpool.Pool

	// Queries returns the sqlc queries instance.
	Queries() *db.Queries
}

type service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// New creates a new database service.
func New(cfg *config.Config) Service {
	poolConfig, err := pgxpool.ParseConfig(cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Printf("Connected to database: %s", cfg.DB.Name)

	queries := db.New(pool)

	return &service{
		pool:    pool,
		queries: queries,
	}
}

// Health checks the health of the database connection.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	err := s.pool.Ping(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"

	poolStats := s.pool.Stat()
	stats["total_conns"] = fmt.Sprintf("%d", poolStats.TotalConns())
	stats["acquired_conns"] = fmt.Sprintf("%d", poolStats.AcquiredConns())
	stats["idle_conns"] = fmt.Sprintf("%d", poolStats.IdleConns())

	return stats
}

// Close closes the database connection pool.
func (s *service) Close() error {
	s.pool.Close()
	log.Println("Disconnected from database")
	return nil
}

// Pool returns the underlying connection pool.
func (s *service) Pool() *pgxpool.Pool {
	return s.pool
}

// Queries returns the sqlc queries instance.
func (s *service) Queries() *db.Queries {
	return s.queries
}
