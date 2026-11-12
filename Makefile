-include .env
export

# POS Application Makefile

# Version of the golang-migrate CLI used by the migrate-* targets
MIGRATE_VERSION := v4.19.1

.PHONY: all build test run proto sqlc migrate-install migrate-up migrate-down migrate-version docker-up docker-down lint clean swagger docs

# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main ./cmd/server

# Run the application
run:
	@go run ./cmd/server

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@buf generate

# Generate sqlc code
sqlc:
	@echo "Generating sqlc code..."
	@sqlc generate

# Install the golang-migrate CLI (required by migrate-up / migrate-down)
# NOTE: the postgres build tag is required, otherwise the CLI reports
# "database driver: unknown driver postgres (forgotten import?)"
migrate-install:
	@echo "Installing golang-migrate CLI $(MIGRATE_VERSION)..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

migrate-check:
	@command -v migrate >/dev/null 2>&1 || (echo "migrate CLI not found, running: make migrate-install"; $(MAKE) migrate-install)

# Run database migrations
migrate-up: migrate-check
	@echo "Running migrations up..."
	@migrate -path db/migration -database "postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=disable" up

migrate-down: migrate-check
	@echo "Running migrations down..."
	@migrate -path db/migration -database "postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=disable" down

migrate-version:
	@migrate -version

# Docker commands
docker-up:
	@echo "Starting Docker containers..."
	@docker compose up -d --build

docker-down:
	@echo "Stopping Docker containers..."
	@docker compose down

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Serve API docs (Swagger UI + gRPC/REST examples) on :18735
# Override with DOCS_PORT=... if 18735 is already taken.
docs:
	@go run ./cmd/docs

# Generate swagger spec
swagger:
	@echo "Generating swagger spec..."
	@buf generate

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f main
