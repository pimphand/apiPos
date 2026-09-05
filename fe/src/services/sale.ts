import http from '@/lib/http';
import type {
  Sale,
  CreateSaleRequest,
  SaleDetailResponse,
  SaleItem,
  ListSalesResponse,
  Payment,
} from '@/types';

function num(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function str(v: unknown): string {
  return v == null ? '' : String(v);
}

function optStr(snake: unknown, camel: unknown): string | undefined {
  if (snake !== undefined && snake !== null && String(snake) !== '') return str(snake);
  if (camel !== undefined && camel !== null && String(camel) !== '') return str(camel);
  return undefined;
}

function normalizeSale(raw: Record<string, unknown>): Sale {
  return {
    id: str(raw.id),
    invoice_number: str(raw.invoice_number ?? raw.invoiceNumber),
    store_id: str(raw.store_id ?? raw.storeId),
    store_name: optStr(raw.store_name, raw.storeName),
    cashier_id: str(raw.cashier_id ?? raw.cashierId),
    cashier_name: optStr(raw.cashier_name, raw.cashierName),
    customer_id: str(raw.customer_id ?? raw.customerId),
    customer_name: optStr(raw.customer_name, raw.customerName),
    subtotal: num(raw.subtotal),
    discount: num(raw.discount),
    tax: num(raw.tax),
    grand_total: num(raw.grand_total ?? raw.grandTotal),
    status: (str(raw.status) || 'PENDING') as Sale['status'],
    item_count:
      raw.item_count !== undefined ? num(raw.item_count) : raw.itemCount !== undefined ? num(raw.itemCount) : undefined,
    created_at: str(raw.created_at ?? raw.createdAt),
    updated_at: str(raw.updated_at ?? raw.updatedAt),
  };
}

function normalizeItem(raw: Record<string, unknown>): SaleItem {
  return {
    id: str(raw.id),
    sale_id: str(raw.sale_id ?? raw.saleId),
    product_id: str(raw.product_id ?? raw.productId),
    product_name: optStr(raw.product_name, raw.productName),
    sku_id: optStr(raw.sku_id, raw.skuId),
    sku_code: optStr(raw.sku_code, raw.skuCode),
    quantity: num(raw.quantity),
    unit_price: num(raw.unit_price ?? raw.unitPrice),
    discount: num(raw.discount),
    subtotal: num(raw.subtotal),
    created_at: str(raw.created_at ?? raw.createdAt),
  };
}

function normalizePayment(raw: Record<string, unknown>): Payment {
  return {
    id: str(raw.id),
    sale_id: str(raw.sale_id ?? raw.saleId),
    payment_method: (str(raw.payment_method ?? raw.paymentMethod) || 'OTHER') as Payment['payment_method'],
    amount: num(raw.amount),
    reference: str(raw.reference),
    paid_at: str(raw.paid_at ?? raw.paidAt),
    created_at: str(raw.created_at ?? raw.createdAt),
  };
}

function normalizeDetail(raw: Record<string, unknown>): SaleDetailResponse {
  const saleRaw = (raw.sale ?? raw) as Record<string, unknown>;
  return {
    sale: normalizeSale(saleRaw),
    items: Array.isArray(raw.items)
      ? (raw.items as Record<string, unknown>[]).map(normalizeItem)
      : [],
    payments: Array.isArray(raw.payments)
      ? (raw.payments as Record<string, unknown>[]).map(normalizePayment)
      : [],
    amount_paid: num(raw.amount_paid ?? raw.amountPaid),
    change: num(raw.change),
  };
}

function normalizeList(data: Record<string, unknown>): ListSalesResponse {
  return {
    sales: Array.isArray(data.sales)
      ? (data.sales as Record<string, unknown>[]).map(normalizeSale)
      : [],
    total_count: num(data.total_count ?? data.totalCount),
    page: num(data.page),
    per_page: num(data.per_page ?? data.perPage),
  };
}

export const saleService = {
  async create(data: CreateSaleRequest): Promise<SaleDetailResponse> {
    const response = await http.post<Record<string, unknown>>('/v1/sales', data);
    return normalizeDetail(response.data);
  },

  async get(id: string): Promise<SaleDetailResponse> {
    const response = await http.get<Record<string, unknown>>(`/v1/sales/${id}`);
    return normalizeDetail(response.data);
  },

  async getByInvoice(invoiceNumber: string): Promise<SaleDetailResponse> {
    const response = await http.get<Record<string, unknown>>(
      `/v1/sales/invoice/${invoiceNumber}`
    );
    return normalizeDetail(response.data);
  },

  async list(params: {
    store_id?: string;
    cashier_id?: string;
    customer_id?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    per_page?: number;
  }): Promise<ListSalesResponse> {
    const response = await http.get<Record<string, unknown>>('/v1/sales', { params });
    return normalizeList(response.data);
  },

  async cancel(id: string, reason?: string): Promise<Sale> {
    const response = await http.post<Record<string, unknown>>(`/v1/sales/${id}/cancel`, { reason });
    const data = response.data as Record<string, unknown>;
    return normalizeSale((data.sale ?? data) as Record<string, unknown>);
  },

  async refund(id: string, reason?: string): Promise<Sale> {
    const response = await http.post<Record<string, unknown>>(`/v1/sales/${id}/refund`, { reason });
    const data = response.data as Record<string, unknown>;
    return normalizeSale((data.sale ?? data) as Record<string, unknown>);
  },
};