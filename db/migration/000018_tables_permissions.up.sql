-- 000018_tables_permissions.up.sql
-- Self ordering: per-store dine-in table management permissions
-- (docs/pos-self-order.md §7). Seeded to admin & manager roles.

INSERT INTO permissions (name, resource, action, description) VALUES
    ('tables.create', 'tables', 'create', 'Create dine-in tables'),
    ('tables.read',   'tables', 'read',   'View dine-in tables'),
    ('tables.update', 'tables', 'update', 'Update dine-in tables'),
    ('tables.delete', 'tables', 'delete', 'Delete dine-in tables')
ON CONFLICT (name) DO NOTHING;

-- Assign to admin and manager.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
  AND p.name IN ('tables.create', 'tables.read', 'tables.update', 'tables.delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;