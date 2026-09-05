-- 000011_cashier_shift_sync_hardening.up.sql
-- Implements docs/pos-cashier-shift.md §5 (sinkronisasi shift dgn transaksi)
-- and §7 hardening (B1, B8, B9, B10).

-- B1: one OPEN shift per user, enforced at the DB level so double-open is
-- impossible no matter how many requests race.
CREATE UNIQUE INDEX uniq_cashier_shifts_one_open_per_user
    ON cashier_shifts(user_id)
    WHERE status = 'OPEN';

-- B8: explicit sale -> shift relation (single source of truth instead of a
-- time window). Nullable so sales made without an open shift (non-cashier
-- roles) keep working with shift_id = NULL.
ALTER TABLE sales
    ADD COLUMN shift_id UUID NULL REFERENCES cashier_shifts(id);

CREATE INDEX idx_sales_shift_id ON sales(shift_id);

-- B9: insert-only per-item ledger for goods going OUT (sale) / IN (refund,
-- cancel) during a shift. Every movement is a permanent row so shift detail
-- is recorded "down to the SKU".
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
    sku_code        VARCHAR(100),
    variant_label   VARCHAR(255),
    quantity        NUMERIC(12,3) NOT NULL CHECK (quantity > 0),
    unit_price      NUMERIC(15,2) NOT NULL,
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    line_amount     NUMERIC(15,2) NOT NULL,
    payment_method  VARCHAR(20),
    reference_no    VARCHAR(100),
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shift_items_shift_id   ON shift_items(shift_id);
CREATE INDEX idx_shift_items_product_id ON shift_items(product_id);
CREATE INDEX idx_shift_items_sku_id     ON shift_items(product_sku_id);
CREATE INDEX idx_shift_items_direction  ON shift_items(shift_id, direction);

-- Snapshot aggregate columns on cashier_shifts: frozen at close time so a
-- closed shift's headline numbers never change even if history is edited.
ALTER TABLE cashier_shifts
    ADD COLUMN total_transactions INT NOT NULL DEFAULT 0,
    ADD COLUMN total_sales_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN total_revenue      NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN revenue_cash       NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN total_items_out    NUMERIC(12,3) NOT NULL DEFAULT 0,
    ADD COLUMN total_items_in     NUMERIC(12,3) NOT NULL DEFAULT 0;

-- B10 (backfill): link historical CLOSED shifts to their sales using the
-- (bounded) opened_at..closed_at window. For each sale we pick the most
-- recently opened shift that covers the sale so overlapping windows never
-- double-assign a sale.
UPDATE sales s
SET shift_id = cs.shift_id
FROM (
    SELECT DISTINCT ON (sl.id) sl.id AS sale_id, cs.id AS shift_id
    FROM sales sl
    JOIN cashier_shifts cs
      ON cs.user_id   = sl.cashier_id
     AND cs.store_id  = sl.store_id
     AND cs.status    = 'CLOSED'
     AND sl.created_at >= cs.opened_at
     AND cs.closed_at IS NOT NULL
     AND sl.created_at <= cs.closed_at
    ORDER BY sl.id, cs.opened_at DESC
) cs
WHERE s.id = cs.sale_id
  AND s.shift_id IS NULL;

-- Frozen snapshots for already-closed shifts so old rows also show numbers
-- (best effort; figures are still recomputable from linked sales).
UPDATE cashier_shifts cs
SET total_transactions = sub.cnt,
    total_sales_amount = sub.amt,
    total_revenue      = sub.amt
FROM (
    SELECT shift_id,
           COUNT(*)::int                AS cnt,
           COALESCE(SUM(grand_total),0) AS amt
    FROM sales
    WHERE shift_id IS NOT NULL
      AND status = 'PAID'
    GROUP BY shift_id
) sub
WHERE cs.id = sub.shift_id
  AND cs.status = 'CLOSED';
