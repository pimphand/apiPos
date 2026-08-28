import http from '@/lib/http';
import type {
  Sale,
  CreateSaleRequest,
  SaleDetailResponse,
  ListSalesResponse,
} from '@/types';

export const saleService = {
  async create(data: CreateSaleRequest): Promise<SaleDetailResponse> {
    const response = await http.post<SaleDetailResponse>('/v1/sales', data);
    return response.data;
  },

  async get(id: string): Promise<SaleDetailResponse> {
    const response = await http.get<SaleDetailResponse>(`/v1/sales/${id}`);
    return response.data;
  },

  async getByInvoice(invoiceNumber: string): Promise<SaleDetailResponse> {
    const response = await http.get<SaleDetailResponse>(
      `/v1/sales/invoice/${invoiceNumber}`
    );
    return response.data;
  },

  async list(params: {
    store_id?: string;
    cashier_id?: string;
    customer_id?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListSalesResponse> {
    const response = await http.get<ListSalesResponse>('/v1/sales', { params });
    return response.data;
  },

  async cancel(id: string, reason?: string): Promise<Sale> {
    const response = await http.post<{ sale: Sale }>(`/v1/sales/${id}/cancel`, { reason });
    return response.data.sale;
  },

  async refund(id: string, reason?: string): Promise<Sale> {
    const response = await http.post<{ sale: Sale }>(`/v1/sales/${id}/refund`, { reason });
    return response.data.sale;
  },
};
