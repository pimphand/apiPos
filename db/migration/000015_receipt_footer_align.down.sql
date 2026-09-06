-- 000015_receipt_footer_align.down.sql
ALTER TABLE store_settings
    DROP CONSTRAINT IF EXISTS chk_store_settings_footer_align,
    DROP COLUMN IF EXISTS receipt_footer_align,
    ADD COLUMN receipt_footer_position VARCHAR(10) NOT NULL DEFAULT 'bottom'
    CONSTRAINT chk_store_settings_footer_position
        CHECK (receipt_footer_position IN ('top', 'middle', 'bottom'));
