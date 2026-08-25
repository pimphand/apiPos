-- name: OpenCashierShift :one
INSERT INTO cashier_shifts (user_id, store_id, opening_cash, status)
VALUES ($1, $2, $3, 'OPEN')
RETURNING *;

-- name: CloseCashierShift :one
UPDATE cashier_shifts
SET status = 'CLOSED',
    closing_cash = $2,
    cash_sales = $3,
    cash_refund = $4,
    expected_cash = $5,
    actual_cash = $6,
    cash_difference = $7,
    closed_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND status = 'OPEN'
RETURNING *;

-- name: GetCurrentShift :one
SELECT * FROM cashier_shifts
WHERE user_id = $1 AND status = 'OPEN'
ORDER BY opened_at DESC
LIMIT 1;

-- name: GetShiftByID :one
SELECT * FROM cashier_shifts WHERE id = $1;

-- name: ListShifts :many
SELECT * FROM cashier_shifts
WHERE ($1::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR user_id = $1)
  AND ($2::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR store_id = $2)
  AND ($3::VARCHAR = '' OR status = $3)
ORDER BY opened_at DESC
LIMIT $4 OFFSET $5;

-- name: UpdateShiftCashSales :exec
UPDATE cashier_shifts
SET cash_sales = cash_sales + $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'OPEN';

-- name: UpdateShiftCashRefund :exec
UPDATE cashier_shifts
SET cash_refund = cash_refund + $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'OPEN';

-- name: CountOpenShifts :one
SELECT COUNT(*) FROM cashier_shifts WHERE status = 'OPEN';
