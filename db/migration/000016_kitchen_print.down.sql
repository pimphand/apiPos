-- 000016_kitchen_print.down.sql
ALTER TABLE store_settings DROP COLUMN IF EXISTS show_kitchen_print;
ALTER TABLE sales DROP COLUMN IF EXISTS notes;
