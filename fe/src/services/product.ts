import http from '@/lib/http';
import type {
  Product,
  CreateProductRequest,
  UpdateProductRequest,
  ListProductsResponse,
  ProductSKU,
} from '@/types';

function num(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function toDateOnly(v: string): string {
  if (!v) return '';
  // Backend may return a full ISO timestamp (e.g. "2026-08-30T00:00:00Z")
  // but <input type="date"> requires "YYYY-MM-DD".
  const m = v.match(/^\d{4}-\d{2}-\d{2}/);
  return m ? m[0] : v;
}

function normalizeSKU(raw: Record<string, unknown>): ProductSKU {
  return {
    id: String(raw.id ?? ''),
    product_id: String(raw.product_id ?? raw.productId ?? ''),
    sku: String(raw.sku ?? ''),
    barcode: String(raw.barcode ?? ''),
    variant: String(raw.variant ?? ''),
    color: String(raw.color ?? ''),
    size: String(raw.size ?? ''),
    unit: String(raw.unit ?? ''),
    cost_price: num(raw.cost_price ?? raw.costPrice),
    selling_price: num(raw.selling_price ?? raw.sellingPrice),
    stock: num(raw.stock),
    minimum_stock: num(raw.minimum_stock ?? raw.minimumStock),
    weight: num(raw.weight),
    volume: num(raw.volume),
    expiry_date: toDateOnly(String(raw.expiry_date ?? raw.expiryDate ?? '')),
    batch_number: String(raw.batch_number ?? raw.batchNumber ?? ''),
    location: String(raw.location ?? ''),
    image_url: String(raw.image_url ?? raw.imageUrl ?? ''),
    is_active: Boolean(raw.is_active ?? raw.isActive),
    metadata: (raw.metadata ?? undefined) as Record<string, string> | undefined,
    created_at: String(raw.created_at ?? raw.createdAt ?? ''),
    updated_at: String(raw.updated_at ?? raw.updatedAt ?? ''),
  };
}

function normalizeProduct(raw: Record<string, unknown>): Product {
  const skus = Array.isArray(raw.skus)
    ? (raw.skus as Record<string, unknown>[]).map(normalizeSKU)
    : undefined;
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? ''),
    description: String(raw.description ?? ''),
    category_id: String(raw.category_id ?? raw.categoryId ?? ''),
    category_name:
      raw.category_name !== undefined
        ? String(raw.category_name)
        : raw.categoryName !== undefined
          ? String(raw.categoryName)
          : undefined,
    cost_price: num(raw.cost_price ?? raw.costPrice),
    selling_price: num(raw.selling_price ?? raw.sellingPrice),
    stock: num(raw.stock),
    minimum_stock: num(raw.minimum_stock ?? raw.minimumStock),
    is_active: Boolean(raw.is_active ?? raw.isActive),
    sku_count: num(raw.sku_count ?? raw.skuCount),
    skus,
    created_at: String(raw.created_at ?? raw.createdAt ?? ''),
    updated_at: String(raw.updated_at ?? raw.updatedAt ?? ''),
  };
}

function normalizeList(data: Record<string, unknown>): ListProductsResponse {
  const rawProducts = (data.products ?? []) as Record<string, unknown>[];
  return {
    products: rawProducts.map(normalizeProduct),
    total_count: num(data.total_count ?? data.totalCount),
    page: num(data.page),
    per_page: num(data.per_page ?? data.perPage),
  };
}

export const productService = {
  async list(params: {
    search?: string;
    page?: number;
    per_page?: number;
    is_active?: string;
    category_id?: string;
    get_sku?: boolean;
  }): Promise<ListProductsResponse> {
    // Filter out undefined/empty params so we don't send empty strings
    const cleaned: Record<string, unknown> = {};
    if (params.search) cleaned.search = params.search;
    if (params.page) cleaned.page = params.page;
    if (params.per_page) cleaned.per_page = params.per_page;
    if (params.is_active) cleaned.is_active = params.is_active;
    if (params.category_id) cleaned.category_id = params.category_id;
    if (params.get_sku) cleaned.get_sku = params.get_sku;
    const response = await http.get<Record<string, unknown>>('/v1/products', { params: cleaned });
    return normalizeList(response.data);
  },

  async get(id: string, getSku = false): Promise<Product> {
    const response = await http.get<Record<string, unknown>>(`/v1/products/${id}`, {
      params: { get_sku: getSku },
    });
    return normalizeProduct((response.data?.product ?? response.data) as Record<string, unknown>);
  },

  async create(data: CreateProductRequest): Promise<Product> {
    const response = await http.post<Record<string, unknown>>('/v1/products', data);
    return normalizeProduct((response.data?.product ?? response.data) as Record<string, unknown>);
  },

  async update(id: string, data: UpdateProductRequest): Promise<Product> {
    const response = await http.put<Record<string, unknown>>(`/v1/products/${id}`, data);
    return normalizeProduct((response.data?.product ?? response.data) as Record<string, unknown>);
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/products/${id}`);
  },

  async activate(id: string): Promise<void> {
    await http.post(`/v1/products/${id}/activate`);
  },

  async deactivate(id: string): Promise<void> {
    await http.post(`/v1/products/${id}/deactivate`);
  },

  async search(params: {
    name: string;
    page?: number;
    per_page?: number;
    is_active?: string;
    get_sku?: boolean;
  }): Promise<ListProductsResponse> {
    const response = await http.get<Record<string, unknown>>('/v1/products/search', { params });
    return normalizeList(response.data);
  },

  async getLowStock(params: {
    page?: number;
    per_page?: number;
    get_sku?: boolean;
  }): Promise<ListProductsResponse> {
    const response = await http.get<Record<string, unknown>>('/v1/products/low-stock', { params });
    return normalizeList(response.data);
  },

  // SKU methods
  async getSKUByBarcode(barcode: string): Promise<ProductSKU> {
    const response = await http.get<Record<string, unknown>>(
      `/v1/product-skus/barcode/${barcode}`
    );
    return normalizeSKU((response.data?.product_sku ?? response.data?.productSku ?? response.data) as Record<string, unknown>);
  },

  async getSKUBySKU(sku: string): Promise<ProductSKU> {
    const response = await http.get<Record<string, unknown>>(
      `/v1/product-skus/sku/${sku}`
    );
    return normalizeSKU((response.data?.product_sku ?? response.data?.productSku ?? response.data) as Record<string, unknown>);
  },

  async listSKUs(productId: string, params?: {
    search?: string;
    variant?: string;
    color?: string;
    size?: string;
    page?: number;
    per_page?: number;
  }): Promise<{ product_skus: ProductSKU[]; total_count: number }> {
    const response = await http.get<Record<string, unknown>>(`/v1/products/${productId}/skus`, { params });
    const data = response.data;
    const raw = (data.product_skus ?? data.productSkus ?? []) as Record<string, unknown>[];
    return {
      product_skus: raw.map(normalizeSKU),
      total_count: num(data.total_count ?? data.totalCount),
    };
  },
};
