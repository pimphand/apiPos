import http from '@/lib/http';
import type { Store } from '@/types';

export interface ListStoresResponse {
  stores: Store[];
  total_count: number;
  page: number;
  per_page: number;
}

export const storeService = {
  async list(params: {
    is_verified?: string;
    is_active?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListStoresResponse> {
    const response = await http.get<ListStoresResponse>('/v1/stores', { params });
    return response.data;
  },

  async get(id: string): Promise<Store> {
    const response = await http.get<{ store: Store }>(`/v1/stores/${id}`);
    return response.data.store;
  },

  async verify(id: string): Promise<Store> {
    const response = await http.post<{ store: Store }>(`/v1/stores/${id}/verify`);
    return response.data.store;
  },
};
