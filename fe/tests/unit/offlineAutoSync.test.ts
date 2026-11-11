import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { createMock } = vi.hoisted(() => ({ createMock: vi.fn() }));
vi.mock('@/services/sale', () => ({ saleService: { create: createMock } }));

import {
  enqueueOfflineSale,
  loadOfflineSales,
  type EnqueueOfflineSaleInput,
} from '@/lib/offlineQueue';
import { maybeAutoSyncOfflineSales, syncOfflineSales } from '@/lib/offlineAutoSync';

// Minimal localStorage for the node test environment.
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

function makeInput(createdAt: string): EnqueueOfflineSaleInput {
  return {
    store_id: 'store-a',
    user_id: 'user-1',
    username: 'kasir1',
    payload: {
      store_id: 'store-a',
      items: [{ product_id: 'p1', sku_id: 's1', quantity: 1, unit_price: 15000 }],
      discount: 0,
      tax_rate: 11,
      payment_method: 'CASH',
      amount_paid: 16650,
    },
    items: [{ name: 'Kopi', sku_code: 'KOPI-1', quantity: 1, unit_price: 15000 }],
    totals: { subtotal: 15000, discount: 0, tax: 1650, total: 16650, paid: 16650 },
    customer_name: '',
    payment_method: 'CASH',
    tax_rate: 11,
    created_at: createdAt,
  };
}

describe('offlineAutoSync (shared runner)', () => {
  beforeEach(() => {
    backing.clear();
    (globalThis as Record<string, unknown>).window = { localStorage: localStorageMock };
    createMock.mockReset();
    vi.stubGlobal('navigator', { onLine: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('syncs queued sales, removes them, and returns the server invoices', async () => {
    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    enqueueOfflineSale(makeInput('2026-11-11T09:00:00.000Z'));
    createMock.mockResolvedValue({ sale: { invoice_number: 'INV-1' } });

    const result = await syncOfflineSales('store-a', 'user-1');

    expect(result.synced).toHaveLength(2);
    expect(result.failed).toHaveLength(0);
    expect(result.offline).toBe(false);
    expect(createMock).toHaveBeenCalledTimes(2);
    expect(loadOfflineSales('store-a', 'user-1')).toHaveLength(0);
  });

  it('keeps a rejected sale with its server message (manual retry)', async () => {
    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    createMock.mockRejectedValueOnce({
      code: 'INVALID_ARGUMENT',
      message: 'Insufficient stock for product',
    });

    const result = await syncOfflineSales('store-a', 'user-1');

    expect(result.synced).toHaveLength(0);
    expect(result.failed[0].message).toBe('Insufficient stock for product');
    const queued = loadOfflineSales('store-a', 'user-1');
    expect(queued).toHaveLength(1);
    expect(queued[0].status).toBe('failed');
    expect(queued[0].error).toBe('Insufficient stock for product');
  });

  it('stops on a transport failure and leaves later entries pending', async () => {
    enqueueOfflineSale(makeInput('2026-11-11T09:00:00.000Z'));
    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    createMock.mockRejectedValueOnce({ code: 'UNAVAILABLE', message: 'Network unavailable' });

    const result = await syncOfflineSales('store-a', 'user-1');

    expect(result.offline).toBe(true);
    expect(result.failed).toHaveLength(1);
    expect(createMock).toHaveBeenCalledTimes(1);
    expect(loadOfflineSales('store-a', 'user-1')).toHaveLength(2);
  });

  it('runs one replay at a time (single-flight, no double POST)', async () => {
    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    let resolveCreate!: (value: unknown) => void;
    createMock.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveCreate = resolve;
      })
    );

    const first = syncOfflineSales('store-a', 'user-1');
    const second = syncOfflineSales('store-a', 'user-1');
    expect(createMock).toHaveBeenCalledTimes(1);

    resolveCreate({ sale: { invoice_number: 'INV-1' } });
    const [a, b] = await Promise.all([first, second]);
    expect(a.synced).toHaveLength(1);
    expect(b.synced).toHaveLength(1);
    expect(loadOfflineSales('store-a', 'user-1')).toHaveLength(0);
  });

  it('auto-sync does nothing for an empty queue or an offline device', async () => {
    expect(await maybeAutoSyncOfflineSales('store-a', 'user-1')).toBeNull();
    expect(createMock).not.toHaveBeenCalled();

    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    vi.stubGlobal('navigator', { onLine: false });
    expect(await maybeAutoSyncOfflineSales('store-a', 'user-1')).toBeNull();
    expect(createMock).not.toHaveBeenCalled();
  });

  it('does not auto-retry sales the server already rejected', async () => {
    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    createMock.mockRejectedValueOnce({
      code: 'INVALID_ARGUMENT',
      message: 'Insufficient stock',
    });
    await syncOfflineSales('store-a', 'user-1');
    createMock.mockClear();

    expect(await maybeAutoSyncOfflineSales('store-a', 'user-1')).toBeNull();
    expect(createMock).not.toHaveBeenCalled();
  });

  it('auto-syncs pending sales once the connection is back', async () => {
    enqueueOfflineSale(makeInput('2026-11-11T08:00:00.000Z'));
    createMock.mockResolvedValue({ sale: { invoice_number: 'INV-2' } });

    const result = await maybeAutoSyncOfflineSales('store-a', 'user-1');

    expect(result?.synced).toHaveLength(1);
    expect(createMock).toHaveBeenCalledTimes(1);
    expect(loadOfflineSales('store-a', 'user-1')).toHaveLength(0);
  });
});
