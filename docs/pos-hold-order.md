# POS — Fitur Hold Order & Restore Hold

> Dokumen spesifikasi & panduan implementasi fitur **hold pesanan** di halaman POS (`/pos`).
> Saat kasir melayani pelanggan lalu pelanggan minta menunggu (mis. "saya cek stok dulu" / lupa
> dompet), kasir bisa **menahan (hold)** keranjang yang sedang diisi, **melayani pelanggan
> berikutnya** dengan keranjang kosong, lalu **memulihkan (restore)** pesanan yang ditahan
> kapan pun — lengkap dengan item, jumlah, diskon, pajak, dan pelanggannya.
>
> **Keputusan desain (disepakati):**
> - Penyimpanan **browser saja (`localStorage`)** — tanpa perubahan backend / migrasi DB / proto.
> - **Banyak hold** sekaligus (daftar antrean), bukan satu slot.
> - Berlaku untuk semua role yang bisa mengakses `/pos` (guard `sales.create`).

---

## 1. Ringkasan & Tujuan

- Tombol **Hold** di area keranjang → menyimpan **snapshot penuh keranjang saat ini**
  (item/SKU, qty, harga, diskon, tax rate, pelanggan, dll.), lalu **mengosongkan keranjang**.
- Setelah hold, keranjang langsung kosong → kasir **bisa membuat pesanan baru** untuk pelanggan lain.
- Tombol/indikator **Restore Hold** (dengan jumlah hold aktif) → membuka **daftar pesanan yang
  ditahan** → kasir memilih satu → isi keranjang **dikembalikan utuh** → tinggal lanjut bayar.
- Hold bisa dihapus (dibatalkan) kapan pun oleh kasir.
- Data hold **tetap ada saat halaman di-reload / pindah route** (localStorage) dan **sinkron
  antar-tab** browser yang sama (event `storage`).
- Hold bersifat lokal per **store + kasir** di browser tersebut (bukan data server, bukan
  multi-perangkat). Catatan batasan ini jelas di UI? Tidak wajib — cukup di dokumentasi.

### Contoh alur nyata

```text
Pelanggan A: 2× Kopi, 1× Roti → klik HOLD
   → keranjang kosong, "Hold #1 — Rp 43.000" masuk daftar

Pelanggan B: 1× Teh → bayar (checkout normal)
   → kasir klik "Restore Hold (1)" → pilih "Hold #1 (A)"
   → 2× Kopi & 1× Roti kembali ke keranjang → pelanggan A bayar
   → hold #1 terhapus dari daftar
```

---

## 2. Aturan Bisnis (Business Rules)

| # | Rule | Sifat |
|---|------|-------|
| R1 | **Hold hanya bisa dilakukan saat keranjang berisi ≥ 1 item** (tombol disabled saat kosong). | FE |
| R2 | Hold menyimpan **snapshot**: item (product_id, sku_id, nama, kode SKU, varian), `unit_price` **sesuai harga saat di-hold**, `quantity`, `discount`, `tax_rate`, `customer_id`, `created_at`. | FE |
| R3 | Setelah hold sukses → **keranjang dikosongkan total** (item, diskon, tax rate kembali default, customer kosong) → kasir bebas membuat pesanan baru. | FE |
| R4 | **Restore** mengembalikan seluruh snapshot ke keranjang **apa adanya** (termasuk diskon, tax rate, customer) — bukan hanya item. | FE |
| R5 | Restore **hanya bisa satu hold per waktu**; jika keranjang sedang terisi saat restore, keranjang lama harus **dikosongkan dulu** (dengan konfirmasi — keranjang lama bisa di-hold dulu). | FE |
| R6 | Stok di-restore memakai snapshot + **divalidasi ulang terhadap stok terbaru** saat restore (lihat §7.3). Item SKU yang stoknya tidak cukup / produk nonaktif → ditandai; hold tetap bisa direstore tapi garis itu **diblokir untuk dibayar** sampai qty dikurangi/dihapus. | FE |
| R7 | Harga memakai **snapshot saat di-hold** (harga yang "dijanjikan" ke pelanggan). Bila ingin harga terbaru, kasir bisa ubah manual di keranjang setelah restore. | FE |
| R8 | Hold dibatasi **maksimal 10 antrean** per store+kasir (cegah UI penuh). Saat penuh, tombol Hold menampilkan error dan kasir harus restore/delete dulu. | FE |
| R9 | Hold yang sudah di-restore / dihapus **tidak bisa dikembalikan** (tidak ada undo; perlu konfirmasi untuk delete). | FE |
| R10 | Shift ditutup / belum dibuka **tidak menghapus hold** — hold adalah data lokal kasir yang bertahan; namun checkout tetap tunduk pada aturan shift (`posShift`). | FE |
| R11 | Data hold **di-scope per `store_id` + `user_id`** (satu browser bisa dipakai beberapa kasir/store — jangan bocor antar akun). | FE |
| R12 | **Tidak ada duplikasi item** saat restore: garis SKU yang sama digabung (qty dijumlah) — sama seperti perilaku `addToCart`. | FE |
| R13 | Hold **bukan transaksi** — tidak menyentuh stok, tidak membuat sale, tidak tercatat di activity log (tidak ada backend). | FE |

---

## 3. Model Data (FE)

Snapshot hold = **data keranjang + metadata hold**. Dipakai untuk tipe TS, serialisasi JSON di
`localStorage`, dan komponen daftar hold.

### 3.1 Tipe TypeScript (tambah di `fe/src/types/index.ts`)

```ts
/** Satu baris snapshot keranjang saat di-hold — bentuknya sama dengan CartItem di POS. */
export interface HeldOrderItem {
  product_id: string;
  sku_id: string;
  name: string;        // nama produk (snapshot utk tampilan)
  sku_code: string;    // kode SKU/variant (snapshot utk tampilan)
  unit_price: number;  // harga jual SAAT di-hold (R7)
  quantity: number;
  stock: number;          // stok saat di-hold (snapshot — referensi tampilan)
  original_stock: number; // sama dengan stock (disimpan agar cocok dgn bentuk CartItem)
}

export interface HeldOrder {
  id: string;            // crypto.randomUUID()
  store_id: string;      // scope tenancy lokal (R11)
  user_id: string;       // scope kasir (R11)
  username: string;      // tampilan singkat bila multi-kasir
  label: string;         // label singkat utk daftar, default "Hold #n — {customer|Walk-in}"
  items: HeldOrderItem[];
  discount: number;      // snapshot diskon keranjang
  tax_rate: number;      // snapshot tax rate (default 11)
  customer_id: string;   // '' = walk-in (NO_CUSTOMER)
  customer_name?: string;// snapshot nama pelanggan utk label (bila ada)
  created_at: string;    // ISO
  updated_at: string;
}

/** Payload yang dibaca/ditulis ke localStorage. */
export interface HeldOrderStore {
  version: 1;
  holds: HeldOrder[];
}
```

> Catatan: `discount` di keranjang POS berbentuk **nilai nominal (Rp)**, bukan persen — ikuti yang
> sama saat hold/restore (konsisten dengan `admin.pos.discount` yang sudah ada).

### 3.2 Struktur `localStorage`

| Key | Isi | Catatan |
|-----|-----|---------|
| `pos.held_orders.v1` | `{ version: 1, holds: HeldOrder[] }` | Satu key untuk semua store/kasir di browser; scope diterapkan saat **baca/tulis** via filter `store_id` + `user_id` (R11). |

**Aturan utilitas penyimpanan (`fe/src/lib/posHold.ts`):**

- `loadHeldOrders(storeId, userId): HeldOrder[]` — baca, filter scope, urut `created_at` DESC,
  tangani JSON korup (try/catch → kembalikan `[]` tanpa melempar).
- `saveHeldOrders(storeId, userId, holds)` — tulis ulang array scope tsb (baca-semua → filter →
  modifikasi → simpan). Dipakai semua mutasi.
- `HOLD_LIMIT = 10` (R8), `HOLD_LIMIT_EXCEEDED` sentinel/message utk error.
- **Versioning:** simpan `version`; jika format berubah, migrasi kecil di `load` (v1 sekarang).
  Bila `version` tak dikenal → buang data lama (anggap korup).
- Jangan simpan apa pun selain snapshot (tanpa token, tanpa data sensitif lain).

---

## 4. Perubahan File (Frontend Saja)

| # | File | Perubahan |
|---|------|-----------|
| 1 | `fe/src/types/index.ts` | Tambah tipe `HeldOrder`, `HeldOrderItem`, `HeldOrderStore` (bagian §3.1). |
| 2 | `fe/src/lib/posHold.ts` **(baru)** | Utilitas localStorage: load/save/filter + `HOLD_LIMIT` + cleanup korup. |
| 3 | `fe/src/composables/useHeldOrders.ts` **(baru)** | Composable reaktif: `holds`, `holdsCount`, `holdCart(snapshot)`, `restoreHold(id)`, `deleteHold(id)`, `clearExpired?`; **sinkronisasi antar-tab** via `window.addEventListener('storage', …)`. |
| 4 | `fe/src/components/pos/POSCheckoutPage.vue` | Tombol **Hold** (header keranjang), badge/daftar **Restore Hold**, dialog daftar hold + konfirmasi delete/restore-dengan-keranjang-terisi, logika snapshot & restore, validasi stok saat restore (§7). |
| 5 | `fe/src/i18n/en.ts` & `id.ts` | Key baru `posHold.*` + parity (bagian §8). Reuse `admin.pos.holdCart` yang **sudah ada** ('Hold'/'Tahan') sebagai teks tombol. |
| 6 | `fe/tests/unit/*` (vitest) | Unit test utilitas & composable (bagian §10). |

> **Tidak ada** perubahan: proto, backend, migrasi DB, permission map, swagger, service FE.

---

## 5. Alur UI di Halaman POS (`/pos`)

Komponen sasaran: `fe/src/components/pos/POSCheckoutPage.vue` — tepatnya area **keranjang** (kolom
kanan, kartu "Cart") dan **total/pembayaran**.

### 5.1 Tombol Hold

- Lokasi: **header kartu Cart**, di samping tombol "Clear Cart".
- Teks: `t('admin.pos.holdCart')` ('Hold'/'Tahan') — key sudah ada.
- `disabled` saat `cart.length === 0` (R1). Tampilkan juga saat tidak ada shift open? Tidak —
  tombol hanya butuh keranjang; layar "Buka Shift" sudah menyembunyikan area keranjang seluruhnya.
- Klik → konfirmasi? **Langsung hold** (aksi non-destruktif; snapshot tetap bisa direstore).
  Toast sukses `posHold.toastHeld`.

### 5.2 Tombol / Indikator Restore Hold

- Lokasi: header kartu Cart di sebelah tombol Hold **atau** baris di atas tombol Bayar — pilih satu
  lokasi yang jelas (disarankan header Cart agar dekat tombol Hold).
- Tampilan: `Restore Hold` + **badge jumlah** hold aktif milik kasir ini (R11), mis.
  "Restore Hold (2)". Disabled saat `holdsCount === 0`.
- Klik → dialog daftar hold (lihat 5.3).

### 5.3 Dialog "Held Orders" (daftar)

| Elemen | Isi |
|--------|-----|
| Title | `posHold.title` ("Held Orders") + deskripsi singkat |
| Daftar | Satu kartu/baris per hold: **label** (`posHold.holdLabel` format, lihat bawah), **jumlah item** (`items.length`), **total nilai** (Σ `unit_price × quantity` − discount + tax, tampilan saja), **waktu** hold (`formatDateTime`), **nama pelanggan** bila ada |
| Empty state | `posHold.empty` ("No held orders") — dialog hanya terbuka saat ada isi, tapi tetap siapkan |
| Aksi per hold | **Restore** (`posHold.restore`) & **Delete** (`posHold.delete`) dengan konfirmasi |
| Footer | Tombol Tutup |

- **Label default** hold: `"Hold #{n}"` — nomor urut tampilan (1..N antrean saat ini) + bila ada
  pelanggan: `"Hold #{n} · {customer_name}"` (R1 di §3.1 `label`). Kasir tidak perlu mengetik
  label — hindari input tambahan (kecepatan kasir).
- Saat hold di-restore/dihapus, penomoran tampilan menyesuaikan (urutan tetap `created_at`).

### 5.4 Restore — konflik dengan keranjang terisi (R5)

1. Kasir klik **Restore** pada sebuah hold.
2. Jika `cart.length === 0` → langsung restore.
3. Jika keranjang **sedang terisi** → dialog konfirmasi:
   - Teks: "Current cart is not empty. Hold the current cart first, or replace it?" — beri 2 aksi:
     a) **Hold current cart & restore** (hold otomatis keranjang lama → kosong → restore hold pilihan);
     b) **Replace** (buang keranjang lama — dengan teks peringatan);
     c) Batal.
   - Rekomendasi implementasi: tawarkan (a) sebagai aksi utama (aman — tidak ada data hilang).

### 5.5 Setelah restore

- Item, `discount`, `tax_rate`, `customer` (pilih pelanggan di dialog payment ikut terset) kembali.
- Hold terhapus dari daftar (R9).
- Toast sukses `posHold.toastRestored`.
- Jika ada garis yang bermasalah (R6) → toast/notice per item (lihat 5.6).

### 5.6 Validasi stok & ketersediaan saat restore (R6)

Karena hold adalah snapshot, antara waktu hold dan restore stok/harga bisa berubah. Saat restore:

1. **Refresh data SKU** dari server (reuse `productService.get(product_id, { get_sku: true })`
   atau `productService.getSKUByBarcode/getSKUBySKU` bila sku_id tak tersedia) **per SKU unik** —
   ambil `stock` & `is_active` terkini. Gagal jaringan → **lanjut pakai snapshot** (offline-friendly);
   validasi final tetap dilakukan server saat `createSale`.
2. Untuk tiap baris hasil restore:
   - SKU **nonaktif / tidak ditemukan** → baris dimasukkan tapi **ditandai disabled** dengan
     notice `posHold.itemUnavailable` (tidak bisa dibayar sampai dihapus).
   - Stok terkini **< qty** → qty baris **dipangkas ke stok terkini** + notice
     `posHold.stockAdjusted` (nama, qty lama → baru). Kalau stok 0 → baris ditandai seperti no. 1.
   - **Harga**: tetap pakai snapshot (R7).
3. `skuRemainingStock` / gate `addToCart` yang sudah ada tetap berlaku — tidak ada bypass.

> Prinsip: **restore secepat mungkin** (jangan blokir karena validasi) tapi **tidak pernah
> mengizinkan pembayaran item yang mustahil**. Pembayaran sendiri sudah divalidasi stok oleh server.

### 5.7 Interaksi dengan fitur lain

- **Shift tertutup / belum dibuka:** hold tetap tersimpan & terlihat, tapi checkout tetap
  terkunci sesuai `posShift` (R10). Restore tidak memerlukan shift open — kasir boleh menyiapkan
  keranjang sebelum buka shift.
- **Clear Cart / selesai bayar:** **tidak** menyentuh daftar hold (hold independen).
- **Logout / ganti akun:** daftar hold di-scope `store_id` + `user_id` → akun lain di browser yang
  sama tidak melihat hold kasir lain (R11).

---

## 6. Sinkronisasi & Edge Cases

| Kasus | Perilaku |
|-------|----------|
| Reload / pindah route | Hold dimuat ulang dari `localStorage` di `onMounted` (via composable) — tidak hilang. |
| Dua tab POS dibuka | Event `storage` browser → daftar hold di tab lain ikut ter-update (composable `useHeldOrders` men-subscribe). Restore tetap per-tab (keranjang milik tab). |
| JSON korup / key tak dikenal | `load` mengembalikan `[]` (atau migrasi) — POS tidak boleh crash karena data lokal rusak. |
| Kapasitas penuh (R8) | Hold ditolak + toast `posHold.toastLimit` (nilai dari `HOLD_LIMIT`). |
| Double-click tombol Hold | Aksi sinkron & cepat; tak perlu async — pastikan tidak ada efek ganda (guard `cart.length===0` sesudah kosongkan). |
| Stok berubah saat pelanggan hold | Diatasi validasi restore (§5.6) + validasi server saat bayar. |
| Harga berubah saat pelanggan hold | Diatasi R7 (snapshot). |

---

## 7. i18n (EN ↔ ID parity)

Grup baru `posHold` di `en.ts` & `id.ts`. Tombol Hold memakai `admin.pos.holdCart` yang **sudah
ada**. Contoh key (nama bebas disesuaikan, wajib parity EN/ID):

| Key | EN | ID |
|-----|----|----|
| `posHold.title` | Held Orders | Pesanan Ditahan |
| `posHold.description` | Held carts can be restored to continue checkout later. | Keranjang yang ditahan bisa dipulihkan untuk melanjutkan pembayaran. |
| `posHold.restore` | Restore | Pulihkan |
| `posHold.restoreHold` | Restore Hold | Pulihkan Hold |
| `posHold.delete` | Delete | Hapus |
| `posHold.empty` | No held orders | Tidak ada pesanan ditahan |
| `posHold.itemCount` | `{count} items` | `{count} item` |
| `posHold.heldAt` | Held {time} | Ditahan {time} |
| `posHold.holdLabel` | Hold #{n} | Hold #{n} |
| `posHold.holdLabelWithCustomer` | Hold #{n} · {customer} | Hold #{n} · {customer} |
| `posHold.walkIn` | Walk-in | Tanpa Nama |
| `posHold.toastHeld` | Order held — cart cleared | Pesanan ditahan — keranjang dikosongkan |
| `posHold.toastRestored` | Held order restored | Pesanan ditahan dipulihkan |
| `posHold.toastDeleted` | Held order deleted | Pesanan ditahan dihapus |
| `posHold.toastLimit` | Hold limit reached ({limit}). Restore or delete one first. | Batas hold tercapai ({limit}). Pulihkan atau hapus salah satu dulu. |
| `posHold.toastHeldCartEmpty` | Nothing to hold | Tidak ada yang bisa ditahan |
| `posHold.confirmDeleteTitle` | Delete held order? | Hapus pesanan ditahan? |
| `posHold.confirmDeleteDesc` | "{label}" will be removed and cannot be restored. | "{label}" akan dihapus dan tidak bisa dikembalikan. |
| `posHold.confirmReplaceTitle` | Restore over current cart? | Pulihkan menggantikan keranjang saat ini? |
| `posHold.confirmReplaceDesc` | Current cart has {count} items. Hold it first, or replace it? | Keranjang saat ini berisi {count} item. Hold dulu, atau ganti? |
| `posHold.actionHoldAndRestore` | Hold current & restore | Hold saat ini & pulihkan |
| `posHold.actionReplace` | Replace | Ganti |
| `posHold.itemUnavailable` | {name} is no longer available | {name} sudah tidak tersedia |
| `posHold.stockAdjusted` | {name}: quantity reduced from {old} to {new} (stock changed) | {name}: jumlah dikurangi dari {old} ke {new} (stok berubah) |

Jalankan test parity i18n yang sudah ada (`tests/unit/i18n.test.ts` style).

---

## 8. Peta Implementasi (checklist)

- [ ] **T1** — Tipe `HeldOrder*` di `types/index.ts` (samakan dgn bentuk `CartItem`).
- [ ] **T2** — `lib/posHold.ts`: load/save + scope store/user + `HOLD_LIMIT` + handling JSON korup.
- [ ] **T3** — `composables/useHeldOrders.ts`: state reaktif + `holdCart` / `restoreHold` /
      `deleteHold` + listener `storage` (cross-tab).
- [ ] **T4** — `POSCheckoutPage.vue`:
  - [ ] panggil composable saat setup; muat hold scope kasir aktif (`authStore.currentUser`).
  - [ ] tombol **Hold** di header Cart (reuse `admin.pos.holdCart`), disabled saat cart kosong.
  - [ ] handler `holdCart()`: snapshot `cart/discount/taxRate/customer` → simpan → `clearCart()` →
        toast → hapus focus dialog payment bila terbuka (hold tidak boleh dari dalam dialog bayar).
  - [ ] tombol/badge **Restore Hold ({n})** di header Cart.
  - [ ] dialog daftar hold: label, item count, total, waktu, aksi Restore/Delete + konfirmasi.
  - [ ] restore: validasi R5 (keranjang terisi → dialog Hold&Restore / Replace), R6 (refresh stok),
        lalu isi keranjang & field diskon/tax/customer → hapus hold dari daftar → toast.
  - [ ] edit qty / hapus item pasca-restore memakai logika `updateQty`/`removeItem` yang ada.
- [ ] **T5** — i18n `posHold.*` EN & ID (parity test hijau).
- [ ] **T6** — Unit test (bagian §9) + `vue-tsc --noEmit` + `npm run build`.

---

## 9. Pengujian

### Unit (Vitest)
- `posHold` storage: simpan→baca utuh; scope store/user tidak bocor; JSON korup → `[]`; format
  tak dikenal → dibuang; batas `HOLD_LIMIT`.
- Composable: `holdCart` dari cart isi → masuk daftar + cart callback kosong; `restoreHold` →
  hapus dari daftar; `deleteHold` → hilang; aksi saat cart terisi → memilih Hold&Restore vs Replace.
- Snapshot: item/`discount`/`tax_rate`/`customer_id` tersimpan & kembali **identik** (R2/R4).

### Manual (uji live di `/pos`)
1. Shift open → isi keranjang (2 SKU, diskon, tax 11, pilih pelanggan) → **Hold** → keranjang
   kosong + toast + badge "Restore Hold (1)".
2. Isi keranjang baru (pelanggan B) → checkout → sukses.
3. **Restore Hold** → pilih hold A → keranjang A kembali (item/qty/diskon/tax/pelanggan) → bayar
   → sukses; hold hilang dari daftar.
4. Hold kosong (klik saat cart 0 item) → tombol disabled.
5. Holds > kapasitas → hold ditolak dgn pesan batas.
6. Keranjang terisi + restore → muncul konfirmasi; pilih "Hold current & restore" → keranjang lama
   ikut ter-hold, keranjang pilihan ter-restore.
7. Stok SKU dikurangi (via halaman inventory) sebelum restore → qty dipangkas + notice; SKU
   dinonaktifkan → item ditandai tidak tersedia & tak bisa dibayar.
8. Reload halaman → daftar hold tetap ada. Dua tab → hold di tab A muncul di tab B.
9. Akun kasir lain login di browser sama → tidak melihat hold kasir sebelumnya (scope).

### E2E (opsional, Playwright)
1. Hold → cart kosong → restore → cart terisi penuh → pay sukses.
2. Restore dgn cart terisi → opsi Hold&Restore bekerja (tidak ada item hilang).

---

## 10. Kriteria Selesai (Definition of Done)

- [ ] Tombol **Hold** & **Restore Hold** muncul di area keranjang POS dan berfungsi saat shift open.
- [ ] Hold menyimpan & mengembalikan **seluruh** state keranjang (item, qty, diskon, tax rate,
      pelanggan) secara utuh.
- [ ] Setelah hold, kasir **bisa langsung membuat pesanan baru** (keranjang kosong).
- [ ] Bisa menahan **banyak pesanan** sekaligus dan memulihkan salah satunya kapan pun.
- [ ] Hold bertahan terhadap reload/pindah route dan sinkron antar-tab browser.
- [ ] Stok/ketersediaan divalidasi ulang saat restore; item bermasalah tidak bisa dibayar.
- [ ] Hold tidak mungkin mengganggu transaksi yang sedang berjalan (R5 konfirmasi).
- [ ] i18n EN/ID parity; vitest hijau; `vue-tsc --noEmit` & `npm run build` lolos.
- [ ] Tidak ada perubahan backend / migrasi / proto (keputusan: localStorage).

---

## Lampiran — Contoh JSON di `localStorage`

```json
{
  "version": 1,
  "holds": [
    {
      "id": "9f1c2b1e-4f7a-4f0d-9c5a-1a2b3c4d5e6f",
      "store_id": "0e4c1a2b-…",
      "user_id": "a1b2c3d4-…",
      "username": "kasir1",
      "label": "Hold #1 · Budi",
      "items": [
        {
          "product_id": "p1…",
          "sku_id": "s1…",
          "name": "Kopi Susu",
          "sku_code": "KOPI-LG",
          "unit_price": 15000,
          "quantity": 2,
          "stock": 40,
          "original_stock": 40
        }
      ],
      "discount": 0,
      "tax_rate": 11,
      "customer_id": "c1…",
      "customer_name": "Budi",
      "created_at": "2026-09-05T09:30:00.000Z",
      "updated_at": "2026-09-05T09:30:00.000Z"
    }
  ]
}
```
