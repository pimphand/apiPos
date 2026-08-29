<script setup lang="ts">
import { reactive, ref, watch } from 'vue';
import { inventoryService, type LowStockProduct } from '@/services/inventory';
import { productService } from '@/services/product';
import { useUiStore } from '@/stores/ui';
import { usePageGuard } from '@/composables/usePageGuard';
import { can } from '@/lib/rbac';
import { apiMessage } from '@/lib/apiError';
import type { InventoryTransaction, Product } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const { isReady, allowed, authStore } = usePageGuard('inventory.read');
const ui = useUiStore();

const canAdjust = ref(false);

const transactions = ref<InventoryTransaction[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const typeFilter = ref('');

const lowStock = ref<LowStockProduct[]>([]);
const lowStockLoading = ref(false);

const adjustOpen = ref(false);
const saving = ref(false);
const products = ref<Product[]>([]);
const productsLoading = ref(false);

const adjustForm = reactive({
  product_id: '',
  direction: 'INCREASE' as 'INCREASE' | 'DECREASE',
  quantity: 1,
  reason: 'ADJUSTMENT',
  description: '',
});

const TYPE_LABELS: Record<string, string> = {
  PURCHASE: 'Purchase',
  SALE: 'Sale',
  RETURN: 'Return',
  ADJUSTMENT: 'Adjustment',
  DAMAGE: 'Damage',
  TRANSFER: 'Transfer',
};

const REASON_OPTIONS = [
  'ADJUSTMENT',
  'DAMAGE',
  'RETURN',
  'TRANSFER',
];

function typeBadgeClass(t: string): string {
  switch (t) {
    case 'PURCHASE':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300';
    case 'SALE':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300';
    case 'RETURN':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300';
    case 'ADJUSTMENT':
      return 'bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300';
    case 'DAMAGE':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300';
    case 'TRANSFER':
      return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
    default:
      return 'bg-muted text-muted-foreground';
  }
}

function formatDate(v: string | undefined): string {
  if (!v) return '—';
  const d = new Date(v);
  return isNaN(d.getTime()) ? v : d.toLocaleString('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function signedQty(t: InventoryTransaction): string {
  const neg = ['SALE', 'DAMAGE', 'TRANSFER'].includes(t.type);
  const sign = neg ? '−' : '+';
  return `${sign}${t.quantity}`;
}

async function load() {
  loading.value = true;
  try {
    const type = typeFilter.value !== 'all' && typeFilter.value ? typeFilter.value : undefined;
    const res = await inventoryService.listTransactions({
      type,
      page: page.value,
      per_page: perPage,
    });
    transactions.value = res.transactions ?? [];
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

async function loadLowStock() {
  lowStockLoading.value = true;
  try {
    const res = await inventoryService.getLowStockAlert({ page: 1, per_page: 10 });
    lowStock.value = res.products ?? [];
  } catch {
    lowStock.value = [];
  } finally {
    lowStockLoading.value = false;
  }
}

async function openAdjust() {
  adjustForm.product_id = '';
  adjustForm.direction = 'INCREASE';
  adjustForm.quantity = 1;
  adjustForm.reason = 'ADJUSTMENT';
  adjustForm.description = '';
  adjustOpen.value = true;
  if (products.value.length === 0) {
    productsLoading.value = true;
    try {
      const res = await productService.list({ is_active: 'true', page: 1, per_page: 100 });
      products.value = res.products ?? [];
    } catch {
      products.value = [];
    } finally {
      productsLoading.value = false;
    }
  }
}

async function saveAdjust() {
  if (!adjustForm.product_id) {
    ui.addToast('error', 'Please select a product');
    return;
  }
  const storeId = authStore.user?.store_id ?? '';
  if (!storeId) {
    ui.addToast('error', 'No store assigned to your account');
    return;
  }
  if (adjustForm.quantity <= 0) {
    ui.addToast('error', 'Quantity must be greater than zero');
    return;
  }
  saving.value = true;
  try {
    await inventoryService.adjustStock({
      product_id: adjustForm.product_id,
      store_id: storeId,
      direction: adjustForm.direction,
      quantity: adjustForm.quantity,
      reason: adjustForm.reason,
      description: adjustForm.description || undefined,
    });
    ui.addToast('success', 'Stock adjusted');
    adjustOpen.value = false;
    applyFilter();
    loadLowStock();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

watch(isReady, (r) => {
  if (r) {
    canAdjust.value = can(authStore.currentUser, 'inventory.adjust');
    loadLowStock();
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
    <h3 class="text-xl font-semibold">Access restricted</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">You don't have permission to view inventory.</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">Inventory</h2>
        <p class="text-muted-foreground mt-1">Track stock movements, adjust quantities, and monitor low-stock items.</p>
      </div>
      <Button v-if="canAdjust" @click="openAdjust">+ Adjust Stock</Button>
    </div>

    <Card>
      <CardHeader>
        <CardTitle class="text-base flex items-center gap-2">
          Low Stock Alerts
          <span v-if="lowStock.length" class="rounded-full bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300 text-xs px-2 py-0.5">
            {{ lowStock.length }}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div v-if="lowStockLoading" class="text-center py-6 text-muted-foreground">Loading alerts...</div>
        <div v-else-if="lowStock.length === 0" class="text-center py-6 text-muted-foreground">
          All products are sufficiently stocked.
        </div>
        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="p in lowStock" :key="p.product_id" class="rounded-lg border p-3 space-y-1">
            <div class="font-medium text-sm truncate">{{ p.product_name }}</div>
            <div class="text-xs text-muted-foreground truncate">{{ p.category_name || '—' }}</div>
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">Stock</span>
              <span class="font-semibold tabular-nums text-rose-600 dark:text-rose-400">{{ p.total_stock }}</span>
            </div>
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">Min</span>
              <span class="tabular-nums">{{ p.minimum_stock }}</span>
            </div>
            <div class="text-xs text-rose-600 dark:text-rose-400 font-medium">
              Short by {{ p.deficit }} unit{{ p.deficit === 1 ? '' : 's' }}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <div class="flex flex-wrap items-end gap-3 p-4 border-b">
          <div class="space-y-1">
            <Label>Transaction type</Label>
            <Select v-model="typeFilter" @update:model-value="applyFilter">
              <SelectTrigger class="w-48"><SelectValue placeholder="All types" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All types</SelectItem>
                <SelectItem v-for="(label, key) in TYPE_LABELS" :key="key" :value="key">{{ label }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button @click="applyFilter">Filter</Button>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Type</TableHead>
              <TableHead>Product</TableHead>
              <TableHead class="text-right">Qty</TableHead>
              <TableHead class="text-right">Before</TableHead>
              <TableHead class="text-right">After</TableHead>
              <TableHead>Reference</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>By</TableHead>
              <TableHead>Date</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="9" class="text-center py-8 text-muted-foreground">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="transactions.length === 0">
                <TableCell :colspan="9" class="text-center py-8 text-muted-foreground">No transactions found</TableCell>
              </TableRow>
              <TableRow v-for="t in transactions" :key="t.id">
                <TableCell>
                  <Badge :class="typeBadgeClass(t.type)">{{ TYPE_LABELS[t.type] || t.type }}</Badge>
                </TableCell>
                <TableCell class="font-medium">{{ t.product_name || t.product_id.slice(0, 8) }}</TableCell>
                <TableCell class="text-right font-semibold tabular-nums">
                  <span :class="t.type === 'SALE' || t.type === 'DAMAGE' ? 'text-rose-600 dark:text-rose-400' : 'text-emerald-600 dark:text-emerald-400'">
                    {{ signedQty(t) }}
                  </span>
                </TableCell>
                <TableCell class="text-right tabular-nums text-muted-foreground">{{ t.before_stock }}</TableCell>
                <TableCell class="text-right tabular-nums font-medium">{{ t.after_stock }}</TableCell>
                <TableCell class="text-muted-foreground">{{ t.reference_type || 'MANUAL' }}</TableCell>
                <TableCell class="text-muted-foreground max-w-56 truncate">{{ t.description || '—' }}</TableCell>
                <TableCell class="text-muted-foreground">{{ t.creator_name || '—' }}</TableCell>
                <TableCell class="text-muted-foreground whitespace-nowrap">{{ formatDate(t.created_at) }}</TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
        <div class="flex justify-center py-4">
          <PaginationBar :page="page" :per-page="perPage" :total="total" @update:page="page = $event; load()" />
        </div>
      </CardContent>
    </Card>

    <Dialog v-model:open="adjustOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Adjust Stock</DialogTitle>
          <DialogDescription>Increase or decrease the stock level of a product. Every change is recorded as an inventory transaction.</DialogDescription>
        </DialogHeader>

        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>Product</Label>
            <Select v-model="adjustForm.product_id">
              <SelectTrigger class="w-full"><SelectValue placeholder="Select product" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-if="productsLoading" value="__loading" disabled>Loading products...</SelectItem>
                <SelectItem v-for="p in products" :key="p.id" :value="p.id">
                  {{ p.name }} (stock: {{ p.stock }})
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>Direction</Label>
            <Select v-model="adjustForm.direction">
              <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="INCREASE">Increase (+)</SelectItem>
                <SelectItem value="DECREASE">Decrease (−)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>Quantity</Label>
            <Input v-model.number="adjustForm.quantity" type="number" min="0" :step="1" />
          </div>
          <div class="space-y-2">
            <Label>Reason</Label>
            <Select v-model="adjustForm.reason">
              <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="r in REASON_OPTIONS" :key="r" :value="r">{{ r.charAt(0) + r.slice(1).toLowerCase() }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>Description</Label>
            <Input v-model="adjustForm.description" placeholder="Optional note" autocomplete="off" />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="adjustOpen = false">Cancel</Button>
          <Button :disabled="saving" @click="saveAdjust">{{ saving ? 'Saving...' : 'Adjust Stock' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
