<script setup lang="ts">
import { ref, watch } from 'vue';
import { saleService } from '@/services/sale';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { can } from '@/lib/rbac';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { Sale, SaleDetailResponse } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed, authStore } = usePageGuard('sales.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const sales = ref<Sale[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const statusFilter = ref('');
const fromDate = ref('');
const toDate = ref('');

const canCancel = ref(false);
const canRefund = ref(false);

const detailOpen = ref(false);
const detail = ref<SaleDetailResponse | null>(null);
const detailLoading = ref(false);
const actionBusy = ref(false);

const saleStatusKey = (s: string) =>
  `admin.sales.status${s.charAt(0)}${s.slice(1).toLowerCase()}` as const;

function saleStatusVariant(s: string) {
  if (s === 'PAID') return 'default' as const;
  if (s === 'PENDING') return 'secondary' as const;
  if (s === 'CANCELLED') return 'outline' as const;
  return 'destructive' as const;
}

async function load() {
  loading.value = true;
  try {
    const res = await saleService.list({
      status: ['PENDING', 'PAID', 'CANCELLED', 'REFUNDED'].includes(statusFilter.value) ? statusFilter.value : undefined,
      start_date: fromDate.value || undefined,
      end_date: toDate.value || undefined,
      page: page.value,
      per_page: perPage,
    });
    sales.value = res.sales ?? [];
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

async function openDetail(sale: Sale) {
  detail.value = null;
  detailOpen.value = true;
  detailLoading.value = true;
  try {
    detail.value = await saleService.get(sale.id);
  } catch (err) {
    ui.addToast('error', apiMessage(err));
    detailOpen.value = false;
  } finally {
    detailLoading.value = false;
  }
}

async function cancelSale() {
  if (!detail.value) return;
  actionBusy.value = true;
  try {
    await saleService.cancel(detail.value.sale.id, 'Cancelled by admin');
    ui.addToast('success', t('admin.sales.toastCancelled'));
    detailOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    actionBusy.value = false;
  }
}

async function refundSale() {
  if (!detail.value) return;
  actionBusy.value = true;
  try {
    await saleService.refund(detail.value.sale.id, 'Refunded by admin');
    ui.addToast('success', t('admin.sales.toastRefunded'));
    detailOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    actionBusy.value = false;
  }
}

watch(isReady, (r) => {
  if (r) {
    canCancel.value = can(authStore.currentUser, 'sales.cancel');
    canRefund.value = can(authStore.currentUser, 'sales.refund');
    load();
  }
}, { immediate: true });
</script>

<template>
  <DashboardShell>
  <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
  </div>

  <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.sales.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.sales.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.sales.description') }}</p>
      </div>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>{{ t('admin.sales.status') }}</Label>
          <Select v-model="statusFilter">
            <SelectTrigger class="w-40"><SelectValue :placeholder="t('admin.shared.all')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.shared.all') }}</SelectItem>
              <SelectItem value="PENDING">{{ t('admin.sales.statusPending') }}</SelectItem>
              <SelectItem value="PAID">{{ t('admin.sales.statusPaid') }}</SelectItem>
              <SelectItem value="CANCELLED">{{ t('admin.sales.statusCancelled') }}</SelectItem>
              <SelectItem value="REFUNDED">{{ t('admin.sales.statusRefunded') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.sales.fromDate') }}</Label>
          <Input v-model="fromDate" type="date" class="w-40" />
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.sales.toDate') }}</Label>
          <Input v-model="toDate" type="date" class="w-40" />
        </div>
        <Button variant="outline" @click="applyFilter">{{ t('admin.shared.filter') }}</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('admin.sales.invoice') }}</TableHead>
              <TableHead>{{ t('admin.sales.customer') }}</TableHead>
              <TableHead>{{ t('admin.sales.cashier') }}</TableHead>
              <TableHead>{{ t('admin.sales.items') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.sales.grandTotal') }}</TableHead>
              <TableHead>{{ t('admin.sales.status') }}</TableHead>
              <TableHead>{{ t('admin.sales.date') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="8" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="sales.length === 0">
                <TableCell :colspan="8" class="text-center py-8 text-muted-foreground">{{ t('admin.sales.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="sale in sales" :key="sale.id">
                <TableCell class="font-mono text-xs">{{ sale.invoice_number }}</TableCell>
                <TableCell>{{ sale.customer_name || '—' }}</TableCell>
                <TableCell class="text-muted-foreground">{{ sale.cashier_name || '—' }}</TableCell>
                <TableCell>{{ sale.item_count ?? '—' }}</TableCell>
                <TableCell class="text-right font-medium">{{ formatCurrency(sale.grand_total) }}</TableCell>
                <TableCell>
                  <Badge :variant="saleStatusVariant(sale.status)">{{ t(saleStatusKey(sale.status)) }}</Badge>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ formatDateTime(sale.created_at) }}</TableCell>
                <TableCell class="text-right">
                  <Button variant="outline" size="sm" @click="openDetail(sale)">{{ t('admin.sales.viewDetail') }}</Button>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
        <div class="flex justify-center py-4">
          <PaginationBar :page="page" :per-page="perPage" :total="total" @update:page="page = $event; load()" />
        </div>
      </CardContent>
    </Card>

    <Dialog v-model:open="detailOpen">
      <DialogContent class="sm:max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('admin.sales.detailTitle') }}</DialogTitle>
          <DialogDescription v-if="detail">
            {{ detail.sale.invoice_number }}
            <Badge class="ml-2" :variant="saleStatusVariant(detail.sale.status)">{{ t(saleStatusKey(detail.sale.status)) }}</Badge>
          </DialogDescription>
        </DialogHeader>

        <div v-if="detailLoading" class="text-center py-10 text-muted-foreground">{{ t('admin.shared.loading') }}</div>

        <template v-else-if="detail">
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
            <div>
              <div class="text-muted-foreground">{{ t('admin.sales.customer') }}</div>
              <div class="font-medium">{{ detail.sale.customer_name || '—' }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.sales.cashier') }}</div>
              <div class="font-medium">{{ detail.sale.cashier_name || '—' }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.sales.date') }}</div>
              <div class="font-medium">{{ formatDateTime(detail.sale.created_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.sales.paymentMethod') }}</div>
              <div class="font-medium flex flex-wrap gap-1">
                <Badge v-for="p in detail.payments" :key="p.id" variant="outline">{{ p.payment_method }}</Badge>
              </div>
            </div>
          </div>

          <div class="mt-4">
            <h4 class="font-semibold mb-2">{{ t('admin.sales.saleItems') }}</h4>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('admin.sales.product') }}</TableHead>
                  <TableHead class="text-center">{{ t('admin.sales.qty') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.sales.unitPrice') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.sales.lineSubtotal') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="item in detail.items" :key="item.id">
                  <TableCell>{{ item.product_name || item.product_id }}</TableCell>
                  <TableCell class="text-center">{{ item.quantity }}</TableCell>
                  <TableCell class="text-right">{{ formatCurrency(item.unit_price) }}</TableCell>
                  <TableCell class="text-right">{{ formatCurrency(item.subtotal) }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <div class="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="space-y-1 text-sm">
              <div class="flex justify-between"><span class="text-muted-foreground">{{ t('admin.sales.subtotal') }}</span><span>{{ formatCurrency(detail.sale.subtotal) }}</span></div>
              <div class="flex justify-between"><span class="text-muted-foreground">{{ t('admin.sales.discount') }}</span><span>-{{ formatCurrency(detail.sale.discount) }}</span></div>
              <div class="flex justify-between"><span class="text-muted-foreground">{{ t('admin.sales.tax') }}</span><span>{{ formatCurrency(detail.sale.tax) }}</span></div>
              <div class="flex justify-between font-semibold"><span>{{ t('admin.sales.grandTotal') }}</span><span>{{ formatCurrency(detail.sale.grand_total) }}</span></div>
            </div>
            <div class="space-y-1 text-sm">
              <div class="flex justify-between"><span class="text-muted-foreground">{{ t('admin.sales.amountPaid') }}</span><span>{{ formatCurrency(detail.amount_paid) }}</span></div>
              <div class="flex justify-between"><span class="text-muted-foreground">{{ t('admin.sales.change') }}</span><span>{{ formatCurrency(detail.change) }}</span></div>
            </div>
          </div>

          <div v-if="detail.payments.length" class="mt-4">
            <h4 class="font-semibold mb-2">{{ t('admin.sales.paymentsLabel') }}</h4>
            <div class="space-y-1 text-sm">
              <div v-for="p in detail.payments" :key="p.id" class="flex justify-between">
                <span>{{ p.payment_method }} <span v-if="p.reference" class="text-muted-foreground">· {{ p.reference }}</span></span>
                <span>{{ formatCurrency(p.amount) }}</span>
              </div>
            </div>
          </div>
        </template>

        <DialogFooter v-if="detail && (canCancel || canRefund)" class="flex justify-end gap-2">
          <AlertDialog v-if="canCancel">
            <AlertDialogTrigger as-child>
              <Button variant="outline" :disabled="actionBusy || detail.sale.status === 'CANCELLED' || detail.sale.status === 'REFUNDED'">{{ t('admin.sales.cancel') }}</Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{{ t('admin.sales.cancelTitle') }}</AlertDialogTitle>
                <AlertDialogDescription>
                  {{ t('admin.sales.cancelDescription', { invoice: detail.sale.invoice_number }) }}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                <AlertDialogAction @click="cancelSale">{{ t('admin.sales.cancel') }}</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          <AlertDialog v-if="canRefund">
            <AlertDialogTrigger as-child>
              <Button variant="destructive" :disabled="actionBusy || detail.sale.status === 'REFUNDED'">{{ t('admin.sales.refund') }}</Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{{ t('admin.sales.refundTitle') }}</AlertDialogTitle>
                <AlertDialogDescription>
                  {{ t('admin.sales.refundDescription', { invoice: detail.sale.invoice_number }) }}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                <AlertDialogAction @click="refundSale">{{ t('admin.sales.refund') }}</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
