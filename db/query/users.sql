-- name: CreateRole :one
INSERT INTO roles (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY name;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name, role_id, store_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE ($1::UUID = '00000000-0000-0000-0000-000000000000'::UUID OR role_id = $1)
  AND ($2::TEXT = '' OR is_active = ($2::TEXT = 'true'))
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateUser :one
UPDATE users
SET full_name = COALESCE($2, full_name),
    email = COALESCE($3, email),
    role_id = COALESCE($4, role_id),
    store_id = COALESCE($5, store_id),
    is_active = COALESCE($6, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = true;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1 AND is_revoked = false;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = true
WHERE token = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET is_revoked = true
WHERE user_id = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < NOW() OR is_revoked = true;
