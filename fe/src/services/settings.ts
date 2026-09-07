import http from '@/lib/http';
import type { StoreSettings, StoreInfo } from '@/types';

function bool(v: unknown): boolean {
  if (v == null) return false;
  if (typeof v === 'boolean') return v;
  if (typeof v === 'number') return v !== 0;
  return String(v) === 'true' || String(v) === '1';
}

function num(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function str(v: unknown): string {
  return v == null ? '' : String(v);
}

function normalizeSettings(raw: Record<string, unknown> | null | undefined): StoreSettings {
  const r = raw ?? {};
  const storeId = str(r.store_id ?? r.storeId);
  // proto3 JSON omits empty strings, so a missing align means 'center'.
  const footerAlign = str(r.receipt_footer_align ?? r.receiptFooterAlign).toLowerCase();
  return {
    store_id: storeId,
    // proto3 JSON omits zero fields (protojson without EmitUnpopulated), so a
    // missing field means the stored value is its zero value — false/0 — NOT the
    // 11% "not configured yet" default. A store that never saved settings gets
    // the real 11% default from the backend (DefaultTaxRate), which is non-zero
    // and therefore present in the response. Any tax rate that was actually
    // saved as 0 comes back absent and must stay 0 here.
    tax_rate: num(r.tax_rate ?? r.taxRate),
    receipt_header: str(r.receipt_header ?? r.receiptHeader),
    receipt_footer: str(r.receipt_footer ?? r.receiptFooter),
    receipt_footer_align: footerAlign === 'left' || footerAlign === 'right' ? footerAlign : 'center',
    show_store_name: bool(r.show_store_name ?? r.showStoreName),
    show_store_contact: bool(r.show_store_contact ?? r.showStoreContact),
    show_cashier: bool(r.show_cashier ?? r.showCashier),
    show_customer: bool(r.show_customer ?? r.showCustomer),
    show_item_sku: bool(r.show_item_sku ?? r.showItemSku),
    show_item_barcode: bool(r.show_item_barcode ?? r.showItemBarcode),
    show_discount: bool(r.show_discount ?? r.showDiscount),
    show_tax: bool(r.show_tax ?? r.showTax),
    show_payment: bool(r.show_payment ?? r.showPayment),
    show_change: bool(r.show_change ?? r.showChange),
    show_kitchen_print: bool(r.show_kitchen_print ?? r.showKitchenPrint),
    self_order_enabled: bool(r.self_order_enabled ?? r.selfOrderEnabled),
    updated_by: str(r.updated_by ?? r.updatedBy),
    updated_at: str(r.updated_at ?? r.updatedAt),
  };
}

function normalizeStoreInfo(raw: Record<string, unknown>): StoreInfo {
  return {
    name: str(raw.name),
    address: str(raw.address ?? raw.Address),
    phone: str(raw.phone ?? raw.Phone),
  };
}

export const settingsService = {
  /** GET /api/v1/settings — settings of the authenticated user's store. */
  async get(): Promise<{ settings: StoreSettings; store_info: StoreInfo }> {
    const response = await http.get<Record<string, unknown>>('/v1/settings');
    const data = (response.data ?? {}) as Record<string, unknown>;
    const settingsRaw = (data.settings ?? data) as Record<string, unknown>;
    const infoRaw = (data.store_info ?? {}) as Record<string, unknown>;
    return {
      settings: normalizeSettings(settingsRaw ?? {}),
      store_info: normalizeStoreInfo(infoRaw ?? {}),
    };
  },

  /** PUT /api/v1/settings — save (upsert) the store settings. */
  async update(data: Partial<StoreSettings>): Promise<{ settings: StoreSettings; store_info: StoreInfo }> {
    const response = await http.put<Record<string, unknown>>('/v1/settings', data);
    const body = (response.data ?? {}) as Record<string, unknown>;
    const settingsRaw = (body.settings ?? body) as Record<string, unknown>;
    const infoRaw = (body.store_info ?? {}) as Record<string, unknown>;
    return {
      settings: normalizeSettings(settingsRaw ?? {}),
      store_info: normalizeStoreInfo(infoRaw ?? {}),
    };
  },

  /** PUT /api/v1/settings/store-info — save the store's address & phone. */
  async updateStoreInfo(data: { address: string; phone: string }): Promise<StoreInfo> {
    const response = await http.put<Record<string, unknown>>('/v1/settings/store-info', data);
    const body = (response.data ?? {}) as Record<string, unknown>;
    const infoRaw = (body.store_info ?? body) as Record<string, unknown>;
    return normalizeStoreInfo(infoRaw ?? {});
  },
};
