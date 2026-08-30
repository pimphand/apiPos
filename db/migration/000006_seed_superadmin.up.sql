-- Seed a default superadmin so the system can bootstrap store verification.
-- Password: SuperAdmin123!  (change after first login)
-- This is idempotent; run only if a superadmin user does not already exist OR
-- the username 'superadmin' is free.

INSERT INTO users (username, email, password_hash, full_name, role_id, store_id, is_active)
SELECT
    'superadmin',
    'superadmin@example.com',
    '$2a$10$KVz92BKQyXj9rTO5g.tf.epQlYJ3Hu4kWZigKLlAVptHaGqOeLiTm',
    'Super Admin',
    id,
    '00000000-0000-0000-0000-000000000001',
    true
FROM roles
WHERE name = 'superadmin'
  AND NOT EXISTS (SELECT 1 FROM users WHERE username = 'superadmin')
ON CONFLICT (username) DO NOTHING;
