-- 000014_receipt_footer_position.down.sql
ALTER TABLE store_settings
    DROP CONSTRAINT IF EXISTS chk_store_settings_footer_position,
    DROP COLUMN IF EXISTS receipt_footer_position;
