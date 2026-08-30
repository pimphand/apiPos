-- 000005_store_multitenancy.up.sql
-- Adds store-level multi-tenancy: a NOT NULL store_id on every tenant-scoped
-- table, seeded with a single default/primary store so existing rows remain
-- valid, plus a protected 'superadmin' role used to verify stores.

-- 1. Add a verification flag to stores and seed a default store (idempotent).
--    All existing rows are backfilled with this store's id until real tenant
--    scoping is applied.
ALTER TABLE stores ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT false;

INSERT INTO stores (id, name, address, phone, is_active, is_verified)
SELECT '00000000-0000-0000-0000-000000000001', 'Default Store', NULL, NULL, true, true
WHERE NOT EXISTS (SELECT 1 FROM stores WHERE id = '00000000-0000-0000-0000-000000000001');

-- 2. Add the protected 'superadmin' role (idempotent).
INSERT INTO roles (name, description)
SELECT 'superadmin', 'System-wide role: manages admins and verifies stores'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'superadmin');

-- 3. Add store_id to every tenant-scoped table, backfilled to the default store.
ALTER TABLE users                     ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE categories                ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE products                  ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE product_skus              ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE customers                 ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE sales                     ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE sale_items                ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE payments                  ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE inventory_transactions    ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE cashier_shifts            ADD COLUMN store_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES stores(id);
ALTER TABLE activity_logs             ADD COLUMN store_id UUID NULL REFERENCES stores(id);

-- 4. Indexes for every tenant-scoped table store_id.
CREATE INDEX IF NOT EXISTS idx_users_store_id                ON users(store_id);
CREATE INDEX IF NOT EXISTS idx_categories_store_id           ON categories(store_id);
CREATE INDEX IF NOT EXISTS idx_products_store_id             ON products(store_id);
CREATE INDEX IF NOT EXISTS idx_product_skus_store_id         ON product_skus(store_id);
CREATE INDEX IF NOT EXISTS idx_customers_store_id            ON customers(store_id);
CREATE INDEX IF NOT EXISTS idx_sales_store_id                ON sales(store_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_store_id           ON sale_items(store_id);
CREATE INDEX IF NOT EXISTS idx_payments_store_id             ON payments(store_id);
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_store_id ON inventory_transactions(store_id);
CREATE INDEX IF NOT EXISTS idx_cashier_shifts_store_id       ON cashier_shifts(store_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_store_id        ON activity_logs(store_id);

-- 5. Grant superadmin the same permissions as admin (all permissions).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'superadmin'
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    JOIN roles rr ON rr.id = rp.role_id
    WHERE rr.name = 'superadmin' AND rp.permission_id = p.id
);
