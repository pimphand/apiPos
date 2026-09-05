import http from '@/lib/http';
import type { Permission } from '@/types';

interface ListPermissionsResponse {
  permissions: Permission[];
  total_count: number;
  page: number;
  per_page: number;
}

export const permissionService = {
  async list(params: {
    page?: number;
    per_page?: number;
  }): Promise<ListPermissionsResponse> {
    const response = await http.get<ListPermissionsResponse>('/v1/permissions', { params });
    return response.data;
  },

  /** Get all permissions of the currently authenticated user */
  async getMy(): Promise<Permission[]> {
    const response = await http.get<{ permissions: Permission[] }>('/v1/permissions/my');
    return response.data.permissions ?? [];
  },

  async get(id: string): Promise<Permission> {
    const response = await http.get<{ permission: Permission }>(`/v1/permissions/${id}`);
    return response.data.permission;
  },

  async create(data: {
    name: string;
    resource: string;
    action: string;
    description?: string;
  }): Promise<Permission> {
    const response = await http.post<{ permission: Permission }>('/v1/permissions', data);
    return response.data.permission;
  },

  async update(id: string, data: {
    name?: string;
    resource?: string;
    action?: string;
    description?: string;
  }): Promise<Permission> {
    const response = await http.put<{ permission: Permission }>(`/v1/permissions/${id}`, data);
    return response.data.permission;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/permissions/${id}`);
  },

  async listByResource(resource: string, params?: {
    page?: number;
    per_page?: number;
  }): Promise<ListPermissionsResponse> {
    const response = await http.get<ListPermissionsResponse>(
      `/v1/permissions/resource/${resource}`,
      { params }
    );
    return response.data;
  },
};
