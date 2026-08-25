-- name: GetDailySalesReport :many
SELECT
    DATE(s.created_at) as sale_date,
    COUNT(s.id) as total_transactions,
    SUM(s.subtotal) as total_subtotal,
    SUM(s.discount) as total_discount,
    SUM(s.tax) as total_tax,
    SUM(s.grand_total) as total_grand_total
FROM sales s
WHERE s.status = 'PAID'
  AND s.created_at >= $1
  AND s.created_at <= $2
GROUP BY DATE(s.created_at)
ORDER BY sale_date DESC;

-- name: GetSalesByDateRange :many
SELECT * FROM sales
WHERE status = 'PAID'
  AND created_at >= $1
  AND created_at <= $2
ORDER BY created_at DESC;

-- name: GetTopSellingProducts :many
SELECT
    p.id,
    p.sku,
    p.name,
    SUM(si.quantity) as total_quantity_sold,
    SUM(si.subtotal) as total_revenue
FROM sale_items si
JOIN products p ON si.product_id = p.id
JOIN sales s ON si.sale_id = s.id
WHERE s.status = 'PAID'
  AND s.created_at >= $1
  AND s.created_at <= $2
GROUP BY p.id, p.sku, p.name
ORDER BY total_quantity_sold DESC
LIMIT $3;

-- name: GetInventorySummaryReport :many
SELECT
    p.id,
    p.sku,
    p.name,
    c.name as category_name,
    p.stock,
    p.minimum_stock,
    p.cost_price,
    p.selling_price,
    (p.stock * p.cost_price) as stock_value,
    CASE WHEN p.stock <= p.minimum_stock THEN true ELSE false END as is_low_stock
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
WHERE p.is_active = true
ORDER BY p.name;

-- name: GetPaymentMethodSummary :many
SELECT
    payment_method,
    COUNT(*) as transaction_count,
    SUM(amount) as total_amount
FROM payments
WHERE paid_at >= $1
  AND paid_at <= $2
GROUP BY payment_method
ORDER BY total_amount DESC;
