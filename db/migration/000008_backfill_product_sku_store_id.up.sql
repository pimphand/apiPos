-- 000008_backfill_product_sku_store_id.up.sql
-- Rows created before store-level tenant scoping was enforced fell back to the
-- legacy default store (00000000-0000-0000-0000-000000000001) while their
-- parent product belongs to a real store. Adopt the parent product's store so
-- these SKUs are visible/usable within the correct tenant.
UPDATE product_skus s
SET store_id = p.store_id
FROM products p
WHERE s.product_id = p.id
  AND s.store_id = '00000000-0000-0000-0000-000000000001'
  AND p.store_id <> '00000000-0000-0000-0000-000000000001';