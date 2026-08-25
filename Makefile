# POS Application Makefile

.PHONY: all build test run proto sqlc migrate-up migrate-down docker-up docker-down lint clean

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

# Run database migrations
migrate-up:
	@echo "Running migrations up..."
	@migrate -path db/migration -database "postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=disable" up

migrate-down:
	@echo "Running migrations down..."
	@migrate -path db/migration -database "postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=disable" down

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

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f main
