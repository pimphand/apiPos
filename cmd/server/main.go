package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"apiPos/internal/config"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/server"
)

func main() {
	// Load configuration
	cfg := config.Load()

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
