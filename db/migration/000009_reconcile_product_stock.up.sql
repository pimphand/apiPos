-- 000009_reconcile_product_stock.up.sql
-- products.stock is the aggregate of its SKUs' stock (per the spec). Rows can
-- drift apart when SKUs were created before tenant scoping was enforced, when a
-- sale was recorded without a sku_id, or when SKU stock was edited directly.
-- Reconcile the aggregate from the variants.
UPDATE products p
SET stock = agg.total_stock
FROM (
    SELECT product_id, SUM(stock)::int AS total_stock
    FROM product_skus
    GROUP BY product_id
) agg
WHERE p.id = agg.product_id
  AND p.stock <> agg.total_stock;