import type { OfflineCacheStore, OfflineSnapshot } from '@/types';

/**
 * Last-known catalogue/settings per store, so the POS can still be used while
 * the gateway is unreachable (docs/pos-offline-sync.md).
 *
 * Only data the POS already received is cached. Nothing here is invented or
 * seeded: an empty cache means "no offline data yet", and the UI says so.
 */

export const OFFLINE_CACHE_STORAGE_KEY = 'pos.offline_cache.v1';
export const OFFLINE_CACHE_VERSION = 1;
/** Keep the snapshot bounded; the POS requests at most 100 products per page. */
export const OFFLINE_CACHE_PRODUCT_LIMIT = 300;

function readStore(): OfflineCacheStore | null {
  if (typeof window === 'undefined') return null;
  let raw: string | null = null;
  try {
    raw = window.localStorage.getItem(OFFLINE_CACHE_STORAGE_KEY);
  } catch {
    return null;
  }
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as OfflineCacheStore;
    if (
      !parsed ||
      typeof parsed !== 'object' ||
      parsed.version !== OFFLINE_CACHE_VERSION ||
      !parsed.stores ||
      typeof parsed.stores !== 'object'
    ) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function writeStore(store: OfflineCacheStore): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(OFFLINE_CACHE_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // Quota / private mode: offline data is best-effort, never fatal.
  }
}

/** Merge fresh data into the store's snapshot (partial updates are fine). */
export function saveOfflineSnapshot(
  storeId: string,
  patch: Partial<Omit<OfflineSnapshot, 'updated_at'>>
): void {
  if (!storeId) return;
  const store = readStore() ?? { version: OFFLINE_CACHE_VERSION, stores: {} };
  const previous = store.stores[storeId];
  const next: OfflineSnapshot = {
    ...previous,
    ...patch,
    updated_at: new Date().toISOString(),
  };
  if (next.products && next.products.length > OFFLINE_CACHE_PRODUCT_LIMIT) {
    next.products = next.products.slice(0, OFFLINE_CACHE_PRODUCT_LIMIT);
  }
  store.stores[storeId] = next;
  writeStore(store);
}

/** Snapshot for one store, or null when nothing was cached yet. */
export function loadOfflineSnapshot(storeId: string): OfflineSnapshot | null {
  if (!storeId) return null;
  const store = readStore();
  return store?.stores?.[storeId] ?? null;
}

/** Drop the snapshot for one store (used by tests/maintenance actions). */
export function clearOfflineSnapshot(storeId: string): void {
  const store = readStore();
  if (!store || !store.stores[storeId]) return;
  delete store.stores[storeId];
  writeStore(store);
}
