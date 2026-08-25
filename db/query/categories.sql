-- name: CreateCategory :one
INSERT INTO categories (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- name: ListCategories :many
SELECT * FROM categories
WHERE ($1::BOOLEAN = false OR is_active = $1)
ORDER BY name;

-- name: UpdateCategory :one
UPDATE categories
SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    is_active = COALESCE($4, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-- name: DeactivateCategory :exec
UPDATE categories
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;
