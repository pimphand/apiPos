<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { paymentService } from '@/services/payment';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { can } from '@/lib/rbac';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { Payment, PaymentSummary } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const { isReady, allowed, authStore } = usePageGuard('payments.read');
const canDelete = computed(() => can(authStore.currentUser, 'payments.delete'));
// Hide the Actions column entirely when the user has no mutating permissions.
const showActions = computed(() => canDelete.value);
const ui = useUiStore();
const t = useLocaleStore().t;

const payments = ref<Payment[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const methodFilter = ref('');
const saleIdFilter = ref('');

// Summary dialog
const summaryOpen = ref(false);
const summarySaleId = ref('');
const summary = ref<PaymentSummary | null>(null);
const summaryLoading = ref(false);

const deleteTarget = ref<Payment | null>(null);
const deleting = ref(false);

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

async function openSummary(saleId: string) {
  summarySaleId.value = saleId;
  summary.value = null;
  summaryOpen.value = true;
  summaryLoading.value = true;
  try {
    summary.value = await paymentService.getSummary(saleId);
  } catch (err) {
    ui.addToast('error', apiMessage(err));
    summaryOpen.value = false;
  } finally {
    summaryLoading.value = false;
  }
}

async function deletePayment() {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await paymentService.delete(deleteTarget.value.id);
    ui.addToast('success', t('admin.payments.toastDeleted'));
    deleteTarget.value = null;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    deleting.value = false;
  }
}

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
    <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
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
                <TableHead v-if="showActions" class="text-right">{{ t('admin.shared.actions') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="showActions ? 6 : 5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="payments.length === 0">
                  <TableCell :colspan="showActions ? 6 : 5" class="text-center py-8 text-muted-foreground">{{ t('admin.payments.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="p in payments" :key="p.id">
                  <TableCell class="font-mono text-xs">
                    <button class="text-primary hover:underline cursor-pointer" @click="openSummary(p.sale_id)">{{ p.sale_id.slice(0, 8) }}…</button>
                  </TableCell>
                  <TableCell>
                    <Badge :variant="methodBadgeVariant(p.payment_method)">{{ methodLabel(p.payment_method) }}</Badge>
                  </TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(p.amount) }}</TableCell>
                  <TableCell class="text-muted-foreground">{{ p.reference || '—' }}</TableCell>
                  <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDateTime(p.paid_at || p.created_at) }}</TableCell>
                  <TableCell v-if="showActions" class="text-right">
                    <AlertDialog v-if="canDelete">
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">{{ t('admin.shared.delete') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.payments.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>{{ t('admin.payments.deleteDescription') }}</AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="deleteTarget = p; deletePayment()">{{ t('admin.shared.delete') }}</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
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

      <!-- Payment Summary Dialog -->
      <Dialog v-model:open="summaryOpen">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{{ t('admin.payments.summaryTitle') }}</DialogTitle>
            <DialogDescription>{{ t('admin.payments.summaryDescription') }}</DialogDescription>
          </DialogHeader>
          <div v-if="summaryLoading" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</div>
          <template v-else-if="summary">
            <div class="grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
              <div>
                <div class="text-muted-foreground">{{ t('admin.payments.grandTotal') }}</div>
                <div class="font-medium text-lg">{{ formatCurrency(summary.grand_total) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.payments.totalPaid') }}</div>
                <div class="font-medium text-lg">{{ formatCurrency(summary.total_paid) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.payments.remaining') }}</div>
                <div class="font-medium" :class="summary.remaining > 0 ? 'text-destructive' : ''">{{ formatCurrency(summary.remaining) }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">{{ t('admin.payments.change') }}</div>
                <div class="font-medium">{{ formatCurrency(summary.change) }}</div>
              </div>
            </div>
            <div class="mt-4">
              <div class="text-muted-foreground text-sm mb-1">{{ t('admin.payments.status') }}</div>
              <Badge :variant="summary.status === 'PAID' ? 'default' : summary.status === 'PARTIAL' ? 'secondary' : 'outline'">
                {{ summary.status }}
              </Badge>
            </div>
            <div v-if="summary.payments?.length" class="mt-4">
              <h4 class="font-semibold text-sm mb-2">{{ t('admin.payments.paymentsLabel') }}</h4>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('admin.payments.method') }}</TableHead>
                    <TableHead class="text-right">{{ t('admin.payments.amount') }}</TableHead>
                    <TableHead>{{ t('admin.payments.paidAt') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="pay in summary.payments" :key="pay.id">
                    <TableCell>
                      <Badge :variant="methodBadgeVariant(pay.payment_method)">{{ methodLabel(pay.payment_method) }}</Badge>
                    </TableCell>
                    <TableCell class="text-right font-medium">{{ formatCurrency(pay.amount) }}</TableCell>
                    <TableCell class="text-muted-foreground">{{ formatDateTime(pay.paid_at) }}</TableCell>
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
