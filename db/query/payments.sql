-- name: CreatePayment :one
INSERT INTO payments (sale_id, payment_method, amount, reference, paid_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1;

-- name: GetPaymentsBySaleID :many
SELECT * FROM payments WHERE sale_id = $1 ORDER BY paid_at;

-- name: ListPayments :many
SELECT * FROM payments
WHERE ($1::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR sale_id = $1)
  AND ($2::VARCHAR = '' OR payment_method = $2)
  AND ($3::TIMESTAMPTZ = '1970-01-01' OR paid_at >= $3)
  AND ($4::TIMESTAMPTZ = '1970-01-01' OR paid_at <= $4)
ORDER BY paid_at DESC
LIMIT $5 OFFSET $6;

-- name: DeletePayment :exec
DELETE FROM payments WHERE id = $1;

-- name: GetTotalPaymentsBySaleID :one
SELECT COALESCE(SUM(amount), 0) as total_paid FROM payments WHERE sale_id = $1;
