<script setup lang="ts">
import { ref, watch } from 'vue';
import { paymentService } from '@/services/payment';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { Payment } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const { isReady, allowed } = usePageGuard('payments.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const payments = ref<Payment[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const methodFilter = ref('');
const saleIdFilter = ref('');

const methodBadgeVariant = (m: string) => {
  if (m === 'CASH') return 'default' as const;
  if (m === 'CARD') return 'secondary' as const;
  return 'outline' as const;
};

const methodLabel = (m: string) => {
  const map: Record<string, string> = {
    CASH: 'admin.payments.cash',
    CARD: 'admin.payments.card',
    BANK_TRANSFER: 'admin.payments.bankTransfer',
    EWALLET: 'admin.payments.ewallet',
    OTHER: 'admin.payments.other',
  };
  return map[m] ? t(map[m]) : m;
};

async function load() {
  loading.value = true;
  try {
    const res = await paymentService.list({
      sale_id: saleIdFilter.value || undefined,
      payment_method: methodFilter.value || undefined,
      page: page.value,
      per_page: perPage,
    });
    payments.value = res.payments ?? [];
    total.value = res.total_count ?? 0;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

function applyFilter() {
  page.value = 1;
  load();
}

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
    <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
    </div>

    <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
      <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.payments.noPermissionAction') }) }}</p>
    </div>

    <div v-else class="flex flex-col gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.payments.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.payments.description') }}</p>
      </div>

      <Card>
        <CardContent class="flex flex-wrap items-end gap-3 pt-4">
          <div class="space-y-1 flex-1 min-w-48">
            <Label>{{ t('admin.payments.filterBySale') }}</Label>
            <Input v-model="saleIdFilter" placeholder="Sale ID" @keyup.enter="applyFilter" />
          </div>
          <div class="space-y-1">
            <Label>{{ t('admin.payments.method') }}</Label>
            <Select v-model="methodFilter">
              <SelectTrigger class="w-44"><SelectValue :placeholder="t('admin.payments.allMethods')" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{{ t('admin.payments.allMethods') }}</SelectItem>
                <SelectItem value="CASH">{{ t('admin.payments.cash') }}</SelectItem>
                <SelectItem value="CARD">{{ t('admin.payments.card') }}</SelectItem>
                <SelectItem value="BANK_TRANSFER">{{ t('admin.payments.bankTransfer') }}</SelectItem>
                <SelectItem value="EWALLET">{{ t('admin.payments.ewallet') }}</SelectItem>
                <SelectItem value="OTHER">{{ t('admin.payments.other') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button variant="outline" @click="applyFilter">{{ t('admin.shared.filter') }}</Button>
        </CardContent>
      </Card>

      <Card>
        <CardContent class="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.payments.saleId') }}</TableHead>
                <TableHead>{{ t('admin.payments.method') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.payments.amount') }}</TableHead>
                <TableHead>{{ t('admin.payments.reference') }}</TableHead>
                <TableHead>{{ t('admin.payments.paidAt') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="payments.length === 0">
                  <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">{{ t('admin.payments.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="p in payments" :key="p.id">
                  <TableCell class="font-mono text-xs">{{ p.sale_id.slice(0, 8) }}…</TableCell>
                  <TableCell>
                    <Badge :variant="methodBadgeVariant(p.payment_method)">{{ methodLabel(p.payment_method) }}</Badge>
                  </TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(p.amount) }}</TableCell>
                  <TableCell class="text-muted-foreground">{{ p.reference || '—' }}</TableCell>
                  <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDateTime(p.paid_at || p.created_at) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          <div class="flex justify-center py-4">
            <PaginationBar :page="page" :per-page="perPage" :total="total" @update:page="page = $event; load()" />
          </div>
        </CardContent>
      </Card>
    </div>
  </DashboardShell>
</template>
