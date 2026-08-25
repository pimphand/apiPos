-- name: CreateProduct :one
INSERT INTO products (sku, barcode, name, description, category_id, cost_price, selling_price, stock, minimum_stock)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = $1;

-- name: GetProductByBarcode :one
SELECT * FROM products WHERE barcode = $1 AND barcode IS NOT NULL;

-- name: ListProducts :many
SELECT * FROM products
WHERE ($1::BOOLEAN = false OR is_active = $1)
  AND ($2::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR category_id = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountProducts :one
SELECT COUNT(*) FROM products
WHERE ($1::BOOLEAN = false OR is_active = $1)
  AND ($2::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR category_id = $2);

-- name: SearchProductsByName :many
SELECT * FROM products
WHERE name ILIKE '%' || $1 || '%'
  AND ($2::BOOLEAN = false OR is_active = $2)
ORDER BY name
LIMIT $3 OFFSET $4;

-- name: CountSearchProductsByName :one
SELECT COUNT(*) FROM products
WHERE name ILIKE '%' || $1 || '%'
  AND ($2::BOOLEAN = false OR is_active = $2);

-- name: UpdateProduct :one
UPDATE products
SET sku = COALESCE($2, sku),
    barcode = COALESCE($3, barcode),
    name = COALESCE($4, name),
    description = COALESCE($5, description),
    category_id = COALESCE($6, category_id),
    cost_price = COALESCE($7, cost_price),
    selling_price = COALESCE($8, selling_price),
    minimum_stock = COALESCE($9, minimum_stock),
    is_active = COALESCE($10, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProductStock :exec
UPDATE products
SET stock = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: DecreaseProductStock :exec
UPDATE products
SET stock = stock - $2,
    updated_at = NOW()
WHERE id = $1 AND stock >= $2;

-- name: IncreaseProductStock :exec
UPDATE products
SET stock = stock + $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ActivateProduct :exec
UPDATE products
SET is_active = true,
    updated_at = NOW()
WHERE id = $1;

-- name: DeactivateProduct :exec
UPDATE products
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;

-- name: GetLowStockProducts :many
SELECT * FROM products
WHERE stock <= minimum_stock AND is_active = true
ORDER BY stock ASC
LIMIT $1 OFFSET $2;

-- name: CountLowStockProducts :one
SELECT COUNT(*) FROM products
WHERE stock <= minimum_stock AND is_active = true;

-- name: GetProductStockForUpdate :one
SELECT id, stock FROM products WHERE id = $1 FOR UPDATE;
