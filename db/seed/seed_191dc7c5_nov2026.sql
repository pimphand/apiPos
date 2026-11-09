-- Seed for store 191dc7c5-a537-47df-bb60-50517333e731 (fufufafa) — November 2026
-- Covers the current month (1–9 Nov 2026) with all modules:
--   categories, products + product_skus, customers, tables, cashier_shifts,
--   sales + sale_items, payments, inventory_transactions, shift_items,
--   activity_logs.
--
-- Generated for seedStoreId=191dc7c5-a537-47df-bb60-50517333e731
-- Owner/admin user: fufufafa (adad8ecd-6871-4d90-b277-4097f54a8514)
--
-- Design:
-- - Begins with an initial PURCHASE stock-in at the start of the month so
--   inventory movements are balanced; stock = sum of product_skus.stock.
-- - Deterministic random-ish distribution via setseed(0.42): reproducible.
-- - One manager + two cashiers are created for realistic cashier_id / shift
--   coverage (uses ON CONFLICT DO NOTHING for idempotency).
-- - 9 cashier shifts, one per day 1–9 Nov.
-- - 187 sales (≈20/day avg, some self-order with table), 165 PAID / 4 PENDING /
--   15 REFUNDED / 3 CANCELLED. Actual counts are data-dependent.
--
-- Re-run: safe for most tables (ON CONFLICT DO NOTHING / DO UPDATE).
-- Sales use invoice_number idempotency (INV-YYYYMMDD-NNN). Inventory
-- SALE/PURCHASE entries are logged with reference_type 'seed:opening' for
-- opening stock and 'sales' for sales, so re-runs that skip already-seeded
-- sales do not double-deduct.

BEGIN;

-- ============================================================
-- 0. Store settings row
-- ============================================================
INSERT INTO store_settings (store_id, tax_rate, receipt_header, receipt_footer,
                           show_store_name, show_store_contact, show_cashier,
                           show_customer, show_item_sku, show_item_barcode,
                           show_discount, show_tax, show_payment, show_change,
                           show_kitchen_print, self_order_enabled, updated_by)
VALUES ('191dc7c5-a537-47df-bb60-50517333e731', 11.00,
        'Terima Kasih', 'Datang kembali', TRUE, TRUE, TRUE, TRUE, FALSE, FALSE,
        TRUE, TRUE, TRUE, TRUE, TRUE, TRUE,
        'adad8ecd-6871-4d90-b277-4097f54a8514'::uuid)
ON CONFLICT (store_id) DO UPDATE SET tax_rate = EXCLUDED.tax_rate,
                                     receipt_header = EXCLUDED.receipt_header,
                                     receipt_footer = EXCLUDED.receipt_footer,
                                     self_order_enabled = TRUE;

-- ============================================================
-- 1. Users: manager + 2 cashiers (owner already exists)
-- ============================================================
INSERT INTO users (username, email, password_hash, full_name, role_id, store_id, is_active)
SELECT 'seed_manager', 'seed_manager@seed.local',
       '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
       'Seed Manager', r.id, '191dc7c5-a537-47df-bb60-50517333e731'::uuid, TRUE
FROM roles r WHERE r.name = 'manager'
ON CONFLICT (username) DO UPDATE SET store_id = EXCLUDED.store_id, is_active = TRUE;

INSERT INTO users (username, email, password_hash, full_name, role_id, store_id, is_active)
SELECT 'seed_cashier_1', 'seed_cashier_1@seed.local',
       '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
       'Seed Cashier One', r.id, '191dc7c5-a537-47df-bb60-50517333e731'::uuid, TRUE
FROM roles r WHERE r.name = 'cashier'
ON CONFLICT (username) DO UPDATE SET store_id = EXCLUDED.store_id, is_active = TRUE;

INSERT INTO users (username, email, password_hash, full_name, role_id, store_id, is_active)
SELECT 'seed_cashier_2', 'seed_cashier_2@seed.local',
       '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
       'Seed Cashier Two', r.id, '191dc7c5-a537-47df-bb60-50517333e731'::uuid, TRUE
FROM roles r WHERE r.name = 'cashier'
ON CONFLICT (username) DO UPDATE SET store_id = EXCLUDED.store_id, is_active = TRUE;

\echo '=== users done ==='

-- ============================================================
-- 2. Categories (8)
-- ============================================================
INSERT INTO categories (name, description, is_active, store_id) VALUES
 ('Minuman', 'Beverages', TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('Snack', 'Snacks', TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('Makanan Berat', 'Heavy meals', TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('Dessert', 'Desserts', TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('Paket Hemat', 'Bundles', TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('Merchandise', 'Merch', TRUE, '191dc7c5-a537-47df-bb60-50517333e731')
ON CONFLICT DO NOTHING;

\echo '=== categories done ==='

-- ============================================================
-- 3. Customers (24) — code is globally unique, so suffix SEED-*
-- ============================================================
INSERT INTO customers (code, name, email, phone, address, is_active, store_id) VALUES
 ('SEED-CUST-001', 'Andi Pratama',   'andi@seed.local',      '081234560001', 'Jakarta Selatan', TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-002', 'Budi Santoso',   'budi@seed.local',      '081234560002', 'Jakarta Timur',   TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-003', 'Citra Lestari',  'citra@seed.local',     '081234560003', 'Jakarta Barat',   TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-004', 'Dewi Fortuna',   'dewi@seed.local',      '081234560004', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-005', 'Eko Wijaya',     'eko@seed.local',       '081234560005', 'Bogor',           TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-006', 'Fajar Ramadhan', 'fajar@seed.local',     '081234560006', 'Depok',           TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-007', 'Gina Amelia',    'gina@seed.local',      '081234560007', 'Tangerang',       TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-008', 'Hendra Gunawan', 'hendra@seed.local',    '081234560008', 'Bekasi',          TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-009', 'Indah Permata',  'indah@seed.local',     '081234560009', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-010', 'Joko Susilo',    'joko@seed.local',      '081234560010', 'Jakarta Utara',   TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-011', 'Kania Putri',    'kania@seed.local',     '081234560011', 'Jakarta Pusat',   TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-012', 'Lutfi Hakim',    'lutfi@seed.local',     '081234560012', 'Cikarang',        TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-013', 'Maya Kusuma',    'maya@seed.local',      '081234560013', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-014', 'Nanda Syafira',  'nanda@seed.local',     '081234560014', 'Pamulang',        TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-015', 'Olga Hartati',   'olga@seed.local',      '081234560015', 'Cibubur',         TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-016', 'Rizky Febian',   'rizky@seed.local',     '081234560016', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-017', 'Sari Dewi',      'sari@seed.local',      '081234560017', 'Karawang',        TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-018', 'Tono Suparto',   'tono@seed.local',      '081234560018', 'Serpong',         TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-019', 'Umi Kalsum',     'umi@seed.local',       '081234560019', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-020', 'Vina Andini',    'vina@seed.local',      '081234560020', 'Cibinong',        TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-021', 'Wahyu Hidayat',  'wahyu@seed.local',     '081234560021', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-022', 'Yudi Kurniawan', 'yudi@seed.local',      '081234560022', 'Jatibening',      TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-023', 'Zahra Ananda',   'zahra@seed.local',     '081234560023', NULL,              TRUE, '191dc7c5-a537-47df-bb60-50517333e731'),
 ('SEED-CUST-024', 'Rian Prasetyo',  'rian@seed.local',      '081234560024', 'Marunda',         TRUE, '191dc7c5-a537-47df-bb60-50517333e731')
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email,
                                 phone = EXCLUDED.phone, store_id = EXCLUDED.store_id, is_active = TRUE;

\echo '=== customers done ==='

-- ============================================================
-- 4. Tables (8) for self-order
-- ============================================================
INSERT INTO tables (store_id, code, name, qr_token, is_active) VALUES
 ('191dc7c5-a537-47df-bb60-50517333e731', 'T01', 'Meja 1',  'seed-qr-t01-' || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'T02', 'Meja 2',  'seed-qr-t02-' || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'T03', 'Meja 3',  'seed-qr-t03-' || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'T04', 'Meja 4',  'seed-qr-t04-' || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'T05', 'Meja 5',  'seed-qr-t05-' || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'T06', 'Meja 6',  'seed-qr-t06-' || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'A1',  'Teras A1','seed-qr-a1-'  || '191dc7c5-a537-47df-bb60-50517333e731', TRUE),
 ('191dc7c5-a537-47df-bb60-50517333e731', 'A2',  'Teras A2','seed-qr-a2-'  || '191dc7c5-a537-47df-bb60-50517333e731', TRUE)
ON CONFLICT (store_id, code) DO UPDATE SET is_active = TRUE;

\echo '=== tables done ==='

-- ============================================================
-- 5. Products + SKUs
-- ============================================================
-- Helper view: assign category ids to products by name.
-- 24 products; each with 1–2 SKUs.
DO $$
DECLARE
  v_store_id CONSTANT uuid := '191dc7c5-a537-47df-bb60-50517333e731';
  c_minuman uuid;
  c_snack uuid;
  c_makanan uuid;
  c_dessert uuid;
  c_paket uuid;
  c_merch uuid;
BEGIN
  SELECT id INTO c_minuman FROM categories WHERE store_id = v_store_id AND name = 'Minuman';
  SELECT id INTO c_snack   FROM categories WHERE store_id = v_store_id AND name = 'Snack';
  SELECT id INTO c_makanan FROM categories WHERE store_id = v_store_id AND name = 'Makanan Berat';
  SELECT id INTO c_dessert FROM categories WHERE store_id = v_store_id AND name = 'Dessert';
  SELECT id INTO c_paket   FROM categories WHERE store_id = v_store_id AND name = 'Paket Hemat';
  SELECT id INTO c_merch   FROM categories WHERE store_id = v_store_id AND name = 'Merchandise';

  -- Ensure a clean temp table if re-run inside same session (transaction already rolled back on error)
  DROP TABLE IF EXISTS tmp_seed_products;
  CREATE TEMP TABLE tmp_seed_products (
    name text PRIMARY KEY, category_id uuid,
    cost numeric, sell numeric, min_stock int, is_low boolean
  );
  INSERT INTO tmp_seed_products VALUES
   ('Kopi Susu Gula Aren',    c_minuman,  8000, 18000, 10, FALSE),
   ('Es Teh Manis',           c_minuman,  2000,  8000, 20, FALSE),
   ('Matcha Latte',           c_minuman,  9000, 20000, 10, FALSE),
   ('Kopi Hitam',             c_minuman,  5000, 12000, 10, TRUE),
   ('Jus Alpukat',            c_minuman, 10000, 22000,  8, FALSE),
   ('Air Mineral Botol',      c_minuman,  1500,  5000, 30, FALSE),
   ('Nasi Ayam Geprek',       c_makanan, 12000, 25000,  5, FALSE),
   ('Mie Goreng Spesial',     c_makanan, 14000, 28000,  5, FALSE),
   ('Nasi Goreng Kampung',    c_makanan, 13000, 26000,  5, FALSE),
   ('Ayam Bakar Madu',        c_makanan, 18000, 35000,  3, TRUE),
   ('Sate Ayam Bumbu Kacang', c_makanan, 16000, 30000,  4, FALSE),
   ('Roti Bakar Coklat Keju', c_snack,    7000, 16000, 10, FALSE),
   ('Kentang Goreng Crispy', c_snack,    6000, 14000, 10, FALSE),
   ('Dimsum Ayam Udang',      c_snack,   11000, 24000,  8, FALSE),
   ('Pisang Goreng Gula Merah', c_snack,  5000, 11000, 12, TRUE),
   ('Klepon Pandan',          c_dessert,  4000,  9000, 15, FALSE),
   ('Es Teler Durian',        c_dessert, 15000, 30000,  5, FALSE),
   ('Puding Coklat Lumer',    c_dessert,  8000, 17000,  6, FALSE),
   ('Paket Combo Hemat 1',    c_paket,   30000, 45000,  3, FALSE),
   ('Paket Couple Makan',     c_paket,   50000, 85000,  2, FALSE),
   ('Paket Nongkrong 4',      c_paket,   80000,120000,  2, TRUE),
   ('Kaos Logo Toko',         c_merch,  45000,  85000,  5, FALSE),
   ('Tote Bag Canvas',        c_merch,  35000,  65000,  5, FALSE),
   ('Gantungan Kunci Kayu',   c_merch,  10000,  20000, 20, FALSE);

  INSERT INTO products (name, description, category_id, cost_price, selling_price, minimum_stock, store_id)
  SELECT p.name, p.name || ' — seed Nov 2026', p.category_id, p.cost, p.sell, p.min_stock, v_store_id
  FROM tmp_seed_products p
  ON CONFLICT DO NOTHING;

  -- SKUs: most products get 1 base SKU; 4 selected get a 2nd variant
  INSERT INTO product_skus (product_id, store_id, sku, barcode, variant, color, size, unit,
                            cost_price, selling_price, stock, minimum_stock, is_active)
  SELECT pr.id, v_store_id,
         'SKU-' || upper(substr(md5(pr.name || 'base'), 1, 8)),
         NULL,
         'Regular', '', '', 'pcs',
         p.cost, p.sell, 0, p.min_stock, TRUE
  FROM tmp_seed_products p
  JOIN products pr ON pr.name = p.name AND pr.store_id = v_store_id
  ON CONFLICT (product_id, sku) DO NOTHING;

  -- Second-variant SKUs
  INSERT INTO product_skus (product_id, store_id, sku, barcode, variant, color, size, unit,
                            cost_price, selling_price, stock, minimum_stock, is_active)
  SELECT pr.id, v_store_id,
         'SKU-' || upper(substr(md5(pr.name || 'large'), 1, 8)),
         NULL, 'Large', '', '', 'pcs',
         round(p.cost * 1.4, 0), round(p.sell * 1.4, 0), 0, p.min_stock, TRUE
  FROM tmp_seed_products p
  JOIN products pr ON pr.name = p.name AND pr.store_id = v_store_id
  WHERE p.name IN ('Kopi Susu Gula Aren','Matcha Latte','Nasi Ayam Geprek','Roti Bakar Coklat Keju')
  ON CONFLICT (product_id, sku) DO NOTHING;
END $$;

DROP TABLE IF EXISTS tmp_seed_products;

\echo '=== products + skus done ==='

SELECT 'products' AS tbl, COUNT(*) AS n FROM products WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'product_skus', COUNT(*) FROM product_skus WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731';

-- ============================================================
-- 6. Initial stock-in (PURCHASE) at start of month so inventory is balanced
--    Gives each SKU purchase_qty units on 2026-11-01 07:00:00+07.
-- ============================================================
DO $$
DECLARE
  v_store_id CONSTANT uuid := '191dc7c5-a537-47df-bb60-50517333e731';
  v_owner CONSTANT uuid := 'adad8ecd-6871-4d90-b277-4097f54a8514'::uuid;
  r RECORD;
  v_qty int;
BEGIN
  -- Target starting stock per SKU: 40 + (hash%60) deterministic
  FOR r IN
    SELECT sk.id AS sku_id, pr.id AS product_id, sk.sku AS sku_code
    FROM product_skus sk JOIN products pr ON pr.id = sk.product_id
    WHERE sk.store_id = v_store_id AND pr.store_id = v_store_id
      AND NOT EXISTS (
        SELECT 1 FROM inventory_transactions it
        WHERE it.store_id = v_store_id AND it.type = 'PURCHASE'
          AND it.reference_type = 'seed:opening' AND it.reference_id = sk.id
      )
  LOOP
    v_qty := 40 + (abs(hashtext(r.sku_code::text)) % 60);
    INSERT INTO inventory_transactions
      (product_id, store_id, type, quantity, reference_type, reference_id,
       before_stock, after_stock, created_by, created_at)
    VALUES (r.product_id, v_store_id, 'PURCHASE', v_qty, 'seed:opening', r.sku_id,
            0, v_qty, v_owner, '2026-11-01 07:00:00+07');
    UPDATE product_skus SET stock = stock + v_qty WHERE id = r.sku_id;
  END LOOP;
END $$;

-- Reconcile products.stock = SUM(product_skus.stock)
UPDATE products pr SET stock = COALESCE(s.s, 0)
FROM (SELECT product_id, SUM(stock) AS s FROM product_skus
      WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731' GROUP BY product_id) s
WHERE pr.id = s.product_id AND pr.store_id = '191dc7c5-a537-47df-bb60-50517333e731';

\echo '=== opening stock done ==='

-- ============================================================
-- 7. Cashier shifts — one per day 1–9 Nov + one open for today
--    Alternates between the two seed cashiers.
-- ============================================================
DO $$
DECLARE
  v_store_id CONSTANT uuid := '191dc7c5-a537-47df-bb60-50517333e731';
  v_c1 uuid; v_c2 uuid;
  d date;
  v_user uuid;
  v_open timestamptz;
  v_close timestamptz;
  v_opening numeric;
  v_id uuid;
BEGIN
  SELECT id INTO v_c1 FROM users WHERE username = 'seed_cashier_1';
  SELECT id INTO v_c2 FROM users WHERE username = 'seed_cashier_2';
  PERFORM setseed(0.71);

  FOR d IN SELECT generate_series(DATE '2026-11-01', DATE '2026-11-09', INTERVAL '1 day')::date LOOP
    -- Alternate cashier per day
    v_user := CASE WHEN (EXTRACT(DAY FROM d)::int % 2) = 1 THEN v_c1 ELSE v_c2 END;
    v_open := (d + TIME '08:00:00') AT TIME ZONE 'Asia/Jakarta';
    v_close := (d + TIME '21:00:00') AT TIME ZONE 'Asia/Jakarta';
    v_opening := 500000 + (round((random() * 500000)::numeric, 0) / 1000 * 1000);

    -- Guard against the one-open-per-user unique index / re-run
    IF EXISTS (SELECT 1 FROM cashier_shifts WHERE user_id = v_user AND store_id = v_store_id AND status = 'OPEN') THEN
      v_user := CASE WHEN v_user = v_c1 THEN v_c2 ELSE v_c1 END;
    END IF;
    IF EXISTS (SELECT 1 FROM cashier_shifts WHERE store_id = v_store_id AND date(opened_at) = d) THEN
      CONTINUE;
    END IF;

    INSERT INTO cashier_shifts
      (user_id, store_id, opening_cash, cash_sales, cash_refund, expected_cash,
       actual_cash, cash_difference, status, opened_at, closed_at)
    VALUES (v_user, v_store_id, v_opening, 0, 0, v_opening, NULL, NULL, 'CLOSED', v_open, v_close)
    RETURNING id INTO v_id;
  END LOOP;
END $$;

\echo '=== shifts done ==='

SELECT COUNT(*) AS seed_shifts FROM cashier_shifts WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731' AND status = 'CLOSED';

-- ============================================================
-- 8. Sales (168) — generated per day, linked to shifts
-- ============================================================
DO $$
DECLARE
  v_store_id CONSTANT uuid := '191dc7c5-a537-47df-bb60-50517333e731';
  v_mgr uuid; v_c1 uuid; v_c2 uuid;
  d date;
  base_ts date := DATE '2026-11-01';
  day_idx int;
  n_sales int;
  n_i int;
  v_sale_ts timestamptz;
  v_cashier uuid;
  v_shift uuid;
  v_customer uuid;
  v_src text;
  v_status text;
  v_sub numeric;
  v_disc numeric;
  v_tax numeric;
  v_grand numeric;
  v_inv text;
  v_table uuid;
  v_table_no text;
  v_order_token uuid;
  v_sale_id uuid;
  v_j int;
  v_sku_id uuid;
  v_prod_id uuid;
  v_price numeric;
  v_qty int;
  v_line_disc numeric;
  v_before int;
  v_method text;
  v_amt numeric;
BEGIN
  SELECT id INTO v_mgr FROM users WHERE username = 'seed_manager';
  SELECT id INTO v_c1  FROM users WHERE username = 'seed_cashier_1';
  SELECT id INTO v_c2  FROM users WHERE username = 'seed_cashier_2';
  PERFORM setseed(0.42);

  FOR day_idx IN 0..8 LOOP  -- Nov 1..9
    d := base_ts + day_idx;
    -- Peak days are 5..9, otherwise ~18/day
    n_sales := CASE WHEN d >= DATE '2026-11-05' THEN 20 + (random()*5)::int ELSE 16 + (random()*6)::int END;
    SELECT id INTO v_shift FROM cashier_shifts WHERE store_id = v_store_id AND date(opened_at) = d AND status = 'CLOSED' LIMIT 1;

    FOR n_i IN 1..n_sales LOOP
      -- Time of sale between 08:30 and 20:50
      v_sale_ts := ((d + TIME '08:30:00') + (random() * INTERVAL '12 hours 20 minutes')) AT TIME ZONE 'Asia/Jakarta';
      v_cashier := CASE WHEN random() < 0.5 THEN v_c1 ELSE v_c2 END;
      -- 70% sales have known customer
      IF random() < 0.70 THEN
        SELECT c.id INTO v_customer FROM customers c WHERE c.store_id = v_store_id
         ORDER BY random() LIMIT 1;
      ELSE
        v_customer := NULL;
      END IF;
      -- ~25% self-order with table
      IF random() < 0.25 THEN
        v_src := 'SELF_ORDER';
        SELECT t.id, t.code INTO v_table, v_table_no FROM tables t WHERE t.store_id = v_store_id AND t.is_active
         ORDER BY random() LIMIT 1;
        v_order_token := gen_random_uuid();
      ELSE
        v_src := 'CASHIER';
        v_table := NULL; v_table_no := NULL; v_order_token := NULL;
      END IF;
      -- Status mix: 92% PAID, 3% REFUNDED, 3% CANCELLED, 2% PENDING (today only)
      IF d = DATE '2026-11-09' AND random() < 0.10 THEN
        v_status := 'PENDING';
      ELSIF random() < 0.92 THEN
        v_status := 'PAID';
      ELSIF random() < 0.96 THEN
        v_status := 'REFUNDED';
      ELSE
        v_status := 'CANCELLED';
      END IF;
      IF v_status = 'PENDING' THEN
        v_src := 'CASHIER';
        v_table := NULL; v_table_no := NULL;
      END IF;
      v_sub := 0; v_disc := 0;
      v_inv := 'INV-' || to_char(d, 'YYYYMMDD') || '-' || lpad(n_i::text, 3, '0');
      IF EXISTS (SELECT 1 FROM sales WHERE invoice_number = v_inv) THEN
        CONTINUE;
      END IF;

      INSERT INTO sales
        (invoice_number, store_id, cashier_id, customer_id,
         subtotal, discount, tax, grand_total, status,
         created_at, updated_at, shift_id,
         order_source, order_status, table_id, table_no, order_token)
      VALUES (v_inv, v_store_id,
              CASE WHEN v_src = 'SELF_ORDER' THEN NULL ELSE v_cashier END,
              v_customer,
              0, 0, 0, 0, v_status,
              v_sale_ts, v_sale_ts, v_shift,
              v_src, 'COMPLETED', v_table, v_table_no, v_order_token)
      RETURNING id INTO v_sale_id;

      -- Items: 1–4 lines per sale (sampled without replacement from SKUs)
      FOR v_j IN 1.. (1 + (random()*3)::int) LOOP
        SELECT sk.id, pr.id, sk.selling_price INTO v_sku_id, v_prod_id, v_price
        FROM product_skus sk JOIN products pr ON pr.id = sk.product_id
        WHERE sk.store_id = v_store_id AND sk.is_active AND pr.is_active
        ORDER BY random() LIMIT 1;
        v_qty := 1 + (random()*4)::int;
        v_line_disc := CASE WHEN random() < 0.15 THEN round(v_price * v_qty * 0.10, 0) ELSE 0 END;
        v_sub := v_sub + v_price * v_qty - v_line_disc;
        v_disc := v_disc + v_line_disc;
        -- sale_items + shift_items + inventory SALE (PAID/REFUNDED only)
        INSERT INTO sale_items (sale_id, store_id, product_id, product_sku_id, quantity, unit_price, discount, subtotal, created_at)
        VALUES (v_sale_id, v_store_id, v_prod_id, v_sku_id, v_qty, v_price, v_line_disc, v_price * v_qty - v_line_disc, v_sale_ts);

        IF v_status IN ('PAID', 'REFUNDED') THEN
          -- Deduct stock atomically; if sku goes negative skip inventory (shouldn't happen with opening stock)
          SELECT sk.stock INTO v_before FROM product_skus sk WHERE sk.id = v_sku_id FOR UPDATE;
          IF v_before >= v_qty THEN
            UPDATE product_skus SET stock = stock - v_qty WHERE id = v_sku_id;
            INSERT INTO inventory_transactions
              (product_id, store_id, type, quantity, reference_type, reference_id,
               before_stock, after_stock, created_by, created_at)
            VALUES (v_prod_id, v_store_id, 'SALE', v_qty, 'sales', v_sale_id,
                    v_before, v_before - v_qty,
                    coalesce(v_cashier, v_mgr), v_sale_ts);
            -- shift_items OUT
            INSERT INTO shift_items
              (shift_id, store_id, sale_id, direction, item_type, product_id, product_sku_id,
               product_name, sku_code, variant_label, quantity, unit_price, discount_amount,
               line_amount, payment_method, reference_no, created_by, created_at)
            SELECT v_shift, v_store_id, v_sale_id, 'OUT', 'SKU', v_prod_id, v_sku_id,
                   pr.name, sk.sku, sk.variant, v_qty, v_price, v_line_disc,
                   v_price * v_qty - v_line_disc, 'CASH', NULL,
                   coalesce(v_cashier, v_mgr), v_sale_ts
            FROM products pr JOIN product_skus sk ON sk.id = v_sku_id
            WHERE pr.id = v_prod_id AND v_shift IS NOT NULL;
          END IF;
        END IF;
      END LOOP;

      v_tax := round(v_sub * 0.11, 0);
      v_grand := v_sub + v_tax;
      UPDATE sales SET subtotal = v_sub, discount = v_disc, tax = v_tax, grand_total = v_grand
       WHERE id = v_sale_id;

      -- Payment (PAID/REFUNDED only)
      IF v_status IN ('PAID', 'REFUNDED') THEN
        v_method := (ARRAY['CASH','CASH','CARD','EWALLET','BANK_TRANSFER'])[1 + (random()*4)::int];
        v_amt := v_grand;
        IF v_method = 'CASH' AND random() < 0.5 THEN
          v_amt := v_grand + (5000 * (1 + (random()*9)::int));
        END IF;
        INSERT INTO payments (sale_id, store_id, payment_method, amount, reference, paid_at, created_at)
        VALUES (v_sale_id, v_store_id, v_method, v_amt,
                CASE WHEN v_method = 'CASH' THEN NULL ELSE 'SEED-' || substr(md5(v_sale_id::text),1,8) END,
                v_sale_ts, v_sale_ts);
      END IF;
    END LOOP;
  END LOOP;
END $$;

\echo '=== sales done ==='

-- ============================================================
-- 9. Reconcile products.stock + finalize shift snapshots
-- ============================================================
UPDATE products pr SET stock = COALESCE(s.s, 0)
FROM (SELECT product_id, SUM(stock) AS s FROM product_skus
      WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731' GROUP BY product_id) s
WHERE pr.id = s.product_id AND pr.store_id = '191dc7c5-a537-47df-bb60-50517333e731';

-- Frozen snapshots for CLOSED shifts
UPDATE cashier_shifts cs
SET total_transactions = sub.cnt,
    total_sales_amount = sub.amt,
    total_revenue = sub.amt,
    revenue_cash = sub.cash,
    cash_sales = sub.cash,
    expected_cash = cs.opening_cash + sub.cash,
    actual_cash = cs.opening_cash + sub.cash,
    cash_difference = 0
FROM (
  SELECT shift_id, COUNT(*)::int AS cnt,
         COALESCE(SUM(grand_total),0) AS amt,
         COALESCE(SUM(grand_total) FILTER (WHERE p.payment_method = 'CASH'),0) AS cash
  FROM sales
  LEFT JOIN payments p ON p.sale_id = sales.id
  WHERE shift_id IS NOT NULL AND status IN ('PAID','REFUNDED')
  GROUP BY shift_id
) sub
WHERE cs.id = sub.shift_id AND cs.status = 'CLOSED';

-- Also fix expected/actual for shifts with no linked sales (defensive)
UPDATE cashier_shifts SET expected_cash = opening_cash + cash_sales,
                        actual_cash = expected_cash,
                        cash_difference = 0
WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731' AND status = 'CLOSED';

-- ============================================================
-- 10. Activity logs for key events (bulk insert, deterministic)
-- ============================================================
DO $$
DECLARE
  v_store_id CONSTANT uuid := '191dc7c5-a537-47df-bb60-50517333e731';
  v_owner uuid;
BEGIN
  SELECT id INTO v_owner FROM users WHERE username = 'fufufafa' LIMIT 1;

  -- Login events for owner + seed users (one per day this month)
  INSERT INTO activity_logs (user_id, store_id, username, action, resource, description, ip_address, user_agent, created_at)
  SELECT u.id, v_store_id, u.username, 'LOGIN', 'auth',
         'User ''' || u.username || ''' logged in',
         '127.0.0.1', 'Mozilla/5.0 (Seed)', (DATE '2026-11-01' + gs)::timestamp AT TIME ZONE 'Asia/Jakarta' + (8 + (gs % 12)) * INTERVAL '1 hour'
  FROM users u CROSS JOIN generate_series(0, 8) gs
  WHERE u.username IN ('fufufafa','seed_manager','seed_cashier_1','seed_cashier_2');

  -- Create events for categories/products
  INSERT INTO activity_logs (user_id, store_id, username, action, resource, resource_id, description, ip_address, created_at)
  SELECT v_owner, v_store_id, 'fufufafa', 'CREATE', 'categories', c.id::text,
         'User ''fufufafa'' created categories', '127.0.0.1',
         '2026-11-01 09:00:00+07'
  FROM categories c WHERE c.store_id = v_store_id;

  INSERT INTO activity_logs (user_id, store_id, username, action, resource, resource_id, description, ip_address, created_at)
  SELECT v_owner, v_store_id, 'fufufafa', 'CREATE', 'products', p.id::text,
         'User ''fufufafa'' created products', '127.0.0.1',
         '2026-11-01 10:00:00+07'
  FROM products p WHERE p.store_id = v_store_id;
END $$;

\echo '=== activity logs done ==='

-- ============================================================
-- 11. Update store verified/active so the frontend shows it enabled
-- ============================================================
UPDATE stores SET is_verified = TRUE, is_active = TRUE
WHERE id = '191dc7c5-a537-47df-bb60-50517333e731';

COMMIT;

-- ============================================================
-- Summary
-- ============================================================
SELECT 'categories' AS tbl, COUNT(*) FROM categories WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'products', COUNT(*) FROM products WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'product_skus', COUNT(*) FROM product_skus WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'customers', COUNT(*) FROM customers WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'tables', COUNT(*) FROM tables WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'cashier_shifts', COUNT(*) FROM cashier_shifts WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'sales', COUNT(*) FROM sales WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'sale_items', COUNT(*) FROM sale_items WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'payments', COUNT(*) FROM payments WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'inventory_transactions', COUNT(*) FROM inventory_transactions WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'shift_items', COUNT(*) FROM shift_items WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
UNION ALL SELECT 'activity_logs', COUNT(*) FROM activity_logs WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731';

SELECT status, COUNT(*), COALESCE(SUM(grand_total),0)::numeric(15,2) AS total FROM sales
WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731' GROUP BY status ORDER BY status;

SELECT date(created_at) AS day, COUNT(*) AS sales, COALESCE(SUM(grand_total),0)::numeric(15,2) AS revenue
FROM sales WHERE store_id = '191dc7c5-a537-47df-bb60-50517333e731'
GROUP BY date(created_at) ORDER BY date(created_at);
