<script setup lang="ts">
/**
 * Sinkronisasi transaksi offline (docs/pos-offline-sync.md).
 *
 * Shows the durable queue written by the POS while the gateway was
 * unreachable, replays each entry through the existing POST /v1/sales, and
 * keeps failures with their server error so a cashier can fix the cause
 * (stock, shift, permission) and retry. No new endpoint, no mock data.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import {
  isOnline as hasNetwork,
  loadOfflineSales,
  removeOfflineSale,
} from '@/lib/offlineQueue';
import { syncOfflineSales } from '@/lib/offlineAutoSync';
import { loadOfflineSnapshot } from '@/lib/offlineCache';
import type { OfflineSaleRecord } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import { Card, CardContent } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';

const { isReady, allowed, authStore } = usePageGuard('sales.create');
const ui = useUiStore();
const t = useLocaleStore().t;

const storeId = computed(() => authStore.currentUser?.store_id ?? '');
const userId = computed(() => authStore.currentUser?.id ?? '');

const entries = ref<OfflineSaleRecord[]>([]);
const isOnline = ref(hasNetwork());
const syncingAll = ref(false);
const busyId = ref<string | null>(null);
const cacheUpdatedAt = ref<string | null>(null);

const pendingCount = computed(
  () => entries.value.filter((e) => e.status !== 'failed').length
);
const failedCount = computed(
  () => entries.value.filter((e) => e.status === 'failed').length
);

const PAY_METHOD_KEYS: Record<string, string> = {
  CASH: 'cash',
  CARD: 'card',
  BANK_TRANSFER: 'bankTransfer',
  EWALLET: 'ewallet',
  OTHER: 'other',
};

function paymentLabel(method: string): string {
  return t(`admin.payments.${PAY_METHOD_KEYS[method] ?? 'other'}`);
}

function offlineRef(record: OfflineSaleRecord): string {
  return `OFFLINE-${record.id.slice(0, 8).toUpperCase()}`;
}

function statusLabel(status: OfflineSaleRecord['status']): string {
  if (status === 'failed') return t('offlineSync.statusFailed');
  if (status === 'syncing') return t('offlineSync.statusSyncing');
  return t('offlineSync.statusPending');
}

function statusVariant(
  status: OfflineSaleRecord['status']
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'failed') return 'destructive';
  if (status === 'syncing') return 'outline';
  return 'secondary';
}

function reload() {
  entries.value = loadOfflineSales(storeId.value, userId.value);
  cacheUpdatedAt.value = loadOfflineSnapshot(storeId.value)?.updated_at ?? null;
}

async function syncOne(record: OfflineSaleRecord): Promise<'synced' | 'failed' | 'offline'> {
  if (!isOnline.value) {
    ui.addToast('warning', t('offlineSync.toastOffline'));
    return 'offline';
  }
  busyId.value = record.id;
  try {
    // Shared single-flight runner: if the automatic reconnect sync is already
    // replaying this entry, this call waits for that run instead of posting it
    // twice.
    const result = await syncOfflineSales(storeId.value, userId.value, {
      ids: [record.id],
    });
    if (result.synced.length > 0) {
      ui.addToast(
        'success',
        t('offlineSync.toastSynced', {
          invoice: result.synced[0].invoice || offlineRef(record),
        })
      );
      return 'synced';
    }
    const message = result.failed[0]?.message ?? 'Sync failed';
    ui.addToast(
      'error',
      t('offlineSync.toastSyncFailed', { invoice: offlineRef(record), error: message })
    );
    if (result.offline) {
      isOnline.value = false;
      return 'offline';
    }
    return 'failed';
  } finally {
    busyId.value = null;
    reload();
  }
}

async function syncAll() {
  if (syncingAll.value) return;
  if (!isOnline.value) {
    ui.addToast('warning', t('offlineSync.toastOffline'));
    return;
  }
  if (entries.value.length === 0) {
    ui.addToast('info', t('offlineSync.toastNothingToSync'));
    return;
  }
  syncingAll.value = true;
  try {
    const result = await syncOfflineSales(storeId.value, userId.value);
    if (result.synced.length > 0) {
      ui.addToast('success', t('offlineSync.toastAllSynced', { count: result.synced.length }));
    } else {
      ui.addToast('warning', t('offlineSync.toastNoneSynced'));
    }
  } finally {
    syncingAll.value = false;
    reload();
  }
}

function discard(record: OfflineSaleRecord) {
  if (removeOfflineSale(record.store_id, record.user_id, record.id)) {
    ui.addToast('success', t('offlineSync.toastDeleted'));
    reload();
  }
}

function handleOnline() {
  isOnline.value = true;
  reload();
}

function handleOffline() {
  isOnline.value = false;
}

watch(isReady, (ready) => {
  if (ready) reload();
}, { immediate: true });

onMounted(() => {
  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);
});

onUnmounted(() => {
  window.removeEventListener('online', handleOnline);
  window.removeEventListener('offline', handleOffline);
});
</script>

<template>
  <DashboardShell>
    <div
      v-if="!allowed"
      class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border"
    >
      <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">
        {{ t('admin.shared.noPermission', { action: 'access offline sync' }) }}
      </p>
    </div>

    <div v-else class="flex flex-col gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('offlineSync.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('offlineSync.description') }}</p>
      </div>

      <Card>
        <CardContent class="flex flex-col gap-4 pt-4">
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="isOnline ? 'default' : 'destructive'">
              {{ isOnline ? t('offlineSync.online') : t('offlineSync.offline') }}
            </Badge>
            <Badge variant="secondary">
              {{ t('offlineSync.pendingBadge', { count: pendingCount }) }}
            </Badge>
            <Badge v-if="failedCount > 0" variant="destructive">
              {{ t('offlineSync.failedBadge', { count: failedCount }) }}
            </Badge>
            <span class="text-sm text-muted-foreground">
              {{
                cacheUpdatedAt
                  ? t('offlineSync.cacheLabel', { time: formatDateTime(cacheUpdatedAt) })
                  : t('offlineSync.cacheEmpty')
              }}
            </span>
          </div>

          <div class="flex flex-wrap gap-2">
            <Button :disabled="syncingAll || busyId !== null" @click="syncAll">
              {{ syncingAll ? t('offlineSync.syncing') : t('offlineSync.syncAll') }}
            </Button>
            <Button variant="outline" :disabled="syncingAll" @click="reload">
              {{ t('offlineSync.refresh') }}
            </Button>
          </div>

          <p class="text-sm text-muted-foreground">
            <span class="font-medium text-foreground">{{ t('offlineSync.infoTitle') }}:</span>
            {{ t('offlineSync.infoBody') }}
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardContent class="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('offlineSync.when') }}</TableHead>
                <TableHead>{{ t('offlineSync.invoice') }}</TableHead>
                <TableHead>{{ t('offlineSync.customer') }}</TableHead>
                <TableHead>{{ t('offlineSync.items') }}</TableHead>
                <TableHead>{{ t('offlineSync.total') }}</TableHead>
                <TableHead>{{ t('offlineSync.payment') }}</TableHead>
                <TableHead>{{ t('offlineSync.status') }}</TableHead>
                <TableHead>{{ t('offlineSync.actions') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="entries.length === 0">
                <TableCell colspan="8" class="py-10 text-center">
                  <div class="font-medium">{{ t('offlineSync.empty') }}</div>
                  <p class="text-sm text-muted-foreground mt-1">
                    {{ t('offlineSync.emptyDesc') }}
                  </p>
                </TableCell>
              </TableRow>

              <TableRow v-for="record in entries" :key="record.id">
                <TableCell class="whitespace-nowrap text-sm text-muted-foreground">
                  {{ formatDateTime(record.created_at) }}
                </TableCell>
                <TableCell class="font-mono text-xs whitespace-nowrap">
                  {{ offlineRef(record) }}
                </TableCell>
                <TableCell>{{ record.customer_name || t('offlineSync.walkIn') }}</TableCell>
                <TableCell class="text-sm">
                  {{ t('offlineSync.itemCount', { count: record.items.length }) }}
                </TableCell>
                <TableCell class="tabular-nums whitespace-nowrap">
                  {{ formatCurrency(record.totals.total) }}
                </TableCell>
                <TableCell class="text-sm">{{ paymentLabel(record.payment_method) }}</TableCell>
                <TableCell>
                  <Badge :variant="statusVariant(record.status)">
                    {{ statusLabel(record.status) }}
                  </Badge>
                </TableCell>
                <TableCell class="max-w-[240px]">
                  <p
                    v-if="record.error"
                    class="truncate text-xs text-destructive"
                    :title="record.error"
                  >
                    {{ record.error }}
                  </p>
                  <div class="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      :disabled="syncingAll || busyId !== null || !isOnline"
                      @click="syncOne(record)"
                    >
                      {{ t('offlineSync.retry') }}
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger as-child>
                        <Button variant="ghost" size="sm" :disabled="busyId !== null">
                          {{ t('offlineSync.discard') }}
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('offlineSync.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('offlineSync.deleteDescription', { invoice: offlineRef(record) }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="discard(record)">
                            {{ t('offlineSync.discard') }}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  </DashboardShell>
</template>
