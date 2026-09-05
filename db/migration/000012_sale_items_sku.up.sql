-- 000012_sale_items_sku.up.sql
-- The POS sells per SKU, but sale_items only stored product_id. On refund /
-- cancel the code restored products.stock directly even though SKU sales keep
-- products.stock = SUM(product_skus.stock) — the aggregate drifted apart from
-- the variants. Record which SKU was sold so restore is exact (and the shift
-- IN ledger can name the variant).
ALTER TABLE sale_items
    ADD COLUMN product_sku_id UUID NULL REFERENCES product_skus(id) ON DELETE SET NULL;

CREATE INDEX idx_sale_items_product_sku_id ON sale_items(product_sku_id);
