-- 000017_tables_self_order.down.sql
ALTER TABLE store_settings DROP COLUMN IF EXISTS self_order_enabled;

DROP INDEX IF EXISTS idx_sales_table_id;
DROP INDEX IF EXISTS idx_sales_order_status;
DROP INDEX IF EXISTS idx_sales_order_source;

ALTER TABLE sales DROP COLUMN IF EXISTS order_token;
ALTER TABLE sales DROP COLUMN IF EXISTS table_no;
ALTER TABLE sales DROP COLUMN IF EXISTS table_id;
ALTER TABLE sales DROP COLUMN IF EXISTS order_status;
ALTER TABLE sales DROP COLUMN IF EXISTS order_source;

-- Kembalikan NOT NULL (gagal bila masih ada baris NULL — wajar saat rollback
-- migrasi ini, karena belum ada sale self-order yang tersisa).
ALTER TABLE inventory_transactions ALTER COLUMN created_by SET NOT NULL;
ALTER TABLE sales ALTER COLUMN cashier_id SET NOT NULL;

DROP TABLE IF EXISTS tables;