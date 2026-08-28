import http from '@/lib/http';
import type {
  InventoryTransaction,
  ListTransactionsResponse,
} from '@/types';

interface StockSummary {
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

export const inventoryService = {
  async createTransaction(data: {
    product_id: string;
    store_id: string;
    type: string;
    quantity: number;
    reference_type?: string;
    reference_id?: string;
    description?: string;
  }): Promise<InventoryTransaction> {
    const response = await http.post<{ transaction: InventoryTransaction }>(
      '/v1/inventory/transactions',
      data
    );
    return response.data.transaction;
  },

  async getTransaction(id: string): Promise<InventoryTransaction> {
    const response = await http.get<{ transaction: InventoryTransaction }>(
      `/v1/inventory/transactions/${id}`
    );
    return response.data.transaction;
  },

  async listTransactions(params: {
    product_id?: string;
    store_id?: string;
    type?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListTransactionsResponse> {
    const response = await http.get<ListTransactionsResponse>(
      '/v1/inventory/transactions',
      { params }
    );
    return response.data;
  },

  async adjustStock(data: {
    product_id: string;
    store_id: string;
    direction: 'INCREASE' | 'DECREASE';
    quantity: number;
    reason?: string;
    description?: string;
  }): Promise<InventoryTransaction> {
    const response = await http.post<{ transaction: InventoryTransaction }>(
      '/v1/inventory/adjust',
      data
    );
    return response.data.transaction;
  },

  async getStockSummary(productId: string): Promise<StockSummary> {
    const response = await http.get<StockSummary>(
      `/v1/inventory/stock/${productId}`
    );
    return response.data;
  },

  async getLowStockAlert(params?: {
    store_id?: string;
    page?: number;
    per_page?: number;
  }): Promise<{
    products: Array<{
      product_id: string;
      product_name: string;
      category_name: string;
      total_stock: number;
      minimum_stock: number;
      deficit: number;
    }>;
    total_count: number;
  }> {
    const response = await http.get('/v1/inventory/low-stock', { params });
    return response.data;
  },
};
