import { saleService } from '@/services/sale';
import { apiMessage } from '@/lib/apiError';
import {
  isOfflineError,
  loadOfflineSales,
  markOfflineSaleFailed,
  markOfflineSaleSyncing,
  removeOfflineSale,
} from '@/lib/offlineQueue';

/**
 * Shared offline-queue replay used by BOTH the automatic reconnect sync and
 * the manual "Sinkronkan Semua"/retry buttons (docs/pos-offline-sync.md).
 *
 * Single-flight: at most one replay runs at a time, so a reconnect firing
 * while the cashier clicks Sync All cannot post the same sale twice. Success
 * removes the entry; failure keeps it with the server message.
 */
export interface OfflineSyncResult {
  synced: { id: string; invoice: string }[];
  failed: { id: string; message: string }[];
  /** True when the run stopped because the gateway became unreachable. */
  offline: boolean;
}

let activeRun: Promise<OfflineSyncResult> | null = null;
let lastAutoAttempt = 0;
/** Avoid a burst of automatic retries when the online event flaps. */
const AUTO_RETRY_WINDOW_MS = 3000;

export function isOfflineSyncRunning(): boolean {
  return activeRun !== null;
}

/** Replay queued sales (optionally only the given ids). Never throws. */
export function syncOfflineSales(
  storeId: string,
  userId: string,
  opts: { ids?: string[] } = {}
): Promise<OfflineSyncResult> {
  if (activeRun) return activeRun;

  const run = (async (): Promise<OfflineSyncResult> => {
    const result: OfflineSyncResult = { synced: [], failed: [], offline: false };
    const queue = loadOfflineSales(storeId, userId).filter(
      (entry) => !opts.ids || opts.ids.includes(entry.id)
    );

    for (const entry of queue) {
      markOfflineSaleSyncing(storeId, userId, entry.id);
      try {
        const detail = await saleService.create(entry.payload);
        removeOfflineSale(storeId, userId, entry.id);
        result.synced.push({
          id: entry.id,
          invoice: detail.sale.invoice_number || '',
        });
      } catch (err) {
        const message = apiMessage(err);
        markOfflineSaleFailed(storeId, userId, entry.id, message);
        result.failed.push({ id: entry.id, message });
        if (isOfflineError(err)) {
          result.offline = true;
          break;
        }
      }
    }

    return result;
  })();

  activeRun = run.finally(() => {
    activeRun = null;
  });
  return activeRun;
}

/**
 * Reconnect/boot hook: syncs only when the device is online, the scope has a
 * non-empty queue, and the last automatic attempt was not just a moment ago.
 * Returns null when there was nothing to do.
 */
export async function maybeAutoSyncOfflineSales(
  storeId: string,
  userId: string
): Promise<OfflineSyncResult | null> {
  if (!storeId || !userId) return null;
  if (typeof navigator !== 'undefined' && navigator.onLine === false) return null;

  const queue = loadOfflineSales(storeId, userId);
  // Failed entries (rejected by the server, e.g. stock/shift) are left for the
  // cashier to retry manually; auto-sync only uploads untouched sales.
  const pendingIds = queue.filter((entry) => entry.status !== 'failed').map((entry) => entry.id);
  if (pendingIds.length === 0) return null;

  const now = Date.now();
  if (now - lastAutoAttempt < AUTO_RETRY_WINDOW_MS) return null;
  lastAutoAttempt = now;

  return syncOfflineSales(storeId, userId, { ids: pendingIds });
}
