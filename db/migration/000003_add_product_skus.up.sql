-- Create product_skus table for product variants/SKU items
CREATE TABLE IF NOT EXISTS product_skus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,
    barcode VARCHAR(100),
    variant VARCHAR(255),
    color VARCHAR(100),
    size VARCHAR(50),
    unit VARCHAR(50) DEFAULT 'pcs',
    cost_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    selling_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    stock INTEGER NOT NULL DEFAULT 0,
    minimum_stock INTEGER NOT NULL DEFAULT 0,
    weight NUMERIC(10,2),
    volume NUMERIC(10,2),
    expiry_date TIMESTAMP,
    batch_number VARCHAR(100),
    location VARCHAR(100),
    image_url VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(product_id, sku)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_product_skus_product_id ON product_skus(product_id);
CREATE INDEX IF NOT EXISTS idx_product_skus_sku ON product_skus(sku);
CREATE INDEX IF NOT EXISTS idx_product_skus_barcode ON product_skus(barcode);
CREATE INDEX IF NOT EXISTS idx_product_skus_variant ON product_skus(variant);
CREATE INDEX IF NOT EXISTS idx_product_skus_is_active ON product_skus(is_active);
CREATE INDEX IF NOT EXISTS idx_product_skus_expiry_date ON product_skus(expiry_date);
