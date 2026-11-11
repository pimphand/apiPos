/**
 * Public self-order API client (docs/pos-self-order.md). All calls are
 * unauthenticated — the backend resolves the tenant from the table token.
 * Uses a bare axios instance so the auth token interceptor is never involved.
 */
import axios from 'axios';
import { API_BASE_URL } from '@/lib/apiBase';
import type {
  PublicTableInfo,
  PublicMenu,
  SelfOrder,
  CreatePublicOrderRequest,
  PayPublicOrderRequest,
} from '@/types';

function num(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function str(v: unknown): string {
  return v == null ? '' : String(v);
}

function bool(v: unknown): boolean {
  return v === true || String(v) === 'true' || String(v) === '1';
}

const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
});

function normalizeTable(raw: Record<string, unknown>): PublicTableInfo {
  return {
    id: str(raw.id),
    store_id: str(raw.store_id ?? raw.storeId),
    store_name: str(raw.store_name ?? raw.storeName),
    code: str(raw.code),
    name: str(raw.name),
    self_order_enabled: bool(raw.self_order_enabled ?? raw.selfOrderEnabled),
  };
}

function normalizeMenu(raw: Record<string, unknown>): PublicMenu {
  const products = Array.isArray(raw.products)
    ? (raw.products as Record<string, unknown>[]).map((p) => ({
        id: str(p.id),
        name: str(p.name),
        description: str(p.description),
        category_id: str(p.category_id ?? p.categoryId),
        category_name: str(p.category_name ?? p.categoryName),
        image_url: str(p.image_url ?? p.imageUrl),
        variants: Array.isArray(p.variants)
          ? (p.variants as Record<string, unknown>[]).map((v) => ({
              sku_id: str(v.sku_id ?? v.skuId),
              sku_code: str(v.sku_code ?? v.skuCode),
              variant: str(v.variant),
              color: str(v.color),
              size: str(v.size),
              selling_price: num(v.selling_price ?? v.sellingPrice),
              available: bool(v.available),
            }))
          : [],
      }))
    : [];
  return {
    store_name: str(raw.store_name ?? raw.storeName),
    table_code: str(raw.table_code ?? raw.tableCode),
    table_name: str(raw.table_name ?? raw.tableName),
    categories: Array.isArray(raw.categories)
      ? (raw.categories as Record<string, unknown>[]).map((c) => ({
          id: str(c.id),
          name: str(c.name),
        }))
      : [],
    products,
  };
}

function normalizeOrder(raw: Record<string, unknown>): SelfOrder {
  return {
    id: str(raw.id),
    invoice_number: str(raw.invoice_number ?? raw.invoiceNumber),
    store_id: str(raw.store_id ?? raw.storeId),
    table_id: str(raw.table_id ?? raw.tableId),
    table_no: str(raw.table_no ?? raw.tableNo),
    status: (str(raw.status) || 'PENDING') as SelfOrder['status'],
    order_status: str(raw.order_status ?? raw.orderStatus),
    order_source: str(raw.order_source ?? raw.orderSource),
    customer_name: str(raw.customer_name ?? raw.customerName),
    subtotal: num(raw.subtotal),
    discount: num(raw.discount),
    tax: num(raw.tax),
    grand_total: num(raw.grand_total ?? raw.grandTotal),
    order_token: str(raw.order_token ?? raw.orderToken),
    created_at: str(raw.created_at ?? raw.createdAt),
    updated_at: str(raw.updated_at ?? raw.updatedAt),
  };
}

export const selfOrderService = {
  /** GET /api/v1/public/table/:token — resolve table + store. */
  async resolveTable(token: string): Promise<PublicTableInfo> {
    const res = await http.get<Record<string, unknown>>(`/v1/public/table/${encodeURIComponent(token)}`);
    const data = (res.data?.table ?? res.data) as Record<string, unknown>;
    return normalizeTable(data);
  },

  /** GET /api/v1/public/menu/:token — safe dine-in menu. */
  async getMenu(token: string): Promise<PublicMenu> {
    const res = await http.get<Record<string, unknown>>(`/v1/public/menu/${encodeURIComponent(token)}`);
    return normalizeMenu(res.data);
  },

  /** POST /api/v1/public/orders — submit (PENDING, stock locked). */
  async createOrder(data: CreatePublicOrderRequest): Promise<SelfOrder> {
    const res = await http.post<Record<string, unknown>>('/v1/public/orders', data);
    const order = (res.data?.order ?? res.data) as Record<string, unknown>;
    return normalizeOrder(order);
  },

  /** POST /api/v1/public/orders/:token/pay — pay → PAID. */
  async payOrder(data: PayPublicOrderRequest): Promise<SelfOrder> {
    const res = await http.post<Record<string, unknown>>(
      `/v1/public/orders/${encodeURIComponent(data.order_token)}/pay`,
      { payment_method: data.payment_method, reference: data.reference ?? '' }
    );
    const order = (res.data?.order ?? res.data) as Record<string, unknown>;
    return normalizeOrder(order);
  },

  /** GET /api/v1/public/orders/:token — order status. */
  async getOrderStatus(token: string): Promise<SelfOrder> {
    const res = await http.get<Record<string, unknown>>(`/v1/public/orders/${encodeURIComponent(token)}`);
    const order = (res.data?.order ?? res.data) as Record<string, unknown>;
    return normalizeOrder(order);
  },
};

export default selfOrderService;