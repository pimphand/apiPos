-- 000007_product_sku_optional.down.sql
ALTER TABLE products ADD CONSTRAINT products_barcode_key UNIQUE (barcode);
ALTER TABLE products ADD CONSTRAINT products_sku_key UNIQUE (sku);
ALTER TABLE products ALTER COLUMN sku SET NOT NULL;