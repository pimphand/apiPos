-- name: CreateCustomer :one
INSERT INTO customers (code, name, email, phone, address)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCustomerByID :one
SELECT * FROM customers WHERE id = $1;

-- name: GetCustomerByCode :one
SELECT * FROM customers WHERE code = $1;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE ($1::BOOLEAN = false OR is_active = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SearchCustomers :many
SELECT * FROM customers
WHERE (name ILIKE '%' || $1 || '%'
    OR code ILIKE '%' || $1 || '%'
    OR phone ILIKE '%' || $1 || '%')
  AND ($2::BOOLEAN = false OR is_active = $2)
ORDER BY name
LIMIT $3 OFFSET $4;

-- name: UpdateCustomer :one
UPDATE customers
SET name = COALESCE($2, name),
    email = COALESCE($3, email),
    phone = COALESCE($4, phone),
    address = COALESCE($5, address),
    is_active = COALESCE($6, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;

-- name: DeactivateCustomer :exec
UPDATE customers
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: CountCustomers :one
SELECT COUNT(*) FROM customers;

-- name: GetCustomerPurchaseHistory :many
SELECT
    s.id,
    s.invoice_number,
    s.subtotal,
    s.discount,
    s.tax,
    s.grand_total,
    s.status,
    s.created_at
FROM sales s
WHERE s.customer_id = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;
