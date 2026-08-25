-- name: CreateInventoryTransaction :one
INSERT INTO inventory_transactions (product_id, store_id, type, quantity, reference_type, reference_id, before_stock, after_stock, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetInventoryTransactionByID :one
SELECT * FROM inventory_transactions WHERE id = $1;

-- name: ListInventoryTransactions :many
SELECT * FROM inventory_transactions
WHERE ($1::BIGINT = 0 OR product_id = $1)
  AND ($2::BIGINT = 0 OR store_id = $2)
  AND ($3::VARCHAR = '' OR type = $3)
  AND ($4::TIMESTAMPTZ = '1970-01-01' OR created_at >= $4)
  AND ($5::TIMESTAMPTZ = '1970-01-01' OR created_at <= $5)
ORDER BY created_at DESC
LIMIT $6 OFFSET $7;

-- name: GetInventoryTransactionsByProduct :many
SELECT * FROM inventory_transactions
WHERE product_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetInventorySummary :many
SELECT
    p.id as product_id,
    p.sku,
    p.name as product_name,
    p.stock as current_stock,
    p.minimum_stock,
    COUNT(it.id) as total_transactions
FROM products p
LEFT JOIN inventory_transactions it ON p.id = it.product_id
WHERE ($1::BIGINT = 0 OR p.category_id = $1)
  AND p.is_active = true
GROUP BY p.id, p.sku, p.name, p.stock, p.minimum_stock
ORDER BY p.name;
