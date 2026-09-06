-- 000015_receipt_footer_align.up.sql
-- Replace the vertical footer position (top/middle/bottom, 000014) with the
-- footer text alignment (left/center/right) — the actual requirement.

ALTER TABLE store_settings
    DROP CONSTRAINT IF EXISTS chk_store_settings_footer_position,
    DROP COLUMN IF EXISTS receipt_footer_position,
    ADD COLUMN receipt_footer_align VARCHAR(10) NOT NULL DEFAULT 'center'
    CONSTRAINT chk_store_settings_footer_align
        CHECK (receipt_footer_align IN ('left', 'center', 'right'));
