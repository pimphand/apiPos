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

### Phase 1 ✅

- [x] Project setup
- [x] PostgreSQL configuration
- [x] Docker & Docker Compose
- [x] gRPC server
- [x] Protobuf setup with buf
- [x] Health check service
- [x] Configuration system

### Phase 2 ✅

- [x] Database migrations
- [x] GORM models
- [x] Repository layer

### Phase 3 ✅

- [x] Authentication
- [x] Users & Roles
- [x] JWT
- [x] Middleware

### Phase 4 ✅

- [x] Categories
- [x] Products
- [x] Product SKUs

### Phase 5 ✅

- [x] Inventory
- [x] Stock transactions
- [x] Atomic stock updates

### Phase 6 ✅

- [x] Customers
- [x] Sales
- [x] Sale items
- [x] Atomic checkout

### Phase 7 ✅

- [x] Payments
- [x] Cashier shifts
- [x] Cash reconciliation

### Phase 8 ✅

- [x] Reports
- [x] Daily sales
- [x] Sales by date/cashier/product/category
- [x] Payment method summary
- [x] Inventory summary
- [x] Low stock alert

### Phase 9 ✅

- [x] Testing - Unit tests for all services
- [x] Logging - Structured logging with slog
- [x] Error handling - Comprehensive apperror package
- [x] Performance optimization - Row locking, batch queries
- [x] Documentation - Swagger, README, grpcurl examples

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
grpcurl -plaintext localhost:50051 list product.v1.ProductService
```

### gRPC Examples

```bash
# Login
grpcurl -plaintext -d '{"username":"admin","password":"admin123"}' \
  localhost:50051 auth.v1.AuthService/Login

# Create Product
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  -d '{"name":"Coffee","selling_price":25000,"sku":"COF001"}' \
  localhost:50051 product.v1.ProductService/CreateProduct

# List Products
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  localhost:50051 product.v1.ProductService/ListProducts

# Create Sale (Checkout)
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  -d '{"store_id":"uuid","items":[{"product_id":"uuid","quantity":2,"unit_price":25000}],"payment_method":"CASH","amount_paid":60000}' \
  localhost:50051 sale.v1.SaleService/CreateSale

# Get Daily Sales Report
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  'localhost:50051/report.v1.ReportService/GetDailySalesReport?start_date=2026-01-01&end_date=2026-12-31'
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

## API Documentation

### Swagger UI

Run the documentation server:

```bash
make docs
```

Then open http://localhost:8080 in your browser.

### gRPC Examples

Visit http://localhost:8080/examples for grpcurl examples.

### REST API (via gRPC-Gateway)

REST endpoints are available through gRPC-Gateway:

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'

# List users
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## License

MIT
