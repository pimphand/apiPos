import http from '@/lib/http';
import type {
  Category,
  CreateCategoryRequest,
  UpdateCategoryRequest,
  ListCategoriesResponse,
} from '@/types';

function normalizeCategory(raw: Record<string, unknown>): Category {
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? ''),
    description: String(raw.description ?? ''),
    is_active: Boolean(raw.is_active ?? raw.isActive),
    created_at: String(raw.created_at ?? raw.createdAt ?? ''),
    updated_at: String(raw.updated_at ?? raw.updatedAt ?? ''),
  };
}

export const categoryService = {
  async list(params: {
    is_active?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListCategoriesResponse> {
    const response = await http.get<Record<string, unknown>>('/v1/categories', { params });
    const data = response.data as Record<string, unknown>;
    const rawCategories = (data.categories ?? []) as Record<string, unknown>[];
    return {
      categories: rawCategories.map(normalizeCategory),
      total_count: Number(data.total_count ?? data.totalCount ?? 0),
      page: Number(data.page ?? params.page ?? 1),
      per_page: Number(data.per_page ?? data.perPage ?? params.per_page ?? 20),
    };
  },

  async get(id: string): Promise<Category> {
    const response = await http.get<Record<string, unknown>>(`/v1/categories/${id}`);
    return normalizeCategory((response.data?.category ?? response.data) as Record<string, unknown>);
  },

  async create(data: CreateCategoryRequest): Promise<Category> {
    const response = await http.post<Record<string, unknown>>('/v1/categories', data);
    return normalizeCategory((response.data?.category ?? response.data) as Record<string, unknown>);
  },

  async update(id: string, data: UpdateCategoryRequest): Promise<Category> {
    const response = await http.put<Record<string, unknown>>(`/v1/categories/${id}`, data);
    return normalizeCategory((response.data?.category ?? response.data) as Record<string, unknown>);
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/categories/${id}`);
  },

  async deactivate(id: string): Promise<void> {
    await http.post(`/v1/categories/${id}/deactivate`);
  },
};
