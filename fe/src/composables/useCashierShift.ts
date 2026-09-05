import { computed, ref } from 'vue';
import { cashierService } from '@/services/cashier';
import { useAuthStore } from '@/stores/auth';
import { apiMessage } from '@/lib/apiError';
import type { CashierShift, ShiftSummary } from '@/types';

/**
 * Shared cashier-shift logic used by /pos (POSCheckoutPage) and (later)
 * /admin/cashier so both pages behave identically:
 *
 * - one "current open shift" source of truth,
 * - idempotent recovery for "already open" / "already closed" responses,
 * - reconciliation numbers always come from the server summary.
 *
 * The backend is the source of truth: local state is only mutated after a
 * successful server response.
 */
export function useCashierShift() {
  const authStore = useAuthStore();

  const currentShift = ref<CashierShift | null>(null);
  /** True while the initial "does a shift exist?" check is running. */
  const shiftLoading = ref(false);
  const opening = ref(false);
  const closing = ref(false);
  const summary = ref<ShiftSummary | null>(null);
  const summaryLoading = ref(false);

  const isOpen = computed(() => currentShift.value?.status === 'OPEN');

  function isIdempotentClose(err: unknown): boolean {
    const msg = apiMessage(err).toLowerCase();
    return (
      msg.includes('already closed') ||
      msg.includes('not found') ||
      msg.includes('no open shift')
    );
  }

  /** Reload the current open shift from the server. 404 → null (no shift). */
  async function refreshShift(): Promise<CashierShift | null> {
    try {
      const shift = await cashierService.getCurrentShift();
      currentShift.value = shift && shift.status === 'OPEN' ? shift : null;
    } catch {
      currentShift.value = null;
    }
    return currentShift.value;
  }

  /** Load a shift's full summary (reconciliation + live numbers). */
  async function loadSummary(shiftId?: string): Promise<ShiftSummary | null> {
    const id = shiftId ?? currentShift.value?.id;
    if (!id) return null;
    summaryLoading.value = true;
    try {
      summary.value = await cashierService.getShiftSummary(id);
      return summary.value;
    } finally {
      summaryLoading.value = false;
    }
  }

  /** Open a shift. Returns the opened (or already-open) shift. */
  async function openShift(openingCash: number): Promise<CashierShift> {
    const storeId = authStore.currentUser?.store_id;
    if (!storeId) throw new Error('No store assigned');

    opening.value = true;
    try {
      const shift = await cashierService.openShift({
        store_id: storeId,
        opening_cash: Number(openingCash) || 0,
      });
      currentShift.value = shift;
      return shift;
    } catch (err) {
      // Recovery: the shift may already exist (opened from another tab or a
      // racing double-submit). Re-sync with the server instead of failing.
      const existing = await refreshShift();
      if (existing) return existing;
      throw err;
    } finally {
      opening.value = false;
    }
  }

  /**
   * Close a shift. Returns the final (CLOSED) summary. Double-close and
   * not-found responses are treated as idempotent success: the shift is
   * already closed, so we load its summary and return it.
   */
  async function closeShift(
    shiftId: string,
    actualCash: number,
    closingCash?: number
  ): Promise<ShiftSummary> {
    closing.value = true;
    try {
      const closed = await cashierService.closeShift({
        shift_id: shiftId,
        closing_cash: closingCash ?? actualCash,
        actual_cash: Number(actualCash) || 0,
      });
      currentShift.value = null;
      const finalSummary = await loadSummary(closed.id);
      return finalSummary ?? (await loadSummary(shiftId))!;
    } catch (err) {
      if (isIdempotentClose(err)) {
        currentShift.value = null;
        const finalSummary = await loadSummary(shiftId);
        if (finalSummary) return finalSummary;
      }
      throw err;
    } finally {
      closing.value = false;
    }
  }

  /** Server-side truth check right before checkout (see spec §4.3). */
  async function ensureShiftOpen(): Promise<boolean> {
    if (currentShift.value?.status === 'OPEN') return true;
    const shift = await refreshShift();
    return shift?.status === 'OPEN';
  }

  return {
    currentShift,
    shiftLoading,
    opening,
    closing,
    summary,
    summaryLoading,
    isOpen,
    refreshShift,
    loadSummary,
    openShift,
    closeShift,
    ensureShiftOpen,
  };
}
