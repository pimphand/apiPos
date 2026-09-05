import http from '@/lib/http';
import type {
  CashierShift,
  ShiftSummary,
  ShiftItem,
  ListShiftsResponse,
} from '@/types';

export const cashierService = {
  async openShift(data: {
    store_id: string;
    opening_cash: number;
  }): Promise<CashierShift> {
    const response = await http.post<{ shift: CashierShift }>(
      '/v1/cashier/shifts/open',
      data
    );
    return response.data.shift;
  },

  async closeShift(data: {
    shift_id: string;
    closing_cash: number;
    actual_cash: number;
  }): Promise<CashierShift> {
    const response = await http.post<{ shift: CashierShift }>(
      '/v1/cashier/shifts/close',
      data
    );
    return response.data.shift;
  },

  async getCurrentShift(): Promise<CashierShift> {
    const response = await http.get<{ shift: CashierShift }>(
      '/v1/cashier/shifts/current'
    );
    return response.data.shift;
  },

  async getShift(id: string): Promise<CashierShift> {
    const response = await http.get<{ shift: CashierShift }>(
      `/v1/cashier/shifts/${id}`
    );
    return response.data.shift;
  },

  async listShifts(params: {
    user_id?: string;
    store_id?: string;
    status?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListShiftsResponse> {
    const response = await http.get<ListShiftsResponse>(
      '/v1/cashier/shifts',
      { params }
    );
    return response.data;
  },

  async getShiftSummary(id: string): Promise<ShiftSummary> {
    const response = await http.get<ShiftSummary>(
      `/v1/cashier/shifts/${id}/summary`
    );
    return response.data;
  },

  async listShiftItems(params: {
    id: string;
    direction?: 'OUT' | 'IN';
    page?: number;
    per_page?: number;
  }): Promise<{ items: ShiftItem[]; total_count: number; page: number; per_page: number }> {
    const { id, ...query } = params;
    const response = await http.get<{
      items: ShiftItem[];
      total_count: number;
      page: number;
      per_page: number;
    }>(`/v1/cashier/shifts/${id}/items`, { params: query });
    return response.data;
  },
};
