import { computed, onMounted, onUnmounted, watch } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { maybeAutoSyncOfflineSales } from '@/lib/offlineAutoSync';

/**
 * Automatically replays the offline queue when the connection returns (and
 * once when an authenticated page boots online).
 *
 * Mounted from the sidebar, which exists on every authenticated page, so a
 * cashier who reconnects while on the dashboard still gets their queued sales
 * uploaded. Manual sync on the Sinkron Data page shares the same single-flight
 * runner, so both paths can never double-post.
 */
export function useOfflineAutoSync() {
  const authStore = useAuthStore();
  const ui = useUiStore();
  const t = useLocaleStore().t;

  const storeId = computed(() => authStore.currentUser?.store_id ?? '');
  const userId = computed(() => authStore.currentUser?.id ?? '');

  let attempting = false;

  async function attempt() {
    if (attempting) return;
    attempting = true;
    try {
      const result = await maybeAutoSyncOfflineSales(storeId.value, userId.value);
      if (!result) return;
      if (result.synced.length > 0) {
        ui.addToast(
          'success',
          t('offlineSync.toastAutoSynced', { count: result.synced.length })
        );
      }
      if (result.failed.length > 0) {
        ui.addToast(
          'warning',
          t('offlineSync.toastAutoSyncPartial', { count: result.failed.length })
        );
      }
    } catch {
      // Background sync must never break navigation or the POS.
    } finally {
      attempting = false;
    }
  }

  const handleOnline = () => {
    void attempt();
  };

  watch([storeId, userId], () => {
    void attempt();
  });

  onMounted(() => {
    void attempt();
    window.addEventListener('online', handleOnline);
  });

  onUnmounted(() => {
    window.removeEventListener('online', handleOnline);
  });
}
