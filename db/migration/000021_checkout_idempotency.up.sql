ALTER TABLE sales ADD COLUMN checkout_id UUID;
CREATE UNIQUE INDEX idx_sales_checkout ON sales (store_id, cashier_id, checkout_id) WHERE checkout_id IS NOT NULL;
