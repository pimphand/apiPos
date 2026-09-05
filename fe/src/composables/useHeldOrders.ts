import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { generateUUID } from '@/lib/utils';
import {
  HELD_ORDERS_STORAGE_KEY,
  HOLD_LIMIT,
  HoldLimitError,
  loadHeldOrders,
  saveHeldOrders,
} from '@/lib/posHold';
import type { HeldOrder, HeldOrderItem } from '@/types';

/**
 * Shared "held order" state for the POS checkout (docs/pos-hold-order.md).
 *
 * Scope (R11): holds are per `store_id` + `user_id` of the signed-in user and
 * live in `localStorage` (no backend). The list is kept reactive in memory and
 * re-synced from storage on every cross-tab `storage` event so two POS tabs
 * show the same holds.
 */
export interface HoldCartSnapshot {
  label: string;
  items: HeldOrderItem[];
  discount: number;
  tax_rate: number;
  customer_id: string;
  customer_name?: string;
}

export function useHeldOrders() {
  const authStore = useAuthStore();
  const holds = ref<HeldOrder[]>([]);

  const holdsCount = computed(() => holds.value.length);
  const isAtLimit = computed(() => holds.value.length >= HOLD_LIMIT);

  /** Current scope from the auth store. Null when not signed in / no store. */
  function scope(): { storeId: string; userId: string; username: string } | null {
    const user = authStore.currentUser;
    if (!user?.store_id || !user?.id) return null;
    return { storeId: user.store_id, userId: user.id, username: user.username };
  }

  /** Re-read this cashier's holds from localStorage (newest first). */
  function reload(): void {
    const s = scope();
    holds.value = s ? loadHeldOrders(s.storeId, s.userId) : [];
  }

  /** Hold the current cart. Throws when the hold limit is reached (R8). */
  function holdCart(snapshot: HoldCartSnapshot): HeldOrder | null {
    const s = scope();
    if (!s) return null;
    if (snapshot.items.length === 0) {
      throw new Error('Nothing to hold');
    }
    if (holds.value.length >= HOLD_LIMIT) {
      throw new HoldLimitError();
    }
    const now = new Date().toISOString();
    const order: HeldOrder = {
      id: generateUUID(),
      store_id: s.storeId,
      user_id: s.userId,
      username: s.username,
      label: snapshot.label,
      items: snapshot.items.map((i) => ({ ...i })),
      discount: Number(snapshot.discount) || 0,
      tax_rate: Number(snapshot.tax_rate) || 0,
      customer_id: snapshot.customer_id,
      customer_name: snapshot.customer_name,
      created_at: now,
      updated_at: now,
    };
    const sScope = scope();
    if (!sScope) return null;
    const next = [order, ...holds.value];
    saveHeldOrders(sScope.storeId, sScope.userId, next);
    holds.value = next;
    return order;
  }

  /** Restore a held order (returns it and removes it from the list). */
  function restoreHold(id: string): HeldOrder | null {
    const s = scope();
    if (!s) return null;
    const order = holds.value.find((h) => h.id === id);
    if (!order) return null;
    const next = holds.value.filter((h) => h.id !== id);
    saveHeldOrders(s.storeId, s.userId, next);
    holds.value = next;
    return { ...order, items: order.items.map((i) => ({ ...i })) };
  }

  /** Delete a held order permanently (R9 — requires UI confirmation). */
  function deleteHold(id: string): boolean {
    const s = scope();
    if (!s) return false;
    const existing = holds.value.find((h) => h.id === id);
    if (!existing) return false;
    const next = holds.value.filter((h) => h.id !== id);
    saveHeldOrders(s.storeId, s.userId, next);
    holds.value = next;
    return true;
  }

  function onStorage(e: StorageEvent) {
    if (e.key === HELD_ORDERS_STORAGE_KEY) reload();
  }

  onMounted(() => {
    reload();
    window.addEventListener('storage', onStorage);
  });
  onUnmounted(() => {
    window.removeEventListener('storage', onStorage);
  });

  return {
    holds,
    holdsCount,
    isAtLimit,
    reload,
    holdCart,
    restoreHold,
    deleteHold,
  };
}
