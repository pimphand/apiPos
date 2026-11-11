-- 000019_accounting.down.sql
-- Roll back accounting engine (Phase 0 + Phase 1). Shadow money columns are kept
-- (they are audit-only and harmless); tables/view/triggers are dropped.

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('accounting.read','accounting.post','accounting.close')
);

DELETE FROM permissions
WHERE name IN ('accounting.read','accounting.post','accounting.close');

DROP VIEW IF EXISTS general_ledger;

DROP TRIGGER IF EXISTS trg_jl_period_guard ON journal_lines;
DROP TRIGGER IF EXISTS trg_je_period_guard ON journal_entries;
DROP TRIGGER IF EXISTS trg_je_balanced ON journal_entries;

DROP FUNCTION IF EXISTS prevent_closed_period_write();
DROP FUNCTION IF EXISTS validate_journal_balanced();

DROP TABLE IF EXISTS journal_lines;
DROP TABLE IF EXISTS journal_entries;
DROP TABLE IF EXISTS chart_of_accounts;
DROP TABLE IF EXISTS accounting_periods;

-- Shadow columns (optional drop — kept for audit; drop only on full rollback):
-- ALTER TABLE sales DROP COLUMN IF EXISTS grand_total_decimal;
-- ALTER TABLE sales DROP COLUMN IF EXISTS tax_decimal;
-- ALTER TABLE sales DROP COLUMN IF EXISTS discount_decimal;
-- ALTER TABLE sales DROP COLUMN IF EXISTS subtotal_decimal;
-- ALTER TABLE payments DROP COLUMN IF EXISTS amount_decimal;
