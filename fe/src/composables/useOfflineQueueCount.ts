import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { loadOfflineSales, subscribeOfflineQueue } from '@/lib/offlineQueue';
import { setTrayOfflineCount } from '@/lib/platform';

/**
 * Number of unsynced offline sales for the current store + cashier.
 *
 * One module-level ref is shared by every component on the page (sidebar,
 * POS, sync page), and `subscribeOfflineQueue` refreshes it whenever the queue
 * is written, so a badge appears the moment an offline sale is queued and
 * disappears right after a successful sync.
 */
const count = ref(0);

export function useOfflineQueueCount() {
  const authStore = useAuthStore();
  const storeId = computed(() => authStore.currentUser?.store_id ?? '');
  const userId = computed(() => authStore.currentUser?.id ?? '');

  function refresh() {
    if (!storeId.value || !userId.value) {
      count.value = 0;
      return;
    }
    count.value = loadOfflineSales(storeId.value, userId.value).length;
  }

  watch([storeId, userId], refresh, { immediate: true });
  // Desktop shell: paint the same number on the tray icon/tooltip. The helper
  // is a no-op outside Tauri and dedupes repeated counts.
  watch(
    count,
    (value) => {
      void setTrayOfflineCount(value);
    },
    { immediate: true }
  );

  let unsubscribe: (() => void) | null = null;
  onMounted(() => {
    refresh();
    unsubscribe = subscribeOfflineQueue(refresh);
    // Another tab may have queued/synced sales; re-read on focus-like events.
    window.addEventListener('storage', refresh);
    window.addEventListener('online', refresh);
  });

  onUnmounted(() => {
    unsubscribe?.();
    unsubscribe = null;
    window.removeEventListener('storage', refresh);
    window.removeEventListener('online', refresh);
  });

  return { count: computed(() => count.value), refresh };
}
