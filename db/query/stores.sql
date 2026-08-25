-- name: CreateStore :one
INSERT INTO stores (name, address, phone)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetStoreByID :one
SELECT * FROM stores WHERE id = $1;

-- name: ListStores :many
SELECT * FROM stores
WHERE ($1::BOOLEAN = false OR is_active = $1)
ORDER BY name;

-- name: UpdateStore :one
UPDATE stores
SET name = COALESCE($2, name),
    address = COALESCE($3, address),
    phone = COALESCE($4, phone),
    is_active = COALESCE($5, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateStore :exec
UPDATE stores
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteStore :exec
DELETE FROM stores WHERE id = $1;

-- name: CountStores :one
SELECT COUNT(*) FROM stores;
