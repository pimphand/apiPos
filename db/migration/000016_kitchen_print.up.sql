-- 000016_kitchen_print.up.sql
-- Kitchen receipt feature: toggle in settings + order notes on sales.

-- Store settings: kitchen receipt button toggle
ALTER TABLE store_settings
  ADD COLUMN show_kitchen_print BOOLEAN NOT NULL DEFAULT TRUE;

-- Sales: optional order notes (kitchen instructions, etc.)
ALTER TABLE sales
  ADD COLUMN notes TEXT NULL DEFAULT NULL;
