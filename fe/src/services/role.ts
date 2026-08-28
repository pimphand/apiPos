import http from '@/lib/http';
import type { Role, Permission } from '@/types';

interface ListRolesResponse {
  roles: Role[];
  total_count: number;
  page: number;
  per_page: number;
}

interface RolePermissionsResponse {
  permissions: Permission[];
}

export const roleService = {
  async list(params: {
    page?: number;
    per_page?: number;
  }): Promise<ListRolesResponse> {
    const response = await http.get<ListRolesResponse>('/v1/roles', { params });
    return response.data;
  },

  async get(id: string): Promise<Role> {
    const response = await http.get<{ role: Role }>(`/v1/roles/${id}`);
    return response.data.role;
  },

  async create(data: { name: string; description?: string }): Promise<Role> {
    const response = await http.post<{ role: Role }>('/v1/roles', data);
    return response.data.role;
  },

  async update(id: string, data: { name?: string; description?: string }): Promise<Role> {
    const response = await http.put<{ role: Role }>(`/v1/roles/${id}`, data);
    return response.data.role;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/roles/${id}`);
  },

  async getPermissions(roleId: string): Promise<Permission[]> {
    const response = await http.get<RolePermissionsResponse>(
      `/v1/roles/${roleId}/permissions`
    );
    return response.data.permissions;
  },

  async assignPermissions(roleId: string, permissionIds: string[]): Promise<Permission[]> {
    const response = await http.post<RolePermissionsResponse>(
      `/v1/roles/${roleId}/permissions`,
      { permission_ids: permissionIds }
    );
    return response.data.permissions;
  },

  async removePermission(roleId: string, permissionId: string): Promise<void> {
    await http.delete(`/v1/roles/${roleId}/permissions/${permissionId}`);
  },
};
