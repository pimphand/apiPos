# Frontend Specification — POS (Vue.js + Astro + shadcn/ui)

> Frontend guide for the POS application. This is the **frontend** counterpart to `AGENTS.md`.
> It consumes the **Go + gRPC backend** described in `AGENTS.md`, exposed through the
> **grpc-gateway REST API (HTTP)**.

> ⚠️ **WORK IN PROGRESS — FOR REVIEW.** This document is a draft. The user is reviewing
> it manually before implementation. Nothing below is implemented yet.

---

## 1. Overview

The frontend is a **Point of Sale (POS)** web application built with:

- **Astro 5.x** — island architecture / shell / static rendering
- **Vue.js 3.x** — interactive UI islands (POS checkout, dashboards, admin CRUD)
- **shadcn/ui (Vue port)** — accessible, high-quality UI components
- **Pinia** — Vue state management
- **Vue Router** — client-side routing for interactive islands
- **TanStack Query (Vue)** — server state, caching, optimistic updates
- **Axios / fetch** — REST client to the Go gateway

It talks to the backend REST API (grpc-gateway) served by the Go app, **not** directly over
gRPC (browsers cannot speak gRPC natively).

---

## 2. Backend API Contract (from Go backend)

The frontend consumes the following Go backend. All details are authoritative from `AGENTS.md` /
`internal/handler/*` / `proto/**`.

### 2.1 Connectivity

| Protocol | Port | Client |
|----------|------|--------|
| gRPC | `50051` | server-to-server only (**not** used by browser) |
| HTTP/REST (grpc-gateway) | `8990` | **Frontend uses this** |

The Browser ↔ Backend boundary is **strictly HTTP** (REST/JSON via grpc-gateway). A small
frontend proxy/`vite` dev-server proxy forwards `/api/v1/*` to the gateway to avoid CORS during
development.

**Environment Variables:**
```bash
GRPC_PORT=50051      # gRPC server port
HTTP_PORT=8990       # REST API port (grpc-gateway)
```

### 2.2 Auth endpoints

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| POST | `/api/v1/auth/register` | public | `{username,email,password,full_name,store_name,device_id}` → **auto-creates a new store** (using the required **`store_name`**) and assigns the **`admin`** role + that `store_id` |
| POST | `/api/v1/auth/login` | public | `{username,password,device_id}` → `{access_token,refresh_token,expires_at,user}` |
| POST | `/api/v1/auth/refresh` | public | `{refresh_token}` → new token pair |
| POST | `/api/v1/auth/logout` | Bearer | revokes refresh token |

> **Multi-tenant registration:** Registering a new user **auto-creates an inactive/pending store**
> and assigns the registering user the **`admin`** role for that store (`store_id` set in the
> returned user/token). The store name is a **required** field on the register form (`store_name`).
> That store remains `inactive` until a `superadmin` verifies it (see the **Stores** module). Do
> **not** send a `store_id` on register — it is derived server-side.

**Token Strategy:**
- **Access token** (TTL 15m): JWT in `Authorization: Bearer <token>` header
- **Refresh token** (TTL 30d): Returned in JSON response body

**Security Note:** Since gRPC-gateway doesn't support HttpOnly cookies, the frontend must:
1. Store refresh token in **memory only** (Pinia store + module variable)
2. Never persist to localStorage/sessionStorage (XSS risk)
3. Re-authenticate user if page reloads (refresh token lost)
4. Use short-lived access tokens for API calls

> ⚠️ **Trade-off:** This approach requires re-login on page refresh. For better UX,
> consider implementing a custom HTTP middleware for cookie-based auth in the future.

### 2.3 Domain endpoints (all require Bearer unless noted)

All **list** endpoints support pagination: `page` (default 1), `per_page` (default 20, max 100), and
return `{ ..., total_count, page, per_page }`.

| Module | Base path | Resources |
|--------|-----------|-----------|
| Users | `/api/v1/users` | create, get, list, update, delete, update-role |
| Roles | `/api/v1/roles` | CRUD + role perms (`/{id}/permissions`) |
| Permissions | `/api/v1/permissions` | get, list, by-resource, create, update, delete |
| Categories | `/api/v1/categories` | CRUD + deactivate |
| Products | `/api/v1/products` | CRUD, search, low-stock, activate/deactivate, `?get_sku=true` |
| Product SKUs | `/api/v1/products/{pid}/skus`, `/api/v1/product-skus` | CRUD, by-barcode, by-sku, by-SKU-code |
| Inventory | `/api/v1/inventory/transactions`, `/adjust`, `/stock/{pid}`, `/low-stock` | |
| Customers | `/api/v1/customers` | CRUD, search, purchase-history |
| Sales | `/api/v1/sales` | create, get, by-invoice, list, cancel, refund |
| Payments | `/api/v1/payments`, `/api/v1/sales/{sid}/payments` | create, get, list, summary |
| Cashier | `/api/v1/cashier/shifts` | open, close, current, by-id, list, summary |
| Reports | `/api/v1/reports/*` | daily-sales, by-date, by-cashier, by-product, by-category, payment-summary, inventory-summary, low-stock |
| Activity Logs | `/api/v1/activity-logs` | list, by-id, by-user, by-resource |
| Stores | `/api/v1/stores` | list, get, **verify** — **superadmin only** (system-wide, not tenant-scoped) |

### 2.4 Product SKU Features

**SKU Lookup (for POS):**
```bash
# Lookup by barcode (fast - optimized for POS scanning)
GET /api/v1/product-skus/barcode/{barcode}

# Lookup by SKU code
GET /api/v1/product-skus/sku/{sku}

# List SKUs for a product
GET /api/v1/products/{product_id}/skus?variant=Large&color=Brown
```

**Product with SKUs:**
```bash
# Get product with SKUs included
GET /api/v1/products/{id}?get_sku=true

# Update product with SKU changes
PUT /api/v1/products/{id}
{
  "name": "Coffee Latte",
  "sku_updates": [
    {"id": "", "sku": "COF-L-001", "variant": "Large", "selling_price": 25000},
    {"id": "existing-sku-uuid", "selling_price": 20000}
  ],
  "sku_ids_to_delete": ["sku-to-delete-uuid"]
}
```

### 2.5 Stores (Superadmin) — tenant onboarding & verification

These endpoints are **system-wide** (not tenant-scoped) and restricted to the **`superadmin`** role.
They are used on a dedicated **Stores/Superadmin** admin area that manages registered stores and
approves pending tenants.

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| GET | `/api/v1/stores` | superadmin | list all stores; filters `is_verified` (`"true"`/`"false"`) and `is_active`; pagination |
| GET | `/api/v1/stores/{id}` | superadmin | get a store by ID |
| POST | `/api/v1/stores/{id}/verify` | superadmin | approve a pending store (`is_verified=true, is_active=true`) |

**Store status semantics**
- A store registered via `/auth/register` is created `is_active=false, is_verified=false` (pending).
- Only a `superadmin` can verify it (`POST /verify`), which activates it.
- A non-superadmin receives `403 PERMISSION_DENIED` on all `/stores` calls.

**Store object (relevant fields):**
```json
{
  "id": "uuid",
  "name": "Acme Store",
  "address": "123 Main St",
  "phone": "+6281234567890",
  "is_active": "false",
  "is_verified": "false",
  "created_at": "2026-08-30T04:57:55Z",
  "updated_at": "..."
}
```

### 2.6 Core business rules the FE must respect

- **POS checkout is atomic** server-side. The FE submits the whole sale in one request
  (`CreateSale`). Do **not** decrement stock client-side or partially save.
- Payment must be sufficient: server rejects if `total paid < grand_total`; will not validate the
  `change` provided by FE — compute `change = amount_paid - grand_total` for display only.
- Sale statuses: `PENDING`, `PAID`, `CANCELLED`, `REFUNDED`.
- Payment methods: `CASH`, `CARD`, `BANK_TRANSFER`, `EWALLET`, `OTHER`.
- Inventory types: `PURCHASE`, `SALE`, `RETURN`, `ADJUSTMENT`, `DAMAGE`, `TRANSFER`.
- Roles: `superadmin`, `admin`, `manager`, `cashier` (default roles must never be deleted;
  `superadmin` is the system-wide role that manages admins and **verifies/approves stores**).
- Permissions format: `{resource}.{action}` (e.g. `products.create`, `sales.read`).
- **Store multi-tenancy:** every tenant gets its own `store_id`. The backend automatically scopes
  all data by the authenticated user's `store_id` (per-token). The FE must **never** try to switch
  or spoof a `store_id` — it is carried in the JWT and derived server-side. Role/permission-driven
  UI must show a tenant only its own data.
- UUID primary keys throughout (string in JSON).

### 2.7 Error contract

Backend returns gRPC status errors mapped to HTTP by the gateway. The FE must interpret
gRPC-style error codes to drive UI (toasts, form errors, redirects):

| gRPC code | HTTP | FE action |
|-----------|------|-----------|
| `UNAUTHENTICATED` | 401 | clear tokens → redirect to login |
| `PERMISSION_DENIED` | 403 | show "not authorized", offer nothing sensitive |
| `INVALID_ARGUMENT` | 400 | field-level form errors |
| `NOT_FOUND` | 404 | show not-found page/toast |
| `ALREADY_EXISTS` | 409 | show "already exists" inline |
| `INTERNAL` | 500 | generic error toast (no raw DB details leaked) |

---

## 3. Backend Phases & Features Reference

The backend has been fully implemented across 9 phases. The frontend must cover **all** these modules.

### Phase 1: Project Foundation

| Component | Status | Details |
|-----------|--------|---------|
| Go project | ✅ | `cmd/server/main.go` entry point |
| Docker Compose | ✅ | PostgreSQL + pos-app containers |
| gRPC server | ✅ | Port `50051` (internal) |
| HTTP gateway | ✅ | Port `8990` (frontend uses this) |
| Configuration | ✅ | Environment variables, typed config package |
| Health check | ✅ | `/api/v1/health` endpoint |

### Phase 2: Database & ORM

| Component | Status | Details |
|-----------|--------|---------|
| GORM models | ✅ | `internal/database/gorm/models.go` |
| Migrations | ✅ | All tables with indexes, constraints |
| Repository layer | ✅ | `internal/repository/*_gorm_repository.go` |

**Tables:** users, roles, stores, categories, products, product_skus, customers, sales, sale_items, payments, inventory_transactions, cashier_shifts, refresh_tokens, activity_logs, permissions, role_permissions

### Phase 3: Authentication & Users

| Module | Endpoints | Features |
|--------|-----------|----------|
| Auth | 3 | Login, register, refresh token |
| Users | 5 | CRUD + role assignment |
| Roles | 6 | CRUD + permission assignment |
| Permissions | 5 | CRUD + list by resource |
| Activity Logs | 4 | List, get, by-user, by-resource |

**Auth Features:**
- JWT access token (15m TTL)
- Refresh token (30d TTL)
- Password hashing (bcrypt)
- **Store-scoped multi-tenancy** (every request auto-filtered by `store_id`)
- Role-based authorization (`superadmin`/admin/manager/cashier)
- **Register auto-creates a pending store + assigns `admin` role**
- Activity logging for all operations

### Phase 4: Categories & Products

| Module | Endpoints | Features |
|--------|-----------|----------|
| Categories | 5 | CRUD + deactivate |
| Products | 8 | CRUD + search + activate/deactivate + low-stock |
| Product SKUs | 8 | CRUD + barcode/SKU lookup + variant management |

**Product SKU Features:**
- One-to-many: 1 product → many SKUs
- Variant fields: size, color, variant, unit
- Price fields: cost_price, selling_price per SKU
- Stock fields: stock, minimum_stock per SKU
- Tracking: expiry_date, batch_number, location
- POS optimization: barcode lookup endpoint
- Conditional loading: `?get_sku=true` parameter

**Frontend Product Pricing Behavior:**
- The product **create/edit** form no longer has base **Cost price / Selling price** inputs.
  Prices are entered **per SKU** only (in the "SKUs / Variants" section).
- On save, the product-level `cost_price` / `selling_price` are derived server-side from the
  first active SKU so the backend contract is unaffected.
- The **Products list** "Selling Price" column shows the SKU price range:
  - exactly **1 SKU** → single selling price (e.g. `Rp 10.000`)
  - **more than 1 SKU** → lowest–highest range (e.g. `Rp 8.000 – Rp 12.000`)
  - no valid SKU price → falls back to the product-level `selling_price`
- The list/search calls pass `?get_sku=true` so the SKU prices are available to compute the range.

### Phase 5: Inventory

| Module | Endpoints | Features |
|--------|-----------|----------|
| Inventory | 6 | Transactions, adjust, stock summary, low-stock |

**Inventory Features:**
- Transaction types: PURCHASE, SALE, RETURN, ADJUSTMENT, DAMAGE,_TRANSFER
- Atomic stock updates with row locking (`FOR UPDATE`)
- Stock summary per product with all SKUs
- Low stock alerts
- User tracking for all adjustments

### Phase 6: Customers & Sales

| Module | Endpoints | Features |
|--------|-----------|----------|
| Customers | 8 | CRUD + search + purchase history |
| Sales | 6 | Create, get, list, cancel, refund |

**Sales Features:**
- Atomic checkout flow (all-or-nothing)
- Row-level locking prevents race conditions
- Automatic inventory transactions per item
- Invoice number generation
- Sale statuses: PENDING, PAID, CANCELLED, REFUNDED
- Cancel/Refund with stock restoration

### Phase 7: Payments & Cashier Shifts

| Module | Endpoints | Features |
|--------|-----------|----------|
| Payments | 5 | Create, get, list, delete, summary |
| Cashier | 6 | Open, close, current, list, get, summary |

**Payment Features:**
- Methods: CASH, CARD, BANK_TRANSFER, EWALLET, OTHER
- Multiple payments per sale
- Payment summary with remaining/change
- Auto-update sale status when fully paid

**Cashier Shift Features:**
- Open/close shift management
- Cash reconciliation (opening + sales - refund = expected)
- Cash difference calculation
- Shift summary with sales list

### Phase 8: Reports

| Module | Endpoints | Features |
|--------|-----------|----------|
| Reports | 8 | 8 report types with date/cashier/product filters |

**Report Types:**
1. Daily sales report
2. Sales by date range
3. Sales by cashier
4. Top selling products
5. Sales by category
6. Payment method summary
7. Inventory summary by category
8. Low stock products

**Report Parameters:**
- `start_date` / `end_date` (YYYY-MM-DD)
- `store_id` (filter by store)
- `cashier_id` (filter by cashier)
- `limit` (default: 10)

### Phase 9: Testing & Documentation

| Component | Status | Details |
|-----------|--------|---------|
| Unit tests | ✅ | 98 tests across 8 test files |
| Structured logging | ✅ | `slog` in middleware |
| Error handling | ✅ | `apperror` package with PG error mapping |
| Swagger docs | ✅ | `docs/swagger-enhanced.json` |
| README | ✅ | Setup instructions + grpcurl examples |

### Complete API Endpoint Summary

| Module | Count | Base Path |
|--------|-------|-----------|
| Auth | 3 | `/api/v1/auth` |
| Users | 5 | `/api/v1/users` |
| Roles | 6 | `/api/v1/roles` |
| Permissions | 5 | `/api/v1/permissions` |
| Categories | 5 | `/api/v1/categories` |
| Products | 8 | `/api/v1/products` |
| Product SKUs | 8 | `/api/v1/product-skus` |
| Customers | 8 | `/api/v1/customers` |
| Sales | 6 | `/api/v1/sales` |
| Payments | 5 | `/api/v1/payments` |
| Cashier | 6 | `/api/v1/cashier` |
| Inventory | 6 | `/api/v1/inventory` |
| Reports | 8 | `/api/v1/reports` |
| Activity Logs | 4 | `/api/v1/activity-logs` |
| Stores | 3 | `/api/v1/stores` |
| **Total** | **86** | |

### Frontend Coverage Requirements

The frontend must implement UI for **all 83 endpoints** across these functional areas:

| Frontend Area | Backend Modules | Priority |
|---------------|----------------|----------|
| **Authentication** | Auth | P0 (must have) |
| **POS Checkout** | Sales, Payments, Product SKUs, Cashier | P0 |
| **Dashboard** | Reports | P1 |
| **Product Management** | Products, Product SKUs, Categories | P1 |
| **Inventory Management** | Inventory | P1 |
| **Customer Management** | Customers | P1 |
| **User Management** | Users, Roles, Permissions | P2 |
| **Tenant / Store Management (Superadmin)** | Stores | P2 |
| **Cashier Management** | Cashier | P2 |
| **Reports & Analytics** | Reports | P2 |
| **Activity Logs** | Activity Logs | P3 |

---

## 4. Project Structure

```text
fe/
├── astro.config.mjs
├── package.json
├── tsconfig.json
├── .env.example
├── .env.development
├── .env.production
├── .gitignore
├── Dockerfile
├── nginx.conf
├── vitest.config.ts
├── playwright.config.ts
├── public/
│   ├── favicon.svg
│   └── (static assets, PWA)
├── src/
│   ├── layouts/
│   │   ├── BaseLayout.astro
│   │   ├── AuthLayout.astro
│   │   └── DashboardLayout.astro
│   ├── pages/
│   │   ├── index.astro            # landing / redirect to dashboard
│   │   ├── login.astro
│   │   ├── register.astro
│   │   ├── dashboard.astro
│   │   ├── pos.astro              # POS checkout island (full-screen)
│   │   ├── reports.astro
│   │   ├── [admin]/
│   │   │   ├── [...path].astro   # dynamic admin prefix; resolves from PUBLIC_ADMIN_PATH_PREFIX
│   │   │   ├── users.astro
│   │   │   ├── roles.astro
│   │   │   ├── permissions.astro
│   │   │   ├── stores.astro          # superadmin: list + verify pending stores
│   │   │   ├── categories.astro
│   │   │   ├── products.astro
│   │   │   ├── inventory.astro
│   │   │   ├── customers.astro
│   │   │   ├── sales.astro
│   │   │   ├── payments.astro
│   │   │   └── activity-logs.astro
│   ├── components/                # shadcn/ui + domain components
│   │   ├── ui/                    # generated shadcn/ui components
│   │   │   ├── button/
│   │   │   ├── input/
│   │   │   ├── dialog/
│   │   │   ├── toast/
│   │   │   ├── table/
│   │   │   ├── card/
│   │   │   └── ...
│   │   ├── auth/
│   │   ├── layout/                # sidebar, topbar, mobile nav
│   │   ├── pos/
│   │   │   ├── PosCheckout.vue
│   │   │   ├── ProductGrid.vue
│   │   │   ├── CartPanel.vue
│   │   │   ├── PaymentModal.vue
│   │   │   └── BarcodeScanner.vue
│   │   └── ...
│   ├── lib/
│   │   ├── http.ts                # Axios instance + interceptors
│   │   ├── auth.ts                # token helpers
│   │   ├── rbac.ts                # permission checks
│   │   ├── utils.ts               # cn(), formatters (currency, date)
│   │   └── validators.ts          # Zod schemas
│   ├── stores/                    # Pinia
│   │   ├── auth.ts
│   │   ├── cart.ts
│   │   ├── cashier.ts
│   │   └── ui.ts
│   ├── composables/
│   │   ├── useProducts.ts
│   │   ├── useCart.ts
│   │   ├── usePermission.ts
│   │   ├── useToast.ts
│   │   └── usePagination.ts
│   ├── services/                  # API clients per domain
│   │   ├── auth.ts
│   │   ├── user.ts
│   │   ├── product.ts
│   │   ├── sale.ts
│   │   ├── report.ts
│   │   ├── store.ts              # list, get, verify stores (superadmin)
│   │   └── ...
│   ├── types/                     # TypeScript types mirroring proto messages
│   │   ├── auth.ts
│   │   ├── product.ts
│   │   ├── sale.ts
│   │   └── index.ts
│   ├── middleware/                # Astro middleware
│   │   ├── index.ts               # orchestrator
│   │   ├── auth.ts
│   │   ├── rbac.ts
│   │   ├── security.ts            # security headers / CSP
│   │   ├── request-id.ts
│   │   └── error.ts
│   └── styles/
│       └── globals.css
└── tests/
    ├── unit/
    └── e2e/
```

---

## 5. Security Hardening (no FE loopholes)

This section is **mandatory**. Every frontend security concern must be addressed.

### 4.1 Token & session security

- **Access token** (15m): memory only (`Pinia` + module variable). Never persist in
  `localStorage`/`sessionStorage` to avoid XSS theft.
- **Refresh token** (30d): `HttpOnly` + `Secure` + `SameSite=Strict` **cookie**, set by backend
  (or by a secure proxy), never accessible by JS.
- **CSRF note:** because the refresh cookie is not `SameSite=None`, cross-site requests cannot
  attach it; add a CSRF token header check anyway for defense-in-depth if cookies are used for
  state-changing requests.
- **Auto-refresh:** an Axios interceptor detects 401 → calls `/auth/refresh` once → retries the
  original request. Queue concurrent requests during refresh. On refresh failure → logout.
- **No tokens in URLs / query strings / logs.**
- **Logout** must call `/auth/logout` (revokes refresh on server) then purge local state.
- **Idle timeout:** auto logout after N minutes of inactivity (configurable).
- **Session fixation:** always use a fresh token pair after login; never trust client-stored role.

### 4.2 RBAC (server is source of truth)

- Enforce **permissions on every route** (navigation guard) and **on every UI action** (v-if).
- **Never gate security on the client alone.** Client permissions are only for UX.
  The Go backend authorizes every request (`PERMISSION_DENIED` must be handled gracefully).
- Map roles → permissions from the backend `/roles/{id}/permissions`; cache in Pinia, refresh on
  login and periodically.
- Central helper: `can(user, 'products.create')` used consistently everywhere.

### 4.3 Input validation & sanitization

- Validate **all inputs client-side** with **Zod** (mirror backend rules: quantity > 0, price >= 0,
  email format, required fields, pagination bounds).
- Treat all backend responses as **untrusted**:
  - Never `v-html` raw HTML (XSS).
  - Encode/sanitize any string rendered from API data.
  - Do not render `metadata` / JSONB fields as HTML.
- Reject oversized uploads, cap `per_page`, set timeouts on all requests.

### 4.4 Headers / CSP (shipped via middleware)

Set in middleware and nginx:
- `Content-Security-Policy` (strict, no `unsafe-inline`/`unsafe-eval` in prod for scripts).
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Strict-Transport-Security` (prod / TLS)
- `Permissions-Policy`

### 4.5 Dependencies & supply chain

- Lockfiles committed (`pnpm-lock.yaml`).
- Audit deps in CI (`pnpm audit`), no outdated Security advisories.
- Keep shadcn/ui components only as needed; no unknown transitive scripts.

### 4.6 Secret handling

- **No secrets in the FE bundle.** `VITE_*` env vars are public — only non-secret config.
- API base URL via `PUBLIC_API_BASE_URL` (Astro) / `VITE_API_BASE_URL` (Vue).
- Never commit `.env*` (except `.env.example`).

### 4.7 Authentication bypass / middleware ordering

- Auth must run **before** any protected page renders (see §6).
- Never render protected data before auth check completes (no flash of unauthorized content).
- Server-side (Astro middleware) validation is prefered; client guards are defense-in-depth.

### 4.8 Rate limiting / abuse

- Rely on backend rate limiting for login/refresh; FE should debounce and disable duplicate
  submits (e.g., double-click on checkout).

---

## 6. shadcn/ui Usage

- Use the official **shadcn-vue** port (`shadcn-vue.com`).
- Components live under `src/components/ui`.
- Configure `components.json` (TypeScript, path alias `@/*`, CSS variables theming, dark mode).
- Standard palette via Tailwind v4.
- Summary of used components (add as needed):

  `button, input, label, form/field, card, dialog, sheet, drawer, table, tabs, badge, avatar,
  dropdown-menu, select, checkbox, radio-group, switch, textarea, tooltip, toast/sonner,
  skeleton, popover, sheet, command, separator, scroll-area, progress, tabs, alert, alert-dialog`

---

## 7. Routing & Middleware

### 6.1 Astro middleware (`src/middleware/index.ts`)

Order (first → last):
1. `request-id` — attach `X-Request-ID` (mirror backend).
2. `security` — emit security headers/CSP.
3. `auth` — verify access token; if absent/expired, attempt refresh via cookie; else redirect to
   `/login?next=...`. Attach `locals.user` + `locals.session`.
4. `rbac` — for protected routes, check required permission(s). Deny → 403 page.
5. `error` — catch errors → friendly error page / toast.

Astro middleware decides **page access**; Vue islands enforce **action-level** permissions.

### 6.2 Route → permission map (example)

| Route | Required permission |
|-------|---------------------|
| `/dashboard` | authenticated |
| `/pos` | `sales.create` |
| `/reports` | `reports.read` |
| `/admin/users` | `users.read` |
| `/admin/roles` | `roles.read` |
| `/admin/stores` | **role `superadmin`** (store approval is superadmin-only) |
| `/admin/inventory` | `inventory.adjust` / `inventory.read` |

### 6.2.1 Obfuscation of admin URL (hide admin routes)

Do **not** expose predictable `/admin/*` paths that make the admin surface trivial to discover
(no security by obscurity, but it reduces probing/scanning noise and accidental links):

- Use a **hard-to-guess prefix** for admin/dashboard routes generated from a secret at
  build/deploy time stored in `PUBLIC_ADMIN_PATH_PREFIX` (env). Example:
  `/.ad/8f3a-c1d2/.../users` instead of `/admin/users`.
- The prefix is **randomized per deployment** and **never committed**; it lives only in
  the deployed env. A user without the prefix cannot form a valid admin URL.
- Actual authorization is **still enforced server-side** (middleware + RBAC in §6.1, and the Go
  backend). The obfuscated URL is **defense-in-depth only** — never a substitute for
  permission checks.
- Official links are never hard-coded; the app builds the admin URL at runtime from
  `PUBLIC_ADMIN_PATH_PREFIX`. Keep the prefix out of static HTML source where possible.
- Log abnormal attempts on the canonical/admin prefix (fail/403) for anomaly detection.

| Route (effective) | Required permission |
|-------------------|---------------------|
| `${PUBLIC_ADMIN_PATH_PREFIX}/users` | `users.read` |
| `${PUBLIC_ADMIN_PATH_PREFIX}/roles` | `roles.read` |
| `${PUBLIC_ADMIN_PATH_PREFIX}/stores` | role `superadmin` |
| `${PUBLIC_ADMIN_PATH_PREFIX}/inventory` | `inventory.adjust` / `inventory.read` |

### 6.3 Auth guard in Vue (defense-in-depth)

A router `beforeEach` guard checks `useAuthStore().isAuthenticated` and permissions before
mounting any protected island, syncing with `locals.user` from Astro when available.

---

## 8. API Client (TypeScript types mirror proto)

`src/lib/http.ts` is the single Axios instance:

```ts
// Applies:
//  1. baseURL from env
//  2. Authorization: Bearer <accessToken> (memory) when available
//  3. request interceptor -> CSRF-safe (same-origin cookie refresh)
//  4. response interceptor -> error mapping + 401 refresh flow
```

Domain services (`src/services/*`) are thin typed wrappers, e.g.:

```ts
// services/sale.ts
export async function createSale(payload: CreateSaleInput): Promise<SaleDetail> {
  return http.post('/api/v1/sales', payload).then(r => r.data)
}
```

Types mirror the proto messages 1:1 (see `proto/**` and `gen/proto` for field names). Keep a
`types/` module in sync with backend changes.

Error mapping helper (`lib/http.ts`) converts backend gRPC-style errors → typed
`{ code, message, status }` for UI.

---

## 9. Core Flows

### 8.1 Login / Refresh

```
GET /login (public)
  |-- user submits {username,password,device_id}
  |-- POST /api/v1/auth/login -> { access_token, refresh_token, expires_at, user }
  |-- access_token kept in memory; refresh_token -> HttpOnly cookie (via backend/proxy)
  |-- push /dashboard
  `-- subsequent requests attach Bearer access_token

On 401 -> interceptor -> POST /api/v1/auth/refresh -> new pair -> retry queue
```

**Register (self-serve tenant onboarding — store name is required)**
```
POST /api/v1/auth/register  {username,email,password,full_name,store_name,device_id}
  |-- validates store_name (required, max 100 chars) client-side + server-side
  |-- backend auto-creates a NEW store (inactive/pending) named `store_name` + assigns the `admin` role
  |-- returns token pair + user (with the new store_id bound to the token)
  `-- UX: tell the user their store is pending; it will work once superadmin verifies it
```

### 8.2 POS Checkout (atomic)

```
1. Open shift (if not already) — cashier module
2. Scan barcode -> GET /api/v1/product-skus/barcode/{barcode}
3. Add to cart (client-side only, quantity > 0, stock check display)
4. Select customer (optional; search customers)
5. Apply discount/tax rate (display only; server recomputes authoritative totals)
6. Select payment method(s); for CASH compute change = paid - grand_total (display)
7. POST /api/v1/sales  (one atomic request — all items, payments, amounts)
8. On success: show receipt / clear cart; on failure: show error, cart unchanged
```

### 8.3 Roles & Permissions admin

```
1. List roles (GET /api/v1/roles)
2. Assign permissions to a role (POST /api/v1/roles/{id}/permissions)
3. Default roles (superadmin/admin/manager/cashier) cannot be deleted — UI must disable delete
```

### 8.5 Store onboarding & verification (superadmin)

```
1. Login as a `superadmin` (system-wide account seeded by the backend)
2. Stores area: GET /api/v1/stores?is_verified=false&is_active=false   -> pending tenants
3. Review a pending store: GET /api/v1/stores/{id}
4. Approve: POST /api/v1/stores/{id}/verify   -> store becomes is_verified=true, is_active=true
5. Refresh the list (pending -> verified). Tenant admin can now operate.
   `-- sessions issued before verification remain valid but the store must be verified to be operational
```

### 8.4 Reports

Reusable table islands with date-range pickers, pagination, and summary cards fed by the
`ReportService` endpoints. Export to CSV/PDF client-side.

---

## 10. State Management (Pinia)

| Store | Purpose |
|-------|---------|
| `auth` | current user, access token (memory), role, permissions, `store_id`, idle timer |
| `cart` | POS line items, totals, discount/tax inputs |
| `cashier` | current shift info, opening/expected/actual cash |
| `ui` | theme, sidebar state, toasts, loading |
| `locale` | current language (`en` / `id`), `t()` translator, language toggling |

---

## 10.1 Bilingual UI (i18n / Localization)

The frontend is **bilingual**: **English** (`en`) and **Indonesian** (`id`). Users switch languages
from a toggle in the **topbar** (shows `ID` in English mode and `EN` in Indonesian mode). The
choice is **persisted to `localStorage`** (`locale` key) and applied on next load.

### Locale store

`src/stores/locale.ts` (a Pinia store):
- `locale` — current locale (`'en'` or `'id'`), seeded from `localStorage`, default `'en'`.
- `t(key, params?)` — returns the localized string for the current locale.
- `setLocale(locale)` / `toggleLocale()` — switch language and persist to `localStorage`.

### Translation dictionaries

`src/i18n/messages.ts` defines a nested dictionary for **both** locales:

```ts
export const messages: Record<Locale, NestedDict> = {
  en: { sidebar: { dashboard: 'Dashboard', ... }, ... },
  id: { sidebar: { dashboard: 'Dashboard', products: 'Produk', ... }, ... },
};

// Resolver
translate(locale, 'sidebar.products') // -> 'Products' (en) / 'Produk' (id)
```

Keys are **dot-delimited** and grouped by feature (e.g. `auth.login.welcome`,
`dashboard.totalSales`, `admin.products.noData`). Nullable placeholders use `{name}` and are
interpolated via the `params` argument, e.g. `t('dashboard.welcome', { name: userName })`.

### Usage in components

```vue
<script setup lang="ts">
import { useLocaleStore } from '@/stores/locale';
const t = useLocaleStore().t;
</script>

<template>
  <h2>{{ t('admin.products.title') }}</h2>
  <p>{{ t('admin.products.deleteDescription', { name: product.name }) }}</p>
</template>
```

`t` keeps the component reactive to language switches because it reads the `locale` ref. Data
values (API responses, names, SKU codes, permission/resource enums like `p.name`, `r.name`,
`CREATE`, `products.create`) are **never** run through `t()`.

### Coverage

All user-facing text is localized:
- **Shell & navigation** — sidebar groups/items, topbar title/profile/logout, dashboard
  heading/stat cards/quick actions.
- **Auth** — login and register cards (including the required `store_name` field and its
  pending-store hint).
- **Module placeholder pages** — title/description/hint passed via an `i18nKey` prop
  (e.g. `i18nKey="reports"`) mapped to `modules.<key>.*`.
- **Admin CRUD pages** — Categories, Products (+ SKU form), Inventory, Users, Roles,
  Permissions, and Activity Logs: page titles, descriptions, filters, table headers,
  action buttons, dialogs, delete/activate confirmations, toasts, and placeholders.
- Common strings (`Cancel`, `Save`, `Delete`, `Loading...`, `Filter`, `Active`/`Inactive`,
  etc.) live under `admin.shared.*` and are reused across all admin pages.

### Tests

`tests/unit/i18n.test.ts` verifies:
- deep key resolution for `en` and `id`,
- fallback to English for unknown locales,
- parameter interpolation,
- **key parity** — every key present in `en` must also exist in `id` (and vice-versa), so a
  language can never show a raw key.

### Adding a new translated string

1. Add the key (and its value) to **both** `messages.en.<group>` and `messages.id.<group>`.
2. Consume it in the component via `t('group.key', params?)`.
3. Run `npx vitest run tests/unit/i18n.test.ts` to confirm key parity, then
   `npx vue-tsc --noEmit` + `npm run build`.

---

## 11. Environment Variables (`.env.example`)

```
# Astro / public
PUBLIC_API_BASE_URL=/api            # same-origin proxy in dev; absolute in prod
PUBLIC_APP_NAME=POS
PUBLIC_SESSION_MAX_AGE=2592000      # 30d
PUBLIC_ADMIN_PATH_PREFIX=           # random admin prefix, set at deploy (never committed)

# Vue-only (prefixed VITE_ if defined at the Vue build, else use PUBLIC_)
# Refresh cookie is HttpOnly -> handled by backend; FE only needs PUBLIC_API_BASE_URL
```

**Never** put `JWT_SECRET`, DB creds, or backend secrets here.

---

## 12. Build, Run, Test

### 11.1 Scripts

```bash
pnpm install
pnpm dev        # astro dev (proxy /api -> http://localhost:8990)
pnpm build      # astro build
pnpm preview
pnpm test       # vitest + @vue/test-utils
pnpm test:e2e   # playwright
pnpm lint
pnpm typecheck
```

### 11.2 Dev proxy (`astro.config.mjs` / vite server.proxy)

```ts
server: {
  proxy: {
    '/api': { target: 'http://localhost:8990', changeOrigin: true },
  },
}
```

### 11.3 Docker / production

- `nginx.conf` serves the Astro static build and reverse-proxies `/api` + refresh-cookie
  handling to the Go gateway (port `8990`).
- Healthcheck on the FE container.

```bash
docker compose -f docker-compose.fe.yml up -d
```

---

## 13. Testing Strategy

### Unit (Vitest + Vue Test Utils)
- Cart calculations (subtotal, discount, tax, change).
- RBAC helper `can()` — role/permission matrix.
- Auth store: login, refresh, logout, idle timeout.
- Validators (Zod): quantity, price, email, pagination.
- HTTP interceptor: 401-refresh-queue behavior, error mapping.

### Component
- Payment modal: change computation, insufficient-fund rejection, multi-payment.
- Barcode scanner handling (manual + device input).
- Forms: validation states.

### E2E (Playwright)
- Full login → POS checkout → payment → receipt flow.
- Logout & token expiry → redirect to login.
- RBAC: cashier denied from admin pages (403), manager allowed sales/reports.
- Refresh-token rotation (cookie) mid-session.
- XSS attempt (inject script via customer name) → rendered safely.

### Accessibility & performance
- Lighthouse budgets; `axe` checks in E2E.
- Code-split islands; lazy-load non-POS routes.

---

## 14. Accessibility / UX

- Keyboard navigation for POS (fast item selection, shortcuts).
- Radix + shadcn a11y built in; ensure contrast, focus states, ARIA labels on custom components.
- Touch-friendly for tablet POS, but responsive for desktop admin.

---

## 15. Deliverables / Definition of Done

- Astro shell + Vue islands wired correctly.
- Single Axios client with auth refresh + error mapping.
- Pinia stores; RBAC guard + permission `can()`.
- Astro middleware auth + rbac + security headers.
- shadcn/ui components integrated theming + dark mode.
- Zod validation on all forms.
- Vitest unit + Playwright E2E for critical flows.
- nginx/Docker serving with security headers + `/api` proxy.
- README with setup + env docs.

---

## 16. Proposed Build Order

1. **Scaffold:** Astro + Vue + Tailwind + shadcn-ui + alias + CI.
2. **HTTP client & types:** mirror proto messages; error mapping; refresh flow.
3. **Auth:** login/register/logout, Pinia `auth`, middleware auth + guards.
4. **RBAC:** permissions from backend; guard + `can()`.
5. **Layout & shell:** dashboard layout, sidebar, topbar, dark mode.
6. **POS checkout:** cart, barcode, payment modal, sale submission.
7. **Admin CRUD:** users, roles, permissions, categories, products, customers.
8. **Inventory & cashier.**
9. **Reports.**
10. **Activity logs.**
11. **Tests + hardening pass + docs.**
