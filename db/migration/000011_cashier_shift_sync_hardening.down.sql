-- 000011_cashier_shift_sync_hardening.down.sql
DROP TABLE IF EXISTS shift_items;

DROP INDEX IF EXISTS idx_sales_shift_id;
ALTER TABLE sales DROP COLUMN IF EXISTS shift_id;

ALTER TABLE cashier_shifts
    DROP COLUMN IF EXISTS total_transactions,
    DROP COLUMN IF EXISTS total_sales_amount,
    DROP COLUMN IF EXISTS total_revenue,
    DROP COLUMN IF EXISTS revenue_cash,
    DROP COLUMN IF EXISTS total_items_out,
    DROP COLUMN IF EXISTS total_items_in;

DROP INDEX IF EXISTS uniq_cashier_shifts_one_open_per_user;
