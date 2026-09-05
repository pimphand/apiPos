# POS — Fitur Open Shift & Close Shift Cashier

> Dokumen spesifikasi & panduan implementasi untuk fitur **shift kasir** di halaman **POS** (`/pos`).
> Backend cashier (`CashierService`) sudah ada dan dipakai halaman admin `/admin/cashier`; dokumen ini
> menjelaskan cara menempatkan fitur yang sama di halaman POS beserta **penguncian transaksi**
> (tidak bisa checkout sebelum shift dibuka) dan **pengamanan maksimal** agar buka/tutup shift
> tidak pernah menghasilkan kondisi error/inkonsisten.

---

## 1. Ringkasan & Tujuan

- Setiap transaksi (checkout) di halaman POS **hanya boleh dilakukan saat kasir punya shift `OPEN`**.
- Jika **belum open shift** → halaman POS menampilkan layar *"Buka Shift"* dan **seluruh tombol checkout nonaktif**.
- Kasir **membuka shift** dengan memasukkan **uang kas awal (opening cash)**.
- Selama shift terbuka: kasir bisa checkout seperti biasa; UI menampilkan **status shift** (kasir aktif, waktu buka, total penjualan kas sementara).
- Kasir **menutup shift** melalui dialog konfirmasi yang menampilkan **rekonsiliasi kas**:
  `opening cash`, `cash sales`, `cash refund`, `expected cash`, `actual cash` (input kasir), `cash difference`.
- Setelah shift `CLOSED`: transaksi kembali terkunci sampai shift baru dibuka.
- Angka shift (pendapatan, barang keluar/masuk) **sinkron real-time** dengan transaksi yang terjadi selama shift dan **beku** setelah shift ditutup.

### Contoh rekonsiliasi (patokan dari AGENTS.md)

```text
Opening Cash      100.00
Cash Sales        500.00
Cash Refund        50.00
Expected Cash     550.00     (= opening + cash sales − cash refund)
Actual Cash       548.00     (dihitung kasir saat tutup)
Difference         −2.00     (= actual − expected)
```

---

## 2. Aturan Bisnis Inti (Business Rules)

| # | Rule | Sifat |
|---|------|-------|
| R1 | Checkout ditolak jika user tidak punya shift `OPEN`. | Wajib (FE gate + validasi ulang sebelum submit) |
| R2 | Satu user hanya boleh punya **satu** shift `OPEN` dalam satu waktu. | Backend sudah memeriksa; DB perlu *partial unique index* (lihat §7) |
| R3 | `opening_cash`, `closing_cash`, `actual_cash` tidak boleh negatif. | FE (Zod) + backend |
| R4 | Shift hanya bisa ditutup oleh kasir pemiliknya (dan dalam store yang sama). | **Backend perlu diperketat** (lihat §7) |
| R5 | Shift yang sudah `CLOSED` tidak boleh ditutup ulang (idempoten: kalau sudah closed → perlakukan sebagai sukses + muat ringkasan). | FE + backend |
| R6 | `expected cash` dihitung server-side; FE **tidak boleh** menghitung ulang untuk nilai otoritatif (hanya untuk *preview* saat dialog). | Backend otoritatif |
| R7 | Buka/tutup shift dicatat ke **activity log** (`OPEN_SHIFT`, `CLOSE_SHIFT`). | Backend sudah |
| R8 | Multi-tab / multi-perangkat: state shift harus selalu disinkronkan ulang dari server sebelum aksi kritis. | FE |

---

## 3. Endpoint Backend (sudah ada)

Dikutip dari `proto/cashier/v1/cashier.proto` & `internal/server/server.go` (permission map):

| Method | Endpoint | Body | Permission | Response (inti) |
|--------|----------|------|------------|-----------------|
| POST | `/api/v1/cashier/shifts/open` | `{ store_id, opening_cash }` | `cashier.open_shift` | `{ shift }` |
| POST | `/api/v1/cashier/shifts/close` | `{ shift_id, closing_cash, actual_cash }` | `cashier.close_shift` | `{ shift }` |
| GET | `/api/v1/cashier/shifts/current` | – | `cashier.view_shift` | `{ shift }` atau **404** jika tidak ada shift open |
| GET | `/api/v1/cashier/shifts/{id}/summary` | – | `cashier.view_shift` | `ShiftSummary` (rekonsiliasi + daftar sales) |
| GET | `/api/v1/cashier/shifts` | filter `status/user_id/store_id`, pagination | `cashier.view_shift` | `{ shifts, total_count, ... }` |

**Catatan `store_id`:** kirim `store_id` dari `authStore.currentUser.store_id` (JWT). Backend tetap mem-*scope* data sesuai token — FE tidak boleh mengirim `store_id` milik store lain.

### Bentuk `ShiftSummary` (dipakai untuk dialog close)

```ts
interface ShiftSummary {
  shift_id: string;
  cashier_name: string;
  store_name: string;
  opening_cash: number;
  cash_sales: number;
  cash_refund: number;
  expected_cash: number;
  actual_cash: number;
  cash_difference: number;
  total_transactions: number;
  total_sales_amount: number;
  status: 'OPEN' | 'CLOSED';
  opened_at: string;
  closed_at?: string;
  sales: ShiftSale[]; // invoice_number, grand_total, payment_method, created_at
}
```

Client FE sudah tersedia: `cashierService.openShift / closeShift / getCurrentShift / getShiftSummary`
(`src/services/cashier.ts`), tipe di `src/types/index.ts` (`CashierShift`, `ShiftSummary`, `ShiftSale`).

### Pemetaan error (FE harus menanganinya secara eksplisit)

| Situasi | Backend (gRPC → HTTP) | Aksi FE |
|---------|------------------------|---------|
| Sudah punya shift open | `INVALID_ARGUMENT` 400 "You already have an open shift" | **Recovery**: panggil `getCurrentShift()` → jadikan shift aktif. **Bukan** error fatal |
| Tidak ada shift open saat `getCurrentShift` | `NOT_FOUND` 404 "No open shift found" | State normal "Belum Buka Shift" — jangan toast error |
| Shift tidak ditemukan saat close | `NOT_FOUND` 404 | Muat ulang state shift (mungkin sudah ditutup di tab lain) |
| Shift sudah closed saat close | `INVALID_ARGUMENT` 400 "Shift is already closed" | **Idempoten**: anggap sukses → muat ringkasan & status "closed" |
| Opening cash negatif | `INVALID_ARGUMENT` 400 | Validasi form (Zod) |
| Store/user tidak valid | `INVALID_ARGUMENT` 400 | Muat ulang sesi (kemungkinan token basi) |

> Prinsip: **backend adalah sumber kebenaran**. Setiap respon error open/close yang berarti "shift
> sudah dalam kondisi yang kita inginkan" → FE *recover* ke kondisi server, bukan menampilkan error.

---

## 4. Alur UI di Halaman POS (`/pos`)

Komponen sasaran: `src/components/pos/POSCheckoutPage.vue` (route `/pos`, guard `sales.create`).
Kasir juga membuka/menutup shift dari `/pos` — **tidak** perlu pindah ke `/admin/cashier`.

### 4.1 State shift (bagian atas halaman / status bar)

- **Belum open** → banner/panel besar: ikon jam + "Shift belum dibuka" + tombol **Buka Shift**.
  Area keranjang & tombol *Bayar/Checkout* disembunyikan/dinonaktifkan (`disabled`) — keranjang tidak bisa diisi.
- **Sedang open** → chip status hijau: kasir aktif, `opened_at` (jam lokal), tombol **Tutup Shift**.
  Checkout normal berjalan.
- **Sedang loading/check** → spinner kecil; jangan tampilkan form checkout sebelum status shift diketahui
  (hindari flash "boleh transaksi" padahal shift tertutup).

### 4.2 Dialog Buka Shift

1. Klik **Buka Shift**.
2. Dialog: label **Uang Kas Awal** (`opening_cash`), nilai default `0`, input angka (≥ 0, maks. 2 desimal).
3. Tombol submit dinonaktifkan saat `saving` (cegah double-click / submit ganda).
4. `POST /shifts/open` dengan `store_id = currentUser.store_id`.
5. Sukses → toast sukses → **refresh `currentShift`** → UI masuk mode "open", checkout aktif.
6. Respon error "already have an open shift" → **jangan** tampilkan error; panggil `getCurrentShift()`
   dan lanjut ke mode "open" (recovery — mis. shift dibuka dari tab/perangkat lain).

### 4.3 Gate transaksi (checkout)

Sebelum `createSale` dikirim (dan saat tombol bayar diklik):

```
1. currentShift masih ada & status == 'OPEN'?  → lanjut
2. Tidak / ragu (mis. sudah > N menit sejak load) → GET /shifts/current
      ├─ 200 + OPEN  → lanjut checkout
      └─ 404         → blokir: muat ulang state shift → tampilkan layar "Buka Shift"
```

`CreateSale` **tidak** boleh dikirim kalau shift tertutup. Jika respon `CreateSale` menolak karena
shift (server hardening — lihat §7) → otomatis refresh state shift & tampilkan layar open-shift,
keranjang **dipertahankan** (tidak dihapus) supaya transaksi tinggal dibuka-shift lalu bayar lagi.

### 4.4 Dialog Tutup Shift (rekonsiliasi)

1. Klik **Tutup Shift** → muat `GET /shifts/{current.id}/summary` (sekali; tombol "Muat Ulang" kecil bila perlu).
2. Tampilkan ringkasan read-only:
   `Opening Cash`, `Cash Sales`, `Cash Refund`, `Expected Cash`, `Total Transaksi`, `Total Penjualan`,
   plus daftar sales (invoice, metode bayar, nominal) yang bisa di-*scroll*.
3. Input kasir:
   - **Actual Cash** (`actual_cash`) — uang tunai aktual di laci. Default saran = `expected_cash`, boleh diedit.
   - *Preview* **Cash Difference** (`actual − expected`) tampil live, warnai merah bila ≠ 0
     (hanya preview — nilai final dihitung server).
   - Opsional: **Closing Cash** (`closing_cash`) = uang tunai yang disisakan untuk shift berikutnya (default = actual).
4. Konfirmasi: tombol **Tutup Shift** (`variant="destructive"`) + dialog konfirmasi kedua bila `difference ≠ 0`
   (mis. teks "Selisih kas −Rp 2.000. Tetap tutup shift?") — mencegah tutup tanpa sadar saat ada selisih.
5. `POST /shifts/close` `{ shift_id, closing_cash, actual_cash }`.
6. Sukses → toast + tampilkan **ringkasan akhir** (opsional: cetak struk tutup shift) → state shift jadi
   `CLOSED`/kosong → UI kembali ke layar "Buka Shift" → checkout terkunci.
7. Error "already closed" → idempoten: muat ringkasan & tampilkan status closed (tanpa error).

### 4.5 Interaksi dengan halaman `/admin/cashier`

Fitur shift di `/pos` dan `/admin/cashier` **berbagi logika yang sama**. Jangan duplikasi — tarik ke
satu **composable** + **Pinia store** bersama:

- `src/composables/useCashierShift.ts` → `{ currentShift, shiftLoading, openShift(), closeShift(), refreshShift() }`
- state disimpan di Pinia (`stores/cashier.ts`) sehingga perpindahan route `/pos` ↔ `/admin/*` tidak
  memanggil ulang `current` dan tidak ada dua sumber kebenaran di UI.

---

## 5. Sinkronisasi Data Shift dengan Transaksi (Pendapatan & Barang Keluar/Masuk)

Tujuan: angka yang tampil di POS (status bar & dialog tutup shift) harus **sinkron dan konsisten dengan
transaksi yang benar-benar terjadi selama shift berjalan** — pendapatan, jumlah barang **keluar** (terjual),
barang **masuk** (retur/batal), lengkap sampai detail per produk/SKU — dan **tidak bergeser** setelah shift ditutup.

### 5.1 Kondisi saat ini (kenapa belum sinkron)

- Summary shift dihitung memakai **window waktu**: `sales WHERE cashier_id = user AND created_at >= opened_at`
  — tanpa relasi eksplisit sale↔shift dan **tanpa batas atas**:
  - sale yang terjadi **setelah** shift ditutup ikut terhitung di shift lama;
  - kalau kasir buka shift lagi di hari yang sama, satu sale bisa terhitung di shift lama & baru;
  - refund/ubah status sesudah close dapat mengubah angka shift yang sudah ditutup.
- Tidak ada **barang keluar/masuk**: unit produk/SKU per shift belum dihitung sama sekali.
- Kolom `cash_sales`/`cash_refund`/`expected_cash` di `cashier_shifts` **belum diisi** oleh alur sale
  (`UpdateCashSales/UpdateCashRefund` ada tapi belum dipanggil) → rekonsiliasi kas tampil `0`.
- `sale_items` hanya menyimpan `product_id` (tanpa SKU) → detail varian (warna/ukuran/harga per SKU)
  tidak tersedia untuk ditampilkan per shift.

### 5.2 Prinsip desain

1. **Relasi eksplisit:** setiap `sales` dicatat `shift_id` saat transaksi (bukan ditebak dari waktu).
2. **Ledger detail (insert-only):** tiap pergerakan barang terkait shift ditulis sebagai **baris permanen**
   di tabel baru `shift_items` — satu baris per item per transaksi, arah `OUT`/`IN` — **di dalam transaksi DB
   yang sama** dengan `CreateSale` / refund-batal (gagal → rollback semua, mustahil tidak sinkron).
3. **Turunan, bukan duplikat:** pendapatan, breakdown metode bayar, dan rekap barang dihitung dari data
   yang ter-link (`sales` + `payments` + `shift_items`) — satu sumber kebenaran, bisa diverifikasi silang.
4. **Snapshot saat close:** begitu `CLOSED`, agregat final dibekukan (`closed_at` + hasil hitung disimpan);
   tidak ada lagi baris yang boleh masuk ke shift tertutup.

### 5.3 Definisi metrik

| Metrik | Definisi | Sumber data |
|--------|----------|-------------|
| Total transaksi | jumlah `sales` dengan `shift_id = shift` (status `PAID`) | `sales` |
| Pendapatan (gross) | Σ `grand_total` sale shift | `sales` |
| Pendapatan per metode bayar | Σ `payments.amount` per `payment_method` utk sale shift | `payments` ⨝ `sales` |
| Cash sales (rekon kas) | Σ `payments.amount` metode `CASH` pada sale shift | `payments` ⨝ `sales` |
| Refund shift | uang kas keluar karena refund/batal **yang terjadi selama shift ini** | alur refund |
| Expected cash | `opening_cash + cash_sales − cash_refund` (server) | shift + payments + refund |
| Barang keluar (`OUT`) | unit barang keluar stok karena penjualan dalam shift (per produk & SKU) | `shift_items` arah `OUT` |
| Barang masuk (`IN`) | unit barang kembali ke stok karena retur/batal dalam shift | `shift_items` arah `IN` |
| Net barang | `Σ OUT − Σ IN` per produk/SKU | `shift_items` |

> **Batasan scope:** yang dihitung sebagai "masuk" shift hanya **retur/batal penjualan** (mempengaruhi kas
> & stok penjualan). Pembelian supplier (`PURCHASE`), `ADJUSTMENT`, `DAMAGE`, `TRANSFER` **bukan** bagian
> shift kasir — tetap tercatat sebagai inventory transaction biasa dan tampil di laporan inventori.

### 5.4 Perubahan skema (migrasi baru `000011_*`)

**(a) `sales.shift_id` — relasi eksplisit sale → shift**

```sql
ALTER TABLE sales
  ADD COLUMN shift_id UUID NULL REFERENCES cashier_shifts(id);

CREATE INDEX idx_sales_shift_id ON sales(shift_id);
```

**(b) Tabel baru `shift_items` — ledger detail (agar "tertulis detail")**

```sql
CREATE TABLE shift_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shift_id        UUID NOT NULL REFERENCES cashier_shifts(id) ON DELETE CASCADE,
    store_id        UUID NOT NULL REFERENCES stores(id),
    sale_id         UUID NULL REFERENCES sales(id),
    direction       VARCHAR(10) NOT NULL CHECK (direction IN ('OUT','IN')),
    item_type       VARCHAR(10) NOT NULL CHECK (item_type IN ('PRODUCT','SKU')),
    product_id      UUID NOT NULL REFERENCES products(id),
    product_sku_id  UUID NULL REFERENCES product_skus(id),
    product_name    VARCHAR(200) NOT NULL,
    sku_code        VARCHAR(50),
    variant_label   VARCHAR(255),              -- "Large / Red / M" utk tampilan cepat
    quantity        NUMERIC(12,3) NOT NULL CHECK (quantity > 0),  -- selalu positif; arah di `direction`
    unit_price      NUMERIC(15,2) NOT NULL,    -- harga jual saat transaksi (snapshot)
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    line_amount     NUMERIC(15,2) NOT NULL,    -- quantity*unit_price − discount (snapshot)
    payment_method  VARCHAR(20),               -- metode sale; utk rekon kas cepat
    reference_no    VARCHAR(100),              -- no. invoice / no. refund pemicu baris ini
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_shift_items_shift_id   ON shift_items(shift_id);
CREATE INDEX idx_shift_items_product_id ON shift_items(product_id);
CREATE INDEX idx_shift_items_sku_id     ON shift_items(product_sku_id);
CREATE INDEX idx_shift_items_direction  ON shift_items(shift_id, direction);
```

Keterangan:
- Baris `shift_items` bersifat **insert-only** — tidak pernah di-update/di-hapus (audit trail permanen).
- Detail SKU (variant/color/size/harga) diambil dari **payload `CreateSale`** (POS menjual per SKU) sehingga
  `sale_items` tidak perlu diubah. Opsional: tambahkan `product_sku_id` di `sale_items` utk histori jangka panjang.
- POS menjual per SKU → `item_type = 'SKU'` + `product_sku_id` terisi; `product_id` (parent) selalu diisi.

**(c) Snapshot agregat saat close (disarankan)** — kolom tambahan di `cashier_shifts` agar angka final
cepat diakses dan kebal terhadap perubahan data historis:

```sql
ALTER TABLE cashier_shifts
  ADD COLUMN total_transactions INT DEFAULT 0,
  ADD COLUMN total_sales_amount NUMERIC(15,2) DEFAULT 0,
  ADD COLUMN total_revenue      NUMERIC(15,2) DEFAULT 0,
  ADD COLUMN revenue_cash       NUMERIC(15,2) DEFAULT 0,
  ADD COLUMN total_items_out    NUMERIC(12,3) DEFAULT 0,
  ADD COLUMN total_items_in     NUMERIC(12,3) DEFAULT 0;
```

> Catatan: kolom agregat hanyalah **snapshot saat close**. Selama shift `OPEN`, angka live dihitung dari
> query atas data ter-link (atau counter transaksional bila performa menuntut) — jangan update kolom
> agregat di luar transaksi sale.

### 5.5 Alur penulisan & penyajian (sinkronisasi)

```
[CreateSale]   ── transaksi DB atomik ──►  1. pastikan shift OPEN (B5)
                                           2. INSERT sales (shift_id = current_shift.id)
                                           3. per item → INSERT shift_items (direction=OUT, dari payload SKU)
                                           4. kurangi stok + INSERT inventory_transactions (SALE)
                                           5. INSERT payments
                                           commit / rollback (gagal → tidak ada yang setengah jalan)

[Refund/Cancel] ── transaksi DB atomik ──► 1. pastikan sale milik shift
                                           2. INSERT shift_items (direction=IN, qty kembali, ref no. refund)
                                           3. kembalikan stok + INSERT inventory_transactions (RETURN)
                                           4. catat cash_refund pada shift tempat refund terjadi
                                           commit / rollback

[UI live]  ── setelah checkout sukses ──►  panggil ulang ringkasan shift (total transaksi, pendapatan,
                                           breakdown metode bayar, barang keluar teratas)
                                           → angka langsung ter-update
                                           (opsional polling ringan 30–60 dtk selama shift OPEN)
```

- **Gate**: `shift_id` hanya boleh diisi shift `OPEN` milik user yang sedang login (ditentukan server, bukan
  dari klien). Sale tanpa shift open (peran non-kasir) tetap boleh → `shift_id = NULL`.
- **Data lama (backfill)**: sekali, isi `sales.shift_id` dari window `opened_at..closed_at` per user; setelah
  itu relasi murni memakai `shift_id`.

### 5.6 Perluasan API

- `GET /shifts/{id}/summary` diperluas (response `ShiftSummary`):

```ts
interface ShiftSummary {
  // … field lama …
  total_transactions: number;
  total_sales_amount: number;
  revenue_total: number;                    // Σ grand_total sale shift (PAID)
  revenue_by_method: Record<string, number>;// { CASH: 0, CARD: 0, BANK_TRANSFER: 0, … }
  cash_sales: number;                       // = revenue_by_method.CASH
  cash_refund: number;
  expected_cash: number;                    // opening + cash_sales − cash_refund
  items_out_total: number;                  // unit barang keluar
  items_in_total: number;                   // unit barang masuk (retur/batal)
  items: ShiftItem[];                       // detail OUT & IN (opsional)
}
```

- `GET /shifts/{id}/items?direction=OUT|IN&page=..&per_page=..` → baris detail `shift_items`
  (produk, SKU, varian, qty, harga, nominal, invoice, waktu). Pagination standar
  (`total_count` / `page` / `per_page`).
- Daftar sales pada summary diubah berbasis `shift_id` — bukan window waktu.

### 5.7 Sinkronisasi FE

- Setelah **setiap checkout sukses** dalam shift: refresh ringkasan shift (transaksi, pendapatan, barang
  keluar) supaya tampilan live.
- Saat dialog **Tutup Shift**: angka yang tampil = hasil server terbaru; FE tidak menghitung ulang.
- Setelah close: ringkasan beku. Refund atas sale shift tsb yang terjadi **setelah** close dicatat pada shift
  **saat refund terjadi** (shift berikutnya), bukan mengubah shift lama.
- Multi-tab: storage-event / polling ringan opsional (lihat §6.7).

### 5.8 Validasi silang saat close (anti angka salah)

Server memverifikasi sebelum menutup shift:
1. `expected_cash` = `opening + Σ cash payment − Σ refund` (dari data shift, bukan input FE).
2. Σ `shift_items(direction=OUT)` shift = Σ pengurangan stok via inventory `SALE` yang ter-link shift (referensi).
3. Tidak ada baris (`sales`/`shift_items`) ber-`shift_id` shift ini yang ditulis **setelah** titik close.

Jika ada ketidakcocokan → **close ditolak** dengan pesan yang bisa ditindaklanjuti (mis. "Ada transaksi
belum tersinkron — hubungi admin"); shift tidak boleh ditutup memakai angka yang salah.

---

## 6. Pengamanan Maksimal — "Tanpa Error saat Buka/Tutup Shift"

Bagian ini adalah syarat mutlak (definition of done). Tiap butir = alasan mengapa error bisa terjadi + solusinya.

### 6.1 Race double-open (dua klik / dua tab bersamaan)

**Masalah:** backend memeriksa "user sudah punya open shift" lalu insert — dua request paralel sama-sama
lolos pemeriksaan → **dua shift OPEN** untuk satu user (tidak ada unique constraint di DB saat ini).

**Solusi (berlapis):**
1. FE: tombol submit di-*disable* saat `saving` (single-flight); dialog tidak bisa dibuka dua kali.
2. FE: **satu sumber kebenaran** — panggilan open dijaga `pendingPromise` agar dua pemicu (klik ganda,
   komponen lain) berbagi satu request.
3. Backend (wajib): **partial unique index** di DB —
   `CREATE UNIQUE INDEX uniq_open_shift_per_user ON cashier_shifts(user_id) WHERE status = 'OPEN';`
   → insert kedua gagal `23505` → `MapError` → FE recovery (jadikan shift yang sudah ada sebagai current).
4. Backend (opsional, rapi): bungkus cek+insert dalam transaksi + `FOR UPDATE`/`INSERT ... ON CONFLICT DO NOTHING`.

### 6.2 Double-close / close yang sudah closed

**Masalah:** dua request close paralel; yang kedua dapat "Shift is already closed".

**Solusi:**
1. FE: tombol tutup disable saat `saving`; dialog hanya bisa dibuka jika `currentShift.status === 'OPEN'`.
2. FE: **idempoten** — error `already closed`/`404` pada close → muat ringkasan shift tsb dan tampilkan
   status `CLOSED` sebagai hasil sukses (bukan error merah).
3. Backend: `CloseShift` sebaiknya atomic `UPDATE ... WHERE id=? AND status='OPEN'` lalu cek `RowsAffected`
   (0 baris → sudah closed → kembalikan respon sukses/idempoten alih-alih error).

### 6.3 Close tanpa shift / shift bukan miliknya

**Masalah:** endpoint close menerima `shift_id` bebas; user store yang sama bisa menutup shift kasir lain.

**Solusi (backend hardening, §7):** service hanya boleh menutup **shift open milik user yang sedang login**
(user_id dari JWT) — bukan id arbitrer; pemilik dicek server-side. FE hanya pernah mengirim `currentShift.id`.

### 6.4 Nilai negatif / format tidak valid

- Zod di FE: `opening/closing/actual_cash` → `number` ≥ 0, maks 2 desimal, tolak `NaN`/`Infinity`.
- Backend sudah menolak `opening_cash < 0`; perketat juga `closing/actual < 0` bila perlu.
- **Format angka**: simpan sebagai number (Rupiah utuh atau 2 desimal, konsisten); hindari string yang bisa
  jadi `"NaN"` saat input kosong (default `0`).

### 6.5 Konsistensi rekonsiliasi (expected vs actual)

**Kondisi saat ini:** kolom `expected_cash`/`cash_sales`/`cash_refund` pada shift **belum diisi otomatis**
oleh alur sale (repo `UpdateCashSales/UpdateCashRefund` ada tapi belum dipanggil). Tanpa pengisian itu,
dialog close akan menampilkan `expected = 0` — angka rekonsiliasi salah.

**Solusi backend (wajib sebelum fitur dianggap benar) — implementasi final mengikuti rancangan §5**
(relasi `shift_id` + ledger `shift_items`). Ringkasannya:
- Hitung di `GetShiftSummary` sebagai **satu sumber otoritatif**:
  - `cash_sales` = total sales ber-metode `CASH` pada rentang shift
  - `cash_refund` = total refund kas pada rentang shift
  - `expected_cash = opening_cash + cash_sales − cash_refund`
- Atau wire `UpdateCashSales/UpdateCashRefund` saat `CreateSale`/refund (jaga `expected_cash` selalu sinkron
  walau shift dibuka ulang / lintas waktu). Pilih salah satu — jangan keduanya untuk menghindari double count.
- `cash_difference = actual_cash − expected_cash` dihitung server saat close (repo sudah melakukannya).

### 6.6 Sesi/token expired di tengah dialog

- Semua request memakai axios yang sudah auto-refresh (401 interceptor) → dialog jangan menganggap
  "shift error" saat sebenarnya token sedang di-refresh; tetap tampilkan loading.
- Kalau refresh gagal → redirect login (sudah ditangani interceptor global) — dialog batal otomatis.

### 6.7 Multi-tab / multi-perangkat

- Shift dibuka di tab A; tab B masih "belum open" → tab B **wajib** cek ulang sebelum checkout (lihat 4.3).
- Setelah close dari tab A: tab B yang mencoba checkout akan gagal di server (hardening §7) → FE otomatis
  sinkron state → tampilkan layar "Buka Shift", keranjang tidak hilang.
- Opsional ringan: `storage` event (`localStorage`/`sessionStorage` key `pos_cashier_shift`) untuk
  memberitahu tab lain "shift berubah" sehingga UI nyaris real-time tanpa polling.

### 6.8 Kegagalan jaringan / timeout

- Jangan ubah state shift lokal **sebelum** respon sukses tiba (server = source of truth).
- Saat gagal jaringan: toast error + tombol **Coba Lagi**; state tetap (open/closed) tidak berubah.
- Setiap panggilan memakai `finally` untuk melepas `saving` state (tombol tidak terkunci permanen).

### 6.9 Activity log & audit

- Open/close shift otomatis tercatat `OPEN_SHIFT`/`CLOSE_SHIFT` di activity log (backend sudah).
- Jangan tampilkan data sensitif (token) di log FE.

---

## 7. Hardening Backend (disarankan — checklist terpisah dari FE)

| # | Perubahan | File kira-kira | Efek |
|---|-----------|----------------|------|
| B1 | Partial unique index `(user_id) WHERE status='OPEN'` | migrasi baru `000011_*` | Cegah double-open dari race apa pun |
| B2 | `CloseShift` hanya untuk shift open **milik user login** (cek `user_id` claims = shift.user_id; kalau beda → `PERMISSION_DENIED`) | `internal/cashier/service.go`, `handler` | Cegah tutup shift orang lain |
| B3 | `CloseShift` pakai `UPDATE … WHERE id=? AND status='OPEN'` + cek `RowsAffected` → idempoten | `cashier_gorm_repository.go` | Double-close jadi no-op sukses |
| B4 | Hitung rekonsiliasi & agregat mengikuti desain **§5** (query atas data ter-link `shift_id` + ledger `shift_items`); jangan pakai `UpdateCashSales/UpdateCashRefund` agar tak ada double count | `internal/cashier/service.go`, `internal/sale/service.go` | Angka dialog close & laporan shift benar |
| B5 | `CreateSale` menolak user (role cashier/admin kasir) tanpa shift `OPEN` → `FAILED_PRECONDITION` "No open shift" | `internal/sale/service.go` | Garansi server: transaksi tanpa shift mustahil |
| B6 | `GetCurrentShift` diberi `ORDER BY opened_at DESC LIMIT 1` (atau rely pada unique index) | `cashier_gorm_repository.go` | Deterministik kalau ada data ganda historis |
| B7 | Activity log sudah OK | – | – |
| B8 | `sales.shift_id` (nullable FK) + index | migrasi `000011_*`; `internal/sale/service.go` | Relasi eksplisit sale→shift (sumber kebenaran, bukan window waktu) |
| B9 | Tabel baru `shift_items` (ledger detail OUT/IN) ditulis dalam transaksi yang sama dgn sale/refund | migrasi `000011_*`; `internal/sale/service.go` | Detail barang keluar/masuk per shift tersimpan permanen ("tertulis detail") |
| B10 | `GetShiftSummary` & daftar sales berbasis `shift_id` + backfill data lama | `internal/cashier/service.go`, migrasi backfill | Data shift tidak tercampur antar shift / setelah close |
| B11 | Validasi silang saat close (expected vs Σ payment; OUT vs inventory SALE; tidak ada baris setelah close) → mismatch = tolak close | `internal/cashier/service.go` | Shift tidak pernah ditutup memakai angka salah |

> B5 adalah pengaman **paling penting** sisi server: meskipun FE lupa mengecek, server tetap menolak
> checkout saat shift tertutup — sehingga "belum open shift → belum bisa transaksi" berlaku di semua klien.

---

## 8. Lingkup Implementasi Frontend (checklist)

- [ ] `stores/cashier.ts` (Pinia): `currentShift`, `shiftLoading`, `opening`, `closing` + aksi open/close/refresh/summary.
- [ ] `composables/useCashierShift.ts`: API reaktif untuk komponen (`/pos` dan `/admin/cashier` memakainya).
- [ ] `POSCheckoutPage.vue`:
  - [ ] inisialisasi → `refreshShift()` (404 → state "no shift")
  - [ ] layar/panel "Buka Shift" saat tidak open (checkout & keranjang nonaktif)
  - [ ] dialog Buka Shift (opening cash, anti double-submit, recovery "sudah open")
  - [ ] status bar shift + tombol Tutup Shift saat open
  - [ ] status bar menampilkan ringkasan **live** (total transaksi, pendapatan, barang keluar) yang di-refresh setelah setiap checkout sukses
  - [ ] dialog Tutup Shift menampilkan rincian **barang keluar/masuk** dari `GET /shifts/{id}/items`
  - [ ] dialog Tutup Shift dengan ringkasan rekonsiliasi + input actual/closing + preview difference + konfirmasi selisih
  - [ ] gate ulang sebelum `createSale` + penanganan respon shift-closed (keranjang tetap)
- [ ] Refactor `/admin/cashier` (CashierPage) memakai composable yang sama (hapus duplikasi logika).
- [ ] `services/cashier.ts` + `types`: perluasan `ShiftSummary`, tipe `ShiftItem`, `shiftItems()`, `items()`.
- [ ] i18n `pos.shift.*` & `admin.cashier.*` parity EN/ID (jalankan `tests/unit/i18n.test.ts`).
- [ ] Bersih-bersih: tidak ada `console.*`; loading/disabled state konsisten.

## 9. Pengujian

### Unit (Vitest)
- Composable/store: open → `OPEN`; close sukses; 404 current → "no shift"; error "already open"/"already closed" → recovery idempoten; `opening/actual < 0` ditolak Zod.
- Gate checkout: fungsi `assertShiftOpen()` memblokir `createSale` saat shift tidak open.
- Agregat sinkron: setelah `createSale` sukses, total transaksi/pendapatan/`items_out` bertambah; retur menambah `items_in` & `cash_refund`; tidak ada double count.

### E2E (Playwright)
1. Login cashier → `/pos` → tampil "Buka Shift", tombol bayar tidak ada/nonaktif.
2. Buka shift (opening cash 100.000) → checkout berhasil.
3. Tutup shift (actual = expected) → sukses → `/pos` kembali terkunci.
4. Tutup shift dengan selisih → muncul konfirmasi → tetap bisa tutup setelah konfirmasi.
5. Double-click tombol Buka Shift → hanya satu shift open (cek `GET /shifts?status=OPEN&user_id=…`).
6. Close dua kali berturut (double click) → sukses idempoten, tanpa error toast.
7. Shift dibuka dari `/admin/cashier` → buka `/pos` → langsung bisa checkout (state sinkron).
8. Setelah 1 checkout sukses: ringkasan shift di `/pos` bertambah (total transaksi, pendapatan, barang keluar) & detail muncul di `GET /shifts/{id}/items?direction=OUT`.
9. Shift yang sudah ditutup: melakukan checkout → diblokir; refund belakangan tidak mengubah angka shift lama.

## 10. Kriteria Selesai (Definition of Done)

- [ ] `/pos` menampilkan state shift yang benar saat reload, setelah open, setelah close, dan saat shift dibuka dari halaman lain.
- [ ] Checkout **tidak mungkin** terjadi saat shift tertutup — dari UI maupun dari panggilan API langsung (setelah B5).
- [ ] Buka/tutup shift aman terhadap: double-click, double-submit, multi-tab, jaringan gagal, sesi expired, angka negatif.
- [ ] Error "sudah open"/"sudah closed" ditangani sebagai *recovery idempoten*, bukan error.
- [ ] Rekonsiliasi menampilkan angka benar (expected = opening + cash sales − cash refund).
- [ ] Pendapatan & barang keluar/masuk tampil **sinkron** dengan transaksi berjalan (refresh setelah checkout sukses) dan **bebas** setelah shift ditutup.
- [ ] Detail barang per shift tersimpan permanen (`shift_items`) dan bisa dilihat ulang kapan saja.
- [ ] i18n EN/ID parity; unit + e2e hijau; `vue-tsc --noEmit` dan `npm run build` lolos.
