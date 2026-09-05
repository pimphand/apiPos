# Progress — Implementasi Fitur Shift Kasir di POS (`docs/pos-cashier-shift.md`)

> Dokumen pelacak kemajuan (progres log). Di-update setiap milestone selesai agar seluruh
> pekerjaan **terekam** dan bisa diverifikasi ulang kapan pun.
> Spesifikasi lengkap: [`pos-cashier-shift.md`](./pos-cashier-shift.md).

- Status umum: **MILESTONE INTI SELESAI + SIKLUS PERBAIKAN + AUDIT** ✅ (backend + UI `/pos` terverifikasi live)
- Tanggal mulai: 2026-09-05 · Update terakhir: 2026-09-05 (B11 tuntas; `/pos` dipecah menjadi 4 komponen anak)
- Lingkup: backend (gRPC/REST) + frontend (Vue `/pos`) + migrasi DB.

---

## Ringkasan Milestone

| # | Milestone | Status | Catatan |
|---|-----------|--------|---------|
| M1 | Migrasi DB `000011` (index anti-double-open, `sales.shift_id`, tabel `shift_items`, kolom snapshot) | ✅ Selesai | Terpasang di DB dev (`version 11`) |
| M2 | Model GORM baru (`ShiftItem`, `Sale.ShiftID`, snapshot `CashierShift`) | ✅ Selesai | |
| M3 | Hardening cashier service: close milik sendiri, close idempoten, summary berbasis `shift_id` | ✅ Selesai | |
| M4 | Gate `CreateSale` tanpa shift OPEN + tulis ledger `shift_items` OUT (transaksi sama) | ✅ Selesai | |
| M5 | Refund/Cancel: ledger IN + `cash_refund` ke shift open aktor | ✅ Selesai | Refund terverifikasi; Cancel terpasang (uji rinci menyusul) |
| M6 | Proto + endpoint `GET /shifts/{id}/items` + perluasan summary | ✅ Selesai | `buf generate` + permission map diperbarui |
| M7 | FE composable shift + dialog Buka/Tutup di `/pos` + gate checkout | ✅ Selesai | Milestone inti; item lain di bawah |
| M8 | Verifikasi: `go build`, `go test`, `vue-tsc`, vitest, uji API live | ✅ Selesai | Bukti di log bawah |

---

## Checklist Backend (peta dari §7 spesifikasi)

- [x] **B1** — Partial unique index `uniq_cashier_shifts_one_open_per_user` pada `(user_id) WHERE status='OPEN'` (migrasi `000011`).
- [x] **B2** — `CloseShift` menolak shift milik user lain / store lain (`PERMISSION_DENIED`); `UserID` & `StoreID` diambil dari JWT claims.
- [x] **B3** — `CloseShift` atomic `UPDATE … WHERE id=? AND status='OPEN'` + `RowsAffected` → double-close jadi sukses idempoten (diverifikasi: `CLOSE2` → `CLOSED`, tanpa error).
- [x] **B4** — Rekonsiliasi dihitung dari data ter-link (`shift_id` + `payments` + `shift_items`) saat summary/close; `UpdateCashSales/UpdateCashRefund` tidak dipakai untuk `cash_sales` (bebas double-count).
- [x] **B5** — `CreateSale` menolak user shift-capable tanpa shift `OPEN` (`You must open a shift before making a sale` — diverifikasi live).
- [x] **B6** — `GetCurrentShift` memakai `ORDER BY opened_at DESC LIMIT 1`.
- [x] **B7** — Activity log OPEN/CLOSE shift tetap aktif (catatan: ada noise FK store_id nol hanya utk user tanpa store, mis. superadmin — pre-existing, lihat Log).
- [x] **B8** — `sales.shift_id` nullable FK + index; diisi otomatis saat `CreateSale` (diverifikasi di `summary`/DB).
- [x] **B9** — Tabel `shift_items` (ledger insert-only OUT/IN) ditulis **dalam transaksi DB yang sama** dengan sale/refund; detail sampai SKU + snapshot nama/harga.
- [x] **B10** — `GetShiftSummary` berbasis `shift_id` (+ fallback window lama yang dibatasi `closed_at`) & backfill data lama di migrasi.
- [x] **B11** — Validasi silang saat close tahap-2 (anti-mismatch ketat: OUT vs inventory SALE, tidak ada baris setelah close). **Selesai 2026-09-05:** `CloseShift` kini menjalankan reconcile → validasi → close dalam **satu transaksi** yang mengunci baris shift `FOR UPDATE` (penulis sale/refund sudah mengunci baris yang sama via `openShiftOfActor`, jadi terserialisasi — mustahil ada baris `sales`/`shift_items` yang ditulis di antara snapshot & close). `validateShiftIntegrity` menolak close bila ada sale ter-link yang ledger OUT-nya ≠ SALE inventory (per-sale), error `FAILED_PRECONDITION` + nomor invoice. Unit test `TestValidateSaleMovementConsistency` hijau.

## Checklist Frontend (peta dari §8 spesifikasi)

- [x] `fe/src/composables/useCashierShift.ts` — state & aksi shift bersama (recovery idempoten "sudah open/closed").
- [x] `/pos` cek shift saat init (spinner; 404 → layar "Shift belum dibuka").
- [x] `/pos` layar kunci "Buka Shift" saat tidak ada shift open (produk & keranjang & checkout tak bisa dipakai).
- [x] Dialog Buka Shift (anti double-submit `opening`, recovery "sudah open" → jadikan shift aktif).
- [x] Status bar shift OPEN: total transaksi, total penjualan, penjualan tunai, barang keluar/masuk — di-refresh setelah tiap checkout sukses & tombol reload.
- [x] Dialog Tutup Shift: rekonsiliasi server (opening/cash sales/refund/expected) + input actual & closing + preview selisih + konfirmasi saat selisih ≠ 0.
- [x] Daftar barang OUT/IN dari `GET /shifts/{id}/items` (toggle OUT/IN, 50 baris teratas).
- [x] Gate ulang sebelum `createSale` (`ensureShiftOpen`) + keranjang dipertahankan saat shift tertutup.
- [x] i18n EN/ID `posShift.*` (parity test lolos).
- [ ] Refactor `/admin/cashier` memakai composable yang sama (mengurangi duplikasi). **Audit 2026-09-05:** masih terbuka — `CashierPage.vue` memakai `cashierService` + state/dialog sendiri (`openShiftOpen`, `closeShiftOpen`, `currentShift`, dsb.), belum `useCashierShift`.
- [ ] Halaman admin: tampilkan item detail shift (opsional, pasca-utuh). **Audit 2026-09-05:** masih belum (opsional).

---

## Hasil Verifikasi (uji live, container dev)

1. Login superadmin → `GET /cashier/shifts/current` → 200 shift OPEN (route `current` diperbaiki dari konflik `{id}`).
2. `POST /sales` **tanpa shift** → `400 You must open a shift before making a sale` ✅ (B5).
3. Buka shift (opening 100.000) → `POST /sales` 2 pcs Kopi (CASH 30.000) → sukses.
4. `GET /shifts/{id}/summary` → `total_transactions:1, total_sales_amount:30000, cash_sales:30000, expected_cash:130000, items_out_total:2` ✅ (sinkron live).
5. `GET /shifts/{id}/items?direction=OUT` → baris detail `(product_name, sku_code 'TEST-SHIFT-1', variant 'Regular', qty 2, 30000)` ✅ ("tertulis detail").
6. Tutup shift dgn `actual=expected` → `CLOSED`, snapshot `expected_cash=130000`, `cash_sales=30000`; **tutup ulang → sukses idempoten** ✅.
7. Buka shift baru (opening 0) → `POST /sales/{id}/refund` sale lama (CASH 30.000) → sukses; shift baru `cash_refund=30000, items_in_total=2` + baris IN; `expected=-30000`; tutup shift menyimpan refund ✅. Angka shift lama yang sudah ditutup **tidak** berubah.
8. `go build` ✓, `go test ./internal/...` ✓, `vue-tsc --noEmit` ✓, vitest 24/24 ✓, `npm run build` (Astro 20 halaman) ✓.## Log Perubahan / Keputusan

### 2026-09-05 — implementasi B11 (validasi silang ketat saat close)

- **Lokasi:** `internal/cashier/service.go` — `CloseShift`, `computeReconciliationWith`,
  `validateShiftIntegrity` + `loadSaleMovementRows` + `validateSaleMovementConsistency` (murni).
- **Serialisasi:** whole `reconcile → validate → close` sekarang berjalan dalam satu transaksi DB
  yang mengunci baris shift `FOR UPDATE`. Penulis sale/refund (`CreateSale`, refund/cancel) memakai
  kunci yang sama lewat `openShiftOfActor`, sehingga tidak ada `sales`/`shift_items` baru yang bisa
  masuk di antara snapshot rekonsiliasi dan update `CLOSED` — close tidak pernah "kehilangan"
  transaksi dan shift tertutup tidak pernah menerima baris baru (B11 check 3).
- **Cek silang OUT vs SALE inventory (B11 check 2):** untuk tiap sale ter-link `shift_id`,
  `SUM(quantity)` ledger `shift_items(direction=OUT)` harus sama dgn `SUM(quantity)`
  `inventory_transactions(reference_type='SALE')` — keduanya ditulis dalam transaksi sale yang sama.
  Mismatch → `ErrFailedPrecondition` ("Cannot close shift: sale …") berisi nomor invoice.
- **Idempotensi tetap:** status `CLOSED` dicek ulang di dalam lock; double-close tetap sukses tanpa error.
- **Catatan:** `computeReconciliation` di-refactor jadi `computeReconciliationWith(ctx, db, shift)`
  (db = handle tx saat close); query payment dipindah ke handle yang sama (bukan repo terpisah).
  `repository.CloseShift` (metode lama) tidak lagi dipakai service tapi dipertahankan di interface.
- Verifikasi: `go vet` ✓, `go test ./internal/...` ✓ (termasuk test baru cashier), `go build ./...` ✓.

### 2026-09-05 — putaran perbaikan ("fix sekalian jika ada issue")

1. **Drift stok saat refund/cancel penjualan per-SKU** (bug nyata) — penjualan SKU
   menurunkan `product_skus.stock` (dan `products.stock` = Σ SKU), tapi refund/cancel lama
   hanya menambah `products.stock` → agregat & SKU melenceng, ledger IN tak bisa menyebut SKU.
   - Migrasi `000012`: `sale_items.product_sku_id` (nullable FK, `ON DELETE SET NULL`) + index.
   - `CreateSale` menyimpan SKU yang dijual di `sale_items`; `restoreSaleIntoShift` mengembalikan
     ke SKU yang tepat lalu merekomputasi `products.stock`; baris IN kini `item_type=SKU` + kode/varian.
   - Diverifikasi live: jual 5 (stok 20→15), refund (15→20), `products.stock == SKU stock == 20` ✅.
2. **Activity log gagal utk user tanpa store/claim store kosong** (bug lama, memutus audit trail
   superadmin): `logParams` tidak mengisi `store_id` dari claims → insert store UUID nol → pelanggaran
   FK `activity_logs_store_id_fkey` ditelan diam-diam. Sekarang `store_id` diisi dari claims saat ada;
   bila kosong dibiarkan `NULL`. Terverifikasi: aksi superadmin tersimpan tanpa error FK ✅.
3. **Tenancy guard `OpenShift`**: user ber-store sekarang tak bisa membuka shift untuk store lain
   (`PERMISSION_DENIED` bila `store_id` request ≠ store di JWT) — simetris dengan `CloseShift`.
4. **FE re-sync saat shift tertutup di tab/perangkat lain**: jika `createSale` ditolak karena
   shift tertutup/tidak open, halaman POS otomatis `refreshLive()` → kembali ke layar "Buka Shift"
   (keranjang dipertahankan), bukan hanya toast error.
5. **FE koersi angka** saat input actual/closing dibersihkan (`''`) sebelum dikirim ke server.

Verifikasi putaran: `go build`/`go test ./internal/...` ✅, `vue-tsc` ✅, vitest 24/24 ✅,
`npm run build` ✅, migrasi `000012` terpasang (version 12), log FK/product_sk_uid = 0 ✅.

### 2026-09-05 — putaran audit (cek checklist vs kode aktual)

Pemeriksaan ulang seluruh checklist `pos-cashier-shift.md` terhadap kode sekarang:

1. **Backend B1–B10 semua terverifikasi** di kode: index anti-double-open, close milik
   sendiri + idempoten (`RowsAffected`), rekonsiliasi berbasis `shift_id`, gate `CreateSale`
   tanpa shift, ledger `shift_items` dlm transaksi sama, `GetShiftSummary`/backfill.
   **B11 masih terbuka** (tidak ada pemeriksaan ketat mismatch saat close — detail di checklist).
2. **Frontend `/pos` item shift**: semua [x] di checklist valid. Implementasi dialog shift
   kini tinggal di `PosShiftDialogs.vue` (buka/tutup/rekonsiliasi + daftar barang OUT/IN),
   induk `POSCheckoutPage.vue` hanya menyimpan status & aksi (composable `useCashierShift`).
3. **Pecahan komponen `/pos`** (refactor 2026-09-05, bukan ubah perilaku):
   `POSCheckoutPage.vue` (1221 → 732 baris) → `PosHeldOrders.vue`, `PosPaymentDialog.vue`,
   `PosProductBrowser.vue`, `PosShiftDialogs.vue`. Verifikasi: `vue-tsc`, vitest 31/31,
   `npm run build` (20 halaman) hijau.
4. **Fitur hold order** (fitur baru terkait `/pos`, spec `docs/pos-hold-order.md`):
   localStorage `pos.held_orders.v1`, composable `useHeldOrders.ts`, UI di `PosHeldOrders.vue`,
   unit test `tests/unit/posHold.test.ts` (7 test) — tidak menyentuh backend shift.
5. **Masih terbuka setelah audit:** refactor `/admin/cashier` → `useCashierShift`;
   swagger `ListShiftItems` & field summary baru; unit test composable shift; E2E shift/refund;
   B11. (detail di checklist & backlog bawah).

### 2026-09-05 — implementasi inti

- **Keputusan `cash_sales`:** Σ **grand_total** sale ber-metode `CASH` (bukan Σ `payments.amount`).
  Uang di laci bertambah **neto** (tendered − change) = `grand_total`; memakai `payments.amount`
  (bisa > grand total bila ada kembalian) membuat `expected_cash` meleset dari laci nyata.
  Konsisten dengan contoh AGENTS.md (Cash Sales 500 → Expected 550).
- **Keputusan refund:** `cash_refund` & baris `IN` dicatat ke shift **aktif milik aktor saat refund
  terjadi** (shift berikutnya jika sale lama sudah ditutup) — sesuai §5.3/§5.7. Sale yang di-refund
  tetap dihitung di pendapatan shift asalnya (model "uang masuk lalu keluar"); refund adalah arus
  keluar shift tempat refund terjadi (dapat `expected` negatif → wajar bila ditarik dari kas shift itu).
- **Perbaikan ikutan (bug ditemukan saat uji):**
  - Kolom GORM `ProductSKUID` butuh tag `column:product_sku_id` (tanpa tag GORM menulis `product_sk_uid`).
  - `CancelSale`/`RefundSale` sebelumnya `tx.Save(sale)` → menulis ulang `customer_id` UUID nol utk
    sale tanpa pelanggan → melanggar FK. Diubah jadi update kolom `status` saja.
  - Route REST `GET /shifts/current` tersaingi pola `GET /shifts/{id}` (grpc-gateway memenangkan `{id}`).
    Route `GetShift` dipindah ke `/api/v1/cashier/shifts/detail/{id}` sehingga `/current` kembali berfungsi.
- **Catatan aktivitas:** `activity_logs` utk user tanpa store (superadmin) gagal FK `store_id` nol —
  sudah diperbaiki di putaran perbaikan (claims store dipakai bila ada; selain itu NULL).
- **Catatan `schema_migrations`:** saat pengerjaan, tabel di DB dev hanya berisi `version 2 dirty`;
  dipaksa `force 10` (semua objek 1–10 sudah terpasang) lalu `up` → version 11 bersih.

## Pekerjaan Berikutnya (backlog)

- [ ] Refactor `/admin/cashier` ke `useCashierShift` (hapus duplikasi state open/close).
- [x] B11: validasi silang ketat saat close + uji mismatch — **selesai 2026-09-05** (lihat checklist B11 & log implementasi). Uji mismatch unit (`TestValidateSaleMovementConsistency`) + integrasi menyusul via E2E/DB test.
- [ ] Test unit Vitest utk composable shift (`useCashierShift`: recovery idempoten, `ensureShiftOpen`, gate) — saat ini baru ada unit test `posHold` (fitur hold), `i18n`, `validators`, `rbac`.
- [ ] Uji E2E Playwright shift (matriks §9 spesifikasi) — `tests/e2e/` baru berisi `reload-check.spec.ts`; refund + matriks shift belum ada.
- [x] Perbaiki `activity_logs.store_id` utk user tanpa store — **selesai** di putaran perbaikan (claims `store_id` dipakai bila ada; selain itu `NULL`; tidak ada lagi error FK).
- [ ] Update `docs/swagger-enhanced.json` utk `ListShiftItems` (`/shifts/{id}/items`) & field summary baru (`items_out_total`, `items_in_total`, `revenue_by_method`, …) — audit: belum ada.
- [x] Migrasi `000012` (sale_items.product_sku_id) — **sudah ditulis & terpasang** (version 12); tinggal ikut matriks E2E refund di atas.

<!-- di-update manual tiap milestone -->
