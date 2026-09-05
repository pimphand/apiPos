# POS — Pengaturan Toko: Pajak Default & Format Nota

> Dokumen spesifikasi & panduan implementasi fitur **halaman Pengaturan (Settings)** per toko:
> mengatur **pajak default** (dipakai POS saat keranjang baru) dan **format nota/struk**
> (header, footer, dan opsi baris yang dicetak). Frontend dan backend sama-sama berubah.
>
> **Keputusan desain (disepakati):**
> - Penyimpanan **backend, per toko** (tabel `store_settings`, satu baris per `store_id`) —
>   bukan localStorage.
> - Cakupan format nota: **lengkap** — header & footer bebas + opsi baris (SKU, barcode,
>   kasir, pelanggan, diskon, pajak, pembayaran, kembalian).
> - Dipakai langsung di POS: **default pajak** keranjang baru + **dialog struk** setelah
>   pembayaran sukses (saat ini belum ada fitur struk sama sekali).
> - Hak akses: **baca** untuk semua user terautentikasi dalam store (kasir butuh default
>   pajak & format struk); **ubah** hanya role **admin & manager** (dipaksa di service layer).

---

## 1. Ringkasan & Tujuan

- Halaman **Pengaturan** baru di area admin (`/admin/settings`) untuk:
  1. **Pajak default** — persen pajak yang otomatis terisi di POS saat keranjang baru
     (menggantikan angka 11 yang sekarang hardcoded).
  2. **Format nota** — teks header & footer struk + pilihan baris yang dicetak, dengan
     **pratinjau struk live** di kanan form.
- POS (`/pos`) memakai hasilnya:
  - `tax_rate` → nilai awal `taxRate` setiap keranjang baru / reset (masih bisa diubah
    manual per transaksi seperti sekarang).
  - Setelah **pembayaran sukses** → **dialog struk** menampilkan nota mengikuti format
    settings (nama/alamat toko, header, item, total, pembayaran, kembalian, footer) +
    tombol **Print** (cetak via browser).
- Perubahan **per store**: toko A bisa pajak 0 tanpa header, toko B pajak 11% dengan
  footer ucapan terima kasih — tidak saling memengaruhi.

### Contoh alur nyata

```text
Admin toko masuk ke /admin/settings
   → isi pajak default 11, header "TERIMA KASIH", matikan opsi SKU, nyalakan opsi kasir
   → preview struk berubah live → Simpan (PUT /api/v1/settings)

Kasir buka /pos (store yang sama)
   → keranjang baru langsung taxRate = 11 (dari settings)
   → pelanggan bayar → dialog struk muncul sesuai format (header, tanpa SKU, ada kasir)
   → Print struk
```

---

## 2. Aturan Bisnis (Business Rules)

| # | Rule | Sifat |
|---|------|-------|
| R1 | Settings **di-scope per store** (`store_id`). Semua query/tulis memakai `store_id` dari klaim JWT user — tidak ada parameter store dari klien. | BE |
| R2 | **Baca** (`GetSettings`) boleh untuk semua user terautentikasi yang punya store (admin/manager/kasir). `superadmin` tidak punya store → tidak bisa membaca (403/404). | BE |
| R3 | **Ubah** (`UpdateSettings`) hanya role **`admin` & `manager`**; role lain → `PERMISSION_DENIED`. Guard di **service layer** (bukan interceptor). | BE |
| R4 | Settings tidak pernah `NULL`: bila baris belum ada, `GetSettings` mengembalikan **default** (`tax_rate=11`, header/footer kosong, opsi cetak standar). `UpdateSettings` = **upsert** (insert bila belum ada). | BE |
| R5 | Validasi `tax_rate`: 0–100 (0 sah = bebas pajak). Header/footer bebas (teks multi-baris). | BE |
| R6 | **Upsert + baca dalam satu store** — tidak ada kebocoran antar store walau id store dikirim nakal (diabaikan, selalu dari claims). | BE |
| R7 | Default pajak di POS hanya dipakai untuk **keranjang baru/reset**; kasir tetap bisa mengubah pajak per transaksi (perilaku lama dipertahankan). | FE |
| R8 | Struk POS hanya muncul **setelah pembayaran sukses** (sale `PAID`), berisi data sale aktual dari server (invoice, subtotal, diskon, pajak, total, metode bayar, kembalian). | FE |
| R9 | Toggle format nota berlaku di **preview halaman settings** dan **dialog struk POS** sekaligus (satu sumber aturan cetak). | FE |
| R10 | Format nota yang **belum disimpan** tidak memengaruhi POS — POS mengambil settings dari server saat halaman dimuat. | FE |
| R11 | Aktivitas `UPDATE settings` dicatat ke **activity log** (`resource: settings`). | BE |
| R12 | Endpoint tidak di-*permission-gate* oleh interceptor (baca butuh akses kasir juga); **role admin/manager untuk tulis** dijaga di service. | BE |

---

## 3. Model Data

### 3.1 Tabel baru: `store_settings` (migrasi `000013`)

```sql
CREATE TABLE store_settings (
    store_id                 UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
    tax_rate                 NUMERIC(5,2) NOT NULL DEFAULT 11.00,
    receipt_header           TEXT NOT NULL DEFAULT '',
    receipt_footer           TEXT NOT NULL DEFAULT '',
    show_store_name          BOOLEAN NOT NULL DEFAULT TRUE,
    show_store_contact       BOOLEAN NOT NULL DEFAULT TRUE,
    show_cashier             BOOLEAN NOT NULL DEFAULT TRUE,
    show_customer            BOOLEAN NOT NULL DEFAULT TRUE,
    show_item_sku            BOOLEAN NOT NULL DEFAULT FALSE,
    show_item_barcode        BOOLEAN NOT NULL DEFAULT FALSE,
    show_discount            BOOLEAN NOT NULL DEFAULT TRUE,
    show_tax                 BOOLEAN NOT NULL DEFAULT TRUE,
    show_payment             BOOLEAN NOT NULL DEFAULT TRUE,
    show_change              BOOLEAN NOT NULL DEFAULT TRUE,
    updated_by               UUID REFERENCES users(id),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- Satu baris per store (PK = `store_id`, 1:1 dengan `stores`).
- Tidak masuk daftar `tenantTables` scope otomatis — `store_id` selalu di-set/difilter
  eksplisit dari claims di repository/service.
- Rollback (`down`): `DROP TABLE store_settings;`

### 3.2 Tipe TypeScript (tambah di `fe/src/types/index.ts`)

```ts
export interface StoreSettings {
  store_id: string;
  tax_rate: number;             // persen, default 11
  receipt_header: string;
  receipt_footer: string;
  show_store_name: boolean;
  show_store_contact: boolean;
  show_cashier: boolean;
  show_customer: boolean;
  show_item_sku: boolean;
  show_item_barcode: boolean;
  show_discount: boolean;
  show_tax: boolean;
  show_payment: boolean;
  show_change: boolean;
  updated_by: string;
  updated_at: string;
}
```

---

## 4. API (gRPC + REST via grpc-gateway)

Domain baru `settings/v1` (`SettingsService`), tenant-scoped.

| Method | REST | Deskripsi |
|--------|------|-----------|
| `GetSettings` | `GET /api/v1/settings` | Ambil settings store user (default bila belum ada). |
| `UpdateSettings` | `PUT /api/v1/settings` | Simpan/upsert settings store user. |

- Semua `bool` dipetakan 1:1 (proto `bool` — bukan filter, jadi aman).
- Numerik `tax_rate` `double`, teks header/footer `string`.
- Tidak ada pagination (satu baris per store).

---

## 5. Peta Perubahan File

### Backend (Go/gRPC/Postgres)

| # | File | Isi |
|---|------|-----|
| 1 | `db/migration/000013_store_settings.up.sql` / `.down.sql` *(baru)* | Tabel `store_settings` |
| 2 | `internal/database/gorm/models.go` | Model `StoreSetting` (`TableName` → `store_settings`) |
| 3 | `proto/settings/v1/settings.proto` *(baru)* | `SettingsService` + message (lihat §4) |
| 4 | `gen/settings/v1/*` | Hasil `buf generate` |
| 5 | `internal/repository/settings_gorm_repository.go` *(baru)* | Repo `GetByStoreID`, `Upsert` (OnConflict) |
| 6 | `internal/repository/repositories.go` | Daftarkan `Settings` repo |
| 7 | `internal/settings/service.go` *(baru)* | Logika: default fallback, guard role, upsert, validasi, activity-log-aware |
| 8 | `internal/handler/settings.go` *(baru)* | Handler gRPC `GetSettings` / `UpdateSettings` + konversi proto |
| 9 | `internal/server/server.go` | Wire: service, register gRPC + REST gateway, health status |
| 10 | `docs/swagger-enhanced.json` | Endpoint + definisi `StoreSettings` |

### Frontend (Vue 3 / Astro)

| # | File | Isi |
|---|------|-----|
| 11 | `fe/src/types/index.ts` | Tipe `StoreSettings` |
| 12 | `fe/src/services/settings.ts` *(baru)* | `getSettings()` & `updateSettings()` via `http` |
| 13 | `fe/src/i18n/en.ts` & `id.ts` | Grup `admin.settings.*` + `posReceipt.*` (parity) |
| 14 | `fe/src/pages/admin/settings.astro` *(baru)* | Route halaman (pakai `DashboardLayout`) |
| 15 | `fe/src/components/admin/SettingsPage.vue` *(baru)* | Form pajak + format nota + **preview struk live** + simpan |
| 16 | `fe/src/components/layout/AppSidebar.vue` | Menu **Settings** (icon `Settings`, terlihat utk yang bisa `settings.*`/admin-manager) |
| 17 | `fe/src/components/pos/PosReceiptDialog.vue` *(baru)* | Dialog struk POS pasca-bayar (render sesuai settings + tombol Print) |
| 18 | `fe/src/components/pos/POSCheckoutPage.vue` | Default pajak dari settings; buka dialog struk saat sale sukses |

---

## 6. Alur UI

### 6.1 Halaman `/admin/settings`

```
Card "Pajak"                     Card "Format Nota"               Card "Pratinjau Struk"
  [ Pajak default % ]              [ Header (textarea)  ]          (render live, lebar 80mm
                                    [ Footer (textarea)  ]           gaya struk thermal)
                                    [ ] Tampilkan nama toko          ┌────────────────┐
                                    [ ] Tampilkan alamat/telepon     │  NAMA TOKO     │
                                    [x] Tampilkan kasir              │  Alamat        │
                                    [x] Tampilkan pelanggan          │  Jl. …         │
                                    [ ] Tampilkan SKU item           │ ─────────────  │
                                    [ ] Tampilkan barcode item       │  2× Kopi 20.000 │
                                    [x] Tampilkan diskon             │  …              │
                                    [x] Tampilkan pajak              │ ─────────────  │
                                    [x] Tampilkan pembayaran         │  Subtotal  40.0 │
                                    [x] Tampilkan kembalian          │  Pajak 11%  4.4 │
                                                                     │  Total     44.4 │
                                                                     │  Tunai 50 – kemb│
                                                                     │  Terima kasih   │
                                                                     └────────────────┘
                                   [ Simpan ]
```

- `role` selain admin/manager → halaman menampilkan blok akses terbatas (guard halaman),
  sama seperti halaman admin lain.
- Preview memakai **data contoh** (bukan sale nyata) dengan nama/alamat toko dari
  `authStore.currentUser` bila tersedia.

### 6.2 POS `/pos` — default pajak

- Saat POS siap (`isReady`), ambil `GET /api/v1/settings` sekali → simpan `defaultTaxRate`.
- `clearCart()` / keranjang baru & restore-hold → `taxRate = defaultTaxRate ?? 11`.
- Gagal ambil settings (offline/down) → fallback 11 tanpa error mengganggu.

### 6.3 POS `/pos` — dialog struk setelah bayar

- `submitSale` sukses (sale `PAID`) → isi `lastSale` (respons `CreateSale` + payment
  method/change dari payload) → buka `PosReceiptDialog`.
- Dialog: header toko (nama/alamat sesuai toggle), header bebas, daftar item (baris
  memakai opsi SKU/barcode), subtotal–diskon–pajak–total sesuai toggle, pembayaran &
  kembalian sesuai toggle, footer bebas, nomor invoice & waktu.
- Tombol **Cetak** → `window.print()` pada area struk (print CSS) — cetak via browser
  (belum integrasi printer thermal bluetooth/USB).
- Tombol **Selesai / Transaksi Baru** → tutup dialog & `clearCart()` untuk transaksi berikutnya.

---

## 7. i18n (EN ↔ ID)

Grup baru di `fe/src/i18n/en.ts` & `id.ts` (parity penuh, 1:1):

| Key | EN | ID |
|-----|----|----|
| `admin.settings.title` | Store Settings | Pengaturan Toko |
| `admin.settings.description` | Tax default & nota/struk format for your store | Pajak default & format nota/struk toko Anda |
| `admin.settings.taxCard` | Default Tax | Pajak Default |
| `admin.settings.taxRate` | Tax rate (%) | Tarif pajak (%) |
| `admin.settings.taxHint` | Used as default for new carts in POS. Cashier can still override per sale. | Dipakai sebagai default keranjang baru di POS. Kasir tetap bisa mengubah per transaksi. |
| `admin.settings.receiptCard` | Receipt (Nota) Format | Format Nota |
| `admin.settings.receiptHeader` | Header | Kepala nota |
| `admin.settings.receiptHeaderHint` | Printed above items (e.g. thank-you message). | Dicetak di atas item (mis. ucapan terima kasih). |
| `admin.settings.receiptFooter` | Footer | Kaki nota |
| `admin.settings.receiptFooterHint` | Printed below totals (e.g. returns policy). | Dicetak di bawah total (mis. kebijakan retur). |
| `admin.settings.previewCard` | Receipt Preview | Pratinjau Struk |
| `admin.settings.rowOptions` | Printed lines | Baris yang dicetak |
| `admin.settings.showStoreName` | Store name | Nama toko |
| `admin.settings.showStoreContact` | Store address & phone | Alamat & telepon toko |
| `admin.settings.showCashier` | Cashier | Kasir |
| `admin.settings.showCustomer` | Customer | Pelanggan |
| `admin.settings.showItemSku` | Item SKU | SKU item |
| `admin.settings.showItemBarcode` | Item barcode | Barcode item |
| `admin.settings.showDiscount` | Discount row | Baris diskon |
| `admin.settings.showTax` | Tax row | Baris pajak |
| `admin.settings.showPayment` | Payment row | Baris pembayaran |
| `admin.settings.showChange` | Change row | Baris kembalian |
| `admin.settings.save` | Save Settings | Simpan Pengaturan |
| `admin.settings.saving` | Saving… | Menyimpan… |
| `admin.settings.toastSaved` | Settings saved | Pengaturan tersimpan |
| `admin.settings.accessRestrictedAction` | change store settings | mengubah pengaturan toko |
| `posReceipt.title` | Receipt | Struk |
| `posReceipt.print` | Print | Cetak |
| `posReceipt.newSale` | New Transaction | Transaksi Baru |
| `posReceipt.invoice` | Invoice | Invoice |
| `posReceipt.cashier` | Cashier | Kasir |
| `posReceipt.customer` | Customer | Pelanggan |
| `posReceipt.item` | Item | Item |
| `posReceipt.qty` | Qty | Jml |
| `posReceipt.subtotal` | Subtotal | Subtotal |
| `posReceipt.discount` | Discount | Diskon |
| `posReceipt.tax` | Tax | Pajak |
| `posReceipt.total` | Total | Total |
| `posReceipt.paid` | Paid | Dibayar |
| `posReceipt.change` | Change | Kembalian |
| `posReceipt.walkIn` | Walk-in | Walk-in |

---

## 8. Pengujian

### Backend (unit, `internal/settings`)
- `GetSettings` tanpa baris → default (tax 11, opsi standar) & **tidak menulis** DB.
- `UpdateSettings` pertama → insert; kedua → update (upsert idempoten).
- Role selain admin/manager gagal update (`ErrForbidden`).
- `tax_rate` di luar 0–100 → `INVALID_ARGUMENT`.
- Store scope: klaim store A tidak pernah membaca/menimpa store B.
- Aktivitas `UPDATE settings` masuk activity log.

### Frontend
- `fe/tests/unit`: parity i18n tetap lolos (key baru EN=ID).
- Uji manual:
  1. Login sbg admin → menu Settings muncul; role kasir → tidak bisa membuka/ubah.
  2. Ubah pajak default → buka `/pos` → keranjang baru `taxRate` = nilai baru.
  3. Ubah format nota → Simpan → buat sale di `/pos` → dialog struk mengikuti format.
  4. Struk: tombol Cetak memunculkan dialog print browser.
  5. Simpan dengan pajak 0 → keranjang POS baru tanpa pajak (masih bisa diubah manual).

### E2E (opsional)
- Alur: settings → POS bayar → struk tampil → cetak.

---

## 9. Definition of Done

- [ ] Migrasi `000013` up/down tersedia; model GORM sesuai tabel.
- [ ] Proto `settings/v1` + `buf generate` sukses; gRPC & REST jalan (`GET/PUT /api/v1/settings`).
- [ ] Guard role admin/manager & scope store di service; error via `apperror` (tidak bocor SQL).
- [ ] `go vet` + `go test ./...` hijau.
- [ ] Halaman `/admin/settings` (form + preview live + simpan) muncul di sidebar utk admin/manager.
- [ ] POS memakai default pajak dari settings; dialog struk muncul setelah bayar + bisa cetak.
- [ ] i18n EN/ID parity; `vue-tsc`, `vitest`, `npm run build` hijau.
- [ ] `docs/swagger-enhanced.json` memuat endpoint & definisi settings.

---

## 10. Catatan

- Aktivitas dicatat dengan resource `settings` (tipe baru) — tidak menambah permission
  baru; guard utama berbasis **role** (admin/manager), konsisten dengan modul tenant lain.
- Belum ada integrasi printer thermal (USB/Bluetooth) — cetak via browser. Bisa jadi
  follow-up terpisah.
- Keputusan alternatif yang dipertimbangkan & ditolak: settings di `localStorage`
  (tidak lintas perangkat / hilang saat ganti browser), kolom tambahan di tabel `stores`
  (menyatu tapi kurang rapi utk opsi nota), halaman superadmin (settings bersifat per toko).
