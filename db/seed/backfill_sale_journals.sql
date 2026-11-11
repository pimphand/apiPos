-- Backfill SALE journals for pre-accounting PAID sales (idempotent).
-- For every PAID sale without a SALE journal, post one entry. Run once
-- after deploy; idempotent via the SALE:{saleID}:SALE key.
-- Amounts use the legacy float64 columns (rounded to 2dp in Go); the
-- shadow decimal columns are populated best-effort from the same values.

BEGIN;

DO $$
DECLARE
  s RECORD;
  it RECORD;
  v_period uuid;
  v_entry uuid;
  v_entry_number text;
  v_rev NUMERIC(15,2);
  v_tax NUMERIC(15,2);
  v_cogs NUMERIC(15,2);
  v_cash_acct uuid;
  v_rev_acct uuid;
  v_tax_acct uuid;
  v_cogs_acct uuid;
  v_inv_acct uuid;
BEGIN
  FOR s IN
    SELECT sale.id, sale.store_id, sale.invoice_number, sale.subtotal, sale.discount, sale.tax, sale.grand_total, sale.created_at
    FROM sales sale
    WHERE sale.status = 'PAID'
      AND NOT EXISTS (
        SELECT 1 FROM journal_entries je
        WHERE je.store_id = sale.store_id AND je.source_type = 'SALE' AND je.source_id = sale.id AND je.status = 'POSTED'
      )
    ORDER BY sale.created_at
  LOOP
    SELECT id INTO v_period FROM accounting_periods WHERE store_id = s.store_id AND s.created_at::date BETWEEN start_date AND end_date LIMIT 1;
    IF v_period IS NULL THEN
      SELECT id INTO v_period FROM accounting_periods WHERE store_id = s.store_id AND status = 'OPEN' ORDER BY period_name DESC LIMIT 1;
    END IF;
    IF v_period IS NULL THEN CONTINUE; END IF;

    SELECT id INTO v_cash_acct FROM chart_of_accounts WHERE store_id = s.store_id AND code = '1100';
    SELECT id INTO v_rev_acct  FROM chart_of_accounts WHERE store_id = s.store_id AND code = '4100';
    SELECT id INTO v_tax_acct  FROM chart_of_accounts WHERE store_id = s.store_id AND code = '2110';
    SELECT id INTO v_cogs_acct FROM chart_of_accounts WHERE store_id = s.store_id AND code = '5100';
    SELECT id INTO v_inv_acct  FROM chart_of_accounts WHERE store_id = s.store_id AND code = '1200';
    IF v_cash_acct IS NULL OR v_rev_acct IS NULL THEN CONTINUE; END IF;

    v_rev := ROUND(s.subtotal - s.discount, 2);
    v_tax := ROUND(s.tax, 2);
    v_cogs := 0;
    FOR it IN SELECT si.product_id, si.quantity FROM sale_items si WHERE si.sale_id = s.id LOOP
      DECLARE pc NUMERIC(15,2);
      BEGIN
        SELECT cost_price INTO pc FROM products WHERE id = it.product_id;
        v_cogs := v_cogs + ROUND(COALESCE(pc, 0) * it.quantity, 2);
      END;
    END LOOP;

    v_entry_number := 'JE-' || to_char(s.created_at, 'YYYY') || '-' || lpad((SELECT COUNT(*) + 1 FROM journal_entries WHERE store_id = s.store_id)::text, 6, '0');
    INSERT INTO journal_entries (store_id, period_id, entry_number, entry_date, entry_type, source_type, source_id, idempotency_key, description, status, posted_at)
    VALUES (s.store_id, v_period, v_entry_number, s.created_at::date, 'SALE', 'SALE', s.id, 'SALE:' || s.id::text || ':SALE', 'Backfill sale ' || s.invoice_number, 'POSTED', s.created_at)
    RETURNING id INTO v_entry;

    -- Dr Cash (simplified: full grand_total to Cash/1100; split-method handling
    -- requires payment rows, which exist but per-method routing is Go-side.
    -- This SQL backfill posts Cash for simplicity; future sales post correctly.)
    INSERT INTO journal_lines (store_id, entry_id, account_id, debit, credit, memo) VALUES
      (s.store_id, v_entry, v_cash_acct, ROUND(s.grand_total, 2), 0, 'backfill cash'),
      (s.store_id, v_entry, v_rev_acct, 0, v_rev, 'backfill revenue');
    IF v_tax <> 0 AND v_tax_acct IS NOT NULL THEN
      INSERT INTO journal_lines (store_id, entry_id, account_id, debit, credit, memo) VALUES
        (s.store_id, v_entry, v_tax_acct, 0, v_tax, 'backfill tax');
    END IF;
    IF v_cogs <> 0 AND v_cogs_acct IS NOT NULL AND v_inv_acct IS NOT NULL THEN
      INSERT INTO journal_lines (store_id, entry_id, account_id, debit, credit, memo) VALUES
        (s.store_id, v_entry, v_cogs_acct, v_cogs, 0, 'backfill COGS'),
        (s.store_id, v_entry, v_inv_acct, 0, v_cogs, 'backfill inventory out');
    END IF;
  END LOOP;
END $$;

COMMIT;

SELECT 'sale_journals' AS tbl, COUNT(*) FROM journal_entries WHERE entry_type = 'SALE'
UNION ALL SELECT 'paid_sales', COUNT(*) FROM sales WHERE status = 'PAID';
