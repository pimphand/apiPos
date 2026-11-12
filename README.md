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

Start all services (FE + API + PostgreSQL + Adminer):

```bash
make docker-up
```

Stop all services:

```bash
make docker-down
```

Container `fe-app` (nginx) adalah satu-satunya pintu masuk, dan TLS berhenti di
sana juga:

| URL | Tujuan |
|---|---|
| https://localhost:18731/ | Semua path di bawah ini, via TLS edge (self-signed: pakai `curl -k`) |
| http://localhost:18730/ | Frontend (static Astro build) |
| http://localhost:18730/api/ | Go grpc-gateway |
| http://localhost:18730/adminer/ | Adminer (HTTP Basic Auth) |

Semua port host sengaja dipilih tidak lazim (bukan 80/443/5432/8080/50051)
supaya stack tidak mudah di-fingerprint scanner, dan semuanya terikat ke
`127.0.0.1`:

| Port host | Fungsi |
|---|---|
| `18730` | edge HTTP (satu-satunya pintu masuk) |
| `18731` | edge HTTPS |
| `18732` | PostgreSQL (di dalam container tetap 5432) |
| `18733` | gRPC (grpcurl di host) |
| `18734` | gateway HTTP internal (tidak dipublikasikan) |
| `18735` | server docs/swagger (`make docs`, lokal saja) |

Ubah lewat `EDGE_PORT` / `EDGE_TLS_PORT` / `EDGE_BIND` / `EDGE_TLS_BIND` /
`DATABASE_PORT` / `HOST_GRPC_PORT` di `.env`. Gateway tidak dipublikasikan
terpisah — edge mem-proxy `/api` secara internal.

`/adminer/` memakai HTTP Basic Auth dan kredensialnya diambil dari
`ADMINER_USER` / `ADMINER_PASSWORD` di `.env` (wajib diisi; container menolak
start kalau kosong).

#### TLS di edge

Edge sudah menangani HTTPS sendiri, jadi tidak perlu proxy host terpisah untuk
sertifikat:

1. Salin `fullchain.pem` dan `privkey.pem` (mis. hasil Certbot) ke folder
   `certs/` di root repo. Folder ini di-ignore git dan di-mount read-only ke
   `/etc/nginx/certs`.
2. Set `EDGE_SERVER_NAME=pos.example.com` supaya CN/SAN sertifikat self-signed
   fallback masuk akal (sertifikat asli tetap yang dipakai kalau ada).
3. Arahkan domain ke `127.0.0.1:443` (default `EDGE_TLS_PORT`). Kalau sertifikat
   belum ada, container membuat pasangan self-signed agar stack tetap jalan —
   browser akan memperingatkan, jadi jangan dipakai untuk produksi.

`curl` lokal ke sertifikat self-signed: `curl -k https://localhost/`.

Let's Encrypt tidak dijalankan di dalam container ini; perbarui sertifikat di
host (cron Certbot) lalu salin ulang ke `certs/` dan `docker compose restart
fe-app`.

#### Domain / reverse proxy di host (opsional)

Kalau domain tetap mau lewat proxy host (mis. Cloudflare Tunnel, atau proxy yang
sudah memegang sertifikat), teruskan apa adanya ke `http://127.0.0.1:18730`
(jangan menambah prefix dan jangan pakai trailing slash di `proxy_pass`);
jalur TLS edge di atas tidak dipakai. Default compose (`18730` HTTP, `18731`
TLS) sudah tidak memakai 80/443, jadi nginx host yang memegang 443 tidak
bentrok dan `EDGE_TLS_PORT` bisa dibiarkan apa adanya.

Contoh siap pakai untuk `pos.dwiki.id`: `deploy/nginx/pos.dwiki.id.conf`.

nginx:

```nginx
# di blok http
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 443 ssl http2;
    server_name pos.example.com;

    ssl_certificate     /etc/letsencrypt/live/pos.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/pos.example.com/privkey.pem;

    client_max_body_size 64m;   # impor produk / upload bukti bayar

    location / {
        proxy_pass http://127.0.0.1:18730;   # tanpa trailing slash: path apa adanya
        proxy_http_version 1.1;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket (/ws)
        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

Caddy (TLS otomatis):

```
pos.example.com {
    reverse_proxy 127.0.0.1:18730
}
```

Catatan:

- `/`, `/api`, `/ws`, dan `/adminer` diteruskan tanpa diubah; semua routing ada di
  edge. Jangan set `X-Forwarded-Prefix` sendiri — edge selalu menimpanya dengan
  `/adminer` untuk permintaan Adminer.
- TLS berhenti di proxy host. Redirect `/adminer` di edge bersifat relatif, jadi
  skema/domain tetap benar di belakang HTTPS.
- `client_max_body_size` di proxy host harus >= 64m agar impor data tidak gagal
  sebelum sampai ke edge.

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
  localhost:18733 \
  health.v1.HealthService/Check
```

### Using grpcurl

```bash
# List services
grpcurl -plaintext localhost:18733 list

# List methods for a service
grpcurl -plaintext localhost:18733 list product.v1.ProductService
```

### gRPC Examples

```bash
# Login
grpcurl -plaintext -d '{"username":"admin","password":"admin123"}' \
  localhost:18733 auth.v1.AuthService/Login

# Create Product
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  -d '{"name":"Coffee","selling_price":25000,"sku":"COF001"}' \
  localhost:18733 product.v1.ProductService/CreateProduct

# List Products
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  localhost:18733 product.v1.ProductService/ListProducts

# Create Sale (Checkout)
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  -d '{"store_id":"uuid","items":[{"product_id":"uuid","quantity":2,"unit_price":25000}],"payment_method":"CASH","amount_paid":60000}' \
  localhost:18733 sale.v1.SaleService/CreateSale

# Get Daily Sales Report
grpcurl -plaintext -H "Authorization: Bearer TOKEN" \
  'localhost:18733/report.v1.ReportService/GetDailySalesReport?start_date=2026-01-01&end_date=2026-12-31'
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
| GRPC_PORT | 18733 | gRPC server port |
| DATABASE_HOST | localhost | PostgreSQL host |
| DATABASE_PORT | 18732 | PostgreSQL host port (di dalam compose tetap 5432) |
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

Then open http://localhost:18735 in your browser (port docs sengaja terpisah
dari edge `18730`; ubah dengan `DOCS_PORT=...`).

### gRPC Examples

Visit http://localhost:18735/examples for grpcurl examples.

### REST API (via gRPC-Gateway)

REST endpoints are available through gRPC-Gateway:

```bash
# Login
curl -X POST http://localhost:18730/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'

# List users
curl http://localhost:18730/api/v1/users \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## License

MIT
