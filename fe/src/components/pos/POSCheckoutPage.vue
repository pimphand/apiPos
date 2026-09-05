<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue';
import { productService } from '@/services/product';
import { customerService } from '@/services/customer';
import { saleService } from '@/services/sale';
import { cashierService } from '@/services/cashier';
import { useCashierShift } from '@/composables/useCashierShift';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { Product, ProductSKU, Customer, ShiftItem, ShiftSummary } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const { isReady, allowed, authStore } = usePageGuard('sales.create');
const ui = useUiStore();
const t = useLocaleStore().t;

// ---- Cashier shift (docs/pos-cashier-shift.md) --------------------------
const {
  currentShift,
  opening,
  closing,
  summary,
  summaryLoading,
  refreshShift,
  loadSummary,
  openShift,
  closeShift,
  ensureShiftOpen,
} = useCashierShift();

const openDialog = ref(false);
const openForm = reactive({ opening_cash: 0 });
/** Gates the very first paint until the shift state is resolved. */
const checkingShift = ref(true);

const closeDialog = ref(false);
const closeActual = ref(0);
const closeClosing = ref(0);
const diffConfirm = ref(false);
const closedResult = ref<ShiftSummary | null>(null);
const resultDialog = ref(false);

const itemsDirection = ref<'OUT' | 'IN'>('OUT');
const itemsOut = ref<ShiftItem[]>([]);
const itemsIn = ref<ShiftItem[]>([]);
const itemsLoading = ref(false);

const differencePreview = computed(
  () => (Number(closeActual.value) || 0) - (summary.value?.expected_cash ?? 0)
);

function openOpenShift() {
  openForm.opening_cash = 0;
  openDialog.value = true;
}

async function submitOpenShift() {
  if (opening.value) return;
  try {
    await openShift(openForm.opening_cash);
    ui.addToast('success', t('posShift.toastOpened'));
    openDialog.value = false;
    await refreshLive();
    searchProducts(); // product list only renders while a shift is open
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function refreshLive() {
  await refreshShift();
  if (currentShift.value) {
    await loadSummary();
    await loadShiftItems();
  }
}

async function loadShiftItems() {
  const id = currentShift.value?.id;
  if (!id) return;
  itemsLoading.value = true;
  try {
    const [outRes, inRes] = await Promise.all([
      cashierService.listShiftItems({ id, direction: 'OUT', page: 1, per_page: 50 }),
      cashierService.listShiftItems({ id, direction: 'IN', page: 1, per_page: 50 }),
    ]);
    itemsOut.value = outRes.items ?? [];
    itemsIn.value = inRes.items ?? [];
  } catch {
    // Items are auxiliary; ignore failures so the dialogs still work.
  } finally {
    itemsLoading.value = false;
  }
}

async function openCloseDialog() {
  if (!currentShift.value) return;
  itemsDirection.value = 'OUT';
  closeActual.value = 0;
  closeClosing.value = 0;
  diffConfirm.value = false;
  closeDialog.value = true;
  await loadSummary();
  if (summary.value && closeActual.value === 0) {
    // Suggest expected cash; the cashier may edit before confirming.
    closeActual.value = summary.value.expected_cash ?? 0;
    closeClosing.value = closeActual.value;
  }
  await loadShiftItems();
}

function askClose() {
  // Require an explicit confirmation when the counted cash differs.
  if (differencePreview.value !== 0) {
    diffConfirm.value = true;
  } else {
    performClose();
  }
}

async function performClose() {
  const id = currentShift.value?.id;
  if (!id || closing.value) return;
  try {
    // Inputs may come back as strings from the number input when cleared.
    const actual = Number(closeActual.value) || 0;
    const closingCash = Number(closeClosing.value) || 0;
    const finalSummary = await closeShift(id, actual, closingCash);
    ui.addToast('success', t('posShift.toastClosed'));
    closeDialog.value = false;
    diffConfirm.value = false;
    closedResult.value = finalSummary;
    resultDialog.value = true;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

// ---- Cart / POS state ----------------------------------------------------
interface CartItem {
  product_id: string;
  sku_id: string;
  name: string;
  sku_code: string;
  unit_price: number;
  quantity: number;
  stock: number;
  original_stock: number;
}

const searchQuery = ref('');
const searchResults = ref<Product[]>([]);
const searching = ref(false);
const barcodeInput = ref('');
const searchInputRef = ref<HTMLInputElement | null>(null);

const cart = ref<CartItem[]>([]);
const discount = ref(0);
const taxRate = ref(11);
const selectedCustomerId = ref('');
const NO_CUSTOMER = 'none'; // reka-ui disallows empty-string SelectItem values
const customers = ref<Customer[]>([]);
const customersLoading = ref(false);

const paymentOpen = ref(false);
const paymentMethod = ref('CASH');
const amountPaid = ref(0);
const processing = ref(false);

// Ensure amountPaid is always a number (Input component may not propagate type="number")
function setAmountPaid(value: string | number) {
  amountPaid.value = Number(value) || 0;
}

const subtotal = computed(() => cart.value.reduce((sum, item) => sum + item.unit_price * item.quantity, 0));
const taxAmount = computed(() => Math.round(subtotal.value * (taxRate.value / 100)));
const grandTotal = computed(() => subtotal.value + taxAmount.value - discount.value);
const change = computed(() => Math.max(0, amountPaid.value - grandTotal.value));

async function searchProducts() {
  const q = searchQuery.value.trim();
  searching.value = true;
  try {
    const res = await productService.list({
      search: q || undefined,
      page: 1,
      per_page: 100,
      get_sku: true,
    });
    searchResults.value = res.products ?? [];
  } catch (err) {
    ui.addToast('error', apiMessage(err));
    searchResults.value = [];
  } finally {
    searching.value = false;
  }
}

async function lookupBarcode() {
  const code = barcodeInput.value.trim();
  if (!code) return;
  try {
    const sku = await productService.getSKUByBarcode(code);
    const product = await productService.get(sku.product_id, true);
    addToCart(product, sku);
    barcodeInput.value = '';
  } catch {
    try {
      const sku = await productService.getSKUBySKU(code);
      const product = await productService.get(sku.product_id, true);
      addToCart(product, sku);
      barcodeInput.value = '';
    } catch {
      ui.addToast('error', `Product not found for: ${code}`);
    }
  }
}

function addToCart(product: Product, sku?: ProductSKU) {
  const skuData = sku ?? product.skus?.[0];
  if (!skuData) return;

  // Calculate remaining stock for this SKU (original stock minus what's already in cart)
  const inCart = cart.value.filter(i => i.sku_id === skuData.id).reduce((sum, i) => sum + i.quantity, 0);
  const remaining = skuData.stock - inCart;
  if (remaining <= 0) {
    ui.addToast('error', t('admin.pos.toastStockIssue', { name: product.name }));
    return;
  }

  const existing = cart.value.find(i => i.sku_id === skuData.id);
  if (existing) {
    existing.quantity++;
  } else {
    cart.value.push({
      product_id: product.id,
      sku_id: skuData.id,
      name: product.name,
      sku_code: skuData.sku,
      unit_price: skuData.selling_price,
      quantity: 1,
      stock: skuData.stock,
      original_stock: skuData.stock,
    });
  }
}

function updateQty(item: CartItem, delta: number) {
  const newQty = item.quantity + delta;
  if (newQty <= 0) {
    cart.value = cart.value.filter(i => i.sku_id !== item.sku_id);
  } else if (newQty <= item.stock) {
    item.quantity = newQty;
  } else {
    ui.addToast('error', t('admin.pos.toastStockIssue', { name: item.name }));
  }
}

function removeItem(item: CartItem) {
  cart.value = cart.value.filter(i => i.sku_id !== item.sku_id);
  ui.addToast('success', t('admin.pos.toastRemoved'));
}

/** Calculate remaining stock for a SKU after cart deductions */
function skuRemainingStock(skuId: string, originalStock: number): number {
  const inCart = cart.value.filter(i => i.sku_id === skuId).reduce((sum, i) => sum + i.quantity, 0);
  return originalStock - inCart;
}

function clearCart() {
  cart.value = [];
  discount.value = 0;
  taxRate.value = 11;
  selectedCustomerId.value = '';
}

function openPayment() {
  if (cart.value.length === 0) return;
  // Biarkan amountPaid kosong agar user bisa input sendiri
  amountPaid.value = 0;
  paymentMethod.value = 'CASH';
  paymentOpen.value = true;
  loadCustomers();
}

async function loadCustomers() {
  if (customers.value.length > 0) return;
  customersLoading.value = true;
  try {
    const res = await customerService.list({ is_active: 'true', page: 1, per_page: 100 });
    customers.value = res.customers ?? [];
  } catch {
    customers.value = [];
  } finally {
    customersLoading.value = false;
  }
}

async function submitSale() {
  // Gate: never send a sale while the shift is not OPEN (server enforces it
  // too — this keeps the cart intact and lets the cashier re-try after
  // opening/re-opening the shift).
  if (!(await ensureShiftOpen())) {
    ui.addToast('error', t('posShift.notOpened'));
    return;
  }
  if (amountPaid.value < grandTotal.value) {
    ui.addToast('error', t('admin.pos.insufficient'));
    return;
  }
  const storeId = authStore.currentUser?.store_id;
  if (!storeId) {
    ui.addToast('error', 'No store assigned');
    return;
  }
  processing.value = true;
  try {
    await saleService.create({
      store_id: storeId,
      customer_id: selectedCustomerId.value && selectedCustomerId.value !== NO_CUSTOMER ? selectedCustomerId.value : undefined,
      items: cart.value.map(i => ({
        product_id: i.product_id,
        sku_id: i.sku_id,
        quantity: i.quantity,
        unit_price: i.unit_price,
      })),
      discount: discount.value,
      tax_rate: taxRate.value,
      payment_method: paymentMethod.value,
      amount_paid: amountPaid.value,
    });
    ui.addToast('success', t('admin.pos.toastSaleCreated'));
    paymentOpen.value = false;
    clearCart();
    searchProducts(); // Refresh stock levels shown in the product list
    loadSummary(); // Live sync: totals/items in the shift status bar
    loadShiftItems();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
    // The shift may have been closed from another tab/device while the
    // payment dialog was open — re-sync so the POS shows the lock screen
    // (the cart is intentionally kept).
    if (/open a shift|no open shift|shift is closed|already closed/i.test(apiMessage(err))) {
      await refreshLive();
    }
  } finally {
    processing.value = false;
  }
}

watch(isReady, async (r) => {
  if (r) {
    // Resolve the shift state before showing the checkout area so there is no
    // flash of "checkout allowed" while the shift is actually closed.
    checkingShift.value = true;
    try {
      await refreshLive();
      if (currentShift.value) searchProducts();
    } finally {
      checkingShift.value = false;
    }
    searchInputRef.value?.focus();
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
      <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: 'access POS' }) }}</p>
    </div>

    <!-- Shift state -->
    <div v-else-if="checkingShift" class="flex items-center justify-center h-[60vh]">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
    </div>

    <!-- No open shift → POS locked -->
    <div v-else-if="!currentShift" class="flex flex-col items-center justify-center min-h-[calc(100vh-10rem)] bg-card rounded-xl border">
      <div class="flex flex-col items-center text-center max-w-md px-6">
        <div class="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="h-7 w-7 text-muted-foreground"><circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" /></svg>
        </div>
        <h3 class="mt-4 text-xl font-semibold">{{ t('posShift.notOpened') }}</h3>
        <p class="mt-2 text-sm text-muted-foreground">{{ t('posShift.notOpenedDesc') }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('posShift.checkoutLocked') }}</p>
        <Button class="mt-6" @click="openOpenShift">{{ t('posShift.openShift') }}</Button>
      </div>
    </div>

    <!-- Open shift → POS usable -->
    <div v-else class="flex flex-col gap-4 min-h-[calc(100vh-8rem)]">
      <!-- Shift status bar (live numbers synced with transactions) -->
      <Card>
        <CardContent class="p-3 flex flex-wrap items-center gap-x-6 gap-y-2">
          <div class="flex items-center gap-2">
            <Badge variant="default">{{ t('posShift.shiftStatusOpen') }}</Badge>
            <span class="text-sm text-muted-foreground">{{ t('posShift.openedSince', { time: formatDateTime(currentShift.opened_at) }) }}</span>
          </div>
          <div class="text-sm">
            <span class="text-muted-foreground">{{ t('posShift.totalTransactions') }}:</span>
            <span class="ml-1 font-semibold tabular-nums">{{ summary?.total_transactions ?? 0 }}</span>
          </div>
          <div class="text-sm">
            <span class="text-muted-foreground">{{ t('posShift.totalSalesAmount') }}:</span>
            <span class="ml-1 font-semibold tabular-nums">{{ formatCurrency(summary?.total_sales_amount ?? 0) }}</span>
          </div>
          <div class="text-sm">
            <span class="text-muted-foreground">{{ t('posShift.cashSales') }}:</span>
            <span class="ml-1 font-semibold tabular-nums">{{ formatCurrency(summary?.cash_sales ?? 0) }}</span>
          </div>
          <div class="text-sm">
            <span class="text-muted-foreground">{{ t('posShift.itemsOut') }}:</span>
            <span class="ml-1 font-semibold tabular-nums">{{ summary?.items_out_total ?? 0 }}</span>
          </div>
          <div class="text-sm">
            <span class="text-muted-foreground">{{ t('posShift.itemsIn') }}:</span>
            <span class="ml-1 font-semibold tabular-nums">{{ summary?.items_in_total ?? 0 }}</span>
          </div>
          <Button variant="outline" size="sm" class="ml-auto" @click="refreshLive">
            {{ t('posShift.refresh') }}
          </Button>
          <Button variant="destructive" @click="openCloseDialog">{{ t('posShift.closeShift') }}</Button>
        </CardContent>
      </Card>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 flex-1">
        <!-- Left: Search & Product List -->
        <div class="lg:col-span-2 flex flex-col gap-4">
          <div class="flex gap-2">
            <div class="flex-1">
              <Input
                ref="searchInputRef"
                v-model="searchQuery"
                :placeholder="t('admin.pos.searchPlaceholder')"
                @keyup.enter="searchProducts"
                class="text-base h-11"
              />
            </div>
            <Button @click="searchProducts" class="h-11 px-6">{{ t('admin.shared.search') }}</Button>
          </div>

          <div class="flex gap-2">
            <Input
              v-model="barcodeInput"
              placeholder="Barcode / SKU"
              class="flex-1"
              @keyup.enter="lookupBarcode"
            />
            <Button variant="outline" @click="lookupBarcode">{{ t('admin.products.scanBarcode') }}</Button>
          </div>

          <Card>
            <CardContent class="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('admin.pos.productName') }}</TableHead>
                    <TableHead>SKU</TableHead>
                    <TableHead class="text-right">{{ t('admin.pos.price') }}</TableHead>
                    <TableHead class="text-right">{{ t('admin.pos.stock') }}</TableHead>
                    <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-if="searching">
                    <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
                  </TableRow>
                  <template v-else>
                    <TableRow v-if="searchResults.filter(p => p.skus && p.skus.length > 0).length === 0">
                      <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">
                        {{ t('admin.pos.noProducts') }}
                      </TableCell>
                    </TableRow>
                    <template v-for="p in searchResults.filter(p => p.skus && p.skus.length > 0)" :key="p.id">
                      <!-- Product header row -->
                      <TableRow class="bg-muted/30">
                        <TableCell colspan="5" class="font-semibold">{{ p.name }}</TableCell>
                      </TableRow>
                      <!-- SKU rows -->
                      <TableRow v-for="sku in (p.skus ?? [])" :key="sku.id" class="hover:bg-muted/50">
                        <TableCell class="pl-8">
                          <span class="text-muted-foreground text-xs">{{ sku.sku }}</span>
                          <span v-if="sku.variant" class="text-xs text-muted-foreground ml-1">· {{ sku.variant }}</span>
                          <span v-if="sku.size" class="text-xs text-muted-foreground ml-1">· {{ sku.size }}</span>
                        </TableCell>
                        <TableCell></TableCell>
                        <TableCell class="text-right tabular-nums text-sm">{{ formatCurrency(sku.selling_price) }}</TableCell>
                        <TableCell class="text-right tabular-nums">
                          <Badge :variant="skuRemainingStock(sku.id, sku.stock) > 0 ? 'default' : 'destructive'">
                            {{ skuRemainingStock(sku.id, sku.stock) }}
                          </Badge>
                        </TableCell>
                        <TableCell class="text-right">
                          <Button
                            size="sm"
                            :disabled="skuRemainingStock(sku.id, sku.stock) <= 0"
                            @click="addToCart(p, sku)"
                          >{{ t('admin.pos.addToCart') }}</Button>
                        </TableCell>
                      </TableRow>
                      <!-- No SKUs fallback -->
                      <TableRow v-if="!p.skus?.length">
                        <TableCell colspan="5" class="pl-8 text-sm text-muted-foreground">No SKUs</TableCell>
                      </TableRow>
                    </template>
                  </template>
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>

        <!-- Right: Cart -->
        <div class="flex flex-col gap-4">
          <Card class="flex-1 flex flex-col">
            <CardHeader class="flex flex-row items-center justify-between pb-2">
              <CardTitle class="text-lg">{{ t('admin.pos.cart') }}</CardTitle>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" @click="clearCart" :disabled="cart.length === 0">{{ t('admin.pos.clearCart') }}</Button>
              </div>
            </CardHeader>
            <CardContent class="flex-1 flex flex-col p-0">
              <div v-if="cart.length === 0" class="flex items-center justify-center flex-1 py-12 text-muted-foreground text-sm">
                {{ t('admin.pos.emptyCart') }}
              </div>
              <div v-else class="flex-1 overflow-y-auto max-h-[40vh]">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{{ t('admin.pos.productName') }}</TableHead>
                      <TableHead class="text-center">{{ t('admin.pos.qty') }}</TableHead>
                      <TableHead class="text-right">{{ t('admin.pos.subtotal') }}</TableHead>
                      <TableHead class="w-10"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow v-for="item in cart" :key="item.sku_id">
                      <TableCell>
                        <div class="font-medium text-sm">{{ item.name }}</div>
                        <div class="text-xs text-muted-foreground">{{ item.sku_code }} · {{ formatCurrency(item.unit_price) }}</div>
                      </TableCell>
                      <TableCell>
                        <div class="flex items-center justify-center gap-1">
                          <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="updateQty(item, -1)">−</Button>
                          <span class="w-8 text-center text-sm tabular-nums">{{ item.quantity }}</span>
                          <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="updateQty(item, 1)" :disabled="item.quantity >= item.stock">+</Button>
                        </div>
                      </TableCell>
                      <TableCell class="text-right text-sm tabular-nums">{{ formatCurrency(item.unit_price * item.quantity) }}</TableCell>
                      <TableCell>
                        <Button variant="ghost" size="sm" class="h-7 w-7 p-0 text-destructive" @click="removeItem(item)">×</Button>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>

          <!-- Totals & Payment -->
          <Card>
            <CardContent class="p-4 space-y-3">
              <div class="space-y-2">
                <div class="flex justify-between text-sm">
                  <span class="text-muted-foreground">{{ t('admin.pos.subtotal') }}</span>
                  <span class="tabular-nums">{{ formatCurrency(subtotal) }}</span>
                </div>
                <div class="flex items-center justify-between text-sm">
                  <span class="text-muted-foreground">{{ t('admin.pos.discount') }}</span>
                  <Input v-model.number="discount" type="number" min="0" class="w-28 h-8 text-right text-sm tabular-nums" />
                </div>
                <div class="flex items-center justify-between text-sm">
                  <span class="text-muted-foreground">{{ t('admin.pos.taxRate') }}</span>
                  <Input v-model.number="taxRate" type="number" min="0" max="100" class="w-20 h-8 text-right text-sm tabular-nums" />
                </div>
                <div class="flex justify-between text-sm">
                  <span class="text-muted-foreground">{{ t('admin.pos.taxRate') }}</span>
                  <span class="tabular-nums">{{ formatCurrency(taxAmount) }}</span>
                </div>
                <div class="border-t pt-2 flex justify-between font-semibold">
                  <span>{{ t('admin.pos.total') }}</span>
                  <span class="tabular-nums text-lg">{{ formatCurrency(grandTotal) }}</span>
                </div>
              </div>

              <Button class="w-full h-11 text-base" :disabled="cart.length === 0" @click="openPayment">
                {{ t('admin.pos.pay') }} {{ formatCurrency(grandTotal) }}
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>

    <!-- Open Shift Dialog -->
    <Dialog v-model:open="openDialog">
      <DialogContent class="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ t('posShift.openShiftTitle') }}</DialogTitle>
          <DialogDescription>{{ t('posShift.openShiftDesc') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-2">
          <Label>{{ t('posShift.openingCashLabel') }}</Label>
          <Input v-model.number="openForm.opening_cash" type="number" min="0" :placeholder="t('posShift.placeholderCash')" />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="openDialog = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="opening" @click="submitOpenShift">
            {{ opening ? t('posShift.opening') : t('posShift.openShift') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Close Shift Dialog -->
    <Dialog v-model:open="closeDialog">
      <DialogContent class="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('posShift.closeShiftTitle') }}</DialogTitle>
          <DialogDescription>{{ t('posShift.closeShiftDesc') }}</DialogDescription>
        </DialogHeader>

        <div v-if="summaryLoading" class="text-center py-10 text-muted-foreground">{{ t('admin.shared.loading') }}</div>
        <template v-else-if="summary">
          <!-- Reconciliation (server numbers) -->
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 text-sm rounded-lg bg-muted/40 p-3">
            <div>
              <div class="text-muted-foreground">{{ t('posShift.openingCashLabel') }}</div>
              <div class="font-medium tabular-nums">{{ formatCurrency(summary.opening_cash) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('posShift.cashSales') }}</div>
              <div class="font-medium tabular-nums">{{ formatCurrency(summary.cash_sales) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('posShift.cashRefund') }}</div>
              <div class="font-medium tabular-nums">{{ formatCurrency(summary.cash_refund) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('posShift.totalTransactions') }}</div>
              <div class="font-medium tabular-nums">{{ summary.total_transactions }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('posShift.totalSalesAmount') }}</div>
              <div class="font-medium tabular-nums">{{ formatCurrency(summary.total_sales_amount) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('posShift.expectedCash') }}</div>
              <div class="font-medium tabular-nums">{{ formatCurrency(summary.expected_cash) }}</div>
            </div>
          </div>

          <!-- Cashier inputs -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4">
            <div class="space-y-1">
              <Label>{{ t('posShift.actualCashLabel') }}</Label>
              <Input v-model.number="closeActual" type="number" min="0" class="text-lg h-11 tabular-nums" :placeholder="t('posShift.actualPlaceholder')" />
            </div>
            <div class="space-y-1">
              <Label>{{ t('posShift.closingCashLabel') }}</Label>
              <Input v-model.number="closeClosing" type="number" min="0" class="text-lg h-11 tabular-nums" :placeholder="t('posShift.placeholderCash')" />
            </div>
          </div>

          <!-- Difference preview -->
          <div class="rounded-lg border p-3 mt-3 flex items-center justify-between">
            <div>
              <div class="text-sm font-medium">{{ t('posShift.cashDifferenceLabel') }}</div>
              <div class="text-xs text-muted-foreground">{{ t('posShift.differenceHint') }}</div>
            </div>
            <span
              class="text-xl font-bold tabular-nums"
              :class="differencePreview === 0 ? '' : (differencePreview > 0 ? 'text-emerald-600' : 'text-destructive')"
            >{{ differencePreview > 0 ? '+' : '' }}{{ formatCurrency(differencePreview) }}</span>
          </div>

          <!-- Items moved (goods OUT/IN) -->
          <div class="mt-4">
            <div class="flex items-center justify-between mb-2">
              <h4 class="font-semibold text-sm">{{ t('posShift.itemDetail') }}</h4>
              <div class="flex gap-1">
                <Button variant="outline" size="sm" :class="itemsDirection === 'OUT' ? 'bg-accent' : ''" @click="itemsDirection = 'OUT'">
                  {{ t('posShift.outItems') }} ({{ summary.items_out_total ?? 0 }})
                </Button>
                <Button variant="outline" size="sm" :class="itemsDirection === 'IN' ? 'bg-accent' : ''" @click="itemsDirection = 'IN'">
                  {{ t('posShift.inItems') }} ({{ summary.items_in_total ?? 0 }})
                </Button>
              </div>
            </div>
            <div v-if="itemsLoading" class="text-center py-6 text-muted-foreground text-sm">{{ t('admin.shared.loading') }}</div>
            <div v-else-if="(itemsDirection === 'OUT' ? itemsOut : itemsIn).length === 0" class="text-center py-6 text-muted-foreground text-sm">
              {{ t('posShift.noItems') }}
            </div>
            <Table v-else>
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('posShift.product') }}</TableHead>
                  <TableHead class="text-right">{{ t('posShift.qty') }}</TableHead>
                  <TableHead class="text-right">{{ t('posShift.amount') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="it in (itemsDirection === 'OUT' ? itemsOut : itemsIn)" :key="it.id">
                  <TableCell>
                    <div class="text-sm font-medium">{{ it.product_name }}</div>
                    <div v-if="it.sku_code || it.variant_label" class="text-xs text-muted-foreground">
                      <span v-if="it.sku_code">{{ it.sku_code }}</span><span v-if="it.variant_label"> · {{ it.variant_label }}</span>
                    </div>
                    <div v-if="it.reference_no" class="text-xs text-muted-foreground font-mono">{{ it.reference_no }}</div>
                  </TableCell>
                  <TableCell class="text-right tabular-nums">{{ it.quantity }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ formatCurrency(it.line_amount) }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </template>

        <DialogFooter class="mt-4">
          <Button variant="outline" @click="closeDialog = false">{{ t('admin.shared.cancel') }}</Button>
          <Button variant="destructive" :disabled="closing || !summary" @click="askClose">
            {{ closing ? t('posShift.closing') : t('posShift.close') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Confirm close with difference -->
    <Dialog v-model:open="diffConfirm">
      <DialogContent class="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ t('posShift.confirmTitle') }}</DialogTitle>
          <DialogDescription>
            {{ t('posShift.confirmDescription', { amount: formatCurrency(differencePreview) }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="diffConfirm = false">{{ t('admin.shared.cancel') }}</Button>
          <Button variant="destructive" :disabled="closing" @click="performClose">
            {{ closing ? t('posShift.closing') : t('posShift.close') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Closed result summary -->
    <Dialog v-model:open="resultDialog">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('posShift.resultTitle') }}</DialogTitle>
          <DialogDescription>{{ t('posShift.summaryTitle') }}</DialogDescription>
        </DialogHeader>
        <div v-if="closedResult" class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div class="text-muted-foreground">{{ t('posShift.totalTransactions') }}</div>
            <div class="font-medium tabular-nums">{{ closedResult.total_transactions }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.totalSalesAmount') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(closedResult.total_sales_amount) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.expectedCash') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(closedResult.expected_cash) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.actualCashLabel') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(closedResult.actual_cash) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.cashDifferenceLabel') }}</div>
            <div class="font-medium tabular-nums" :class="closedResult.cash_difference !== 0 ? (closedResult.cash_difference > 0 ? 'text-emerald-600' : 'text-destructive') : ''">
              {{ closedResult.cash_difference > 0 ? '+' : '' }}{{ formatCurrency(closedResult.cash_difference) }}
            </div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.itemsOut') }}</div>
            <div class="font-medium tabular-nums">{{ closedResult.items_out_total ?? 0 }}</div>
          </div>
        </div>
        <DialogFooter>
          <Button @click="resultDialog = false">{{ t('posShift.closeResult') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Payment Dialog -->
    <Dialog v-model:open="paymentOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('admin.pos.paymentTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.pos.paymentDescription') }}</DialogDescription>
        </DialogHeader>

        <div class="space-y-4">
          <div class="space-y-2">
            <Label>{{ t('admin.pos.customer') }}</Label>
            <Select v-model="selectedCustomerId">
              <SelectTrigger class="w-full"><SelectValue :placeholder="t('admin.pos.noCustomer')" /></SelectTrigger>
              <SelectContent>
                <SelectItem :value="NO_CUSTOMER">{{ t('admin.pos.noCustomer') }}</SelectItem>
                <SelectItem v-for="c in customers" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label>{{ t('admin.pos.paymentMethod') }}</Label>
            <Select v-model="paymentMethod">
              <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="CASH">Cash</SelectItem>
                <SelectItem value="CARD">Card</SelectItem>
                <SelectItem value="BANK_TRANSFER">Bank Transfer</SelectItem>
                <SelectItem value="EWALLET">E-Wallet</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="rounded-lg bg-muted p-4 text-center">
            <div class="text-sm text-muted-foreground">{{ t('admin.pos.grandTotal') }}</div>
            <div class="text-2xl font-bold tabular-nums">{{ formatCurrency(grandTotal) }}</div>
          </div>

          <div class="space-y-2">
            <Label>{{ t('admin.pos.amountPaid') }}</Label>
            <Input v-model="amountPaid" type="number" min="0" placeholder="0" class="text-lg h-11 tabular-nums" @update:modelValue="setAmountPaid" />
          </div>

          <div v-if="amountPaid > 0 && amountPaid < grandTotal" class="text-center text-destructive font-medium">
            {{ t('admin.pos.insufficient') }}
          </div>
          <div v-else-if="amountPaid === 0" class="text-center text-muted-foreground text-sm">
            {{ t('admin.pos.inputAmountToSeeChange') }}
          </div>
          <div v-else class="text-center text-emerald-600 font-medium">
            {{ t('admin.pos.change') }}: {{ formatCurrency(change) }}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="paymentOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="processing || amountPaid < grandTotal" @click="submitSale">
            {{ processing ? t('admin.pos.processing') : t('admin.pos.pay') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </DashboardShell>
</template>
