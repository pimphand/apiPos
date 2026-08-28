import http from '@/lib/http';
import type {
  Payment,
  CreatePaymentRequest,
  PaymentSummary,
  ListPaymentsResponse,
} from '@/types';

export const paymentService = {
  async create(data: CreatePaymentRequest): Promise<Payment> {
    const response = await http.post<{ payment: Payment }>('/v1/payments', data);
    return response.data.payment;
  },

  async get(id: string): Promise<Payment> {
    const response = await http.get<{ payment: Payment }>(`/v1/payments/${id}`);
    return response.data.payment;
  },

  async getBySale(saleId: string): Promise<Payment[]> {
    const response = await http.get<ListPaymentsResponse>(
      `/v1/sales/${saleId}/payments`
    );
    return response.data.payments;
  },

  async list(params: {
    sale_id?: string;
    payment_method?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListPaymentsResponse> {
    const response = await http.get<ListPaymentsResponse>('/v1/payments', { params });
    return response.data;
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/v1/payments/${id}`);
  },

  async getSummary(saleId: string): Promise<PaymentSummary> {
    const response = await http.get<PaymentSummary>(
      `/v1/sales/${saleId}/payment-summary`
    );
    return response.data;
  },
};
