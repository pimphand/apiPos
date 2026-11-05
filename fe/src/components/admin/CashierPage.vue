<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue';
import { cashierService } from '@/services/cashier';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { can } from '@/lib/rbac';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { CashierShift, ShiftSummary } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const { isReady, allowed, authStore } = usePageGuard('cashier.view_shift');
const canOpenShift = computed(() => can(authStore.currentUser, 'cashier.open_shift'));
const canCloseShift = computed(() => can(authStore.currentUser, 'cashier.close_shift'));
const ui = useUiStore();
const t = useLocaleStore().t;

const shifts = ref<CashierShift[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const statusFilter = ref('');

const currentShift = ref<CashierShift | null>(null);

const openShiftOpen = ref(false);
const openSaving = ref(false);
const openForm = reactive({ opening_cash: 0 });

const closeShiftOpen = ref(false);
const closeSaving = ref(false);
const closeForm = reactive({ closing_cash: 0, actual_cash: 0 });

const summaryOpen = ref(false);
const summary = ref<ShiftSummary | null>(null);
const summaryLoading = ref(false);

function statusVariant(s: string) {
  return s === 'OPEN' ? ('default' as const) : ('secondary' as const);
}

async function load() {
  loading.value = true;
  try {
    const res = await cashierService.listShifts({
      status: statusFilter.value || undefined,
      page: page.value,
      per_page: perPage,
    });
    shifts.value = res.shifts ?? [];
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

async function loadCurrentShift() {
  try {
    currentShift.value = await cashierService.getCurrentShift();
  } catch {
    currentShift.value = null;
  }
}

function openOpenShift() {
  openForm.opening_cash = 0;
  openShiftOpen.value = true;
}

async function saveOpenShift() {
  const storeId = authStore.currentUser?.store_id;
  if (!storeId) {
    ui.addToast('error', 'No store assigned');
    return;
  }
  openSaving.value = true;
  try {
    await cashierService.openShift({
      store_id: storeId,
      opening_cash: openForm.opening_cash,
    });
    ui.addToast('success', t('admin.cashier.toastOpened'));
    openShiftOpen.value = false;
    loadCurrentShift();
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    openSaving.value = false;
  }
}

function openCloseShift() {
  if (!currentShift.value) return;
  closeForm.closing_cash = 0;
  closeForm.actual_cash = 0;
  closeShiftOpen.value = true;
}

async function saveCloseShift() {
  if (!currentShift.value) return;
  closeSaving.value = true;
  try {
    await cashierService.closeShift({
      shift_id: currentShift.value.id,
      closing_cash: closeForm.closing_cash,
      actual_cash: closeForm.actual_cash,
    });
    ui.addToast('success', t('admin.cashier.toastClosed'));
    closeShiftOpen.value = false;
    currentShift.value = null;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    closeSaving.value = false;
  }
}

async function openSummary(shift: CashierShift) {
  summary.value = null;
  summaryOpen.value = true;
  summaryLoading.value = true;
  try {
    summary.value = await cashierService.getShiftSummary(shift.id);
  } catch (err) {
    ui.addToast('error', apiMessage(err));
    summaryOpen.value = false;
  } finally {
    summaryLoading.value = false;
  }
}

watch(isReady, (r) => {
  if (r) {
    loadCurrentShift();
    load();
  }
}, { immediate: true });
</script>

<template>
  <DashboardShell>
    <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
      <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.cashier.noPermissionAction') }) }}</p>
    </div>

    <div v-else class="flex flex-col gap-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
        <div>
          <h2 class="text-2xl font-bold">{{ t('admin.cashier.title') }}</h2>
          <p class="text-muted-foreground mt-1">{{ t('admin.cashier.description') }}</p>
        </div>
        <div class="flex gap-2">
          <!-- <Button v-if="currentShift && canCloseShift" variant="destructive" @click="openCloseShift">{{ t('admin.cashier.closeShiftTitle') }}</Button> -->
          <Button v-if="!currentShift && canOpenShift" @click="openOpenShift">{{ t('admin.cashier.openShift') }}</Button>
        </div>
      </div>

      <Card v-if="currentShift">
        <CardContent class="p-4">
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex flex-wrap items-center gap-3">
              <Badge variant="default">{{ t('admin.cashier.statusOpen') }}</Badge>
              <span class="text-sm text-muted-foreground">{{ t('admin.cashier.openedAt') }}: {{ formatDateTime(currentShift.opened_at) }}</span>
            </div>
            <div class="text-sm font-medium">{{ t('admin.cashier.openingCash') }}: {{ formatCurrency(currentShift.opening_cash) }}</div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent class="p-0">
          <div class="flex flex-wrap items-end gap-3 p-4 border-b">
            <div class="space-y-1">
              <Label>{{ t('admin.cashier.status') }}</Label>
              <Select v-model="statusFilter">
                <SelectTrigger class="w-40"><SelectValue :placeholder="t('admin.cashier.allStatuses')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{{ t('admin.cashier.allStatuses') }}</SelectItem>
                  <SelectItem value="OPEN">{{ t('admin.cashier.statusOpen') }}</SelectItem>
                  <SelectItem value="CLOSED">{{ t('admin.cashier.statusClosed') }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button variant="outline" @click="applyFilter">{{ t('admin.shared.filter') }}</Button>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.cashier.cashierLabel') }}</TableHead>
                <TableHead>{{ t('admin.cashier.storeLabel') }}</TableHead>
                <TableHead>{{ t('admin.cashier.status') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.cashier.openingCash') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.cashier.closingCash') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.cashier.expectedCash') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.cashier.actualCash') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.cashier.difference') }}</TableHead>
                <TableHead>{{ t('admin.cashier.openedAt') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="10" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="shifts.length === 0">
                  <TableCell :colspan="10" class="text-center py-8 text-muted-foreground">{{ t('admin.cashier.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="shift in shifts" :key="shift.id">
                  <TableCell>{{ shift.user_name || '—' }}</TableCell>
                  <TableCell class="text-muted-foreground">{{ shift.store_name || '—' }}</TableCell>
                  <TableCell>
                    <Badge :variant="statusVariant(shift.status)">
                      {{ shift.status === 'OPEN' ? t('admin.cashier.statusOpen') : t('admin.cashier.statusClosed') }}
                    </Badge>
                  </TableCell>
                  <TableCell class="text-right tabular-nums">{{ formatCurrency(shift.opening_cash) }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ shift.closing_cash > 0 ? formatCurrency(shift.closing_cash) : '—' }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ shift.expected_cash > 0 ? formatCurrency(shift.expected_cash) : '—' }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ shift.actual_cash > 0 ? formatCurrency(shift.actual_cash) : '—' }}</TableCell>
                  <TableCell class="text-right tabular-nums" :class="shift.cash_difference !== 0 ? 'text-destructive font-medium' : ''">
                    {{ shift.cash_difference !== 0 ? formatCurrency(shift.cash_difference) : '—' }}
                  </TableCell>
                  <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDateTime(shift.opened_at) }}</TableCell>
                  <TableCell class="text-right">
                    <Button variant="outline" size="sm" @click="openSummary(shift)">{{ t('admin.cashier.viewSummary') }}</Button>
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

      <!-- Open Shift Dialog -->
      <Dialog v-model:open="openShiftOpen">
        <DialogContent class="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{{ t('admin.cashier.openShiftTitle') }}</DialogTitle>
            <DialogDescription>{{ t('admin.cashier.openShiftDescription') }}</DialogDescription>
          </DialogHeader>
          <div class="space-y-2">
            <Label>{{ t('admin.cashier.openingCashLabel') }}</Label>
            <Input v-model.number="openForm.opening_cash" type="number" min="0" :placeholder="t('admin.cashier.placeholderCash')" />
          </div>
          <DialogFooter>
            <Button variant="outline" @click="openShiftOpen = false">{{ t('admin.shared.cancel') }}</Button>
            <Button :disabled="openSaving" @click="saveOpenShift">{{ openSaving ? t('admin.shared.saving') : t('admin.cashier.openShiftTitle') }}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- Close Shift Dialog -->
      <Dialog v-model:open="closeShiftOpen">
        <DialogContent class="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{{ t('admin.cashier.closeShiftTitle') }}</DialogTitle>
            <DialogDescription>{{ t('admin.cashier.closeShiftDescription') }}</DialogDescription>
          </DialogHeader>
          <div class="space-y-3">
            <div class="space-y-1">
              <div class="text-sm text-muted-foreground">{{ t('admin.cashier.openingCash') }}</div>
              <div class="font-medium">{{ currentShift ? formatCurrency(currentShift.opening_cash) : '—' }}</div>
            </div>
            <div class="space-y-2">
              <Label>{{ t('admin.cashier.closingCashLabel') }}</Label>
              <Input v-model.number="closeForm.closing_cash" type="number" min="0" :placeholder="t('admin.cashier.placeholderCash')" />
            </div>
            <div class="space-y-2">
              <Label>{{ t('admin.cashier.actualCashLabel') }}</Label>
              <Input v-model.number="closeForm.actual_cash" type="number" min="0" :placeholder="t('admin.cashier.placeholderCash')" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="closeShiftOpen = false">{{ t('admin.shared.cancel') }}</Button>
            <Button variant="destructive" :disabled="closeSaving" @click="saveCloseShift">{{ closeSaving ? t('admin.shared.saving') : t('admin.cashier.closeShiftTitle') }}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- Summary Dialog -->
      <Dialog v-model:open="summaryOpen">
        <DialogContent class="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{{ t('admin.cashier.summaryTitle') }}</DialogTitle>
            <DialogDescription>{{ t('admin.cashier.summaryDescription') }}</DialogDescription>
          </DialogHeader>
          <div v-if="summaryLoading" class="text-center py-10 text-muted-foreground">{{ t('admin.shared.loading') }}</div>
          <template v-else-if="summary">
            <div class="grid grid-cols-1 gap-3 text-sm min-[420px]:grid-cols-2 sm:grid-cols-4">
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.openingCash') }}</div>
                <div class="font-medium">{{ formatCurrency(summary.opening_cash) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.expectedCash') }}</div>
                <div class="font-medium">{{ formatCurrency(summary.expected_cash) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.actualCash') }}</div>
                <div class="font-medium">{{ formatCurrency(summary.actual_cash) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.difference') }}</div>
                <div class="font-medium" :class="summary.cash_difference !== 0 ? 'text-destructive' : ''">{{ formatCurrency(summary.cash_difference) }}</div>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 text-sm mt-3">
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.totalTransactions') }}</div>
                <div class="font-medium">{{ summary.total_transactions }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.totalSalesAmount') }}</div>
                <div class="font-medium">{{ formatCurrency(summary.total_sales_amount) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.cashier.status') }}</div>
                <Badge :variant="summary.status === 'OPEN' ? 'default' : 'secondary'">
                  {{ summary.status === 'OPEN' ? t('admin.cashier.statusOpen') : t('admin.cashier.statusClosed') }}
                </Badge>
              </div>
            </div>

            <div v-if="summary.sales?.length" class="mt-4">
              <h4 class="font-semibold mb-2">{{ t('admin.cashier.recentSales') }}</h4>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Invoice</TableHead>
                    <TableHead class="text-right">Total</TableHead>
                    <TableHead>Method</TableHead>
                    <TableHead>{{ t('admin.cashier.openedAt') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="sale in summary.sales" :key="sale.id">
                    <TableCell class="font-mono text-xs">{{ sale.invoice_number }}</TableCell>
                    <TableCell class="text-right font-medium">{{ formatCurrency(sale.grand_total) }}</TableCell>
                    <TableCell>{{ sale.payment_method }}</TableCell>
                    <TableCell class="text-muted-foreground">{{ formatDateTime(sale.created_at) }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          </template>
          <DialogFooter>
            <Button variant="outline" @click="summaryOpen = false">{{ t('admin.shared.cancel') }}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  </DashboardShell>
</template>
