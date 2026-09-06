# Input Nama Pelanggan di POS

Kasir cukup mengetik **nama** pelanggan — tanpa harus membuat profil lengkap
(kode/telepon/alamat) terlebih dahulu.

## Perilaku

1. Di halaman POS (kartu totals) dan di dialog pembayaran tersedia input
   **Nama Pelanggan** (opsional).
2. Kasir mengetik nama apa pun.
   - Kosong → transaksi dianggap **walk-in / pelanggan umum** (`customer_id` NULL).
   - Terisi → nama dikirim ke backend lewat field `customer_name` pada
     `CreateSale`.
3. Backend (`internal/sale/service.go` → `resolveCustomer`) melakukan
   **find-or-create**:
   - Jika sudah ada pelanggan dengan nama yang sama di toko yang sama
     (case-insensitive, `LOWER(name) = LOWER(?)` + `store_id`), transaksi
     **dipakai/ditimpa ke pelanggan lama** tersebut — tidak membuat duplikat.
   - Jika belum ada, pelanggan baru dibuat otomatis (kode dibuat
     `CUS-XXXXXXXX`) dan transaksi dikaitkan ke pelanggan baru itu.
4. Nama yang diketik ikut disimpan ke **hold order** (`customer_name`) sehingga
   saat hold di-restore nama pelanggan tetap terbawa.
5. Nama pelanggan tampil di struk/nota (baris Customer, bila setting
   `show_customer` aktif).

## Data flow

```text
POS (input nama)                 Backend                    Database
    │  customer_name="Budi"          │                          │
    ├────────────────────────────────→ CreateSale               │
    │                                │ resolveCustomer:         │
    │                                │  • LOWER(name)=LOWER('Budi') di store → pakai ID lama
    │                                │  • tidak ada → CREATE customer + pakai ID baru
    │                                └── sale.customer_id = <ID>│
    └── struk menampilkan nama       ←──────────────────────────┘
```

## Catatan teknis

- Tidak ada tombol/modal "buat pelanggan" terpisah di alur bayar; cukup isi
  nama pada input.
- Input memakai `<datalist>` berisi nama pelanggan aktif toko itu sebagai
  saran, tapi teks bebas tetap diperbolehkan.
- Backend memakai satu query name+store untuk reuse sehingga nama yang sama
  tidak menghasilkan pelanggan ganda.
