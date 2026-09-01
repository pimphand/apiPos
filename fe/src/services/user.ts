import http from '@/lib/http';
import type { User } from '@/types';

interface ListUsersResponse {
  users: User[];
  total_count: number;
  page: number;
  per_page: number;
}

interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  full_name: string;
  role_id: string;
  store_id?: string;
}

interface UpdateUserRequest {
  username?: string;
  email?: string;
  full_name?: string;
  role_id?: string;
  store_id?: string;
  is_active?: boolean;
}

export const userService = {
  async list(params: {
    role_id?: string;
    is_active?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListUsersResponse> {
    const response = await http.get<ListUsersResponse>('/v1/users', { params });
    return response.data;
  },

  async get(id: string): Promise<User> {
    const response = await http.get<{ user: User }>(`/v1/users/${id}`);
    return response.data.user;
  },

  async create(data: CreateUserRequest): Promise<User> {
    const response = await http.post<{ user: User }>('/v1/users', data);
    return response.data.user;
  },

  async update(id: string, data: UpdateUserRequest): Promise<User> {
    const response = await http.put<{ user: User }>(`/v1/users/${id}`, data);
    return response.data.user;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/users/${id}`);
  },

  async updateRole(id: string, roleId: string): Promise<User> {
    const response = await http.patch<{ user: User }>(`/v1/users/${id}/role`, {
      role_id: roleId,
    });
    return response.data.user;
  },

  async toggleActive(id: string, isActive: boolean): Promise<User> {
    const response = await http.patch<{ user: User }>(`/v1/users/${id}/active`, {
      action: isActive ? 'activate' : 'deactivate',
    });
    return response.data.user;
  },
};
