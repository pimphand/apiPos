-- 000019_accounting.up.sql
-- Phase 0 + Phase 1: money shadow columns and double-entry accounting engine.
-- No POS logic changes. accounting reads only the decimal (rounded) values.

-- ============================================================
-- Phase 0: money shadow columns (audit only)
-- ============================================================
ALTER TABLE sales ADD COLUMN IF NOT EXISTS subtotal_decimal NUMERIC(15,2) NOT NULL DEFAULT 0;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS discount_decimal NUMERIC(15,2) NOT NULL DEFAULT 0;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS tax_decimal NUMERIC(15,2) NOT NULL DEFAULT 0;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS grand_total_decimal NUMERIC(15,2) NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS amount_decimal NUMERIC(15,2) NOT NULL DEFAULT 0;

-- ============================================================
-- Phase 1: accounting periods
-- ============================================================
CREATE TABLE accounting_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    period_name VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','CLOSED')),
    closed_at TIMESTAMPTZ,
    closed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, period_name)
);
CREATE INDEX idx_periods_store_dates ON accounting_periods(store_id, start_date, end_date);

-- ============================================================
-- Phase 1: chart of accounts (per-store)
-- ============================================================
CREATE TABLE chart_of_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(150) NOT NULL,
    account_type VARCHAR(15) NOT NULL CHECK (account_type IN ('ASSET','LIABILITY','EQUITY','REVENUE','EXPENSE')),
    normal_balance VARCHAR(10) NOT NULL CHECK (normal_balance IN ('DEBIT','CREDIT')),
    parent_id UUID REFERENCES chart_of_accounts(id),
    is_cash_equiv BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, code)
);
CREATE INDEX idx_coa_store_type ON chart_of_accounts(store_id, account_type);

-- ============================================================
-- Phase 1: journal entries + lines (double-entry)
-- ============================================================
CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES accounting_periods(id),
    entry_number VARCHAR(30) NOT NULL,
    entry_date DATE NOT NULL,
    entry_type VARCHAR(20) NOT NULL CHECK (entry_type IN ('OPENING','SALE','PAYMENT','PURCHASE','ADJUSTMENT','REVERSAL','MANUAL')),
    source_type VARCHAR(30),
    source_id UUID,
    idempotency_key VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(10) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','POSTED','REVERSED')),
    reversal_of_id UUID REFERENCES journal_entries(id),
    reversed_by_id UUID REFERENCES journal_entries(id),
    created_by UUID REFERENCES users(id),
    posted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, idempotency_key)
);
CREATE INDEX idx_je_store_date ON journal_entries(store_id, entry_date);
CREATE INDEX idx_je_source ON journal_entries(store_id, source_type, source_id);
CREATE INDEX idx_je_period ON journal_entries(period_id, status);

CREATE TABLE journal_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES chart_of_accounts(id),
    debit NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (credit >= 0),
    memo VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((debit > 0 AND credit = 0) OR (debit = 0 AND credit > 0))
);
CREATE INDEX idx_jl_account_date ON journal_lines(store_id, account_id, entry_id);
CREATE INDEX idx_jl_entry ON journal_lines(entry_id);

-- ============================================================
-- Phase 1: general ledger view (POSTED only)
-- ============================================================
CREATE OR REPLACE VIEW general_ledger AS
SELECT jl.store_id,
       jl.account_id,
       je.entry_date,
       je.entry_number,
       je.source_type,
       je.source_id,
       jl.debit,
       jl.credit,
       jl.memo,
       je.id AS entry_id
FROM journal_lines jl
JOIN journal_entries je ON je.id = jl.entry_id
WHERE je.status = 'POSTED';

-- ============================================================
-- Phase 1: guards — balanced journals + closed-period immutability
-- ============================================================
CREATE OR REPLACE FUNCTION validate_journal_balanced() RETURNS trigger AS $$
DECLARE
    d NUMERIC;
    c NUMERIC;
BEGIN
    SELECT COALESCE(SUM(debit),0), COALESCE(SUM(credit),0)
      INTO d, c FROM journal_lines WHERE entry_id = NEW.id;
    IF NEW.status = 'POSTED' AND d <> c THEN
        RAISE EXCEPTION 'journal % not balanced: debit % != credit %', NEW.entry_number, d, c
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_je_balanced BEFORE UPDATE OF status ON journal_entries
    FOR EACH ROW EXECUTE FUNCTION validate_journal_balanced();

CREATE OR REPLACE FUNCTION prevent_closed_period_write() RETURNS trigger AS $$
BEGIN
  IF TG_TABLE_NAME = 'journal_entries' THEN
    IF EXISTS (SELECT 1 FROM accounting_periods p WHERE p.id = NEW.period_id AND p.status = 'CLOSED') THEN
      RAISE EXCEPTION 'period is closed' USING ERRCODE = '23001';
    END IF;
    RETURN NEW;
  ELSE
    IF EXISTS (
      SELECT 1 FROM accounting_periods p
      JOIN journal_entries je ON je.period_id = p.id
      WHERE je.id = NEW.entry_id AND p.status = 'CLOSED'
    ) THEN
      RAISE EXCEPTION 'period is closed' USING ERRCODE = '23001';
    END IF;
    RETURN NEW;
  END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_je_period_guard BEFORE INSERT OR UPDATE OR DELETE ON journal_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_closed_period_write();

CREATE TRIGGER trg_jl_period_guard BEFORE INSERT OR UPDATE OR DELETE ON journal_lines
    FOR EACH ROW EXECUTE FUNCTION prevent_closed_period_write();

-- ============================================================
-- Phase 2: accounting permissions
-- ============================================================
INSERT INTO permissions (name, resource, action, description) VALUES
    ('accounting.read',  'accounting', 'read',  'View accounting reports'),
    ('accounting.post',  'accounting', 'post',  'Create and post journals'),
    ('accounting.close', 'accounting', 'close', 'Close accounting periods')
ON CONFLICT (name) DO NOTHING;

-- superadmin + admin get all; manager gets read only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('superadmin','admin')
  AND p.name IN ('accounting.read','accounting.post','accounting.close')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name = 'accounting.read'
ON CONFLICT (role_id, permission_id) DO NOTHING;
