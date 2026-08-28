import http from '@/lib/http';
import type {
  DailySalesReportResponse,
  SaleReport,
  CashierSalesReport,
  ProductSalesReport,
  CategorySalesReport,
  PaymentMethodReport,
  InventoryReport,
  LowStockReport,
} from '@/types';

export const reportService = {
  async getDailySales(params: {
    start_date?: string;
    end_date?: string;
    store_id?: string;
  }): Promise<DailySalesReportResponse> {
    const response = await http.get<DailySalesReportResponse>(
      '/v1/reports/daily-sales',
      { params }
    );
    return response.data;
  },

  async getSalesByDate(params: {
    start_date?: string;
    end_date?: string;
    store_id?: string;
    cashier_id?: string;
  }): Promise<{ sales: SaleReport[]; total_count: number; total_amount: number }> {
    const response = await http.get('/v1/reports/sales-by-date', { params });
    return response.data;
  },

  async getSalesByCashier(params: {
    start_date?: string;
    end_date?: string;
    store_id?: string;
  }): Promise<{ reports: CashierSalesReport[] }> {
    const response = await http.get('/v1/reports/sales-by-cashier', { params });
    return response.data;
  },

  async getSalesByProduct(params: {
    start_date?: string;
    end_date?: string;
    limit?: number;
  }): Promise<{ reports: ProductSalesReport[] }> {
    const response = await http.get('/v1/reports/sales-by-product', { params });
    return response.data;
  },

  async getSalesByCategory(params: {
    start_date?: string;
    end_date?: string;
  }): Promise<{ reports: CategorySalesReport[] }> {
    const response = await http.get('/v1/reports/sales-by-category', { params });
    return response.data;
  },

  async getPaymentSummary(params: {
    start_date?: string;
    end_date?: string;
  }): Promise<{ reports: PaymentMethodReport[]; total_amount: number }> {
    const response = await http.get('/v1/reports/payment-summary', { params });
    return response.data;
  },

  async getInventorySummary(): Promise<{
    reports: InventoryReport[];
    total_products: number;
    total_stock: number;
    total_value: number;
  }> {
    const response = await http.get('/v1/reports/inventory-summary');
    return response.data;
  },

  async getLowStock(params: {
    limit?: number;
  }): Promise<{ products: LowStockReport[] }> {
    const response = await http.get('/v1/reports/low-stock', { params });
    return response.data;
  },
};
