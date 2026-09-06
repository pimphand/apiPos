-- 000014_receipt_footer_position.up.sql
-- Let the store choose where the free footer text prints on the nota:
-- top (after header) / middle (after totals, before payment) / bottom (end).

ALTER TABLE store_settings
    ADD COLUMN receipt_footer_position VARCHAR(10) NOT NULL DEFAULT 'bottom'
    CONSTRAINT chk_store_settings_footer_position
        CHECK (receipt_footer_position IN ('top', 'middle', 'bottom'));
