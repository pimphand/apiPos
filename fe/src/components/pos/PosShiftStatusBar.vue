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
</script>

<template>
  <Card>
    <CardContent class="p-4">
      <!-- Summary -->
<div class="grid flex-1 grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-5">
  <div class="rounded-lg border bg-muted/20 p-4">
    <p class="text-xs text-muted-foreground">
      {{ t('posShift.totalTransactions') }}
    </p>
    <p class="mt-1 text-xl font-bold tabular-nums">
      {{ summary?.total_transactions ?? 0 }}
    </p>
  </div>

  <div class="rounded-lg border bg-muted/20 p-4">
    <p class="text-xs text-muted-foreground">
      {{ t('posShift.totalSalesAmount') }}
    </p>
    <p class="mt-1 text-xl font-bold tabular-nums">
      {{ formatCurrency(summary?.total_sales_amount ?? 0) }}
    </p>
  </div>

  <div class="rounded-lg border bg-muted/20 p-4">
    <p class="text-xs text-muted-foreground">
      {{ t('posShift.cashSales') }}
    </p>
    <p class="mt-1 text-xl font-bold tabular-nums">
      {{ formatCurrency(summary?.cash_sales ?? 0) }}
    </p>
  </div>

  <div class="rounded-lg border bg-muted/20 p-4">
    <p class="text-xs text-muted-foreground">
      {{ t('posShift.itemsOut') }}
    </p>
    <p class="mt-1 text-xl font-bold tabular-nums">
      {{ summary?.items_out_total ?? 0 }}
    </p>
  </div>

  <div class="rounded-lg border bg-muted/20 p-4">
    <p class="text-xs text-muted-foreground">
      {{ t('posShift.itemsIn') }}
    </p>
    <p class="mt-1 text-xl font-bold tabular-nums">
      {{ summary?.items_in_total ?? 0 }}
    </p>
  </div>
</div>
    </CardContent>
  </Card>
</template>
