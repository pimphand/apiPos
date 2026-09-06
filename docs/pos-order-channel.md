# POS — Fitur Channel/Status Pesanan: Dine-in vs Take-away

> Dokumen spesifikasi & panduan implementasi fitur **jenis pesanan** (dine-in / take-away)
> di halaman POS (`/pos`). Kasir bisa memilih apakah pesanan untuk **Dine-in**
> atau **Take-away** sebelum memproses pembayaran, dan nilai tersebut
> **terlihat di struk/struk (POS receipt)** dan disimpan minimal (bila backend siap,
> migrasi menyusul nanti).
>
> **Keputusan desain (disepakati via percakapan terbaru):**
> - Fitur UI-first saat ini; hanya frontend yang berubah sekarang.
> - Nilai **per-order**, bukan per-store/customer.
> - Tersedia di keranjang/area totals POS.
> - Default: **Dine-in**.
> - Muncul di struk bila tersedia.
> - Tidak mengubah harga, stok, tax, diskon, atau logika bisnis lainnya.
> - Punya path migrasi & backend-friendly.

Konfirmasi status kode sekarang:
- Tidak ada field `channel`/`order_type` di `Sale` ataupun model GORM saat ini.
- Tidak ada konstanta/boolaian channel dinein/takeaway di frontend.
- Backend *belum* meresapi perubahan sekarang (nilai akan tetap hidup di frontend,
  dan nanti bisa di-*save* jika migrasi dilakukan).


---

## 1. Ringkasan & tujuan

- Tambahkan **selektor jenis pesanan** di POS: **Dine-in** vs **Take-away**.
- Nilai dimunculkan di UI kasir sebelum pembayaran, lalu terlihat di struk.
- Tidak mempengaruhi logika hitung (subtotal, diskon, tax, grand total).
- Tidak mempengaruhi stok atau arah transaksi.

Guna utama:
- kasir bisa menyertakan konteks transaksi tanpa perlu ekstra langkah.
- struk informatif.
- siap untuk migrasi backend & laporan macam-macam nilai (dinamis).


---

## 2. Keputusan desain


| # | Keputusan | Alasan |
|---|-----------|--------|
| D1 | Nilai per-order (bukan per-store/customer) | Mengikuti konteks tiap transaksi yang berubah-ubah. |
| D2 | Dua kategori: Dine-in & Take-away (string stabil) | Lebih sederhana, lebih mudah dikonversi ke field backend. |
| D3 | Default: Dine-in (pilihan pertama) | Kasir bisa langsung mulai; memilih kembali gampang. |
| D4 | UI di sekitar keranjang/totals (terlihat) sebelum pay | Tepat waktu — kasir kepanjangan durasi mudah → gambar mental. |
| D5 | Tampil di struk bila ada | Killer UX: struk terlihat. |
| D6 | Data sementara frontend dulu (dan di-local) | MVP ringan; backend path hypersusable nanti. |
| D7 | Tidak mempengaruhi harga/stok/tax | Overlap risk minimizer. |


---

## 3. Model data FE

### 3.1 Konstanta & tipe

`fe/src/types/index.ts` (atau file konstanta terpisah jika pref didekat masa depan, tapi saat ini tinggal di `index.ts`):

```ts
/**
 * Jenis channel pesanan di POS.
 * Nanti bisa dipetakan ke kolom `order_channel`/`order_type` di tabel sales bila
 * backend pernah migrasi. Saat ini nilai hanya dipakai secara lokal di FE.
 */
export type OrderChannel = 'dinein' | 'takeaway';

export const ORDER_CHANNEL_LABELS: Record<OrderChannel, string> = {
  dinein: 'Dine-in',
  takeaway: 'Take-away',
};
```

### 3.2 Perubahan di keranjang / state checkout

- Tambahkan state internal di `POSCheckoutPage.vue`:
  - `orderChannel: ref<OrderChannel>('dinein')` (default Dine-in)
- Opsional secara struktural: adalah `OrderChannel` mengambil stored kotak
  keranjang sederhana (tidak perlu colume define aside).
- Nilai ini disalin ke payload struk (lihat §5).

---

## 4. UI / UX (pada aplikasi POS)

### 4.1 Tata letak

- Tempat: di sekitar total / sebelum tombol bayar.
- Bentuk: dua *pill toggle* konsisten (bisa merujuk ke pola tombol “channel select”).
- Default: Dine-in terpilih saat halaman diload / reset cart.
- Label: pakai `i18n` key yang sesuai.

### 4.2 Flow

1. Kasir mengisi keranjang.
2. Kasir bisa memilih Dine-in / Take-away sebelum bayar.
3. Saat struk muncul, header/fields struk menginformasikan channel.
4. Kasir boleh saja hilang fokus; selection tetap terjaga (persist di state saat ini).


### 4.3 Komponen dasar

Bisa membuat komponen kecil:

- `PosOrderChannelSelector.vue`
  - props: `modelValue: OrderChannel`
  - emits: `'update:modelValue'`
  - tampilkan dinein/takeaway toggle.
- Gunakan di POS + preview struk.

---

## 5. Penggunaan di struk/struk (receipt)

### 5.1 Di `PosReceiptPaper.vue` & `PosReceiptDialog.vue`

- Tambahkan field `orderChannel?: OrderChannel` pada `ReceiptData`.
- Tampilkan teks di struk: opsi: muncul dekat header atas, atau sebelah deskripsi.
- Fallback: tidak tampil bila nilai kosong (backwards-compatible).

Contoh JSON preview:

```json
{
  "orderChannel": "takeaway",
  ...
}
```

Kalau mengambil referensi Arrengeon layout, tambahkan baris:

```
Jenis Pesanan: Take-away
```

atau sec.

---

## 6. API / stroage (saat ini)

Saat ini **tidak ada perubahan backend API**. Nilai hanya disimpan di sisi FE
dan diekspos ke struk. Backend saat ini cukup; migration hanya:
- Jika backend ternyata mau menyimpan, kolom `order_channel`/`order_type`
  dengan string atau status ENUM kecil.
- Nilai backup sementara bisa disimpan via `POST /v1/sales` metadata (tetapi requires
  approval).
- Akan dokumentasikan migrasi jika planned.

---

## 7. Lokalisasi (EN & ID)

Tambahkan grup `orderChannel` di `en.ts` dan `id.ts` (diri sama pattern dengan yang
lain).

#### EN (`en.ts`)

```json
{
  "orderChannel": {
    "title": "Order type",
    "dineIn": "Dine-in",
    "takeAway": "Take-away",
    "selectToChange": "Select Dine-in or Take-away",
    "showInReceipt": "Order type will appear on the receipt"
  }
}
```

#### ID (`id.ts`)

```json
{
  "orderChannel": {
    "title": "Jenis pesanan",
    "dineIn": "Dine-in",
    "takeAway": "Take-away",
    "selectToChange": "Pilih Dine-in atau Take-away",
    "showInReceipt": "Jenis pesanan muncul di struk"
  }
}
```

---

## 8. Pengujian


### 8.1 Manual (UI/UX)

1. Buka `/pos`, pastikan default Dine-in terpilih.
2. Pilih **Take-away**, periksa label struk bila struk muncul.
3. Reset keranjang, cek default kembali ke Dine-in.
4. Cek tidak ada efek pada subtotal/tax/grand total.
5. Cek struk tidak rusak/berantakan.

### 8.2 Unit (opsional)

- Konstan label/channel bisa diuji ringan:
  - `ORDER_CHANNEL_LABELS` terdengar benar.
  - Default awal ref benar.
- Fungsi datar kecil (converter label) bisa ditest bila dibuat.

---

## 9. Definition of Done

- [ ] Jenis channel constant & type ada di `fe/src/types/index.ts` (atau file konstan).
- [ ] Konsep state FE di `POSCheckoutPage.vue`/kotak order state.
- [ ] Komponen `PosOrderChannelSelector.vue` (toggle pill).
- [ ] Terpakai di POS (tampilan keranjang/totals).
- [ ] Default Dine-in.
- [ ] Muncul di struk (PosReceiptPaper & PosReceiptDialog).
- [ ] Lokalisasi EN & ID.
- [ ] Tidak ada perubahan harga/stok/tax/logika.
- [ ] `vue-tsc --noEmit`, `vitest`, `npm run build` lolos.


## 10. File yang diubah (frontend saat ini)

| File | Catatan |
|------|---------|
| `fe/src/types/index.ts` | Tambah `OrderChannel` + label konstan. |
| `fe/src/i18n/en.ts` / `id.ts` | Tambah grup `orderChannel`. |
| `fe/src/components/pos/PosOrderChannelSelector.vue` (baru) | Toggle Dine-in/Take-away. |
| `fe/src/components/pos/POSCheckoutPage.vue` | State channel, tempat menanam selektor, kirim ke struk. |
| `fe/src/components/pos/PosReceiptPaper.vue` | Tampilkan channel di struk. |
| `fe/src/components/pos/PosReceiptDialog.vue` | Kirim/pass channel ke struk. |


## 11. Penambahan backend (nanti / opsional)

Jika backend harus mendukung:

- Migrasi kecil: `ALTER TABLE sales ADD COLUMN order_channel VARCHAR(20) DEFAULT 'dinein';`
  - atau `order_type`.
- Proto & handler yang sesuai (jika ikut cerita).
- Repo, service, gRPC + swagger minimal diupdate.
- Path: state terdeteksi dini, jadi nanti bisa digabung sebagai “optional” syarat FE.


## 12. Catatan tambahan

- Apabila UI terlalu kecil, bisa diganti dropdown tunggal; walau sekarang pill toggle lebih sesuai.
- Apabila backend sedang cuma FE, tidak ada efek untrusted/lookup.
- Dokumentasi ini digunakan untuk pengingat terstruktur: jadi mudah nanti build atau modifikasi


Selesai. Dokumen berada di `docs/pos-order-channel.md`.
