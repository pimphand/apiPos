import http from '@/lib/http';
import type {
  Customer,
  CreateCustomerRequest,
  UpdateCustomerRequest,
  ListCustomersResponse,
  Sale,
} from '@/types';

export const customerService = {
  async list(params: {
    is_active?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListCustomersResponse> {
    const response = await http.get<ListCustomersResponse>('/v1/customers', { params });
    return response.data;
  },

  async get(id: string): Promise<Customer> {
    const response = await http.get<{ customer: Customer }>(`/v1/customers/${id}`);
    return response.data.customer;
  },

  async create(data: CreateCustomerRequest): Promise<Customer> {
    const response = await http.post<{ customer: Customer }>('/v1/customers', data);
    return response.data.customer;
  },

  async update(id: string, data: UpdateCustomerRequest): Promise<Customer> {
    const response = await http.put<{ customer: Customer }>(`/v1/customers/${id}`, data);
    return response.data.customer;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/customers/${id}`);
  },

  async deactivate(id: string): Promise<void> {
    await http.post(`/v1/customers/${id}/deactivate`);
  },

  async search(params: {
    query: string;
    page?: number;
    per_page?: number;
  }): Promise<ListCustomersResponse> {
    const response = await http.get<ListCustomersResponse>('/v1/customers/search', { params });
    return response.data;
  },

  async getPurchaseHistory(id: string, params?: {
    page?: number;
    per_page?: number;
  }): Promise<{ purchases: Sale[]; total_count: number }> {
    const response = await http.get(`/v1/customers/${id}/purchases`, { params });
    return response.data;
  },
};
