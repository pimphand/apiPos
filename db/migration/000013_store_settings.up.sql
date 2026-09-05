-- 000013_store_settings.up.sql
-- Per-store settings (default tax rate + receipt/nota format). One row per store.

CREATE TABLE store_settings (
    store_id            UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
    tax_rate            NUMERIC(5,2) NOT NULL DEFAULT 11.00,
    receipt_header      TEXT NOT NULL DEFAULT '',
    receipt_footer      TEXT NOT NULL DEFAULT '',
    show_store_name     BOOLEAN NOT NULL DEFAULT TRUE,
    show_store_contact  BOOLEAN NOT NULL DEFAULT TRUE,
    show_cashier        BOOLEAN NOT NULL DEFAULT TRUE,
    show_customer       BOOLEAN NOT NULL DEFAULT TRUE,
    show_item_sku       BOOLEAN NOT NULL DEFAULT FALSE,
    show_item_barcode   BOOLEAN NOT NULL DEFAULT FALSE,
    show_discount       BOOLEAN NOT NULL DEFAULT TRUE,
    show_tax            BOOLEAN NOT NULL DEFAULT TRUE,
    show_payment        BOOLEAN NOT NULL DEFAULT TRUE,
    show_change         BOOLEAN NOT NULL DEFAULT TRUE,
    updated_by          UUID REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
