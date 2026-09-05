-- 000007_product_sku_optional.up.sql
-- Products store aggregate stock; individual SKUs/Barcodes are managed in
-- product_skus. Relax the legacy products.sku/barcode columns so a product
-- without a variant-level SKU can still be created.

ALTER TABLE products ALTER COLUMN sku DROP NOT NULL;

-- Uniqueness is enforced per SKU in product_skus, not at the product level.
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_sku_key;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_barcode_key;