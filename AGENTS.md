# Build a POS Application Using Go and gRPC

You are a senior Go backend engineer and system architect.

Build a production-ready **Point of Sale (POS) application** using **Go, gRPC, Protocol Buffers, PostgreSQL, Docker, and sqlc**.

## 1. Technology Stack

Use:

* Go 1.24+
* gRPC
* Protocol Buffers
* PostgreSQL
* pgx
* sqlc
* golang-migrate
* Docker
* Docker Compose
* JWT authentication
* Makefile
* grpcurl for API testing

Do not use a heavy ORM. Use `sqlc` for type-safe database queries.

## 2. Architecture

Use a clean and modular architecture:

```text
pos/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
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
│   └── proto/
├── db/
│   ├── migration/
│   └── query/
├── config/
├── middleware/
├── pkg/
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── sqlc.yaml
├── buf.yaml
├── buf.gen.yaml
└── go.mod
```

Separate the application into:

```text
gRPC Handler
     ↓
Service / Business Logic
     ↓
Repository
     ↓
PostgreSQL
```

Handlers must not contain business logic or direct database queries.

## 3. Core Modules

Implement the following modules.

### Authentication

Features:

* Login
* Logout
* JWT access token
* Refresh token
* Password hashing using bcrypt/argon2
* Role-based authorization

Roles:

* Admin
* Manager
* Cashier

### Users

Features:

* Create user
* Update user
* Get user
* List users
* Delete/deactivate user
* Assign role

### Products

Product fields:

```text
id
sku
barcode
name
description
category_id
cost_price
selling_price
stock
minimum_stock
is_active
created_at
updated_at
```

Features:

* Create product
* Update product
* Get product
* List products
* Search by name
* Search by SKU
* Search by barcode
* Activate/deactivate product

Barcode lookup must be optimized because it will be frequently used by the POS cashier.

### Categories

Features:

* Create category
* Update category
* List categories
* Delete/deactivate category

### Inventory

Do not modify stock without recording an inventory transaction.

Create an inventory transaction system supporting:

```text
PURCHASE
SALE
RETURN
ADJUSTMENT
DAMAGE
TRANSFER
```

Each transaction should record:

```text
id
product_id
store_id
type
quantity
reference_type
reference_id
before_stock
after_stock
created_by
created_at
```

Stock updates must be atomic and protected against race conditions.

### Customers

Features:

* Create customer
* Update customer
* Get customer
* Search customer
* Customer purchase history

### Sales

Implement the complete POS transaction flow.

Example:

```text
Create Sale
    ↓
Add Items
    ↓
Validate Product
    ↓
Validate Stock
    ↓
Calculate Subtotal
    ↓
Apply Discount
    ↓
Calculate Tax
    ↓
Calculate Grand Total
    ↓
Process Payment
    ↓
Create Sale
    ↓
Create Sale Items
    ↓
Decrease Inventory
    ↓
Create Inventory Transactions
    ↓
Commit Database Transaction
```

The entire checkout process must run inside a PostgreSQL database transaction.

Sale fields:

```text
id
invoice_number
store_id
cashier_id
customer_id
subtotal
discount
tax
grand_total
status
created_at
updated_at
```

Sale statuses:

```text
PENDING
PAID
CANCELLED
REFUNDED
```

Sale item fields:

```text
id
sale_id
product_id
quantity
unit_price
discount
subtotal
```

### Payments

Support multiple payment methods:

```text
CASH
CARD
BANK_TRANSFER
EWALLET
OTHER
```

A sale can contain one or multiple payments.

Payment fields:

```text
id
sale_id
payment_method
amount
reference
paid_at
```

For cash payment, calculate:

```text
change = amount_paid - grand_total
```

Reject payment if the total paid amount is insufficient.

### Cashier Shift

Implement cashier shift management.

Features:

* Open shift
* Close shift
* Current shift
* Shift summary
* Opening cash
* Cash sales
* Expected cash
* Actual cash
* Cash difference

Example:

```text
Opening Cash      100.00
Cash Sales        500.00
Cash Refund        50.00
Expected Cash     550.00
Actual Cash       548.00
Difference         -2.00
```

### Reports

Implement basic reports:

* Daily sales
* Sales by date range
* Sales by cashier
* Sales by product
* Sales by category
* Payment method summary
* Inventory summary
* Low stock products

## 4. Database

Use PostgreSQL.

Create migrations for:

```text
users
roles
stores
categories
products
customers
sales
sale_items
payments
inventory_transactions
cashier_shifts
refresh_tokens
```

Add appropriate:

* Primary keys
* Foreign keys
* Unique constraints
* Indexes
* Check constraints
* Created/updated timestamps

Important indexes:

```text
products.sku
products.barcode
products.name
sales.invoice_number
sales.created_at
sales.cashier_id
sales.store_id
sale_items.product_id
inventory_transactions.product_id
inventory_transactions.created_at
```

Use UUID or BIGINT consistently throughout the system. Choose one approach and use it consistently.

## 5. gRPC API

Create `.proto` files per domain:

```text
auth.proto
user.proto
product.proto
category.proto
inventory.proto
customer.proto
sales.proto
payment.proto
cashier.proto
report.proto
```

Use proper protobuf conventions.

Example:

```protobuf
service ProductService {
    rpc CreateProduct(CreateProductRequest) returns (ProductResponse);
    rpc GetProduct(GetProductRequest) returns (ProductResponse);
    rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);
    rpc UpdateProduct(UpdateProductRequest) returns (ProductResponse);
    rpc DeleteProduct(DeleteProductRequest) returns (DeleteProductResponse);
}
```

Generate Go code automatically using `buf`.

## 6. Error Handling

Use proper gRPC status codes.

Examples:

```text
INVALID_ARGUMENT
UNAUTHENTICATED
PERMISSION_DENIED
NOT_FOUND
ALREADY_EXISTS
FAILED_PRECONDITION
ABORTED
INTERNAL
```

Do not expose internal database errors directly to clients.

Return structured and meaningful errors.

## 7. Authentication Middleware

Create gRPC middleware/interceptors for:

* JWT validation
* Request ID
* Logging
* Recovery
* Authorization

Some endpoints must be public:

```text
Login
RefreshToken
```

Other endpoints require authentication.

Role restrictions should be implemented at the interceptor/service level.

Example:

```text
Admin:
    full access

Manager:
    products
    inventory
    sales
    reports

Cashier:
    sales
    customers
    current shift
```

## 8. Configuration

Use environment variables.

Example:

```env
APP_ENV=development
GRPC_PORT=50051

DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_NAME=pos
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres

JWT_SECRET=change-me
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h
```

Create a typed configuration package.

Do not hardcode credentials.

## 9. Docker

Create:

```text
Dockerfile
docker-compose.yml
```

Docker Compose should start:

```text
pos-app
postgres
```

Add PostgreSQL health checks.

The Go application must wait until PostgreSQL is ready.

## 10. Makefile

Create useful commands:

```text
make run
make build
make test
make proto
make migrate-up
make migrate-down
make sqlc
make docker-up
make docker-down
make lint
```

## 11. Testing

Write unit tests for business logic.

Especially test:

### Sales

```text
valid checkout
insufficient stock
invalid product
invalid quantity
discount calculation
tax calculation
payment validation
cash change calculation
```

### Inventory

```text
stock increase
stock decrease
insufficient stock
concurrent stock updates
inventory transaction creation
```

### Authentication

```text
valid login
invalid password
expired JWT
invalid JWT
role authorization
```

Use database integration tests where appropriate.

## 12. Transaction Safety

Checkout must be atomic.

The following operations must either all succeed or all rollback:

```text
Create Sale
Create Sale Items
Create Payment
Decrease Product Stock
Create Inventory Transaction
```

Never allow a sale to be created while inventory remains unchanged.

Use PostgreSQL transactions and appropriate row locking when necessary.

## 13. Observability

Implement:

* Structured logging
* Request ID
* gRPC request logging
* Error logging
* Recovery interceptor

Use a standard Go logging package such as `slog`.

Do not log:

* passwords
* JWT secrets
* sensitive authentication data

## 14. API Documentation

Provide:

* `.proto` files
* README
* Database schema documentation
* Setup instructions
* Example grpcurl commands
* Example requests/responses

Example:

```bash
grpcurl \
  -plaintext \
  -d '{"sku":"SKU001","name":"Coffee","selling_price":10000}' \
  localhost:50051 \
  product.ProductService/CreateProduct
```

## 15. Development Order

Do not attempt to implement everything at once.

Build incrementally in this order:

### Phase 1

```text
Project setup
PostgreSQL
Docker
gRPC
Protobuf
Health check
Configuration
```

### Phase 2

```text
Database migrations
sqlc
Repository layer
```

### Phase 3

```text
Authentication
Users
Roles
JWT
Middleware
```

### Phase 4

```text
Categories
Products
```

### Phase 5

```text
Inventory
Stock transactions
```

### Phase 6

```text
Customers
Sales
Sale items
```

### Phase 7

```text
Payments
Cashier shifts
```

### Phase 8

```text
Reports
```

### Phase 9

```text
Testing
Logging
Error handling
Performance optimization
Documentation
```

## 16. Coding Rules

Follow these rules:

1. Use idiomatic Go.
2. Keep functions small and focused.
3. Use `context.Context` for database and gRPC operations.
4. Never put SQL directly inside gRPC handlers.
5. Never put business logic inside protobuf handlers.
6. Use dependency injection.
7. Return errors instead of panicking.
8. Use database transactions for financial operations.
9. Validate all input.
10. Use parameterized SQL queries.
11. Avoid global mutable state.
12. Make concurrency-safe code.
13. Add tests for important business logic.
14. Keep interfaces small.
15. Prefer explicit code over unnecessary abstraction.

## 17. Expected Deliverables

At the end of the implementation, the project must be runnable with:

```bash
docker compose up -d
```

and the gRPC server should be available at:

```text
localhost:50051
```

Provide:

1. Complete project structure
2. Go source code
3. Protobuf definitions
4. Generated protobuf setup
5. PostgreSQL migrations
6. sqlc queries
7. Docker configuration
8. Makefile
9. Authentication middleware
10. Unit tests
11. Integration tests
12. README
13. grpcurl examples

Start with **Phase 1 only**.

Do not implement all modules immediately.

First create the Go project, Docker Compose, PostgreSQL connection, gRPC server, protobuf setup, health-check service, configuration system, and basic project structure.

After Phase 1 is complete, explain what was created, how to run it, and what should be implemented next.
