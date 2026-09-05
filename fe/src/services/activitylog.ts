import http from '@/lib/http';
import type { ActivityLog } from '@/types';

interface ListActivityLogsResponse {
  activity_logs: ActivityLog[];
  total_count: number;
  page: number;
  per_page: number;
}

function num(v: unknown): number {
  const n = typeof v === 'number' ? v : Number(v);
  return Number.isFinite(n) ? n : 0;
}

function str(v: unknown): string {
  return v == null ? '' : String(v);
}

export function normalizeLog(raw: Record<string, any>): ActivityLog {
  return {
    id: str(raw.id),
    user_id: str(raw.user_id ?? raw.userId),
    username: str(raw.username),
    action: str(raw.action),
    resource: str(raw.resource),
    resource_id: str(raw.resource_id ?? raw.resourceId),
    description: str(raw.description),
    ip_address: str(raw.ip_address ?? raw.ipAddress),
    user_agent: str(raw.user_agent ?? raw.userAgent),
    metadata: str(raw.metadata),
    created_at: str(raw.created_at ?? raw.createdAt),
    store_id: str(raw.store_id ?? raw.storeId),
    store_name: str(raw.store_name ?? raw.storeName),
  };
}

function normalizeList(raw: any): ListActivityLogsResponse {
  const logs = Array.isArray(raw.logs ?? raw.activity_logs) ? (raw.logs ?? raw.activity_logs) : [];
  return {
    activity_logs: logs.map((l: any) => normalizeLog(l)),
    total_count: num(raw.total_count ?? raw.totalCount),
    page: num(raw.page) || 1,
    per_page: num(raw.per_page ?? raw.perPage) || 20,
  };
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
    const response = await http.get<Record<string, any>>(
      '/v1/activity-logs',
      { params }
    );
    return normalizeList(response.data);
  },

  async get(id: string): Promise<ActivityLog> {
    const response = await http.get<Record<string, any>>(
      `/v1/activity-logs/${id}`
    );
    return normalizeLog(response.data.log ?? response.data.activity_log ?? response.data);
  },

  async listByUser(userId: string, params?: {
    page?: number;
    per_page?: number;
  }): Promise<ListActivityLogsResponse> {
    const response = await http.get<Record<string, any>>(
      `/v1/activity-logs/user/${userId}`,
      { params }
    );
    return normalizeList(response.data);
  },

  async listByResource(resource: string, params?: {
    resource_id?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListActivityLogsResponse> {
    const response = await http.get<Record<string, any>>(
      `/v1/activity-logs/resource/${resource}`,
      { params }
    );
    return normalizeList(response.data);
  },
};
