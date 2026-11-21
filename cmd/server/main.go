package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"apiPos/internal/auth"
	"apiPos/internal/config"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/server"
	"apiPos/pkg/timeutil"
)

func main() {
	// Load configuration and fail fast on unsafe values (e.g. a placeholder or
	// short JWT secret) before touching the database or opening listeners.
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Date/timestamp layout for every API response that renders one; see
	// config.FormatConfig. Set once, before the server starts serving.
	timeutil.Configure(cfg.Format.Date, cfg.Format.DateTime)

	log.Printf("Starting POS application in %s mode", cfg.App.Env)

	// Initialize GORM database
	db := gormdb.New(&gormdb.Config{
		Host:     cfg.DB.Host,
		Port:     fmt.Sprintf("%d", cfg.DB.Port),
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		DBName:   cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
	})
	defer db.Close()

	if err := auth.BootstrapSuperadmin(context.Background(), db.GetDB(), os.Getenv("BOOTSTRAP_SUPERADMIN_USERNAME"), os.Getenv("BOOTSTRAP_SUPERADMIN_EMAIL"), os.Getenv("BOOTSTRAP_SUPERADMIN_PASSWORD")); err != nil {
		log.Fatalf("Superadmin bootstrap failed: %v", err)
	}

	// Create and start gRPC server
	srv := server.New(cfg, db)

	// Handle graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-done
	log.Println("Received shutdown signal")

	srv.Stop()
	log.Println("Server shutdown complete")
}
