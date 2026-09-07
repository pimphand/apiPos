-- 000017_tables_self_order.up.sql
-- Self order (dine-in via QR meja): meja, kolom sales, toggle store_settings.

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