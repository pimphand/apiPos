<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { customerService } from '@/services/customer';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { can } from '@/lib/rbac';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { Customer, Sale } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import Switch from '@/components/ui/switch/Switch.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed, authStore } = usePageGuard('customers.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const customers = ref<Customer[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const activeFilter = ref('');
const search = ref('');

const formOpen = ref(false);
const editing = ref<Customer | null>(null);
const saving = ref(false);
const form = ref<{ code: string; name: string; email: string; phone: string; address: string }>({
  code: '',
  name: '',
  email: '',
  phone: '',
  address: '',
});

const profileOpen = ref(false);
const selected = ref<Customer | null>(null);
const purchases = ref<Sale[]>([]);
const purchasesLoading = ref(false);
const purchasesTotal = ref(0);
const purchasesPage = ref(1);

const saleStatusKey = (s: string) =>
  `admin.sales.status${s.charAt(0)}${s.slice(1).toLowerCase()}` as const;

const saleStatusVariant = (s: string) =>
  s === 'PAID' ? ('default' as const) : s === 'PENDING' ? ('secondary' as const) : ('destructive' as const);

async function load() {
  loading.value = true;
  try {
    const active = ['true', 'false'].includes(activeFilter.value) ? activeFilter.value : undefined;
    const params: Record<string, unknown> = {
      is_active: active,
      page: page.value,
      per_page: perPage,
    };
    const q = search.value.trim();
    const res = q
      ? await customerService.search({ query: q, page: page.value, per_page: perPage })
      : await customerService.list(params);
    let list = res.customers ?? [];
    // Search endpoint doesn't accept is_active; apply it client-side so the
    // status filter still works when a search query is active.
    if (q && active) {
      const wantActive = active === 'true';
      list = list.filter((c) => c.is_active === wantActive);
    }
    customers.value = list;
    total.value = q && active ? list.length : (res.total_count ?? 0);
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

function runSearch() {
  page.value = 1;
  load();
}

function openCreate() {
  editing.value = null;
  form.value = { code: '', name: '', email: '', phone: '', address: '' };
  formOpen.value = true;
}

function openEdit(c: Customer) {
  editing.value = c;
  form.value = {
    code: c.code,
    name: c.name,
    email: c.email || '',
    phone: c.phone || '',
    address: c.address || '',
  };
  formOpen.value = true;
}

async function save() {
  saving.value = true;
  try {
    if (editing.value) {
      await customerService.update(editing.value.id, form.value);
      ui.addToast('success', t('admin.customers.toastUpdated'));
    } else {
      await customerService.create(form.value);
      ui.addToast('success', t('admin.customers.toastCreated'));
    }
    formOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function toggleActive(c: Customer) {
  try {
    if (c.is_active) {
      await customerService.deactivate(c.id);
      ui.addToast('success', t('admin.customers.toastDeactivated'));
    } else {
      await customerService.update(c.id, { is_active: true });
      ui.addToast('success', t('admin.customers.toastActivated'));
    }
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function removeCustomer(c: Customer) {
  try {
    await customerService.delete(c.id);
    ui.addToast('success', t('admin.customers.toastDeleted'));
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function openProfile(c: Customer) {
  selected.value = c;
  purchases.value = [];
  purchasesPage.value = 1;
  purchasesTotal.value = 0;
  profileOpen.value = true;
  await loadPurchases();
}

async function loadPurchases() {
  if (!selected.value) return;
  purchasesLoading.value = true;
  try {
    const res = await customerService.getPurchaseHistory(selected.value.id, {
      page: purchasesPage.value,
      per_page: 10,
    });
    purchases.value = res.purchases ?? [];
    purchasesTotal.value = res.total_count ?? 0;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    purchasesLoading.value = false;
  }
}

const canCreate = computed(() => can(authStore.currentUser, 'customers.create'));
const canUpdate = computed(() => can(authStore.currentUser, 'customers.update'));
const canDelete = computed(() => can(authStore.currentUser, 'customers.delete'));
// Hide the Actions column entirely when the user has no mutating permissions.
const showActions = computed(() => canCreate.value || canUpdate.value || canDelete.value);

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
  <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.customers.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.customers.title') }}</h2>
      </div>
      <Button v-if="canCreate" @click="openCreate">{{ t('admin.customers.newCustomer') }}</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1 flex-1 min-w-52">
          <Label>{{ t('admin.shared.search') }}</Label>
          <div class="flex gap-2">
            <Input v-model="search" :placeholder="t('admin.customers.searchPlaceholder')" @keyup.enter="runSearch" />
            <Button variant="outline" @click="runSearch">{{ t('admin.shared.search') }}</Button>
          </div>
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.shared.status') }}</Label>
          <Select v-model="activeFilter">
            <SelectTrigger class="w-44"><SelectValue :placeholder="t('admin.shared.all')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.shared.all') }}</SelectItem>
              <SelectItem value="true">{{ t('admin.shared.active') }}</SelectItem>
              <SelectItem value="false">{{ t('admin.shared.inactive') }}</SelectItem>
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
              <TableHead>{{ t('admin.customers.code') }}</TableHead>
              <TableHead>{{ t('admin.shared.name') }}</TableHead>
              <TableHead>{{ t('admin.customers.phone') }}</TableHead>
              <TableHead>{{ t('admin.shared.email') }}</TableHead>
              <TableHead>{{ t('admin.shared.status') }}</TableHead>
              <TableHead v-if="showActions" class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="showActions ? 6 : 5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="customers.length === 0">
                <TableCell :colspan="showActions ? 6 : 5" class="text-center py-8 text-muted-foreground">{{ t('admin.customers.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="c in customers" :key="c.id">
                <TableCell class="font-mono text-xs">{{ c.code }}</TableCell>
                <TableCell class="font-medium">{{ c.name }}</TableCell>
                <TableCell class="text-muted-foreground">{{ c.phone || '—' }}</TableCell>
                <TableCell class="text-muted-foreground">{{ c.email || '—' }}</TableCell>
                <TableCell>
                  <Badge :variant="c.is_active ? 'default' : 'secondary'">{{ c.is_active ? t('admin.shared.active') : t('admin.shared.inactive') }}</Badge>
                </TableCell>
                <TableCell v-if="showActions" class="text-right">
                  <div class="flex justify-end items-center gap-2 flex-wrap">
                    <Button variant="outline" size="sm" @click="openProfile(c)">{{ t('admin.customers.viewProfile') }}</Button>
                    <Button v-if="canUpdate" variant="outline" size="sm" @click="openEdit(c)">{{ t('admin.shared.edit') }}</Button>
                    <Switch v-if="canUpdate" :model-value="c.is_active" @update:model-value="toggleActive(c)" />
                    <AlertDialog v-if="canDelete">
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">{{ t('admin.shared.delete') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.customers.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('admin.customers.deleteDescription', { name: c.name }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="removeCustomer(c)">{{ t('admin.shared.delete') }}</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
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

    <Dialog v-model:open="formOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('admin.customers.editTitle') : t('admin.customers.createTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.customers.dialogDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>{{ t('admin.shared.name') }}</Label>
            <Input v-model="form.name" :placeholder="t('admin.customers.namePlaceholder')" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>{{ t('admin.customers.code') }}</Label>
            <Input v-model="form.code" :placeholder="t('admin.customers.codePlaceholder')" autocomplete="off" />
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="space-y-2">
              <Label>{{ t('admin.shared.email') }}</Label>
              <Input v-model="form.email" type="email" :placeholder="t('admin.customers.emailPlaceholder')" autocomplete="off" />
            </div>
            <div class="space-y-2">
              <Label>{{ t('admin.customers.phone') }}</Label>
              <Input v-model="form.phone" :placeholder="t('admin.customers.phonePlaceholder')" autocomplete="off" />
            </div>
          </div>
          <div class="space-y-2">
            <Label>{{ t('admin.customers.address') }}</Label>
            <Textarea v-model="form.address" :placeholder="t('admin.customers.addressPlaceholder')" :rows="2" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="saving" @click="save">{{ saving ? t('admin.shared.saving') : t('admin.shared.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="profileOpen">
      <DialogContent class="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('admin.customers.profile') }}</DialogTitle>
          <DialogDescription v-if="selected">
            {{ selected.name }} — {{ selected.code }}
          </DialogDescription>
        </DialogHeader>
        <div v-if="selected" class="grid gap-2 text-sm">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <div><span class="text-muted-foreground">{{ t('admin.customers.phone') }}:</span> {{ selected.phone || '—' }}</div>
            <div><span class="text-muted-foreground">{{ t('admin.shared.email') }}:</span> {{ selected.email || '—' }}</div>
          </div>
          <div><span class="text-muted-foreground">{{ t('admin.customers.address') }}:</span> {{ selected.address || '—' }}</div>
        </div>

        <div class="mt-2">
          <h4 class="font-semibold mb-2">{{ t('admin.customers.purchaseHistory') }}</h4>
          <div v-if="purchasesLoading" class="text-center py-6 text-muted-foreground">{{ t('admin.shared.loading') }}</div>
          <div v-else-if="purchases.length === 0" class="text-center py-6 text-muted-foreground">{{ t('admin.customers.noPurchases') }}</div>
          <Table v-else>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.customers.invoice') }}</TableHead>
                <TableHead>{{ t('admin.customers.dateLabel') }}</TableHead>
                <TableHead>{{ t('admin.customers.statusLabel') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.customers.totalLabel') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="sale in purchases" :key="sale.id">
                <TableCell class="font-mono text-xs">{{ sale.invoice_number }}</TableCell>
                <TableCell>{{ formatDateTime(sale.created_at) }}</TableCell>
                <TableCell>
                  <Badge :variant="saleStatusVariant(sale.status)">{{ t(saleStatusKey(sale.status)) }}</Badge>
                </TableCell>
                <TableCell class="text-right font-medium">{{ formatCurrency(sale.grand_total) }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <div v-if="purchases.length > 0" class="flex justify-center py-3">
            <PaginationBar :page="purchasesPage" :per-page="10" :total="purchasesTotal" @update:page="purchasesPage = $event; loadPurchases()" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="profileOpen = false">{{ t('admin.shared.cancel') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
