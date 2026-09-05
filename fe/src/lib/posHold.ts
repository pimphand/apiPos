import type { HeldOrder, HeldOrderStore } from '@/types';

/**
 * Browser-only persistence for POS "held orders" (docs/pos-hold-order.md).
 *
 * Design decisions (spec §3.2 / R11):
 * - All holds for every store+cashier of this browser live under ONE key.
 * - Every read/write is scoped to `store_id` + `user_id` so cashiers sharing
 *   a browser never see each other's holds.
 * - Corrupt / unknown-version payloads are treated as empty (never throw).
 */

export const HELD_ORDERS_STORAGE_KEY = 'pos.held_orders.v1';
export const HELD_ORDERS_VERSION = 1;
/** R8 — maximum concurrent holds per store+cashier. */
export const HOLD_LIMIT = 10;

export class HoldLimitError extends Error {
  constructor() {
    super(`Hold limit reached (${HOLD_LIMIT}). Restore or delete one first.`);
    this.name = 'HoldLimitError';
  }
}

/** Read + validate the raw persisted store. Never throws. */
function readStore(): HeldOrderStore | null {
  if (typeof window === 'undefined') return null;
  let raw: string | null = null;
  try {
    raw = window.localStorage.getItem(HELD_ORDERS_STORAGE_KEY);
  } catch {
    return null; // storage unavailable (private mode / quota)
  }
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as HeldOrderStore;
    if (
      !parsed ||
      typeof parsed !== 'object' ||
      parsed.version !== HELD_ORDERS_VERSION ||
      !Array.isArray(parsed.holds)
    ) {
      return null; // unknown format → discard (spec §3.2 versioning)
    }
    return parsed;
  } catch {
    return null; // corrupt JSON → discard
  }
}

function writeStore(store: HeldOrderStore): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(HELD_ORDERS_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // Quota / private mode: silently keep in-memory state; hold is best-effort.
  }
}

function isScopedTo(order: HeldOrder, storeId: string, userId: string): boolean {
  return order.store_id === storeId && order.user_id === userId;
}

/** Load held orders for one store+cashier, newest first. */
export function loadHeldOrders(storeId: string, userId: string): HeldOrder[] {
  const store = readStore();
  if (!store) return [];
  return store.holds
    .filter((h) => isScopedTo(h, storeId, userId))
    .sort((a, b) => b.created_at.localeCompare(a.created_at));
}

/** Persist the full held-order list for one store+cashier (replaces scope). */
export function saveHeldOrders(
  storeId: string,
  userId: string,
  orders: HeldOrder[]
): void {
  const existing = readStore();
  const others = existing
    ? existing.holds.filter((h) => !isScopedTo(h, storeId, userId))
    : [];
  writeStore({ version: HELD_ORDERS_VERSION, holds: [...others, ...orders] });
}
