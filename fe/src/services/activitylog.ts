import http from '@/lib/http';
import type { ActivityLog } from '@/types';

interface ListActivityLogsResponse {
  activity_logs: ActivityLog[];
  total_count: number;
  page: number;
  per_page: number;
}

export const activityLogService = {
  async list(params: {
    user_id?: string;
    action?: string;
    resource?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListActivityLogsResponse> {
    const response = await http.get<ListActivityLogsResponse>(
      '/v1/activity-logs',
      { params }
    );
    return response.data;
  },

  async get(id: string): Promise<ActivityLog> {
    const response = await http.get<{ activity_log: ActivityLog }>(
      `/v1/activity-logs/${id}`
    );
    return response.data.activity_log;
  },

  async listByUser(userId: string, params?: {
    page?: number;
    per_page?: number;
  }): Promise<ListActivityLogsResponse> {
    const response = await http.get<ListActivityLogsResponse>(
      `/v1/activity-logs/user/${userId}`,
      { params }
    );
    return response.data;
  },

  async listByResource(resource: string, params?: {
    resource_id?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListActivityLogsResponse> {
    const response = await http.get<ListActivityLogsResponse>(
      `/v1/activity-logs/resource/${resource}`,
      { params }
    );
    return response.data;
  },
};
