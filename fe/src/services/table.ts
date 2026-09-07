import http from '@/lib/http';
import type { Table, ListTablesResponse, CreateTableRequest, UpdateTableRequest } from '@/types';

function num(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function str(v: unknown): string {
  return v == null ? '' : String(v);
}

function normalizeTable(raw: Record<string, unknown>): Table {
  return {
    id: str(raw.id),
    store_id: str(raw.store_id ?? raw.storeId),
    code: str(raw.code),
    name: str(raw.name),
    qr_url: str(raw.qr_url ?? raw.qrUrl),
    is_active: raw.is_active === true || raw.isActive === true || String(raw.is_active ?? raw.isActive) === 'true',
    created_at: str(raw.created_at ?? raw.createdAt),
    updated_at: str(raw.updated_at ?? raw.updatedAt),
  };
}

function normalizeList(data: Record<string, unknown>): ListTablesResponse {
  return {
    tables: Array.isArray(data.tables)
      ? (data.tables as Record<string, unknown>[]).map(normalizeTable)
      : [],
    total_count: num(data.total_count ?? data.totalCount),
    page: num(data.page),
    per_page: num(data.per_page ?? data.perPage),
  };
}

export const tableService = {
  async list(params: { code?: string; is_active?: string; page?: number; per_page?: number } = {}): Promise<ListTablesResponse> {
    const cleaned: Record<string, unknown> = {};
    if (params.code) cleaned.code = params.code;
    if (params.is_active) cleaned.is_active = params.is_active;
    if (params.page) cleaned.page = params.page;
    if (params.per_page) cleaned.per_page = params.per_page;
    const res = await http.get<Record<string, unknown>>('/v1/tables', { params: cleaned });
    return normalizeList(res.data);
  },

  async get(id: string): Promise<Table> {
    const res = await http.get<Record<string, unknown>>(`/v1/tables/${id}`);
    return normalizeTable((res.data?.table ?? res.data) as Record<string, unknown>);
  },

  async create(data: CreateTableRequest): Promise<Table> {
    const res = await http.post<Record<string, unknown>>('/v1/tables', data);
    return normalizeTable((res.data?.table ?? res.data) as Record<string, unknown>);
  },

  async update(id: string, data: UpdateTableRequest): Promise<Table> {
    const res = await http.put<Record<string, unknown>>(`/v1/tables/${id}`, data);
    return normalizeTable((res.data?.table ?? res.data) as Record<string, unknown>);
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/tables/${id}`);
  },

  async rotateToken(id: string): Promise<Table> {
    const res = await http.post<Record<string, unknown>>(`/v1/tables/${id}/rotate-token`, {});
    return normalizeTable((res.data?.table ?? res.data) as Record<string, unknown>);
  },
};

export default tableService;