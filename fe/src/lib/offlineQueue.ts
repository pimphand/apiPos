import type {
  CreateSaleRequest,
  OfflineQueueStore,
  OfflineSaleItem,
  OfflineSaleRecord,
  OfflineSaleTotals,
} from '@/types';

/**
 * Durable queue for checkout transactions created while the gateway is
 * unreachable (docs/pos-offline-sync.md).
 *
 * Design:
 * - One versioned localStorage key holds every store/user scope, like
 *   `posHold.ts`; reads/writes are scoped so cashiers never see each other's
 *   pending sales.
 * - The exact `POST /v1/sales` body is stored, so syncing replays the existing
 *   endpoint — there is no new backend contract and no mock.
 * - Corrupt or unknown-version payloads are treated as empty; nothing throws.
 */

export const OFFLINE_QUEUE_STORAGE_KEY = 'pos.offline_queue.v1';
export const OFFLINE_QUEUE_VERSION = 1;
/** Upper bound so a device that stays offline for weeks cannot exhaust storage. */
export const OFFLINE_QUEUE_LIMIT = 200;

export class OfflineQueueLimitError extends Error {
  constructor() {
    super(`Offline queue limit reached (${OFFLINE_QUEUE_LIMIT}).`);
    this.name = 'OfflineQueueLimitError';
  }
}

/** Listeners notified after every persisted queue mutation (badge counts). */
const listeners = new Set<() => void>();

/**
 * Subscribe to queue changes (enqueue, status change, removal). Returns an
 * unsubscribe function. Used by `useOfflineQueueCount` so the sidebar/POS
 * badges update immediately after an offline checkout or a sync.
 */
export function subscribeOfflineQueue(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function notifyListeners(): void {
  for (const listener of [...listeners]) {
    try {
      listener();
    } catch {
      // A badge listener must never break the queue write.
    }
  }
}

function readStore(): OfflineQueueStore | null {
  if (typeof window === 'undefined') return null;
  let raw: string | null = null;
  try {
    raw = window.localStorage.getItem(OFFLINE_QUEUE_STORAGE_KEY);
  } catch {
    return null;
  }
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as OfflineQueueStore;
    if (
      !parsed ||
      typeof parsed !== 'object' ||
      parsed.version !== OFFLINE_QUEUE_VERSION ||
      !Array.isArray(parsed.entries)
    ) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function writeStore(store: OfflineQueueStore): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(OFFLINE_QUEUE_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // Quota / private mode: the caller cannot queue further, but nothing throws.
  }
  notifyListeners();
}

function isScopedTo(entry: OfflineSaleRecord, storeId: string, userId: string): boolean {
  return entry.store_id === storeId && entry.user_id === userId;
}

function newId(): string {
  const cryptoApi = typeof globalThis !== 'undefined' ? globalThis.crypto : undefined;
  if (cryptoApi && typeof cryptoApi.randomUUID === 'function') return cryptoApi.randomUUID();
  return `offline-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

/** Pending/failed sales for one store+cashier, newest first. */
export function loadOfflineSales(storeId: string, userId: string): OfflineSaleRecord[] {
  const store = readStore();
  if (!store) return [];
  return store.entries
    .filter((entry) => isScopedTo(entry, storeId, userId))
    .sort((a, b) => b.created_at.localeCompare(a.created_at));
}

export interface EnqueueOfflineSaleInput {
  store_id: string;
  user_id: string;
  username: string;
  payload: CreateSaleRequest;
  items: OfflineSaleItem[];
  totals: OfflineSaleTotals;
  customer_name: string;
  payment_method: string;
  tax_rate: number;
  /** Override for tests; defaults to now. */
  created_at?: string;
}

/** Persist a checkout that could not be sent. Throws only on the queue limit. */
export function enqueueOfflineSale(input: EnqueueOfflineSaleInput): OfflineSaleRecord {
  const store = readStore() ?? { version: OFFLINE_QUEUE_VERSION, entries: [] };
  const scopedCount = store.entries.filter((entry) =>
    isScopedTo(entry, input.store_id, input.user_id)
  ).length;
  if (scopedCount >= OFFLINE_QUEUE_LIMIT) throw new OfflineQueueLimitError();

  const record: OfflineSaleRecord = {
    id: newId(),
    store_id: input.store_id,
    user_id: input.user_id,
    username: input.username,
    created_at: input.created_at ?? new Date().toISOString(),
    payload: input.payload,
    items: input.items,
    totals: input.totals,
    customer_name: input.customer_name,
    payment_method: input.payment_method,
    tax_rate: input.tax_rate,
    status: 'pending',
    attempts: 0,
  };

  writeStore({ version: OFFLINE_QUEUE_VERSION, entries: [...store.entries, record] });
  return record;
}

/** Replace one record in place; returns the updated record (or null). */
function patchRecord(
  storeId: string,
  userId: string,
  id: string,
  patch: Partial<OfflineSaleRecord>
): OfflineSaleRecord | null {
  const store = readStore();
  if (!store) return null;
  let updated: OfflineSaleRecord | null = null;
  const entries = store.entries.map((entry) => {
    if (!isScopedTo(entry, storeId, userId) || entry.id !== id) return entry;
    updated = { ...entry, ...patch };
    return updated;
  });
  if (!updated) return null;
  writeStore({ version: OFFLINE_QUEUE_VERSION, entries });
  return updated;
}

/** Mark a record as being sent right now and count the attempt. */
export function markOfflineSaleSyncing(
  storeId: string,
  userId: string,
  id: string
): OfflineSaleRecord | null {
  const current = loadOfflineSales(storeId, userId).find((entry) => entry.id === id);
  if (!current) return null;
  return patchRecord(storeId, userId, id, {
    status: 'syncing',
    error: undefined,
    attempts: current.attempts + 1,
    last_attempt_at: new Date().toISOString(),
  });
}

/** Keep the record with the server error so the cashier can act on it. */
export function markOfflineSaleFailed(
  storeId: string,
  userId: string,
  id: string,
  error: string
): OfflineSaleRecord | null {
  return patchRecord(storeId, userId, id, {
    status: 'failed',
    error: error || 'Sync failed',
    last_attempt_at: new Date().toISOString(),
  });
}

/** Remove a record (synced successfully, or discarded by the cashier). */
export function removeOfflineSale(storeId: string, userId: string, id: string): boolean {
  const store = readStore();
  if (!store) return false;
  const entries = store.entries.filter(
    (entry) => !(isScopedTo(entry, storeId, userId) && entry.id === id)
  );
  if (entries.length === store.entries.length) return false;
  writeStore({ version: OFFLINE_QUEUE_VERSION, entries });
  return true;
}

/** GET/POST failures with no HTTP response are tagged `UNAVAILABLE` in http.ts. */
export function isOfflineError(err: unknown): boolean {
  return (
    !!err &&
    typeof err === 'object' &&
    (err as { code?: unknown }).code === 'UNAVAILABLE'
  );
}

/** Browser connectivity hint; defaults to online when unknown. */
export function isOnline(): boolean {
  if (typeof navigator === 'undefined') return true;
  return navigator.onLine !== false;
}
