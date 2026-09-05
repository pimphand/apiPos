import http from '@/lib/http';
import type { StoreSettings } from '@/types';

function bool(v: unknown): boolean {
  if (typeof v === 'boolean') return v;
  return String(v ?? '') === 'true';
}

function num(v: unknown): number {
  const n = Number(v ?? 11);
  return Number.isFinite(n) ? n : 11;
}

function str(v: unknown): string {
  return v == null ? '' : String(v);
}

function normalizeSettings(raw: Record<string, unknown>): StoreSettings {
  const storeId = str(raw.store_id ?? raw.storeId);
  // The gateway omits zero values from JSON (protojson). A missing tax_rate on
  // an existing row therefore means the store deliberately saved 0% tax — only
  // rows that never existed fall back to the 11% default.
  const taxRaw = raw.tax_rate ?? raw.taxRate;
  const taxRate = taxRaw === undefined || taxRaw === null
    ? (storeId ? 0 : 11)
    : num(taxRaw);
  return {
    store_id: storeId,
    tax_rate: taxRate,
    receipt_header: str(raw.receipt_header ?? raw.receiptHeader),
    receipt_footer: str(raw.receipt_footer ?? raw.receiptFooter),
    show_store_name: bool(raw.show_store_name ?? raw.showStoreName),
    show_store_contact: bool(raw.show_store_contact ?? raw.showStoreContact),
    show_cashier: bool(raw.show_cashier ?? raw.showCashier),
    show_customer: bool(raw.show_customer ?? raw.showCustomer),
    show_item_sku: bool(raw.show_item_sku ?? raw.showItemSku),
    show_item_barcode: bool(raw.show_item_barcode ?? raw.showItemBarcode),
    show_discount: bool(raw.show_discount ?? raw.showDiscount),
    show_tax: bool(raw.show_tax ?? raw.showTax),
    show_payment: bool(raw.show_payment ?? raw.showPayment),
    show_change: bool(raw.show_change ?? raw.showChange),
    updated_by: str(raw.updated_by ?? raw.updatedBy),
    updated_at: str(raw.updated_at ?? raw.updatedAt),
  };
}

export const settingsService = {
  /** GET /api/v1/settings — settings of the authenticated user's store. */
  async get(): Promise<StoreSettings> {
    const response = await http.get<Record<string, unknown>>('/v1/settings');
    const data = (response.data?.settings ?? response.data) as Record<string, unknown>;
    return normalizeSettings(data ?? {});
  },

  /** PUT /api/v1/settings — save (upsert) the store settings. */
  async update(data: Partial<StoreSettings>): Promise<StoreSettings> {
    const response = await http.put<Record<string, unknown>>('/v1/settings', data);
    const body = (response.data?.settings ?? response.data) as Record<string, unknown>;
    return normalizeSettings(body ?? {});
  },
};
