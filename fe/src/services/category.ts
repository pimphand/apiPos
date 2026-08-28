import http from '@/lib/http';
import type {
  Category,
  CreateCategoryRequest,
  UpdateCategoryRequest,
  ListCategoriesResponse,
} from '@/types';

export const categoryService = {
  async list(params: {
    is_active?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListCategoriesResponse> {
    const response = await http.get<ListCategoriesResponse>('/v1/categories', { params });
    return response.data;
  },

  async get(id: string): Promise<Category> {
    const response = await http.get<{ category: Category }>(`/v1/categories/${id}`);
    return response.data.category;
  },

  async create(data: CreateCategoryRequest): Promise<Category> {
    const response = await http.post<{ category: Category }>('/v1/categories', data);
    return response.data.category;
  },

  async update(id: string, data: UpdateCategoryRequest): Promise<Category> {
    const response = await http.put<{ category: Category }>(`/v1/categories/${id}`, data);
    return response.data.category;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/categories/${id}`);
  },

  async deactivate(id: string): Promise<void> {
    await http.post(`/v1/categories/${id}/deactivate`);
  },
};
