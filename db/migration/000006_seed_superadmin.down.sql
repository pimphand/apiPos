-- Remove the default superadmin seeded by the up migration.
DELETE FROM users WHERE username = 'superadmin' AND email = 'superadmin@example.com' AND full_name = 'Super Admin';
