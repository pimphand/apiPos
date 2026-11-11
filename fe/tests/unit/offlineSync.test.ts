import { describe, it, expect, beforeEach } from 'vitest';
import {
  OFFLINE_QUEUE_LIMIT,
  OFFLINE_QUEUE_STORAGE_KEY,
  OFFLINE_QUEUE_VERSION,
  OfflineQueueLimitError,
  enqueueOfflineSale,
  isOfflineError,
  loadOfflineSales,
  markOfflineSaleFailed,
  markOfflineSaleSyncing,
  removeOfflineSale,
  subscribeOfflineQueue,
  type EnqueueOfflineSaleInput,
} from '@/lib/offlineQueue';
import {
  OFFLINE_CACHE_STORAGE_KEY,
  OFFLINE_CACHE_VERSION,
  clearOfflineSnapshot,
  loadOfflineSnapshot,
  saveOfflineSnapshot,
} from '@/lib/offlineCache';
import type { CreateSaleRequest, OfflineQueueStore, Product } from '@/types';

// Vitest runs in node — polyfill a minimal window.localStorage, mirroring the
// posHold tests, so the storage guards are exercised for real.
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

function makePayload(overrides: Partial<CreateSaleRequest> = {}): CreateSaleRequest {
  return {
    store_id: 'store-a',
    customer_name: 'Budi',
    items: [{ product_id: 'p1', sku_id: 's1', quantity: 2, unit_price: 15000 }],
    discount: 0,
    tax_rate: 11,
    payment_method: 'CASH',
    amount_paid: 40000,
    ...overrides,
  };
}

function makeInput(
  overrides: Partial<EnqueueOfflineSaleInput> = {}
): EnqueueOfflineSaleInput {
  return {
    store_id: 'store-a',
    user_id: 'user-1',
    username: 'kasir1',
    payload: makePayload(),
    items: [{ name: 'Kopi', sku_code: 'KOPI-1', quantity: 2, unit_price: 15000 }],
    totals: { subtotal: 30000, discount: 0, tax: 3300, total: 33300, paid: 40000 },
    customer_name: 'Budi',
    payment_method: 'CASH',
    tax_rate: 11,
    ...overrides,
  };
}

describe('offlineQueue', () => {
  beforeEach(() => {
    backing.clear();
    withWindowLocalStorage();
  });

  it('queues a sale as pending with the exact POST /v1/sales payload', () => {
    const record = enqueueOfflineSale(makeInput());
    expect(record.status).toBe('pending');
    expect(record.attempts).toBe(0);

    const loaded = loadOfflineSales('store-a', 'user-1');
    expect(loaded).toHaveLength(1);
    expect(loaded[0].payload).toEqual(makePayload());
    expect(loaded[0].totals.total).toBe(33300);
  });

  it('scopes the queue per store + user', () => {
    enqueueOfflineSale(makeInput({ created_at: '2026-11-11T08:00:00.000Z' }));
    enqueueOfflineSale(
      makeInput({
        user_id: 'user-2',
        username: 'kasir2',
        created_at: '2026-11-11T09:00:00.000Z',
      })
    );
    enqueueOfflineSale(
      makeInput({
        store_id: 'store-b',
        created_at: '2026-11-11T10:00:00.000Z',
      })
    );

    expect(loadOfflineSales('store-a', 'user-1')).toHaveLength(1);
    expect(loadOfflineSales('store-a', 'user-2')).toHaveLength(1);
    expect(loadOfflineSales('store-b', 'user-1')).toHaveLength(1);

    // Removing one scope never touches another.
    const own = loadOfflineSales('store-a', 'user-1')[0];
    expect(removeOfflineSale('store-a', 'user-1', own.id)).toBe(true);
    expect(loadOfflineSales('store-a', 'user-2')).toHaveLength(1);
    expect(loadOfflineSales('store-b', 'user-1')).toHaveLength(1);
  });

  it('returns newest first', () => {
    enqueueOfflineSale(makeInput({ created_at: '2026-11-11T08:00:00.000Z' }));
    enqueueOfflineSale(makeInput({ created_at: '2026-11-11T10:00:00.000Z' }));
    enqueueOfflineSale(makeInput({ created_at: '2026-11-11T09:00:00.000Z' }));
    const times = loadOfflineSales('store-a', 'user-1').map((e) => e.created_at);
    expect(times).toEqual([
      '2026-11-11T10:00:00.000Z',
      '2026-11-11T09:00:00.000Z',
      '2026-11-11T08:00:00.000Z',
    ]);
  });

  it('enforces the per-scope queue limit', () => {
    for (let i = 0; i < OFFLINE_QUEUE_LIMIT; i++) enqueueOfflineSale(makeInput());
    expect(() => enqueueOfflineSale(makeInput())).toThrow(OfflineQueueLimitError);
    // Another cashier still has room.
    expect(() => enqueueOfflineSale(makeInput({ user_id: 'user-2' }))).not.toThrow();
  });

  it('tracks sync attempts and server errors', () => {
    const record = enqueueOfflineSale(makeInput());
    const syncing = markOfflineSaleSyncing('store-a', 'user-1', record.id);
    expect(syncing?.status).toBe('syncing');
    expect(syncing?.attempts).toBe(1);

    const failed = markOfflineSaleFailed('store-a', 'user-1', record.id, 'Insufficient stock');
    expect(failed?.status).toBe('failed');
    expect(failed?.error).toBe('Insufficient stock');
    expect(failed?.attempts).toBe(1);

    // A retry clears the previous error and counts the new attempt.
    const retried = markOfflineSaleSyncing('store-a', 'user-1', record.id);
    expect(retried?.error).toBeUndefined();
    expect(retried?.attempts).toBe(2);
  });

  it('ignores other users when patching', () => {
    const record = enqueueOfflineSale(makeInput());
    expect(markOfflineSaleSyncing('store-a', 'user-2', record.id)).toBeNull();
    expect(loadOfflineSales('store-a', 'user-1')[0].status).toBe('pending');
  });

  it('treats corrupt JSON / unknown versions as an empty queue', () => {
    backing.set(OFFLINE_QUEUE_STORAGE_KEY, '{not json!!');
    expect(loadOfflineSales('store-a', 'user-1')).toEqual([]);

    const stale = { version: 0, entries: [] } as unknown as OfflineQueueStore;
    backing.set(OFFLINE_QUEUE_STORAGE_KEY, JSON.stringify(stale));
    expect(loadOfflineSales('store-a', 'user-1')).toEqual([]);

    enqueueOfflineSale(makeInput());
    const stored = JSON.parse(backing.get(OFFLINE_QUEUE_STORAGE_KEY)!) as OfflineQueueStore;
    expect(stored.version).toBe(OFFLINE_QUEUE_VERSION);
  });

  it('notifies subscribers after every queue mutation', () => {
    const counts: number[] = [];
    const unsubscribe = subscribeOfflineQueue(() => {
      counts.push(loadOfflineSales('store-a', 'user-1').length);
    });

    const record = enqueueOfflineSale(makeInput());
    markOfflineSaleSyncing('store-a', 'user-1', record.id);
    removeOfflineSale('store-a', 'user-1', record.id);
    unsubscribe();
    enqueueOfflineSale(makeInput());

    // enqueue → 1, status change → 1, remove → 0, and no event after unsubscribe.
    expect(counts).toEqual([1, 1, 0]);
  });

  it('classifies only transport failures as offline errors', () => {
    expect(isOfflineError({ code: 'UNAVAILABLE' })).toBe(true);
    expect(isOfflineError({ code: 'INVALID_ARGUMENT' })).toBe(false);
    expect(isOfflineError(new Error('x'))).toBe(false);
    expect(isOfflineError(null)).toBe(false);
  });
});

describe('offlineCache', () => {
  beforeEach(() => {
    backing.clear();
    withWindowLocalStorage();
  });

  it('merges partial snapshots and stamps updated_at', () => {
    const product = { id: 'p1', name: 'Kopi' } as Product;
    saveOfflineSnapshot('store-a', { products: [product] });
    const first = loadOfflineSnapshot('store-a')!;
    expect(first.products).toHaveLength(1);
    expect(first.updated_at).toBeTruthy();

    saveOfflineSnapshot('store-a', {
      storeInfo: { name: 'Toko A', address: 'Jl. A', phone: '' },
    });
    const merged = loadOfflineSnapshot('store-a')!;
    expect(merged.products).toHaveLength(1);
    expect(merged.storeInfo?.name).toBe('Toko A');
  });

  it('keeps stores isolated and clears one store only', () => {
    saveOfflineSnapshot('store-a', { products: [{ id: 'p1', name: 'Kopi' } as Product] });
    saveOfflineSnapshot('store-b', { products: [{ id: 'p2', name: 'Teh' } as Product] });

    clearOfflineSnapshot('store-a');
    expect(loadOfflineSnapshot('store-a')).toBeNull();
    expect(loadOfflineSnapshot('store-b')?.products?.[0].id).toBe('p2');
  });

  it('treats corrupt JSON / unknown versions as empty', () => {
    backing.set(OFFLINE_CACHE_STORAGE_KEY, '{broken');
    expect(loadOfflineSnapshot('store-a')).toBeNull();

    backing.set(
      OFFLINE_CACHE_STORAGE_KEY,
      JSON.stringify({ version: OFFLINE_CACHE_VERSION + 1, stores: {} })
    );
    expect(loadOfflineSnapshot('store-a')).toBeNull();
  });
});
