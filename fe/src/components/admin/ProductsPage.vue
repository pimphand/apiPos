<script setup lang="ts">
import { reactive, ref, watch } from 'vue';
import { productService } from '@/services/product';
import { categoryService } from '@/services/category';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { Product, ProductSKU, Category } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed } = usePageGuard('products.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const products = ref<Product[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const search = ref('');
const categoryFilter = ref('');
const activeFilter = ref('');

const categories = ref<Category[]>([]);

const formOpen = ref(false);
const editing = ref<Product | null>(null);
const saving = ref(false);

const baseForm = reactive({
  name: '',
  description: '',
  category_id: '',
  minimum_stock: 0,
});

const skuRows = ref<
  (ProductSKU & { _deleted?: boolean; _isNew?: boolean })[]
>([]);

const initialSKUsLoading = ref(false);

async function loadCategories() {
  try {
    const res = await categoryService.list({ is_active: 'true', page: 1, per_page: 100 });
    categories.value = res.categories;
  } catch {
    categories.value = [];
  }
}

async function load() {
  loading.value = true;
  try {
    const active = ['true', 'false'].includes(activeFilter.value) ? activeFilter.value : undefined;
    const category = categoryFilter.value !== 'all' && categoryFilter.value ? categoryFilter.value : undefined;
    const res = search.value
      ? await productService.search({ name: search.value, is_active: active, get_sku: true, page: page.value, per_page: perPage })
      : await productService.list({ is_active: active, category_id: category, get_sku: true, page: page.value, per_page: perPage });
    products.value = res.products ?? [];
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

function searchSubmit() {
  page.value = 1;
  load();
}

function resetForm() {
  baseForm.name = '';
  baseForm.description = '';
  baseForm.category_id = categories.value[0]?.id ?? '';
  baseForm.minimum_stock = 0;
}

function newSkuRow(partial: Partial<ProductSKU> = {}): ProductSKU & { _isNew?: boolean } {
  return {
    id: '',
    product_id: '',
    sku: '',
    barcode: '',
    variant: '',
    color: '',
    size: '',
    unit: '',
    cost_price: partial.cost_price ?? 0,
    selling_price: partial.selling_price ?? 0,
    stock: 0,
    minimum_stock: 0,
    weight: 0,
    volume: 0,
    expiry_date: '',
    batch_number: '',
    location: '',
    image_url: '',
    is_active: true,
    created_at: '',
    updated_at: '',
    _isNew: true,
  };
}

async function openCreate() {
  editing.value = null;
  resetForm();
  skuRows.value = [newSkuRow()];
  formOpen.value = true;
}

function matchesSku(sku: ProductSKU, q: string): boolean {
  const s = q.toLowerCase();
  return (
    sku.sku.toLowerCase().includes(s) ||
    sku.barcode.toLowerCase().includes(s) ||
    sku.variant.toLowerCase().includes(s) ||
    sku.color.toLowerCase().includes(s) ||
    sku.size.toLowerCase().includes(s)
  );
}

async function openEdit(p: Product) {
  editing.value = p;
  baseForm.name = p.name;
  baseForm.description = p.description || '';
  baseForm.category_id = p.category_id;
  baseForm.minimum_stock = p.minimum_stock;
  initialSKUsLoading.value = true;
  try {
    const res = await productService.listSKUs(p.id, { page: 1, per_page: 100 });
    skuRows.value = res.product_skus.map((s) => ({ ...s }));
  } catch {
    skuRows.value = [];
  } finally {
    initialSKUsLoading.value = false;
  }
  formOpen.value = true;
}

function addSkuRow() {
  const first = skuRows.value[0];
  skuRows.value.push(
    newSkuRow({
      cost_price: first?.cost_price ?? 0,
      selling_price: first?.selling_price ?? 0,
    })
  );
}

function removeSkuRow(index: number) {
  const row = skuRows.value[index];
  if (row.id && !row._isNew) {
    row._deleted = true;
  } else {
    skuRows.value.splice(index, 1);
  }
}

function undoDelete(row: ProductSKU & { _deleted?: boolean }) {
  row._deleted = false;
}

async function save() {
  saving.value = true;
  try {
    const skuUpdates = skuRows.value
      .filter((r) => !r._deleted)
      .map((r) => ({
        id: r._isNew ? undefined : r.id,
        sku: r.sku,
        barcode: r.barcode || undefined,
        variant: r.variant || undefined,
        color: r.color || undefined,
        size: r.size || undefined,
        unit: r.unit || undefined,
        cost_price: r.cost_price,
        selling_price: r.selling_price,
        stock: r.stock,
        minimum_stock: r.minimum_stock,
        weight: r.weight || undefined,
        volume: r.volume || undefined,
        expiry_date: r.expiry_date || undefined,
        batch_number: r.batch_number || undefined,
        location: r.location || undefined,
        image_url: r.image_url || undefined,
        is_active: r.is_active,
      }));
    const skuIdsToDelete = skuRows.value.filter((r) => r._deleted).map((r) => r.id);
    const firstSku = skuRows.value.find((r) => !r._deleted) ?? skuRows.value[0];

    if (editing.value) {
      await productService.update(editing.value.id, {
        name: baseForm.name,
        description: baseForm.description || undefined,
        category_id: baseForm.category_id,
        cost_price: firstSku?.cost_price ?? 0,
        selling_price: firstSku?.selling_price ?? 0,
        minimum_stock: baseForm.minimum_stock,
        sku_updates: skuUpdates,
        sku_ids_to_delete: skuIdsToDelete,
      });
      ui.addToast('success', t('admin.products.toastUpdated'));
    } else {
      const first = skuRows.value.find((r) => !r._deleted) ?? skuRows.value[0];
      await productService.create({
        name: baseForm.name,
        description: baseForm.description || undefined,
        category_id: baseForm.category_id,
        cost_price: first?.cost_price ?? 0,
        selling_price: first?.selling_price ?? 0,
        stock: first?.stock ?? 0,
        minimum_stock: baseForm.minimum_stock,
        sku: first?.sku,
        barcode: first?.barcode || undefined,
        variant: first?.variant || undefined,
        color: first?.color || undefined,
        size: first?.size || undefined,
        unit: first?.unit || undefined,
        weight: first?.weight || undefined,
        volume: first?.volume || undefined,
        batch_number: first?.batch_number || undefined,
        location: first?.location || undefined,
      });
      ui.addToast('success', t('admin.products.toastCreated'));
    }
    formOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

const statusTarget = ref<Product | null>(null);
const statusAction = ref<'activate' | 'deactivate'>('deactivate');
const toggling = ref(false);

function openStatusDialog(p: Product) {
  statusAction.value = p.is_active ? 'deactivate' : 'activate';
  statusTarget.value = p;
}

async function confirmStatusToggle() {
  const p = statusTarget.value;
  if (!p) return;
  toggling.value = true;
  try {
    if (statusAction.value === 'deactivate') {
      await productService.deactivate(p.id);
      ui.addToast('success', t('admin.products.toastDeactivated'));
    } else {
      await productService.activate(p.id);
      ui.addToast('success', t('admin.products.toastActivated'));
    }
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    toggling.value = false;
    statusTarget.value = null;
  }
}

async function removeProduct(p: Product) {
  try {
    await productService.delete(p.id);
    ui.addToast('success', t('admin.products.toastDeleted'));
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

function formatCurrency(v: number | undefined | null): string {
  return (v ?? 0).toLocaleString('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 });
}

function formatPriceRange(p: Product): string {
  const prices = (p.skus ?? [])
    .map((s) => s.selling_price)
    .filter((v) => v > 0);
  if (prices.length === 0) return formatCurrency(p.selling_price);
  if (prices.length === 1) return formatCurrency(prices[0]);
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  return `${formatCurrency(min)} – ${formatCurrency(max)}`;
}

watch(isReady, (r) => {
  if (r) {
    loadCategories();
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
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.products.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.products.title') }}</h2>
      </div>
      <Button @click="openCreate">{{ t('admin.products.newProduct') }}</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1 min-w-56 flex-1">
          <Label>{{ t('admin.products.searchByName') }}</Label>
          <Input v-model="search" :placeholder="t('admin.products.searchPlaceholder')" @keyup.enter="searchSubmit" autocomplete="off" />
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.products.category') }}</Label>
          <Select v-model="categoryFilter">
            <SelectTrigger class="w-48"><SelectValue :placeholder="t('admin.products.allCategories')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.products.allCategories') }}</SelectItem>
              <SelectItem v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.shared.status') }}</Label>
          <Select v-model="activeFilter">
            <SelectTrigger class="w-36"><SelectValue :placeholder="t('admin.shared.all')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.shared.all') }}</SelectItem>
              <SelectItem value="true">{{ t('admin.shared.active') }}</SelectItem>
              <SelectItem value="false">{{ t('admin.shared.inactive') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button @click="applyFilter">{{ t('admin.shared.filter') }}</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('admin.shared.name') }}</TableHead>
              <TableHead>{{ t('admin.products.category') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.products.sellingPrice') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.products.stock') }}</TableHead>
              <TableHead class="text-center">{{ t('admin.products.skus') }}</TableHead>
              <TableHead>{{ t('admin.shared.status') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="7" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="products.length === 0">
                <TableCell :colspan="7" class="text-center py-8 text-muted-foreground">{{ t('admin.products.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="p in products" :key="p.id">
                <TableCell class="font-medium">{{ p.name }}</TableCell>
                <TableCell class="text-muted-foreground">{{ p.category_name || '—' }}</TableCell>
                <TableCell class="text-right font-medium tabular-nums">{{ formatPriceRange(p) }}</TableCell>
                <TableCell class="text-right tabular-nums">{{ p.stock }}</TableCell>
                <TableCell class="text-center tabular-nums">{{ p.sku_count ?? 0 }}</TableCell>
                <TableCell>
                  <button
                    type="button"
                    class="inline-flex items-center gap-1.5 rounded px-2 py-1 text-xs font-medium transition-colors hover:opacity-80"
                    :class="p.is_active ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'"
                    @click="openStatusDialog(p)"
                    :title="p.is_active ? t('admin.products.clickToDeactivate') : t('admin.products.clickToActivate')"
                  >
                    <span class="size-1.5 rounded-full" :class="p.is_active ? 'bg-primary' : 'bg-muted-foreground'"></span>
                    {{ p.is_active ? t('admin.shared.active') : t('admin.shared.inactive') }}
                  </button>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end items-center gap-2">
                    <Button variant="outline" size="sm" @click="openEdit(p)">{{ t('admin.shared.edit') }}</Button>
                    <AlertDialog>
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">{{ t('admin.shared.delete') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.products.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('admin.products.deleteDescription', { name: p.name }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="removeProduct(p)">{{ t('admin.shared.delete') }}</AlertDialogAction>
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

    <AlertDialog
      :open="!!statusTarget"
      @update:open="(v) => { if (!v) statusTarget = null }"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ statusAction === 'deactivate' ? t('admin.products.deactivateTitle') : t('admin.products.activateTitle') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ statusAction === 'deactivate' ? t('admin.products.deactivateDescription', { name: statusTarget?.name ?? '' }) : t('admin.products.activateDescription', { name: statusTarget?.name ?? '' }) }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="toggling">{{ t('admin.shared.cancel') }}</AlertDialogCancel>
          <Button
            type="button"
            variant="default"
            :disabled="toggling"
            @click="confirmStatusToggle"
          >
            {{ toggling ? t('admin.shared.saving') : statusAction === 'deactivate' ? t('admin.products.deactivate') : t('admin.products.activate') }}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <Dialog v-model:open="formOpen">
      <DialogContent class="sm:max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('admin.products.editTitle') : t('admin.products.createTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.products.dialogDescription') }}</DialogDescription>
        </DialogHeader>

        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>{{ t('admin.shared.name') }}</Label>
            <Input v-model="baseForm.name" :placeholder="t('admin.products.namePlaceholder')" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>{{ t('admin.shared.description') }}</Label>
            <Textarea v-model="baseForm.description" :placeholder="t('admin.products.descriptionPlaceholder')" :rows="2" />
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-2">
              <Label>{{ t('admin.products.category') }}</Label>
              <Select v-model="baseForm.category_id">
                <SelectTrigger class="w-full"><SelectValue :placeholder="t('admin.products.selectCategory')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2">
              <Label>{{ t('admin.products.minimumStock') }}</Label>
              <Input v-model.number="baseForm.minimum_stock" type="number" min="0" />
            </div>
          </div>
        </div>

        <div class="border-t pt-4">
          <div class="flex items-center justify-between mb-3">
            <div>
              <h3 class="font-semibold">{{ t('admin.products.skuSection') }}</h3>
              <p class="text-xs text-muted-foreground">
                {{ editing ? t('admin.products.editSkuHint') : t('admin.products.createSkuHint') }}
              </p>
            </div>
            <Button variant="outline" size="sm" @click="addSkuRow">{{ t('admin.products.addSku') }}</Button>
          </div>

          <div v-if="initialSKUsLoading" class="text-center py-6 text-muted-foreground">{{ t('admin.products.loadingSkus') }}</div>

          <div v-for="(sku, idx) in skuRows" v-else :key="idx" :class="['border rounded-lg p-4 space-y-3', sku._deleted && 'opacity-50']">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                SKU #{{ idx + 1 }}<span v-if="sku._deleted">{{ t('admin.products.toBeDeleted') }}</span>
              </span>
              <div>
                <Button v-if="sku._deleted" variant="outline" size="sm" @click="undoDelete(sku)">{{ t('admin.products.undo') }}</Button>
                <Button v-else variant="ghost" size="sm" @click="removeSkuRow(idx)">{{ t('admin.products.remove') }}</Button>
              </div>
            </div>
            <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.skuCode') }}</Label>
                <Input v-model="sku.sku" :placeholder="t('admin.products.skuCodePlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.barcode') }}</Label>
                <Input v-model="sku.barcode" :placeholder="t('admin.products.scanBarcode')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.variant') }}</Label>
                <Input v-model="sku.variant" :placeholder="t('admin.products.variantPlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.color') }}</Label>
                <Input v-model="sku.color" :placeholder="t('admin.products.colorPlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.size') }}</Label>
                <Input v-model="sku.size" :placeholder="t('admin.products.sizePlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.unit') }}</Label>
                <Input v-model="sku.unit" :placeholder="t('admin.products.unitPlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.costPrice') }}</Label>
                <Input v-model.number="sku.cost_price" type="number" min="0" step="any" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.sellingPriceShort') }}</Label>
                <Input v-model.number="sku.selling_price" type="number" min="0" step="any" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.stock') }}</Label>
                <Input v-model.number="sku.stock" type="number" min="0" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.minStock') }}</Label>
                <Input v-model.number="sku.minimum_stock" type="number" min="0" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.expiryDate') }}</Label>
                <Input v-model="sku.expiry_date" type="date" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.batchNumber') }}</Label>
                <Input v-model="sku.batch_number" :placeholder="t('admin.products.batchPlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.location') }}</Label>
                <Input v-model="sku.location" :placeholder="t('admin.products.locationPlaceholder')" autocomplete="off" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.weight') }}</Label>
                <Input v-model.number="sku.weight" type="number" min="0" step="any" />
              </div>
              <div class="space-y-1">
                <Label class="text-xs">{{ t('admin.products.volume') }}</Label>
                <Input v-model.number="sku.volume" type="number" min="0" step="any" />
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="saving" @click="save">{{ saving ? t('admin.shared.saving') : t('admin.shared.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
