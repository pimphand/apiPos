# Build a POS Application Using Go and gRPC

You are a senior Go backend engineer and system architect.

Build a production-ready **Point of Sale (POS) application** using **Go, gRPC, Protocol Buffers, PostgreSQL, Docker, and Gorm**.

## 1. Technology Stack

Use:

* Go 1.24+
* gRPC
* Protocol Buffers
* PostgreSQL
* pgx
* GORM (for ORM operations)
* golang-migrate
* Docker
* Docker Compose
* JWT authentication
* Makefile
* grpcurl for API testing

Use `GORM` as the primary ORM for all database operations. Model definitions are in `internal/database/gorm/models.go`.

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
│   ├── store/
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

* Hash password
* Role-based authorization

Roles:

* Superadmin (system-wide; manages admins and verifies stores)
* Admin
* Manager
* Cashier

### Registration Flow

* Register **auto-creates a new store** and assigns the registering user the **`admin`** role for that store (`store_id` set to the new store).
* The new store is created **inactive/pending** until verified by a `superadmin`.

### Store Management (Superadmin)

Superadmin-only endpoints to manage and verify tenant stores. None of these are tenant-scoped (they operate system-wide):

```text
GET  /api/v1/stores              - List all stores (optional is_verified / is_active filters)
GET  /api/v1/stores/{id}         - Get a store by ID
POST /api/v1/stores/{id}/verify  - Verify/approve a pending store (sets is_verified=true, is_active=true)
```

* Only a `superadmin` can verify (approve) a store (`VerifyStore`).
* A store is created `is_active=false, is_verified=false` on registration and becomes operational once verified.
* `StoreService` is a superadmin-only gRPC/REST service; the `superadmin` guard is enforced in the service layer.

### Users

Features:

* Create user
* Update user
* Get user
* List users
* Delete/deactivate user
* Assign role

### Products

Product represents the main product entity. SKU, barcode, and variants are managed separately in the `product_skus` table.

Product fields:

```text
id
name
description
category_id
cost_price
selling_price
stock (total stock across all SKUs)
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
* Activate/deactivate product

### Product SKU (Variants)

Each product can have multiple SKUs/variants. This allows for:
- Different sizes, colors, flavors
- Different prices per variant
- Different stock levels per variant
- Expiry date tracking
- Batch number tracking

ProductSKU fields:

```text
id
product_id (FK to products)
sku (unique within product)
barcode (unique, for POS scanning)
variant (e.g., "Large", "Small", "Spicy")
color (e.g., "Red", "Blue")
size (e.g., "XL", "M", "S")
unit (e.g., "pcs", "kg", "ml")
cost_price
selling_price
stock
minimum_stock
weight
volume
expiry_date
batch_number
location (shelf/bin location)
image_url
is_active
metadata (JSONB for additional data)
created_at
updated_at
```

Features:

* Create product SKU
* Update product SKU
* Get product SKU by ID
* Get product SKU by barcode (POS lookup)
* Get product SKU by SKU code
* List product SKUs by product
* List all product SKUs
* Search product SKUs
* Delete product SKU
* Activate/deactivate product SKU

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

Implement comprehensive reports:

* Daily sales report
* Sales by date range
* Sales by cashier
* Sales by product (top selling)
* Sales by category
* Payment method summary
* Inventory summary by category
* Low stock products

#### Report API Endpoints

```text
GET /api/v1/reports/daily-sales         - Daily sales report
GET /api/v1/reports/sales-by-date       - Sales by date range
GET /api/v1/reports/sales-by-cashier    - Sales by cashier
GET /api/v1/reports/sales-by-product    - Top selling products
GET /api/v1/reports/sales-by-category   - Sales by category
GET /api/v1/reports/payment-summary     - Payment method summary
GET /api/v1/reports/inventory-summary   - Inventory summary
GET /api/v1/reports/low-stock           - Low stock products
```

#### Report Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| start_date | string | Start date (YYYY-MM-DD) |
| end_date | string | End date (YYYY-MM-DD) |
| store_id | string | Filter by store |
| cashier_id | string | Filter by cashier |
| limit | integer | Limit results (default: 10) |

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

### 4.1 Store-based Multi-Tenancy (SaaS)

The POS is multi-tenant at the **store** level. Every tenant-scoped table MUST have a `store_id` column that is a foreign key to `stores(id)`.

Tables that MUST include `store_id`:

```text
categories
products
product_skus
customers
sales
sale_items
payments
inventory_transactions
cashier_shifts
users
activity_logs (via the acting user's store, when available)
```

Global/system tables (NOT tenant-scoped, no `store_id`):

```text
roles
permissions
role_permissions
refresh_tokens
```

Rules:

1. **Every query** on a tenant-scoped table MUST filter by the authenticated user's `store_id`.
2. The token claims carry the user's `store_id`; services must scope queries using `store_id`.
3. `store_id` is NOT NULL for records created inside a store context (e.g. products, sales).
4. Non-administrative users (admin/manager/cashier) are always bound to a single store.
5. Add a `store_id` index on every tenant-scoped table.

### 4.2 Registration Creates a Store

- When a user registers, the system **automatically creates a new store** for them.
- The registering user is assigned the **`admin`** role for that store, and their `store_id` is set to the newly created store.
- The store is created inactive/pending until it is verified by a `superadmin`.

### 4.3 Superadmin Role

- A **`superadmin`** role exists to manage the overall system: it handles registered admins and **verifies/approves stores**.
- `superadmin` is a default/protected role and cannot be deleted.
- Only a `superadmin` can activate (verify) a pending store.

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
store.proto
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

### SQL Error Safety Rules

**CRITICAL:** Never expose raw SQL errors to API clients. Always use the `apperror` package to map errors:

1. **PostgreSQL Unique Violation (23505):** Map to `ALREADY_EXISTS` with user-friendly message:
   - `users_username` → "Username already exists"
   - `users_email` → "Email already exists"
   - `products_sku` → "Product SKU already exists"
   - `products_barcode` → "Barcode already exists"
   - `categories_name` → "Category name already exists"

2. **PostgreSQL Foreign Key Violation (23503):** Map to `INVALID_ARGUMENT` with message:
   - "Invalid category reference"
   - "Invalid product reference"
   - "Invalid role reference"
   - "Invalid store reference"

3. **PostgreSQL Connection Errors:** Map to `INTERNAL` with generic message:
   - Always use: `apperror.Wrap(apperror.ErrInternal, "Friendly message", err)`
   - Never return raw `err` to client

4. **Pattern for all handlers:**
   ```go
   result, err := s.service.DoSomething(ctx, args)
   if err != nil {
       return nil, apperror.MapError(err).GRPCStatus().Err()
   }
   ```

5. **Pattern for service layer:**
   ```go
   result, err := s.repos.Something.Create(ctx, params)
   if err != nil {
       // PostgreSQL errors are automatically caught by MapError
       // For business logic errors, return sentinel errors:
       return nil, apperror.ErrCategoryNotFound
       return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create", err)
   }
   ```

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
make gorm
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

## 13. Observability & Activity Logging

Implement:

* Structured logging
* Request ID
* gRPC request logging
* Error logging
* Recovery interceptor
* **Activity logging** - Record all user activities in the database

Use a standard Go logging package such as `slog`.

Do not log:

* passwords
* JWT secrets
* sensitive authentication data

### Activity Logging Rules

**ALL user activities MUST be recorded in the `activity_logs` table.** This provides a complete audit trail.

#### Activity Log Fields

```text
id          UUID PRIMARY KEY
user_id     UUID (nullable - for system actions)
username    VARCHAR(50)
action      VARCHAR(50) NOT NULL  -- LOGIN, LOGOUT, CREATE, UPDATE, DELETE, VIEW
resource    VARCHAR(100) NOT NULL -- users, products, categories, etc.
resource_id VARCHAR(100)         -- ID of the affected resource
description TEXT NOT NULL         -- Human-readable description
ip_address  VARCHAR(50)
user_agent  TEXT
metadata    JSONB                -- Additional context (old/new values, etc.)
created_at  TIMESTAMPTZ
```

#### Action Types

```text
LOGIN     - User login
LOGOUT    - User logout
CREATE    - Created a new resource
UPDATE    - Updated an existing resource
DELETE    - Deleted a resource
VIEW      - Viewed a resource (optional)
```

#### When to Log

1. **Authentication:** LOGIN, LOGOUT
2. **CRUD Operations:** CREATE, UPDATE, DELETE for all entities
3. **Financial Operations:** All payment and sale operations
4. **Permission Changes:** Role assignment, permission changes
5. **Stock Changes:** Inventory adjustments

#### Activity Log Implementation Pattern

```go
// In handler/service layer:
func (h *Handler) CreateProduct(ctx context.Context, req *CreateProductRequest) (*ProductResponse, error) {
    // Get user from context
    claims, _ := auth.ClaimsFromContext(ctx)
    
    // Perform the operation
    product, err := h.service.CreateProduct(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Log the activity
    h.activityLogService.CreateLog(ctx, activitylog.LogParams{
        UserID:     claims.UserID,
        Username:   claims.Username,
        Action:     "CREATE",
        Resource:   "products",
        ResourceID: product.ID.String(),
        Metadata:   fmt.Sprintf(`{"name":"%s","sku":"%s"}`, product.Name, product.SKU),
    })
    
    return product, nil
}
```

#### Activity Log Service Methods

```go
// CreateLog creates a new activity log entry
func (s *Service) CreateLog(ctx context.Context, params LogParams) (*gormdb.ActivityLog, error)

// ListLogs lists activity logs with filters
func (s *Service) ListLogs(ctx context.Context, userID, action, resource, startDate, endDate string, page, perPage int32) ([]gormdb.ActivityLog, int64, error)

// ListLogsByUser lists activity logs for a specific user
func (s *Service) ListLogsByUser(ctx context.Context, userID string, page, perPage int32) ([]gormdb.ActivityLog, int64, error)

// ListLogsByResource lists activity logs for a specific resource
func (s *Service) ListLogsByResource(ctx context.Context, resource, resourceID string, page, perPage int32) ([]gormdb.ActivityLog, int64, error)
```

#### Activity Log API Endpoints

```text
GET /api/v1/activity-logs              - List all logs (with filters)
GET /api/v1/activity-logs/{id}         - Get a specific log
GET /api/v1/activity-logs/user/{user_id} - Get logs by user
GET /api/v1/activity-logs/resource/{resource} - Get logs by resource
```

#### Example Activity Log Entries

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "username": "admin",
  "action": "CREATE",
  "resource": "products",
  "resource_id": "uuid",
  "description": "User 'admin' created products",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "metadata": `{"name":"Coffee","sku":"SKU001"}`,
  "created_at": "2024-01-01T12:00:00Z"
}
```

## 14. Pagination

**All list endpoints MUST have consistent pagination.**

### Pagination Parameters

Every list endpoint must accept these query parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number (min: 1) |
| `per_page` | integer | 20 | Items per page (min: 1, max: 100) |

### Pagination Response

Every list response must include:

```json
{
  "data": [...],
  "total_count": 100,
  "page": 1,
  "per_page": 20
}
```

### Pagination Rules

1. **Default values:** page=1, per_page=20
2. **Maximum per_page:** 100 (cap at 100)
3. **Minimum values:** page >= 1, per_page >= 1
4. **Calculate offset:** `offset = (page - 1) * per_page`
5. **Always return total_count** for client-side pagination

### Service Layer Pattern

```go
// PaginationResult holds pagination metadata
type PaginationResult struct {
    TotalCount int64
    Page       int32
    PerPage    int32
}

// ListResult holds the result of listing items
type ListResult struct {
    Items      []Model
    Pagination PaginationResult
}

func (s *Service) List(ctx context.Context, page, perPage int32) (*ListResult, error) {
    // Normalize pagination params
    if page <= 0 { page = 1 }
    if perPage <= 0 { perPage = 20 }
    if perPage > 100 { perPage = 100 }
    
    offset := (page - 1) * perPage
    
    // Get items
    items, err := s.repo.List(ctx, offset, perPage)
    
    // Get total count
    total, err := s.repo.Count(ctx)
    
    return &ListResult{
        Items: items,
        Pagination: PaginationResult{
            TotalCount: total,
            Page:       page,
            PerPage:    perPage,
        },
    }, nil
}
```

### Handler Layer Pattern

```go
func (h *Handler) List(ctx context.Context, req *ListRequest) (*ListResponse, error) {
    result, err := h.service.List(ctx, req.Page, req.PerPage)
    
    return &ListResponse{
        Items:      result.Items,
        TotalCount: result.Pagination.TotalCount,
        Page:       result.Pagination.Page,
        PerPage:    result.Pagination.PerPage,
    }, nil
}
```

### Proto Definition Pattern

```protobuf
message ListProductsRequest {
    string name = 1;
    int32 page = 2;
    int32 per_page = 3;
}

message ListProductsResponse {
    repeated Product products = 1;
    int64 total_count = 2;
    int32 page = 3;
    int32 per_page = 4;
}
```

## 15. API Documentation

Provide:

* `.proto` files
* README
* Database schema documentation
* Setup instructions
* Example grpcurl commands
* Example requests/responses
* Swagger/OpenAPI documentation

### Swagger Documentation Requirements

**Every module must have Swagger documentation in `docs/swagger-enhanced.json`.**

When implementing a new module, add the following to the swagger file:

1. **Tags:** Add a tag for the module:
   ```json
   {
     "name": "ModuleName",
     "description": "Module description"
   }
   ```

2. **Paths:** Add all endpoints with:
   - Summary and description
   - operationId
   - Security (BearerAuth for protected endpoints)
   - Request/response schemas with examples
   - Proper error responses ($ref to ErrorResponse)

3. **Definitions:** Add request/response models with:
   - Required fields marked
   - Examples for all fields
   - Proper types (string, integer, boolean, etc.)

4. **Error Response Format:**
   ```json
   {
     "code": "NOT_FOUND",
     "message": "Resource not found"
   }
   ```

### Example Swagger Entry

```json
"/api/v1/categories": {
  "get": {
    "tags": ["Categories"],
    "summary": "List Categories",
    "description": "List all categories.",
    "operationId": "category_list",
    "security": [{"BearerAuth": []}],
    "parameters": [...],
    "responses": {
      "200": {
        "description": "List of categories",
        "schema": { "$ref": "#/definitions/ListCategoriesResponse" }
      }
    }
  }
}
```

Example grpcurl command:

```bash
grpcurl \
  -plaintext \
  -d '{"sku":"SKU001","name":"Coffee","selling_price":10000}' \
  localhost:50051 \
  product.ProductService/CreateProduct
```

## 16. Development Order

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
Gorm
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
Reports (export pdf, excel)
```

### Phase 9

```text
Testing
Logging
Error handling
Performance optimization
Documentation
```

## 17. Roles & Permissions

### Roles

Implement CRUD operations for roles:

```text
CreateRole   - Create a new role
GetRole      - Get a role by ID
ListRoles    - List all roles
UpdateRole   - Update a role
DeleteRole   - Delete a role (prevent deleting default roles)
```

Default roles (cannot be deleted):
- superadmin
- admin
- manager
- cashier

`superadmin` is the system-wide role: it manages registered admins and verifies/approves stores. `admin` is the role auto-assigned to a store owner on registration.

### Permissions

Implement CRUD operations for permissions:

```text
GetPermission          - Get a permission by ID
ListPermissions        - List all permissions
ListPermissionsByResource - List permissions by resource
CreatePermission       - Create a new permission
UpdatePermission       - Update a permission
DeletePermission       - Delete a permission
```

### Role-Permission Assignment

```text
GetRolePermissions    - Get all permissions for a role
AssignPermissions     - Replace all permissions for a role
RemovePermission      - Remove a permission from a role
```

### Permission Format

Permissions follow the format: `{resource}.{action}`

Examples:
```text
users.create
users.read
users.update
users.delete
products.create
products.read
inventory.adjust
sales.create
sales.read
reports.read
```

### Permission Resources

```text
users, roles, permissions, categories, products, inventory,
customers, sales, payments, cashier, reports, activity_logs
```

### Permission Actions

```text
create, read, update, delete, assign, adjust, cancel, refund,
open_shift, close_shift, view_shift
```

## 18. Coding Rules

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
16. **NEVER expose raw SQL errors to API clients** - always use `apperror.MapError()` to convert.
17. **Update Swagger documentation** when adding new endpoints.
18. **Follow the handler-service-repository pattern** consistently.
19. **Use GORM for all database operations** - models are in `internal/database/gorm/models.go`.
20. **Always use GORM transactions** for operations that modify multiple tables.
21. **Log all user activities** - Use `activityLogService.CreateLog()` for audit trail.
22. **Use `is_active` as string in proto** - Never use `bool` for filter fields (proto3 defaults to false).

### GORM Usage Rules

**ALL database operations MUST use GORM.** Follow these patterns:

#### Model Definition
```go
type Product struct {
    ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    SKU          string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"sku"`
    Name         string     `gorm:"type:varchar(200);not null" json:"name"`
    Stock        int32      `gorm:"default:0" json:"stock"`
    IsActive     bool       `gorm:"default:true" json:"is_active"`
    CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
```

#### Create
```go
func (r *ProductRepository) Create(ctx context.Context, product *Product) error {
    return r.db.WithContext(ctx).Create(product).Error
}
```

#### Read (Single)
```go
func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
    var product Product
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error
    if err != nil {
        return nil, err
    }
    return &product, nil
}
```

#### Read (List with Pagination)
```go
func (r *ProductRepository) List(ctx context.Context, page, perPage int32) ([]Product, int64, error) {
    var products []Product
    var total int64
    
    query := r.db.WithContext(ctx).Model(&Product{})
    query.Count(&total)
    
    offset := (page - 1) * perPage
    err := query.Offset(int(offset)).Limit(int(perPage)).Find(&products).Error
    return products, total, err
}
```

#### Update
```go
func (r *ProductRepository) Update(ctx context.Context, product *Product) error {
    return r.db.WithContext(ctx).Save(product).Error
}
```

#### Delete (Soft Delete)
```go
func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
    return r.db.WithContext(ctx).Where("id = ?", id).Delete(&Product{}).Error
}
```

#### Transaction
```go
func (s *Service) Checkout(ctx context.Context, req CheckoutRequest) error {
    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // Create sale
        if err := tx.Create(&sale).Error; err != nil {
            return err
        }
        // Decrease stock
        if err := tx.Model(&Product{}).Where("id = ?", productID).
            Update("stock", gorm.Expr("stock - ?", quantity)).Error; err != nil {
            return err
        }
        return nil
    })
}
```

#### Search with Conditions
```go
func (r *ProductRepository) Search(ctx context.Context, name string) ([]Product, error) {
    var products []Product
    err := r.db.WithContext(ctx).
        Where("name ILIKE ?", "%"+name+"%").
        Where("is_active = ?", true).
        Find(&products).Error
    return products, err
}
```

## 18. Expected Deliverables

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
6. gorm
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
