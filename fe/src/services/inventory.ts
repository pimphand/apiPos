import http from '@/lib/http';
import type {
  InventoryTransaction,
  ListTransactionsResponse,
} from '@/types';

function num(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function str(v: unknown): string {
  return String(v ?? '');
}

export interface LowStockProduct {
  product_id: string;
  product_name: string;
  category_name: string;
  total_stock: number;
  minimum_stock: number;
  deficit: number;
}

export interface LowStockResponse {
  products: LowStockProduct[];
  total_count: number;
  page: number;
  per_page: number;
}

export interface StockSummary {
  product_id: string;
  product_name: string;
  total_stock: number;
  skus: Array<{
    sku_id: string;
    sku_code: string;
    variant: string;
    barcode: string;
    stock: number;
    minimum_stock: number;
    is_low_stock: boolean;
  }>;
}

function normalizeTransaction(raw: Record<string, unknown>): InventoryTransaction {
  return {
    id: str(raw.id ?? raw.id),
    product_id: str(raw.product_id ?? raw.productId),
    product_name:
      raw.product_name !== undefined
        ? str(raw.product_name)
        : raw.productName !== undefined
          ? str(raw.productName)
          : undefined,
    store_id: str(raw.store_id ?? raw.storeId),
    store_name:
      raw.store_name !== undefined
        ? str(raw.store_name)
        : raw.storeName !== undefined
          ? str(raw.storeName)
          : undefined,
    type: str(raw.type ?? '') as InventoryTransaction['type'],
    quantity: num(raw.quantity),
    reference_type: str(raw.reference_type ?? raw.referenceType),
    reference_id: str(raw.reference_id ?? raw.referenceId),
    before_stock: num(raw.before_stock ?? raw.beforeStock),
    after_stock: num(raw.after_stock ?? raw.afterStock),
    description: str(raw.description),
    created_by: str(raw.created_by ?? raw.createdBy),
    creator_name:
      raw.creator_name !== undefined
        ? str(raw.creator_name)
        : raw.creatorName !== undefined
          ? str(raw.creatorName)
          : undefined,
    created_at: str(raw.created_at ?? raw.createdAt),
  };
}

function normalizeTransactionList(data: Record<string, unknown>): ListTransactionsResponse {
  const rawTxns = (data.transactions ?? []) as Record<string, unknown>[];
  return {
    transactions: rawTxns.map(normalizeTransaction),
    total_count: num(data.total_count ?? data.totalCount),
    page: num(data.page),
    per_page: num(data.per_page ?? data.perPage),
  };
}

function normalizeLowStock(data: Record<string, unknown>): LowStockResponse {
  const raw = (data.products ?? []) as Record<string, unknown>[];
  return {
    products: raw.map((p) => ({
      product_id: str(p.product_id ?? p.productId),
      product_name: str(p.product_name ?? p.productName),
      category_name: str(p.category_name ?? p.categoryName),
      total_stock: num(p.total_stock ?? p.totalStock),
      minimum_stock: num(p.minimum_stock ?? p.minimumStock),
      deficit: num(p.deficit),
    })),
    total_count: num(data.total_count ?? data.totalCount),
    page: num(data.page),
    per_page: num(data.per_page ?? data.perPage),
  };
}

function normalizeStockSummary(data: Record<string, unknown>): StockSummary {
  const rawSkus = (data.skus ?? []) as Record<string, unknown>[];
  return {
    product_id: str(data.product_id ?? data.productId),
    product_name: str(data.product_name ?? data.productName),
    total_stock: num(data.total_stock ?? data.totalStock),
    skus: rawSkus.map((s) => ({
      sku_id: str(s.sku_id ?? s.skuId),
      sku_code: str(s.sku_code ?? s.skuCode),
      variant: str(s.variant),
      barcode: str(s.barcode),
      stock: num(s.stock),
      minimum_stock: num(s.minimum_stock ?? s.minimumStock),
      is_low_stock: Boolean(s.is_low_stock ?? s.isLowStock),
    })),
  };
}

function extractTransaction(data: Record<string, unknown>): InventoryTransaction {
  return normalizeTransaction(
    (data.transaction ?? data) as Record<string, unknown>
  );
}

export const inventoryService = {
  async listTransactions(params?: {
    product_id?: string;
    store_id?: string;
    type?: string;
    page?: number;
    per_page?: number;
  }, signal?: AbortSignal): Promise<ListTransactionsResponse> {
    const response = await http.get<Record<string, unknown>>(
      '/v1/inventory/transactions',
      { params, signal }
    );
    return normalizeTransactionList(response.data);
  },

  async getTransaction(id: string): Promise<InventoryTransaction> {
    const response = await http.get<Record<string, unknown>>(
      `/v1/inventory/transactions/${id}`
    );
    return extractTransaction(response.data);
  },

  async adjustStock(data: {
    product_id: string;
    store_id: string;
    direction: 'INCREASE' | 'DECREASE';
    quantity: number;
    reason?: string;
    description?: string;
  }): Promise<InventoryTransaction> {
    const response = await http.post<Record<string, unknown>>(
      '/v1/inventory/adjust',
      data
    );
    return extractTransaction(response.data);
  },

  async createTransaction(data: {
    product_id: string;
    store_id: string;
    type: string;
    quantity: number;
    reference_type?: string;
    reference_id?: string;
    description?: string;
  }): Promise<InventoryTransaction> {
    const response = await http.post<Record<string, unknown>>(
      '/v1/inventory/transactions',
      data
    );
    return extractTransaction(response.data);
  },

  async getLowStockAlert(params?: {
    store_id?: string;
    page?: number;
    per_page?: number;
  }, signal?: AbortSignal): Promise<LowStockResponse> {
    const response = await http.get<Record<string, unknown>>(
      '/v1/inventory/low-stock',
      { params, signal }
    );
    return normalizeLowStock(response.data);
  },

  async getStockSummary(productId: string): Promise<StockSummary> {
    const response = await http.get<Record<string, unknown>>(
      `/v1/inventory/stock/${productId}`
    );
    return normalizeStockSummary(response.data);
  },
};
