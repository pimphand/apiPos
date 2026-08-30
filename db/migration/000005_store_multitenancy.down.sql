-- 000005_store_multitenancy.down.sql
-- Reverses the store multi-tenancy migration.

-- Revoke superadmin role permissions.
DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'superadmin');

-- Drop the superadmin role.
DELETE FROM roles WHERE name = 'superadmin';

-- Drop store_id indexes.
DROP INDEX IF EXISTS idx_users_store_id;
DROP INDEX IF EXISTS idx_categories_store_id;
DROP INDEX IF EXISTS idx_products_store_id;
DROP INDEX IF EXISTS idx_product_skus_store_id;
DROP INDEX IF EXISTS idx_customers_store_id;
DROP INDEX IF EXISTS idx_sales_store_id;
DROP INDEX IF EXISTS idx_sale_items_store_id;
DROP INDEX IF EXISTS idx_payments_store_id;
DROP INDEX IF EXISTS idx_inventory_transactions_store_id;
DROP INDEX IF EXISTS idx_cashier_shifts_store_id;
DROP INDEX IF EXISTS idx_activity_logs_store_id;

-- Drop store_id columns.
ALTER TABLE activity_logs             DROP COLUMN IF EXISTS store_id;
ALTER TABLE cashier_shifts            DROP COLUMN IF EXISTS store_id;
ALTER TABLE inventory_transactions    DROP COLUMN IF EXISTS store_id;
ALTER TABLE payments                  DROP COLUMN IF EXISTS store_id;
ALTER TABLE sale_items                DROP COLUMN IF EXISTS store_id;
ALTER TABLE sales                     DROP COLUMN IF EXISTS store_id;
ALTER TABLE customers                 DROP COLUMN IF EXISTS store_id;
ALTER TABLE product_skus              DROP COLUMN IF EXISTS store_id;
ALTER TABLE products                  DROP COLUMN IF EXISTS store_id;
ALTER TABLE categories                DROP COLUMN IF EXISTS store_id;
ALTER TABLE users                     DROP COLUMN IF EXISTS store_id;

-- Remove the default store only if it is no longer referenced by other rows.
DELETE FROM stores
WHERE id = '00000000-0000-0000-0000-000000000001'
AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                WHERE table_schema = 'public'
                AND (table_name = 'users' AND column_name = 'store_id'))
AND NOT EXISTS (SELECT 1 FROM users WHERE store_id = '00000000-0000-0000-0000-000000000001');

-- Drop the store verification flag.
ALTER TABLE stores DROP COLUMN IF EXISTS is_verified;
