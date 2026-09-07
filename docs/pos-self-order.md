# POS — Self Order (Dine-in via QR Meja)

> Dokumen spesifikasi & panduan implementasi fitur **self order dine-in**:
> pelanggan memesan langsung dari mejanya dengan **scan QR yang ditempel di
> meja**, membayar sendiri, dan pesanan (yang sudah lunas) otomatis masuk ke
> kasir dengan **notifikasi suara** di POS.
>
> **Keputusan desain (disepakati):**
> - **Khusus dine-in** — tidak ada take-away / kiosk di lokasi.
> - **QR per meja** — ditempel di setiap meja; meja adalah entitas data yang
>   **unik per tenant** dan **tidak bisa diakses lintas tenant**.
> - **Bayar langsung oleh pelanggan** — pesanan hanya masuk ke kasir **jika
>   pembayaran sukses** (sale berstatus `PAID`).
> - **Notifikasi suara di POS** ketika ada orderan self order baru.
>
> Dokumen ini mengikuti pola `docs/pos-order-channel.md` dan mengacu pada kode
> yang ada (migrasi `000016_kitchen_print`, `sales.notes`, realtime
> `sale.created`, multi-tenancy per-store).

## 1. Ringkasan & tujuan

- Pelanggan datang ke restoran, duduk di meja, **scan QR meja** dengan HP.
- Browser mengarah ke halaman order khusus meja itu (`/o/:table_token`).
- Pelanggan memilih produk (termasuk varian/SKU), menambah catatan, lalu
  **membayar langsung** (metode non-tunai, mis. QRIS/EWALLET).
- **Jika pembayaran sukses**: sale dibuat **`PAID`** (sumber `SELF_ORDER`),
  dapur menerima Nota Dapur, dan **kasir di POS mendapat notifikasi suara**
  + daftar pesanan masuk.
- **Jika pembayaran gagal/batal**: tidak ada sale, stok tidak terpotong.

Tujuan utama:

- mengurangi antrean & beban kasir untuk pesanan dine-in sederhana;
- pesanan yang masuk ke kasir **sudah lunas** (kasir tinggal siapkan/serahkan);
- data meja aman: setiap meja terikat ke satu store, mustahil diakses tenant lain.

## 2. Keputusan desain

| # | Keputusan | Alasan |
|---|-----------|--------|
| D1 | **QR per meja** → token meja unik global (`tables.qr_token`) | QR yang ditempel hanya memuat token; tidak ada informasi tenant yang bisa dimanipulasi klien. |
| D2 | **Isolasi tenant via resolusi server-side** | Endpoint publik menerima `table_token`; store_id ditentukan **di server** dari tabel `tables`. Klien **tidak pernah** mengirim store_id → mustahil akses lintas tenant. |
| D3 | **`code` meja unique per store** (`UNIQUE(store_id, code)`) + `qr_token` unique global | Dua meja di toko yang sama tidak boleh bernomor sama; token QR dijamin unik di seluruh sistem. |
| D4 | **Pembayaran langsung pelanggan** | Sale hanya dibuat `PAID` setelah pembayaran sukses; metode tunai (CASH) **tidak** tersedia di alur self order (tidak ada kasir di meja). |
| D5 | **Dua langkah: submit (PENDING) → pay (PAID)** | Stok dikunci saat submit agar dapur tidak kehabisan; sale tidak "masuk kasir" sampai lunas. Submit tanpa bayar di-*cleanup* otomatis. |
| D6 | **`order_source='SELF_ORDER'` & `order_status` terpisah dari status pembayaran** | Laporan/shift bisa membedakan; dapur melacak ORDERED→PREPARING→READY→COMPLETED sementara `status` pembayaran tetap PAID. |
| D7 | **Reuse `sales.notes` + kitchen print (000016)** | Catatan pelanggan → Nota Dapur; toggle `show_kitchen_print` tetap berlaku. |
| D8 | **Notifikasi suara POS via event realtime** | `sale.created` memuat `order_source=SELF_ORDER` → POS memutar bunyi + toast/panel (hanya saat `status=PAID`, lihat D12). Tidak ada polling. |
| D9 | **Menu publik hanya data aman** | Tanpa cost price, stok detail (cukup flag tersedia), data internal; token meja/store tidak pernah tampil di log/UI. |
| D10 | **Manajemen meja di halaman admin (per store)** | Admin/manager CRUD meja + cetak QR; hanya meja store-nya sendiri (selalu difilter `store_id`). |
| D11 | **Sale self-order tanpa kasir → `cashier_id`/`created_by` nullable** | Tidak ada user di meja; `sales.cashier_id` & `inventory_transactions.created_by` dibuat nullable (NULL = aksi pelanggan), mengikuti pola `activity_logs.user_id`. |
| D12 | **Satu event `sale.created` hanya saat PAID** | Submit (PENDING) tidak memicu event/notifikasi apa pun; satu event `sale.created` (dengan `status=PAID`) dipublish hanya setelah pembayaran sukses → kasir tidak pernah melihat orderan belum lunas. |
| D13 | **Rate limit per IP pada endpoint publik (interceptor in-process)** | Interceptor gRPC per-IP (sliding window, mis. 10 submit + 60 pay/menit) dipasang hanya pada service `selforder.v1.*`; cukup untuk single-instance MVP, ganti ke store bersama (Redis) bila multi-instance. |
| D14 | **Self-order tidak tercatat di shift kasir** | Tanpa cashier → `sales.shift_id` NULL & tanpa baris `shift_items`; revenue self-order (selalu non-tunai) tidak ikut hitung shift, tetap muncul di laporan sales per store. |

## 3. Alur (flow)

```text
Pelanggan duduk di meja → scan QR yang ditempel
        ↓
GET /o/:table_token (atau /o?table=:table_token) → halaman order (server resolve store + meja)
        ↓
Menu publik dine-in (produk aktif + SKU tersedia; tanpa harga modal)
        ↓
Pilih item & varian → nama (opsional) + catatan (opsional) → Submit
        ↓
POST /api/v1/public/orders            → Sale dibuat PENDING + order_token
        ↓
POST /api/v1/public/orders/:order_token/pay   (QRIS/EWALLET/CARD/BANK_TRANSFER)
        ↓
Jika PAYMENT SUKSES → sale.status = PAID (+ baris payment)  [satu transaksi DB]
        ↓        Publish realtime sale.created (order_source=SELF_ORDER, status=PAID, invoice, table)
        ↓
POS kasir: NOTIFIKASI SUARA + daftar pesanan self order masuk
        Dapur: Nota Dapur otomatis / tombol
        ↓
Dapur siapkan → order_status ORDERED → PREPARING → READY → COMPLETED
        ↓
Pelanggan: layar "Pesanan Dibayar & Diterima" + nomor antrean/invoice
```

Aturan penting:

- **Stok dikurangi saat submit** (bukan saat bayar) supaya dapur tidak
  memasak barang yang sudah habis; jika bayar gagal/batal, stok dikembalikan
  otomatis oleh cleanup.
- **Tidak ada sale yang masuk kasir dalam status PENDING dari self order** —
  kasir hanya melihat orderan **lunas** (`PAID`).
- **Submit tidak memicu event realtime apa pun** — notifikasi POS & Nota
  Dapur hanya muncul lewat satu event `sale.created` saat `PAID` (D12).
- Semua operasi (menu, submit, pay, status) **hanya** untuk store yang
  ter-resolve dari token meja — klien tidak bisa memilih tenant.

## 4. Model data

### 4.1 Tabel meja baru (`000017_tables_self_order.up.sql` — usulan)

```sql
-- Meja dine-in per store. `code` unik per store; `qr_token` unik GLOBAL
-- sehingga QR meja tidak mungkin menunjuk ke tenant lain.
CREATE TABLE tables (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id   UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    code       VARCHAR(20) NOT NULL,               -- "T01", "M2", "TERAS-A"
    name       VARCHAR(100),                        -- "Meja 1", "Teras A-2"
    qr_token   VARCHAR(64) NOT NULL UNIQUE,         -- token untuk QR meja
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tables_store_code ON tables(store_id, code);
CREATE INDEX idx_tables_store_id        ON tables(store_id);
CREATE INDEX idx_tables_qr_token        ON tables(qr_token);
```

### 4.2 Migrasi kolom `sales` (bagian dari migrasi yang sama)

```sql
-- Sumber transaksi & status pemenuhan.
ALTER TABLE sales ADD COLUMN order_source VARCHAR(20) NOT NULL DEFAULT 'CASHIER'
    CHECK (order_source IN ('CASHIER', 'SELF_ORDER'));
ALTER TABLE sales ADD COLUMN order_status VARCHAR(20) NOT NULL DEFAULT 'COMPLETED'
    CHECK (order_status IN ('ORDERED', 'PREPARING', 'READY', 'COMPLETED'));

-- Meja (FK + snapshot denormalisasi agar struk tetap terbaca walau meja dihapus).
-- ON DELETE SET NULL: menghapus meja yang punya riwayat sale tidak boleh gagal.
ALTER TABLE sales ADD COLUMN table_id UUID NULL REFERENCES tables(id) ON DELETE SET NULL;
ALTER TABLE sales ADD COLUMN table_no  VARCHAR(20);

-- Token pelacakan pelanggan tanpa akun (unik per sale).
ALTER TABLE sales ADD COLUMN order_token UUID UNIQUE;

-- Tidak ada kasir/user di meja: kolom aktor dibuat nullable (NULL = aksi pelanggan),
-- mengikuti pola activity_logs.user_id.
ALTER TABLE sales ALTER COLUMN cashier_id DROP NOT NULL;
ALTER TABLE inventory_transactions ALTER COLUMN created_by DROP NOT NULL;

CREATE INDEX idx_sales_order_source ON sales(order_source);
CREATE INDEX idx_sales_order_status ON sales(order_status);
CREATE INDEX idx_sales_table_id     ON sales(table_id);

-- Toggle self order per store (di store_settings).
ALTER TABLE store_settings
    ADD COLUMN self_order_enabled BOOLEAN NOT NULL DEFAULT FALSE;
```

- `sales.status` tetap satu-satunya status pembayaran — untuk self order,
  sale dibuat `PENDING` saat submit lalu `PAID` saat bayar sukses.
- `order_status` default `'COMPLETED'` agar transaksi kasir biasa konsisten.

### 4.3 Model GORM (`internal/database/gorm/models.go`)

```go
// Table meja dine-in. TableName() -> "tables"
type Table struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    StoreID   uuid.UUID `gorm:"type:uuid;not null;index:idx_tables_store_id" json:"store_id"`
    Code      string    `gorm:"type:varchar(20);not null" json:"code"`   // uniqueIndex: idx_tables_store_code
    Name      string    `gorm:"type:varchar(100)" json:"name"`
    QRToken   string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"` // never expose
    IsActive  bool      `gorm:"default:true" json:"is_active"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Pada Sale:
OrderSource  string     `gorm:"type:varchar(20);default:CASHIER" json:"order_source"`
OrderStatus  string     `gorm:"type:varchar(20);default:COMPLETED" json:"order_status"`
TableID      *uuid.UUID `gorm:"type:uuid;index:idx_sales_table_id" json:"table_id"`
TableNo      string     `gorm:"type:varchar(20)" json:"table_no"`
OrderToken   *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"order_token"`
// CashierID tetap uuid.UUID (bukan pointer) — NULL ditulis via
// tx.Omit("CashierID") saat membuat sale self-order, mengikuti pola CustomerID
// yang sudah ada di sale/service.go. Membaca NULL menghasilkan uuid.Nil;
// effectiveActor/laporan sudah menanganinya sebagai "tanpa aktor".
CashierID    uuid.UUID `gorm:"type:uuid" json:"cashier_id"`

// Pada InventoryTransaction:
CreatedBy    uuid.UUID `gorm:"type:uuid" json:"created_by"` // NULL via tx.Omit("CreatedBy")

// Pada StoreSetting:
// Tanpa tag `default:` — ikuti konvensi yang didokumentasikan di models.go:
// GORM meng-omit field zero-value yang punya tag default saat INSERT/upsert,
// sehingga `default:false` membuat nilai false tidak pernah tersimpan.
SelfOrderEnabled bool `gorm:"not null" json:"self_order_enabled"`
```

> `QRToken` diberi `json:"-"` — tidak boleh bocor ke API admin/POS; hanya
> dipakai untuk generate QR dan resolusi endpoint publik.

## 5. Isolasi tenant (aturan wajib)

1. **Resolusi server-side**: `GET /api/v1/public/table/:table_token`
   → server membaca `tables` → `store_id` + meja. Semua respons menu & semua
   pembuatan order memakai `store_id` hasil resolusi ini — **klien tidak pernah
   mengirim `store_id`** di payload/query.
2. **Setiap query meja di sisi admin** (list/create/update/delete) wajib
   difilter `store_id = claims.StoreID` (aturan multi-tenancy §4.1 AGENTS.md).
3. `qr_token` **unique global** (constraint UNIQUE) — satu QR hanya menunjuk
   satu meja satu store; token lain tidak valid.
4. Store harus `is_verified=true && is_active=true && self_order_enabled=true`
   sebelum menerima order; meja harus `is_active=true`.
5. Rate limit per IP pada endpoint publik: interceptor gRPC baru
   `middleware/ratelimit.go` (sliding window in-process, key = client IP dari
   metadata `x-forwarded-for` atau peer) yang hanya aktif untuk method
   `/selforder.v1.SelfOrderService/*` — mis. 10 submit + 60 pay/menit.
   Cukup untuk single-instance MVP; ganti ke store bersama (mis. Redis) bila
   multi-instance.
6. Endpoint status pesanan hanya bisa diakses dengan `order_token` milik sale
   itu sendiri — tidak ada enumerasi lintas tenant.

## 6. Backend

### 6.1 Endpoint publik (tanpa JWT, scoped table_token)

Path REST di bawah berasal dari anotasi `google.api.http` di proto. Yang
**ditambahkan ke daftar `publicEndpoints` di `middleware/auth.go` adalah full
method path gRPC** (daftar itu diisi nama method gRPC, bukan path REST):

```text
REST                                      → method gRPC publik (publicEndpoints)
GET  /api/v1/public/table/:table_token    → /selforder.v1.SelfOrderService/ResolveTable
GET  /api/v1/public/menu/:table_token     → /selforder.v1.SelfOrderService/GetPublicMenu
POST /api/v1/public/orders                → /selforder.v1.SelfOrderService/CreatePublicOrder
POST /api/v1/public/orders/:order_token/pay → /selforder.v1.SelfOrderService/PayPublicOrder
GET  /api/v1/public/orders/:order_token   → /selforder.v1.SelfOrderService/GetOrderStatus
```

Semua method tersebut juga lewat interceptor rate limit per IP (§5 poin 5).

### 6.2 Service layer — `internal/selforder/service.go` (baru)

- `ResolveTable(ctx, tableToken)` → `(Table, StoreID)`; `NOT_FOUND` bila token
  tidak dikenal, `FAILED_PRECONDITION` bila store/meja nonaktif.
- `GetPublicMenu(ctx, tableToken)` — produk aktif + SKU tersedia, field aman.
- `CreatePublicOrder(ctx, tableToken, items, name, notes)`:
  1. resolve meja → scope store;
  2. validasi item (produk/SKU aktif, qty>0, qty ≤ stok);
  3. hitung subtotal/discount/tax (pakai `store_settings.tax_rate`);
  4. **satu transaksi DB**: Sale `PENDING`/`ORDERED` + `order_source=SELF_ORDER`
     + `order_token` + `table_id`/`table_no` + `notes` + SaleItems + kurangi
     stok + InventoryTransaction (reference_type=SALE). `cashier_id` dan
     `created_by` diisi NULL — ini bukan aksi user (D11);
  5. **tidak publish event realtime apa pun** saat submit (D12) — kasir/dapur
     hanya bereaksi pada event saat `PAID`;
  6. activity log CREATE sales (user_id NULL — aksi pelanggan).
- `PayPublicOrder(ctx, orderToken, paymentMethod, reference)`:
  1. ambil sale by `order_token` (wajib `order_source=SELF_ORDER`,
     `status=PENDING`, belum kadaluarsa);
  2. metode **non-tunai** saja (`CARD`, `BANK_TRANSFER`, `EWALLET`, `OTHER`);
     CASH ditolak `INVALID_ARGUMENT`;
  3. validasi `amount == grand_total` (pembayaran kurang → `FAILED_PRECONDITION`);
  4. **satu transaksi DB**: buat Payment + `status=PAID` + isi snapshot
     `table_no`; publish **satu** event `sale.created` (kasir & dapur) dengan
     `data = {order_source: "SELF_ORDER", status: "PAID", invoice_number,
     table_no}`;
  5. activity log.
- `CancelExpiredPending(ctx)` — job berkala membatalkan order PENDING > N menit
  (mis. 15) dan mengembalikan stok via alur cancel yang ada; `created_by`
  transaksi RETURN = NULL (aksi sistem), tanpa baris `shift_items` (D14).

> **StoreID pada event realtime**: endpoint publik tidak membawa JWT, jadi
> `realtime.PublishFromContext` tidak bisa mengisi `StoreID` dari claims —
> service wajib mengisi `Event.StoreID` secara eksplisit saat publish (D2/§5).

> **Shift kasir (D14)**: sale self-order tidak terhubung ke shift mana pun —
> `sales.shift_id` NULL dan tanpa baris `shift_items`. Revenue self-order
> (selalu non-tunai) tidak ikut hitung shift (`total_sales_amount`,
> `revenue_*`, `cash_sales`), tapi tetap muncul di laporan sales per store.

### 6.3 Error handling

- Token meja salah / meja nonaktif / store belum verified → `NOT_FOUND` /
  `FAILED_PRECONDITION` ("QR meja tidak valid atau layanan nonaktif").
- Produk/SKU tidak tersedia → `FAILED_PRECONDITION`.
- Stok kurang saat submit → `FAILED_PRECONDITION`.
- Pembayaran kurang → `FAILED_PRECONDITION`; metode tunai → `INVALID_ARGUMENT`.
- Tetap `apperror.MapError()` — jangan bocorkan error SQL mentah.

### 6.4 Proto

`proto/selforder/v1/selforder.proto` (baru):

```protobuf
service SelfOrderService {
  rpc ResolveTable(ResolveTableRequest) returns (TableResponse);          // GET public/table
  rpc GetPublicMenu(GetPublicMenuRequest) returns (GetPublicMenuResponse);
  rpc CreatePublicOrder(CreatePublicOrderRequest) returns (OrderResponse); // PENDING
  rpc PayPublicOrder(PayPublicOrderRequest) returns (OrderResponse);       // -> PAID
  rpc GetOrderStatus(GetOrderStatusRequest) returns (OrderResponse);
}
```

`CreatePublicOrderRequest`: `table_token`, `items[]` (product_id,
product_sku_id, quantity), `customer_name`, `notes`.
`PayPublicOrderRequest`: `order_token`, `payment_method`, `reference` (mis.
ID transaksi QRIS).

## 7. Manajemen meja (admin, per store)

- Endpoint CRUD di service store/tabel (wajib filter `store_id`):
  `POST/GET/PUT/DELETE /api/v1/tables`, `POST /api/v1/tables/:id/rotate-token`,
  `GET /api/v1/tables/:id/qr` (gambar QR / URL cetak).
- Permission baru: `tables.create/read/update/delete` (di-seed ke role
  admin/manager; lihat pola `role_permissions`).
- Halaman FE `admin/tables.astro` (baru): daftar meja (code, name, status),
  tambah/edit/hapus, tombol **Cetak QR** (PDF berisi QR per meja + label
  "Scan untuk Pesan"), dan **rotasi token** (QR lama langsung tidak berlaku).
- Unik enforced di DB: `UNIQUE(store_id, code)` → error `ALREADY_EXISTS`
  via `apperror` bila nomor meja duplikat.

## 8. Frontend

### 8.1 Halaman publik pelanggan

- Route `fe/src/pages/o/index.astro` (statis; token dibaca dari query
  `/o?table=:token` atau path `/o/:token`) + island
  `fe/src/components/selforder/SelfOrderMenu.vue` (baru). Token QR yang
  dikembalikan backend memakai bentuk query (`/o?table=...`) agar bekerja di
  hosting statis (nginx fallback + rule `location /o/`).
- Alur: resolve meja → header "Meja {table_no}" → menu dine-in → keranjang →
  nama (opsional) + catatan (opsional) → **Submit** → halaman pembayaran →
  pilih metode (QRIS/EWALLET/CARD/BANK_TRANSFER) → **Bayar** → sukses:
  layar "Pesanan Dibayar ✓ No. {invoice}" + status (via realtime/polling
  `GET public/orders/:order_token`).
- Tidak ada login; tidak menampilkan stok detail/harga modal.

### 8.2 POS kasir — notifikasi suara

- Di `POSCheckoutPage.vue` (dan/atau komponen baru `SelfOrderIncoming.vue`):
  - subscribe `sale.created` (pola yang sudah ada) → bila
    `event.data.order_source === 'SELF_ORDER' && event.data.status === 'PAID'`
    (submit PENDING tidak memicu apa pun — D12):
    - **mainkan bunyi** via Web Audio API (nada pendek berulang, mis. 3 beep;
      file audio kecil di `fe/public/sounds/self-order.mp3` atau synth `AudioContext`);
    - tampilkan **toast + badge** "Pesanan baru meja T-01 · No. INV-…";
    - refresh daftar pesanan self order (PENDING sudah tidak ada — hanya PAID).
  - Pengaturan suara: toggle di Settings (`sound_notifications`) + tombol
    mute per sesi; hormati interaksi pengguna (AudioContext butuh gesture —
    mulai setelah user pernah klik di POS).
- Backend harus mengirim `order_source` + `status` di `data` event realtime —
  cukup tambah key saat publish (field `Data` sudah `any`, tidak perlu
  perpanjangan struct).

### 8.3 Dapur & status

- Nota Dapur: reuse `buildKitchenHtml()` di `PosReceiptDialog.vue` — tercetak
  otomatis/button untuk sale `SELF_ORDER` PAID; `notes` pelanggan ikut tercetak.
- Ubah `order_status` dari POS/dapur (permission `sales.update`): daftar
  ORDERED/PREPARING/READY/COMPLETED.

### 8.4 Lokalisasi (EN & ID)

Grup `selfOrder` + `tables` di `fe/src/i18n/en.ts` dan `id.ts` (pola sama):
`scanPrompt`, `tableLabel`, `payNow`, `paymentSuccess`, `queueNumber`,
`orderStatus`, `newSelfOrder`, `soundNotification`, dst.

## 9. Pengujian

### 9.1 Unit (Go)

- `CreatePublicOrder`: valid checkout, item invalid, SKU nonaktif, stok kurang,
  qty ≤ 0, tax benar, token meja salah, store nonaktif/self order off.
- `PayPublicOrder`: sukses (PAID + payment), metode CASH ditolak, pembayaran
  kurang, order sudah PAID, order bukan SELF_ORDER.
- **Isolasi tenant**: token meja store A → tidak bisa submit/lihat menu store B;
  resolusi server-side mengabaikan input klien.
- Atomicity: kegagalan di tengah submit → sale/items/stok/inventory rollback.
- Menu publik tidak membocorkan `cost_price`/stok detail/`qr_token`.
- Event `sale.created` hanya ter-publish saat `PAID` (submit PENDING tidak
  memicu notifikasi) dan `data` memuat `status` + `order_source`.
- Rate limit per IP: submit ke-11 dalam satu menit ditolak
  `RESOURCE_EXHAUSTED`.

### 9.2 FE

- `vue-tsc --noEmit`, `npm run build`.
- Unit kecil: helper notifikasi suara (dipanggil hanya untuk
  `order_source=SELF_ORDER` + `status=PAID`).

### 9.3 End-to-end manual

1. Admin membuat 2 meja (T-01, T-02) + cetak QR + aktifkan self order di Settings.
2. Scan QR T-01 → halaman menampilkan "Meja T-01" + menu store-nya.
3. Submit + bayar sukses → POS berbunyi, toast muncul, daftar bertambah (PAID),
   dapur menerima Nota Dapur, stok berkurang.
4. Bayar gagal/batal → tidak ada sale baru, stok kembali.
5. QR T-02 dari store lain / token asal → `NOT_FOUND`, tidak ada akses.
6. Duplikat nomor meja → error `ALREADY_EXISTS`.
7. Rotasi token meja → QR lama menolak.

## 10. Definition of Done

- [ ] Migrasi `000017` (tables + kolom sales + store_settings; `cashier_id`
      & `inventory_transactions.created_by` di-drop NOT NULL; `table_id`
      `ON DELETE SET NULL`) & model GORM (`CashierID`/`CreatedBy` nullable).
- [ ] Proto `selforder/v1` + `buf generate` + swagger diperbarui.
- [ ] `internal/selforder` service: resolve/table, menu, submit, pay, status,
      cleanup PENDING; transaksi DB & publish realtime dengan `order_source`.
- [ ] Endpoint publik di daftar path tanpa JWT (full method gRPC di
      `middleware/auth.go`) + interceptor rate limit per IP.
- [ ] CRUD meja per store + rotasi token + cetak QR; permission `tables.*`.
- [ ] Halaman `/o/:tableToken` (menu + bayar + status) EN/ID.
- [ ] POS: notifikasi suara + toast + daftar saat `sale.created` dengan
      `order_source=SELF_ORDER` **dan** `status=PAID`.
- [ ] Nota Dapur untuk self order PAID.
- [ ] `go build ./...`, `go test ./...`, `vue-tsc`, `npm run build` lolos.
- [ ] Dokumen diperbarui bila keputusan berubah (lihat §12).

## 11. File yang diperkirakan berubah

| File | Catatan |
|------|---------|
| `db/migration/000017_tables_self_order.up.sql` / `.down.sql` | Baru. |
| `internal/database/gorm/models.go` | `Table` + field `Sale`/`StoreSetting`; `Sale.CashierID` & `InventoryTransaction.CreatedBy` jadi nullable. |
| `proto/selforder/v1/selforder.proto` | Baru. |
| `gen/...` | Hasil `buf generate`. |
| `internal/selforder/service.go` (+ repo) | Baru. |
| `internal/server/server.go` | Registrasi service + handler. |
| `middleware/auth.go` | Method gRPC publik `/selforder.v1.SelfOrderService/*`. |
| `middleware/ratelimit.go` | Baru: rate limit per IP untuk self-order. |
| `internal/role/service.go` / seed permission | `tables.*` permissions. |
| `internal/settings/service.go` | `self_order_enabled` (+ sound toggle). |
| `internal/sale/service.go` | Event realtime memuat `order_source` + `status`; tangani `CashierID` nil. |
| `fe/src/pages/o/index.astro` + `SelfOrderMenu.vue` | Baru (statis; token via query/path). |
| `fe/src/pages/admin/tables.astro` + `TablesPage.vue` | Baru (CRUD + cetak QR). |
| `fe/src/components/pos/POSCheckoutPage.vue` | Notif suara + daftar self order. |
| `fe/src/components/admin/SettingsPage.vue` | Toggle self order + suara. |
| `fe/src/i18n/en.ts` / `id.ts` | Grup `selfOrder` + `tables`. |
| `docs/swagger-enhanced.json` | Endpoint publik + tables + settings. |

## 12. Pertanyaan tersisa (kecil — bisa diputuskan saat implementasi)

1. **Metode pembayaran MVP**: cukup `EWALLET`/QRIS saja, atau semua metode
   non-tunai (`CARD`, `BANK_TRANSFER`, `OTHER`)? (Dokumen mengasumsikan
   semua non-tunai; CASH ditolak.)
2. **Integrasi payment gateway**: mock/reference manual dulu (pelanggan isi
   reference), atau integrasi gateway sungguhan (mis. Midtrans/QRIS) di MVP?
3. **Batas waktu submit→bayar** sebelum cleanup stok (usulan: 15 menit).
4. **Cetak QR**: PDF per meja satu halaman, atau satu lembar berisi semua meja?
5. **Nomor antrean**: cukup `invoice_number`, atau counter antrean per shift?

## 13. Catatan tambahan

- Prinsip: self order **tidak boleh** melewati aturan stok/inventory yang ada —
  setiap pesanan lunas = transaksi stok tercatat, seperti penjualan kasir.
- `qr_token` meja **tidak pernah** dikirim ke klien selain sebagai bagian dari
  URL QR; semua respons API publik/JSON mengecualikannya (`json:"-"`).
- Notifikasi suara wajib hormati autoplay policy browser (AudioContext
  di-resume setelah gesture pertama di POS) dan bisa dimatikan dari Settings.