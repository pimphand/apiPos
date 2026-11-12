<script setup lang="ts">
import { computed } from 'vue';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { CashierShift, ShiftSummary } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent } from '@/components/ui/card';

const props = defineProps<{
  currentShift: CashierShift | null;
  summary?: ShiftSummary | null;
  refresh?: () => void;
  openCloseDialog?: () => void;
}>();

const t = useLocaleStore().t;

const openedAtLabel = computed(() => {
  const openedAt = props.currentShift?.opened_at || props.summary?.opened_at;
  return openedAt ? formatDateTime(openedAt) : '';
});
</script>

<template>
  <Card>
    <CardContent class="p-3 sm:p-4">
      <div class="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex min-w-0 flex-wrap items-center gap-2">
          <Badge variant="secondary">{{ t('posShift.shiftOpen') }}</Badge>
          <span v-if="openedAtLabel" class="text-xs text-muted-foreground truncate">
            {{ t('posShift.openedAt') }}: {{ openedAtLabel }}
          </span>
        </div>
        <div v-if="openCloseDialog" class="flex shrink-0 items-center gap-2">
          <Button variant="outline" size="sm" @click="openCloseDialog">
            {{ t('posShift.closeShift') }}
          </Button>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 sm:gap-3 xl:grid-cols-5">
        <div class="min-w-0 rounded-lg border bg-muted/20 p-3 sm:p-4">
          <p class="text-xs text-muted-foreground">{{ t('posShift.totalTransactions') }}</p>
          <p class="mt-1 truncate text-lg font-bold tabular-nums sm:text-xl">{{ summary?.total_transactions ?? 0 }}</p>
        </div>
        <div class="min-w-0 rounded-lg border bg-muted/20 p-3 sm:p-4">
          <p class="text-xs text-muted-foreground">{{ t('posShift.totalSalesAmount') }}</p>
          <p class="mt-1 truncate text-lg font-bold tabular-nums sm:text-xl">{{ formatCurrency(summary?.total_sales_amount ?? 0) }}</p>
        </div>
        <div class="min-w-0 rounded-lg border bg-muted/20 p-3 sm:p-4">
          <p class="text-xs text-muted-foreground">{{ t('posShift.cashSales') }}</p>
          <p class="mt-1 truncate text-lg font-bold tabular-nums sm:text-xl">{{ formatCurrency(summary?.cash_sales ?? 0) }}</p>
        </div>
        <div class="min-w-0 rounded-lg border bg-muted/20 p-3 sm:p-4">
          <p class="text-xs text-muted-foreground">{{ t('posShift.itemsOut') }}</p>
          <p class="mt-1 truncate text-lg font-bold tabular-nums sm:text-xl">{{ summary?.items_out_total ?? 0 }}</p>
        </div>
        <div class="min-w-0 rounded-lg border bg-muted/20 p-3 sm:p-4">
          <p class="text-xs text-muted-foreground">{{ t('posShift.itemsIn') }}</p>
          <p class="mt-1 truncate text-lg font-bold tabular-nums sm:text-xl">{{ summary?.items_in_total ?? 0 }}</p>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
