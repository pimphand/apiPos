-- 000012_sale_items_sku.down.sql
DROP INDEX IF EXISTS idx_sale_items_product_sku_id;
ALTER TABLE sale_items DROP COLUMN IF EXISTS product_sku_id;
