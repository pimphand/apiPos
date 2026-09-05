-- 000010_backfill_activity_log_store_id.up.sql
-- Login/register logs were written before an authenticated context existed, so
-- their store_id is NULL and the tenant scope (store_id = X OR store_id IS
-- NULL) leaked them to every store. Attribute them to the acting user's store
-- so each tenant only sees its own logs.
UPDATE activity_logs l
SET store_id = u.store_id
FROM users u
WHERE l.user_id = u.id
  AND l.store_id IS NULL;