-- Seed a default superadmin so the system can bootstrap store verification.
-- Password: SuperAdmin123!  (change after first login)
-- This migration is idempotent.

-- 1. Create the superadmin role if it doesn't already exist.
INSERT INTO roles (name, description)
SELECT 'superadmin', 'Full system-wide access (super admin)'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'superadmin');

-- 2. Assign ALL permissions to the superadmin role (idempotent).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'superadmin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- 3. Seed the superadmin user (password: SuperAdmin123!)
--    Uses the role_id from the superadmin role we just guaranteed exists.
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
