-- Disable the seeded account if it still uses its original credentials.
WITH disabled AS (
    UPDATE users
    SET is_active = false, password_hash = 'disabled', updated_at = now()
    WHERE username = 'superadmin' AND password_hash = '$2a$10$KVz92BKQyXj9rTO5g.tf.epQlYJ3Hu4kWZigKLlAVptHaGqOeLiTm'
    RETURNING id
)
UPDATE refresh_tokens
SET is_revoked = true
WHERE user_id IN (SELECT id FROM disabled) AND NOT is_revoked;
