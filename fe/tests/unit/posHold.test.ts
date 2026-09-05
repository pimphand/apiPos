import { describe, it, expect, beforeEach } from 'vitest';
import {
  HELD_ORDERS_STORAGE_KEY,
  HELD_ORDERS_VERSION,
  HOLD_LIMIT,
  HoldLimitError,
  loadHeldOrders,
  saveHeldOrders,
} from '@/lib/posHold';
import type { HeldOrder, HeldOrderStore } from '@/types';

// Vitest runs in node — polyfill a minimal window.localStorage so the module
// (which guards on `typeof window === 'undefined'`) can be exercised.
const backing = new Map<string, string>();
const localStorageMock = {
  getItem: (key: string) => (backing.has(key) ? backing.get(key)! : null),
  setItem: (key: string, value: string) => {
    backing.set(key, String(value));
  },
  removeItem: (key: string) => {
    backing.delete(key);
  },
  clear: () => backing.clear(),
};

function withWindowLocalStorage() {
  (globalThis as Record<string, unknown>).window = {
    localStorage: localStorageMock,
  };
}

function makeOrder(partial: Partial<HeldOrder> = {}): HeldOrder {
  const base: HeldOrder = {
    id: partial.id ?? 'order-1',
    store_id: 'store-a',
    user_id: 'user-1',
    username: 'kasir1',
    label: 'Hold #1',
    items: [
      {
        product_id: 'p1',
        sku_id: 's1',
        name: 'Kopi',
        sku_code: 'KOPI-1',
        unit_price: 15000,
        quantity: 2,
        stock: 40,
        original_stock: 40,
      },
    ],
    discount: 0,
    tax_rate: 11,
    customer_id: '',
    created_at: '2026-09-05T09:30:00.000Z',
    updated_at: '2026-09-05T09:30:00.000Z',
  };
  return { ...base, ...partial };
}

describe('posHold localStorage utils', () => {
  beforeEach(() => {
    backing.clear();
    withWindowLocalStorage();
  });

  it('persists and reloads held orders for a scope', () => {
    const order = makeOrder();
    saveHeldOrders('store-a', 'user-1', [order]);

    const loaded = loadHeldOrders('store-a', 'user-1');
    expect(loaded).toHaveLength(1);
    expect(loaded[0]).toEqual(order);
  });

  it('scopes holds per store + user (R11)', () => {
    saveHeldOrders('store-a', 'user-1', [makeOrder({ id: 'u1-order' })]);
    saveHeldOrders('store-a', 'user-2', [makeOrder({ id: 'u2-order', user_id: 'user-2' })]);
    saveHeldOrders('store-b', 'user-1', [makeOrder({ id: 's2-order', store_id: 'store-b' })]);

    expect(loadHeldOrders('store-a', 'user-1').map((o) => o.id)).toEqual(['u1-order']);
    expect(loadHeldOrders('store-a', 'user-2').map((o) => o.id)).toEqual(['u2-order']);
    expect(loadHeldOrders('store-b', 'user-1').map((o) => o.id)).toEqual(['s2-order']);
    // Saving one scope never wipes another scope.
    saveHeldOrders('store-a', 'user-1', []);
    expect(loadHeldOrders('store-a', 'user-2')).toHaveLength(1);
    expect(loadHeldOrders('store-b', 'user-1')).toHaveLength(1);
  });

  it('returns newest first', () => {
    const older = makeOrder({
      id: 'older',
      created_at: '2026-09-05T08:00:00.000Z',
      updated_at: '2026-09-05T08:00:00.000Z',
    });
    const newer = makeOrder({
      id: 'newer',
      created_at: '2026-09-05T10:00:00.000Z',
      updated_at: '2026-09-05T10:00:00.000Z',
    });
    saveHeldOrders('store-a', 'user-1', [older, newer]);
    expect(loadHeldOrders('store-a', 'user-1').map((o) => o.id)).toEqual(['newer', 'older']);
  });

  it('treats corrupt JSON as an empty store (never throws)', () => {
    backing.set(HELD_ORDERS_STORAGE_KEY, '{not json!!');
    expect(loadHeldOrders('store-a', 'user-1')).toEqual([]);
    // Saving over corrupt data repairs the key.
    saveHeldOrders('store-a', 'user-1', [makeOrder()]);
    expect(loadHeldOrders('store-a', 'user-1')).toHaveLength(1);
  });

  it('discards stores with an unknown version', () => {
    const stale: HeldOrderStore = { version: 0, holds: [makeOrder()] };
    backing.set(HELD_ORDERS_STORAGE_KEY, JSON.stringify(stale));
    expect(loadHeldOrders('store-a', 'user-1')).toEqual([]);
  });

  it('writes the current version marker', () => {
    saveHeldOrders('store-a', 'user-1', [makeOrder()]);
    const raw = backing.get(HELD_ORDERS_STORAGE_KEY)!;
    const parsed = JSON.parse(raw) as HeldOrderStore;
    expect(parsed.version).toBe(HELD_ORDERS_VERSION);
    expect(parsed.holds).toHaveLength(1);
  });

  it('exposes the hold limit and its error type', () => {
    expect(HOLD_LIMIT).toBeGreaterThan(0);
    expect(new HoldLimitError().message).toContain('limit');
    expect(new HoldLimitError()).toBeInstanceOf(Error);
  });
});
