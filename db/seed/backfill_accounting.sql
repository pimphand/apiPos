-- Backfill default COA + opening balance for every existing store.
-- Idempotent: safe to re-run. Run with:
--   docker exec -i apipos-postgres-1 psql -U postgres -d pos -v ON_ERROR_STOP=1 < db/seed/backfill_accounting.sql

BEGIN;

-- 1. Default COA for every store (14 accounts per store).
INSERT INTO chart_of_accounts (store_id, code, name, account_type, normal_balance, is_cash_equiv, is_active)
SELECT s.id, a.code, a.name, a.account_type, a.normal_balance, a.is_cash_equiv, TRUE
FROM stores s
CROSS JOIN (VALUES
  ('1100', 'Cash', 'ASSET', 'DEBIT', TRUE),
  ('1110', 'Bank', 'ASSET', 'DEBIT', TRUE),
  ('1120', 'Input VAT (recoverable)', 'ASSET', 'DEBIT', FALSE),
  ('1130', 'Accounts Receivable', 'ASSET', 'DEBIT', FALSE),
  ('1200', 'Inventory', 'ASSET', 'DEBIT', FALSE),
  ('2100', 'Accounts Payable', 'LIABILITY', 'CREDIT', FALSE),
  ('2110', 'Tax Payable (PPN)', 'LIABILITY', 'CREDIT', FALSE),
  ('3100', 'Owner Capital', 'EQUITY', 'CREDIT', FALSE),
  ('3110', 'Retained Earnings', 'EQUITY', 'CREDIT', FALSE),
  ('4100', 'Sales Revenue', 'REVENUE', 'CREDIT', FALSE),
  ('4200', 'Inventory Gain', 'REVENUE', 'CREDIT', FALSE),
  ('5100', 'Cost of Goods Sold', 'EXPENSE', 'DEBIT', FALSE),
  ('5200', 'Inventory Shrinkage', 'EXPENSE', 'DEBIT', FALSE),
  ('6100', 'Operating Expenses', 'EXPENSE', 'DEBIT', FALSE)
) AS a(code, name, account_type, normal_balance, is_cash_equiv)
ON CONFLICT (store_id, code) DO NOTHING;

-- 2. Accounting period for the current month per store (OPEN).
INSERT INTO accounting_periods (store_id, period_name, start_date, end_date, status)
SELECT s.id, to_char(CURRENT_DATE, 'YYYY-MM'),
       date_trunc('month', CURRENT_DATE)::date,
       (date_trunc('month', CURRENT_DATE) + INTERVAL '1 month - 1 day')::date,
       'OPEN'
FROM stores s
ON CONFLICT (store_id, period_name) DO NOTHING;

-- 3. Opening balance: Dr Inventory / Cr Owner Capital for existing stock.
--    One OPENING journal per store (idempotent by key).
DO $$
DECLARE
  s RECORD;
  inv_value NUMERIC(15,2);
  v_period uuid;
  v_entry uuid;
  v_entry_number text;
  v_inv_id uuid; v_cap_id uuid;
BEGIN
  FOR s IN SELECT id FROM stores WHERE is_verified = TRUE LOOP
    SELECT COALESCE(SUM(stock::numeric * cost_price::numeric), 0) INTO inv_value
    FROM products WHERE store_id = s.id;
    IF inv_value = 0 OR inv_value IS NULL THEN
      CONTINUE;
    END IF;
    SELECT id INTO v_period FROM accounting_periods WHERE store_id = s.id AND status = 'OPEN' ORDER BY period_name DESC LIMIT 1;
    IF v_period IS NULL THEN
      CONTINUE;
    END IF;
    SELECT id INTO v_inv_id FROM chart_of_accounts WHERE store_id = s.id AND code = '1200';
    SELECT id INTO v_cap_id FROM chart_of_accounts WHERE store_id = s.id AND code = '3100';
    IF v_inv_id IS NULL OR v_cap_id IS NULL THEN
      CONTINUE;
    END IF;
    IF EXISTS (SELECT 1 FROM journal_entries WHERE store_id = s.id AND idempotency_key = 'OPENING:' || s.id::text) THEN
      CONTINUE;
    END IF;
    v_entry_number := 'JE-' || to_char(CURRENT_DATE, 'YYYY') || '-' || lpad((SELECT COUNT(*) + 1 FROM journal_entries WHERE store_id = s.id)::text, 6, '0');
    INSERT INTO journal_entries (store_id, period_id, entry_number, entry_date, entry_type, source_type, idempotency_key, description, status, posted_at)
    VALUES (s.id, v_period, v_entry_number, CURRENT_DATE, 'OPENING', 'MANUAL', 'OPENING:' || s.id::text, 'Opening balance', 'POSTED', NOW())
    RETURNING id INTO v_entry;
    INSERT INTO journal_lines (store_id, entry_id, account_id, debit, credit, memo) VALUES
      (s.id, v_entry, v_inv_id, inv_value, 0, 'opening inventory'),
      (s.id, v_entry, v_cap_id, 0, inv_value, 'opening capital');
  END LOOP;
END $$;

COMMIT;

SELECT 'chart_of_accounts' AS tbl, COUNT(*) FROM chart_of_accounts
UNION ALL SELECT 'journal_entries', COUNT(*) FROM journal_entries
UNION ALL SELECT 'accounting_periods', COUNT(*) FROM accounting_periods;
