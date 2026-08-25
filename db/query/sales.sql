-- name: CreateSale :one
INSERT INTO sales (invoice_number, store_id, cashier_id, customer_id, subtotal, discount, tax, grand_total, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetSaleByID :one
SELECT * FROM sales WHERE id = $1;

-- name: GetSaleByInvoiceNumber :one
SELECT * FROM sales WHERE invoice_number = $1;

-- name: ListSales :many
SELECT * FROM sales
WHERE ($1::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR store_id = $1)
  AND ($2::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR cashier_id = $2)
  AND ($3::VARCHAR = '' OR status = $3)
  AND ($4::TIMESTAMPTZ = '1970-01-01' OR created_at >= $4)
  AND ($5::TIMESTAMPTZ = '1970-01-01' OR created_at <= $5)
ORDER BY created_at DESC
LIMIT $6 OFFSET $7;

-- name: UpdateSaleStatus :exec
UPDATE sales
SET status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSale :one
UPDATE sales
SET customer_id = COALESCE($2, customer_id),
    subtotal = COALESCE($3, subtotal),
    discount = COALESCE($4, discount),
    tax = COALESCE($5, tax),
    grand_total = COALESCE($6, grand_total),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteSale :exec
DELETE FROM sales WHERE id = $1;

-- name: CountSales :one
SELECT COUNT(*) FROM sales;

-- name: GetDailySales :many
SELECT
    DATE(created_at) as sale_date,
    COUNT(*) as total_transactions,
    SUM(grand_total) as total_amount,
    SUM(discount) as total_discount,
    SUM(tax) as total_tax
FROM sales
WHERE status = 'PAID'
  AND ($1::TIMESTAMPTZ = '1970-01-01' OR created_at >= $1)
  AND ($2::TIMESTAMPTZ = '1970-01-01' OR created_at <= $2)
GROUP BY DATE(created_at)
ORDER BY sale_date DESC;

-- name: GetSalesByCashier :many
SELECT
    u.id as cashier_id,
    u.full_name as cashier_name,
    COUNT(s.id) as total_transactions,
    SUM(s.grand_total) as total_amount
FROM sales s
JOIN users u ON s.cashier_id = u.id
WHERE s.status = 'PAID'
  AND ($1::TIMESTAMPTZ = '1970-01-01' OR s.created_at >= $1)
  AND ($2::TIMESTAMPTZ = '1970-01-01' OR s.created_at <= $2)
GROUP BY u.id, u.full_name
ORDER BY total_amount DESC;

-- name: GetSalesByProduct :many
SELECT
    p.id as product_id,
    p.sku,
    p.name as product_name,
    SUM(si.quantity) as total_quantity,
    SUM(si.subtotal) as total_amount
FROM sale_items si
JOIN products p ON si.product_id = p.id
JOIN sales s ON si.sale_id = s.id
WHERE s.status = 'PAID'
  AND ($1::TIMESTAMPTZ = '1970-01-01' OR s.created_at >= $1)
  AND ($2::TIMESTAMPTZ = '1970-01-01' OR s.created_at <= $2)
GROUP BY p.id, p.sku, p.name
ORDER BY total_amount DESC;

-- name: GetSalesByCategory :many
SELECT
    c.id as category_id,
    c.name as category_name,
    SUM(si.quantity) as total_quantity,
    SUM(si.subtotal) as total_amount
FROM sale_items si
JOIN products p ON si.product_id = p.id
JOIN categories c ON p.category_id = c.id
JOIN sales s ON si.sale_id = s.id
WHERE s.status = 'PAID'
  AND ($1::TIMESTAMPTZ = '1970-01-01' OR s.created_at >= $1)
  AND ($2::TIMESTAMPTZ = '1970-01-01' OR s.created_at <= $2)
GROUP BY c.id, c.name
ORDER BY total_amount DESC;

-- name: CreateSaleItem :one
INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, discount, subtotal)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSaleItemsBySaleID :many
SELECT * FROM sale_items WHERE sale_id = $1;

-- name: DeleteSaleItemsBySaleID :exec
DELETE FROM sale_items WHERE sale_id = $1;

-- name: GetPaymentSummary :many
SELECT
    payment_method,
    COUNT(*) as count,
    SUM(amount) as total_amount
FROM payments
WHERE ($1::TIMESTAMPTZ = '1970-01-01' OR paid_at >= $1)
  AND ($2::TIMESTAMPTZ = '1970-01-01' OR paid_at <= $2)
GROUP BY payment_method;
