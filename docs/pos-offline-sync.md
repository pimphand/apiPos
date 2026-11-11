# POS Offline — Transaksi & Sinkronisasi

> Status: **implemented** (2026-11-11). Dokumen ini menjelaskan perilaku mode offline
> yang ditambahkan ke POS existing. Ringkasan progres: `docs/progress.md` F9.

## 1. Tujuan

Kasir tetap bisa menyelesaikan transaksi saat koneksi ke gateway terputus, lalu
mengirim transaksi tersebut ke server lewat menu **Sinkron Data** ketika koneksi
kembali. Tidak ada endpoint baru: sinkronisasi memutar ulang `POST /v1/sales`
yang sudah ada.

## 2. Komponen

| Bagian | File | Peran |
|---|---|---|
| Queue transaksi | `fe/src/lib/offlineQueue.ts` | Menyimpan payload `POST /v1/sales` + snapshot struk di localStorage, scoped per `store_id` + `user_id`, batas 200 transaksi |
| Sync otomatis | `fe/src/lib/offlineAutoSync.ts` + `fe/src/composables/useOfflineAutoSync.ts` | Memutar ulang antrean `pending` saat event `online` atau saat halaman boot online; runner single-flight yang sama dipakai tombol manual |
| Cache data | `fe/src/lib/offlineCache.ts` | Snapshot terakhir katalog (produk + SKU), pelanggan, pengaturan toko, dan shift OPEN per toko |
| Klasifikasi error | `fe/src/lib/http.ts` | Request tanpa respons HTTP diberi kode `UNAVAILABLE` (offline/DNS/timeout), bukan `INTERNAL` |
| Integrasi POS | `fe/src/components/pos/POSCheckoutPage.vue` | Fallback katalog/barcode/shift dari cache; checkout offline → queue + struk `OFFLINE-XXXXXXXX` |
| Halaman sinkron | `fe/src/components/admin/OfflineSyncPage.vue` + `/admin/offline-sync` | Daftar antrean, status, error server, sinkronkan semua / per item, hapus |
| Sesi offline | `fe/src/stores/auth.ts` | Saat boot tanpa internet, memakai profil terenkripsi yang tersimpan (`offlineSession`) agar POS tetap terbuka |
| Badge antrean | `fe/src/composables/useOfflineQueueCount.ts` | Jumlah transaksi belum tersinkron per store+user; tampil di sidebar **Sinkron Data**, chip di POS, dan ikon tray desktop, diperbarui lewat `subscribeOfflineQueue` setiap queue berubah |
| Badge tray | `desktop/src-tauri/src/shell.rs` + `fe/src/lib/platform.ts` | Command `tray_set_offline_count` menggambar badge merah + angka putih di ikon tray dan memperbarui tooltip; 0 mengembalikan ikon normal. Web build: no-op |
| Notifikasi native | `fe/src/lib/platform.ts` (`notifyNative`) + `desktop/src-tauri/src/notify.rs` | Saat transaksi baru masuk antrean, build desktop mengirim notifikasi OS berisi referensi `OFFLINE-…`, total, dan jumlah antrean; web tetap memakai toast |

## 3. Alur

1. **Online (normal):** setiap respons sukses memperbarui cache (`cacheSnapshot()`); checkout
   memakai `POST /v1/sales` seperti sebelumnya.
2. **Koneksi putus saat checkout:** error `UNAVAILABLE` → transaksi ditulis ke queue, struk
   ditampilkan dengan nomor `OFFLINE-<8 karakter>` + toast. Di build desktop, shell juga mengirim
   notifikasi native berisi referensi, total, dan jumlah antrean, sehingga transaksi tidak terlewat
   saat window diminimalkan. Keranjang dikosongkan, stok pada katalog cache dikurangi secara lokal.
3. **Checkout saat offline:** shift harus masih tercatat OPEN di cache. Tanpa shift cache,
   checkout ditolak dengan pesan untuk menyambung internet dan membuka shift.
4. **Sinkron:** item dikirim berurutan ke `POST /v1/sales`. Sukses → item dihapus dari queue dan
   invoice server ditampilkan di toast. Gagal → item tetap ada dengan status **Gagal** + pesan
   server, bisa diulang atau dihapus manual.
5. **Kembali online:** banner POS hilang dan antrean `pending` otomatis dikirim oleh
   `useOfflineAutoSync` (dipasang di sidebar, jadi berlaku di semua halaman). Hasilnya muncul
   sebagai toast; entri yang ditolak server tetap **Gagal** dan hanya di-retry manual dari
   halaman Sinkron Data. Auto-sync dan tombol manual memakai runner single-flight yang sama,
   sehingga transaksi tidak pernah dikirim dua kali.
6. **Indikator:** badge di sidebar, chip `Sinkron Data (n)` di POS, dan badge angka pada ikon tray
   desktop menunjukkan jumlah transaksi yang belum tersinkron (0 → badge sidebar/tray disembunyikan).
   Angka berubah langsung setelah checkout offline, retry, atau sync berhasil; tooltip tray memuat
   jumlah persis (ikon memakai `99+` di atas 99).

## 4. Batasan yang disengaja

- **Validasi tetap di server.** Stok, shift, pajak, dan permission divalidasi ulang saat sync.
  Transaksi yang ditolak (mis. stok sudah habis) tetap di daftar agar kasir tahu.
- **Auto-sync tidak mengulang entri `failed`.** Entri yang sudah ditolak server (stok/shift) hanya
  dikirim ulang lewat tombol manual agar tidak membanjiri server setiap kali halaman dibuka.
- **Tidak ada idempotency key baru.** Jika request sync benar-benar sampai ke server tetapi
  responsnya hilang di jaringan, retry dapat membuat duplikat. Backend tidak menyediakan
  kunci idempotensi; ini dicatat sebagai keterbatasan, bukan ditutupi.
- **Sesi offline hanya berlaku bila perangkat benar-benar offline** (`navigator.onLine === false`)
  dan ada profil terenkripsi. Ini berarti siapa pun yang memegang perangkat dapat membuka POS
  offline; perlindungan PIN/lock screen adalah pekerjaan lanjutan.
- **Realtime dan laporan tidak offline.** Hanya katalog, pelanggan, shift, dan checkout yang
  memakai cache.
- **Batas queue 200 transaksi per kasir per toko**; setelah penuh, checkout offline ditolak
  dengan pesan agar sinkron lebih dulu.
- **Hapus antrean bersifat permanen** dan memakai konfirmasi.

## 5. Hasil uji end-to-end (2026-11-11)

Spec: `fe/tests/e2e/offline-sync.spec.ts` (Playwright, gateway nyata + Postgres terisolasi).
Alur yang terbukti: login lewat form → POS memuat produk & shift OPEN → cache lokal terisi →
`context.setOffline(true)` → checkout → struk `OFFLINE-CD2DE38E` + queue `pending` 1 item →
`setOffline(false)` → menu **Sinkron Data** → **Sync All** → `POST /v1/sales` 200
(`INV-20261111-0a6ae4de`), antrean kosong, sale `PAID` grand_total 16.650 di server. **1/1 lulus (2,9 s)**.

Runner auto-sync diuji terpisah di `fe/tests/unit/offlineAutoSync.test.ts` (7 test: sync sukses,
gagal server, transport putus, single-flight, serta skip untuk queue kosong/perangkat offline/entri
`failed`). Total Vitest **66/66** lulus.

Temuan backend saat uji: pembayaran tunai **lebih bayar** (20.000 untuk total 16.650) ditolak
`CreateSale` dengan `journal not balanced: debit 30000 != credit 26650`. Penyebabnya baris kas di
`internal/accounting/posting.go` mendebit nominal pembayaran, sedangkan kredit memakai total
penjualan + PPN + COGS; kembalian tidak dikreditkan. Uang pas seimbang, sehingga spec memakai uang
pas. Ini bug backend di luar lingkup desktop/offline dan dicatat di `docs/progress.md` F9.
