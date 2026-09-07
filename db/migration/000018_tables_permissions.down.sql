-- 000018_tables_permissions.down.sql
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('tables.create', 'tables.read', 'tables.update', 'tables.delete')
);

DELETE FROM permissions
WHERE name IN ('tables.create', 'tables.read', 'tables.update', 'tables.delete');