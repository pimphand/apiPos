# Product Import (CSV / Excel)

## Overview

Admin and manager can bulk-import products from a **CSV** or **Excel (.xlsx)** file.  
The import creates both the **product** and a **default SKU** for each row.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/products/import-sample` | Download CSV template |
| `POST` | `/api/v1/products/import` | Upload CSV/Excel to import |

Both endpoints require authentication.  
`POST /import` also requires `products.create` permission.

## CSV Template

Download from the UI or directly via `GET /api/v1/products/import-sample`.

```csv
name,sku,category,cost_price,selling_price,stock,minimum_stock,unit,barcode,variant
Es Kopi Susu,KOP-001,Minuman,8000,18000,100,10,pcs,8991234567890,Large
Es Kopi Susu,KOP-002,Minuman,8000,15000,100,10,pcs,,Small
Croissant,CRO-001,Makanan,5000,15000,50,5,pcs,,Regular
```

### Column Reference

| Column | Required | Description |
|--------|----------|-------------|
| `name` | ✅ | Product name |
| `sku` | ❌ | SKU code (auto-generated if empty) |
| `category` | ❌ | Category name (auto-created if not found) |
| `cost_price` | ❌ | Cost price (default 0) |
| `selling_price` | ❌ | Selling price (default 0) |
| `stock` | ❌ | Initial stock (default 0) |
| `minimum_stock` | ❌ | Minimum stock threshold (default 0) |
| `unit` | ❌ | Unit of measure (default "pcs") |
| `barcode` | ❌ | Barcode for POS scanning |
| `variant` | ❌ | Variant name (e.g. "Large", "Small") |

## Behavior

1. **Category auto-creation**: If the category name doesn't exist, it's automatically created.
2. **SKU uniqueness**: Duplicate SKU codes within the same store are rejected with an error.
3. **Error reporting**: Each failed row is reported with line number and error message.
4. **Store-scoped**: Products are created for the authenticated user's store only.

## Upload Response

```json
{
  "total": 5,
  "imported": 4,
  "failed": 1,
  "errors": [
    { "row": 3, "msg": "SKU 'KOP-001' already exists" }
  ]
}
```

## UI Flow

1. Go to **Products** page
2. Click **Import** button (next to "+ New Product")
3. Click **Download sample CSV** to get the template
4. Fill in the template and save as CSV or Excel
5. Drag & drop or select the file
6. Click **Import**
7. Results are displayed inline (success count + error table)

## Backend Implementation

- `internal/handler/product_import.go` — HTTP handler (multipart upload, not gRPC)
- `internal/product/service.go` — `CreateProductFromImport()` creates product + default SKU
- Uses `excelize` library for .xlsx parsing
- Registered directly on the HTTP mux in `internal/server/server.go`
