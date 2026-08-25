# POS Application

Production-ready Point of Sale (POS) application built with Go and gRPC.

## Technology Stack

- Go 1.24+
- gRPC
- Protocol Buffers
- PostgreSQL
- pgx (database driver)
- sqlc (type-safe SQL)
- golang-migrate (database migrations)
- Docker & Docker Compose
- JWT Authentication
- buf (protobuf tooling)

## Architecture

```
pos/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── server/
│   ├── database/
│   ├── auth/
│   ├── user/
│   ├── product/
│   ├── category/
│   ├── inventory/
│   ├── customer/
│   ├── sales/
│   ├── payment/
│   ├── cashier/
│   └── report/
├── proto/
├── gen/
├── db/
│   ├── migration/
│   └── query/
├── middleware/
├── pkg/
└── ...
```

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- buf (protobuf tooling)
- sqlc
- golang-migrate
- grpcurl (for API testing)

### Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd pos
```

2. Install dependencies:

```bash
go mod tidy
```

3. Copy and configure environment variables:

```bash
cp .env.example .env
# Edit .env with your configuration
```

### Running with Docker

Start all services:

```bash
make docker-up
```

Stop all services:

```bash
make docker-down
```

### Running Locally

1. Start PostgreSQL:

```bash
docker compose up postgres -d
```

2. Run the application:

```bash
make run
```

### Development

Use live reload with air:

```bash
make watch
```

## Development Phases

### Phase 1 (Current) ✅

- [x] Project setup
- [x] PostgreSQL configuration
- [x] Docker & Docker Compose
- [x] gRPC server
- [x] Protobuf setup with buf
- [x] Health check service
- [x] Configuration system

### Phase 2 (Next)

- [ ] Database migrations
- [ ] sqlc queries
- [ ] Repository layer

### Phase 3

- [ ] Authentication
- [ ] Users & Roles
- [ ] JWT
- [ ] Middleware

### Phase 4

- [ ] Categories
- [ ] Products

### Phase 5

- [ ] Inventory
- [ ] Stock transactions

### Phase 6

- [ ] Customers
- [ ] Sales
- [ ] Sale items

### Phase 7

- [ ] Payments
- [ ] Cashier shifts

### Phase 8

- [ ] Reports

### Phase 9

- [ ] Testing
- [ ] Logging
- [ ] Error handling
- [ ] Performance optimization
- [ ] Documentation

## API Testing

### Health Check

```bash
grpcurl \
  -plaintext \
  localhost:50051 \
  health.v1.HealthService/Check
```

### Using grpcurl

```bash
# List services
grpcurl -plaintext localhost:50051 list

# List methods for a service
grpcurl -plaintext localhost:50051 list health.v1.HealthService
```

## Make Commands

```bash
make all          # Build and test
make build        # Build the application
make run          # Run the application
make test         # Run tests
make proto        # Generate protobuf code
make sqlc         # Generate sqlc code
make migrate-up   # Run database migrations
make migrate-down # Rollback database migrations
make docker-up    # Start Docker containers
make docker-down  # Stop Docker containers
make lint         # Lint the code
make clean        # Clean build artifacts
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| APP_ENV | development | Application environment |
| GRPC_PORT | 50051 | gRPC server port |
| DATABASE_HOST | localhost | PostgreSQL host |
| DATABASE_PORT | 5432 | PostgreSQL port |
| DATABASE_NAME | pos | Database name |
| DATABASE_USER | postgres | Database user |
| DATABASE_PASSWORD | postgres | Database password |
| JWT_SECRET | change-me | JWT secret key |
| JWT_ACCESS_TTL | 15m | JWT access token TTL |
| JWT_REFRESH_TTL | 720h | JWT refresh token TTL |

## License

MIT
