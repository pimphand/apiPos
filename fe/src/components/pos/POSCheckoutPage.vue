<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { productService } from '@/services/product';
import { customerService } from '@/services/customer';
import { saleService } from '@/services/sale';
import { cashierService } from '@/services/cashier';
import { settingsService } from '@/services/settings';
import { useCashierShift } from '@/composables/useCashierShift';
import { useHeldOrders } from '@/composables/useHeldOrders';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import { HOLD_LIMIT } from '@/lib/posHold';
import type {
  Product,
  ProductSKU,
  Customer,
  ShiftItem,
  ShiftSummary,
  HeldOrder,
  HeldOrderItem,
  StoreSettings,
} from '@/types';
import type { ReceiptData } from '@/components/pos/PosReceiptPaper.vue';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import PosHeldOrders from '@/components/pos/PosHeldOrders.vue';
import PosPaymentDialog from '@/components/pos/PosPaymentDialog.vue';
import PosReceiptDialog from '@/components/pos/PosReceiptDialog.vue';
import PosProductBrowser from '@/components/pos/PosProductBrowser.vue';
import PosShiftDialogs from '@/components/pos/PosShiftDialogs.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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

/** Gates the very first paint until the shift state is resolved. */
const checkingShift = ref(true);

/** Template refs to the child views that own the shift-dialog / product UI. */
const shiftDialogsRef = ref<InstanceType<typeof PosShiftDialogs> | null>(null);
const productBrowserRef = ref<InstanceType<typeof PosProductBrowser> | null>(null);

async function refreshLive() {
  await refreshShift();
  if (currentShift.value) {
    await loadSummary();
  }
}

/** PosShiftDialogs → open the shift; refreshes POS state on success. */
async function handleOpenShift(openingCash: number): Promise<boolean> {
  try {
    await openShift(openingCash);
    ui.addToast('success', t('posShift.toastOpened'));
    await refreshLive();
    searchProducts(); // product list only renders while a shift is open
    return true;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
    return false;
  }
}

/** PosShiftDialogs → close the shift (composable handles idempotency). */
async function handleCloseShift(
  shiftId: string,
  actualCash: number,
  closingCash: number
): Promise<ShiftSummary> {
  return closeShift(shiftId, actualCash, closingCash);
}

/** PosShiftDialogs → OUT/IN item ledger rows for the close dialog. */
async function handleFetchShiftItems(
  shiftId: string
): Promise<{ out: ShiftItem[]; in: ShiftItem[] }> {
  try {
    const [outRes, inRes] = await Promise.all([
      cashierService.listShiftItems({ id: shiftId, direction: 'OUT', page: 1, per_page: 50 }),
      cashierService.listShiftItems({ id: shiftId, direction: 'IN', page: 1, per_page: 50 }),
    ]);
    return { out: outRes.items ?? [], in: inRes.items ?? [] };
  } catch {
    // Items are auxiliary; ignore failures so the dialogs still work.
    return { out: [], in: [] };
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

const cart = ref<CartItem[]>([]);
const discount = ref(0);
const taxRate = ref(11);
const selectedCustomerId = ref('');
const NO_CUSTOMER = 'none'; // reka-ui disallows empty-string SelectItem values
const customers = ref<Customer[]>([]);
const customersLoading = ref(false);

const paymentOpen = ref(false);
const processing = ref(false);

// ---- Store settings (docs/pos-store-settings.md) -------------------------
const settings = ref<StoreSettings | null>(null);
/** Default tax % for fresh carts — from store settings, fallback 11. */
const defaultTaxRate = computed(() => {
  const v = settings.value?.tax_rate;
  return typeof v === 'number' && Number.isFinite(v) ? v : 11;
});

/** Last successful sale → receipt/struk dialog after payment. */
const receiptOpen = ref(false);
const lastReceipt = ref<ReceiptData | null>(null);

/** Map payment-method code to a localized label (reuses admin.payments keys). */
const PAY_METHOD_KEYS: Record<string, string> = {
  CASH: 'cash',
  CARD: 'card',
  BANK_TRANSFER: 'bankTransfer',
  EWALLET: 'ewallet',
  OTHER: 'other',
};

async function loadStoreSettings() {
  try {
    settings.value = await settingsService.get();
  } catch {
    // Best-effort only — POS keeps working with the built-in 11% default.
    settings.value = null;
  }
}

const subtotal = computed(() => cart.value.reduce((sum, item) => sum + item.unit_price * item.quantity, 0));
const taxAmount = computed(() => Math.round(subtotal.value * (taxRate.value / 100)));
const grandTotal = computed(() => subtotal.value + taxAmount.value - discount.value);
/** SKU → total quantity in the cart (live stock badges in PosProductBrowser). */
const cartQuantities = computed(() => {
  const quantities: Record<string, number> = {};
  for (const i of cart.value) quantities[i.sku_id] = (quantities[i.sku_id] ?? 0) + i.quantity;
  return quantities;
});

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

async function lookupBarcode(code: string) {
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

function clearCart() {
  cart.value = [];
  discount.value = 0;
  taxRate.value = defaultTaxRate.value;
  selectedCustomerId.value = '';
}

// ---- Held orders (docs/pos-hold-order.md) --------------------------------
const heldOrdersApi = useHeldOrders();
const { holds, holdsCount } = heldOrdersApi;

const restoring = ref(false);
/** Single-flight guard so two rapid clicks cannot hold the same cart twice. */
let holdingCart = false;

/** Snapshot of the current cart lines (mirrors CartItem shape). */
const holdableCartItems = computed<HeldOrderItem[]>(() =>
  cart.value.map((i) => ({
    product_id: i.product_id,
    sku_id: i.sku_id,
    name: i.name,
    sku_code: i.sku_code,
    unit_price: i.unit_price,
    quantity: i.quantity,
    stock: i.stock,
    original_stock: i.original_stock,
  }))
);

/**
 * Hold the current cart (R2–R3): snapshot → save → clear cart so the cashier
 * can serve the next customer. Returns false when nothing was held.
 */
async function holdCurrentCart(opts: { silent?: boolean } = {}): Promise<boolean> {
  if (cart.value.length === 0) {
    if (!opts.silent) ui.addToast('error', t('posHold.toastHeldCartEmpty'));
    return false;
  }
  if (holdsCount.value >= HOLD_LIMIT) {
    ui.addToast('error', t('posHold.toastLimit', { limit: HOLD_LIMIT }));
    return false;
  }
  if (holdingCart) return false;
  holdingCart = true;
  try {
    // Best-effort resolve the customer name for the hold label.
    const hasCustomer =
      selectedCustomerId.value && selectedCustomerId.value !== NO_CUSTOMER;
    let customerName: string | undefined;
    if (hasCustomer) {
      customerName = customers.value.find((c) => c.id === selectedCustomerId.value)?.name;
      if (!customerName && customers.value.length === 0 && !customersLoading.value) {
        await loadCustomers();
        customerName = customers.value.find((c) => c.id === selectedCustomerId.value)?.name;
      }
    }
    const n = holdsCount.value + 1;
    const label = customerName
      ? t('posHold.holdLabelWithCustomer', { n, customer: customerName })
      : t('posHold.holdLabel', { n });
    heldOrdersApi.holdCart({
      label,
      items: holdableCartItems.value,
      discount: discount.value,
      tax_rate: taxRate.value,
      customer_id: hasCustomer ? selectedCustomerId.value : '',
      customer_name: customerName,
    });
    clearCart();
    if (!opts.silent) {
      ui.addToast('success', t('posHold.toastHeld'));
    }
    return true;
  } finally {
    holdingCart = false;
  }
}

/**
 * --- Actions triggered by <PosHeldOrders> ------------------------------
 * The component only asks; cart/hold business logic stays here so the POS page
 * remains the single owner of the cart and the shift state.
 */
function handleRestoreHeld(h: HeldOrder) {
  void performRestore(h); // child guarantees the cart is empty here
}

async function handleHoldCurrentAndRestore(h: HeldOrder) {
  const held = await holdCurrentCart({ silent: true });
  if (!held) return; // limit reached — current cart preserved, error toasted
  await performRestore(h);
}

async function handleReplaceAndRestore(h: HeldOrder) {
  clearCart();
  await performRestore(h);
}

function handleDeleteHeld(h: HeldOrder) {
  if (heldOrdersApi.deleteHold(h.id)) {
    ui.addToast('success', t('posHold.toastDeleted'));
  }
}

/**
 * Live stock/availability revalidation before filling the cart (spec §5.6,
 * R6). Best-effort: if the refresh fails (offline), items are restored from
 * the snapshot and the server still validates stock at payment time.
 */
async function resolveRestoredItems(items: HeldOrderItem[]): Promise<{
  items: HeldOrderItem[];
  notices: { type: 'error' | 'warning' | 'info'; text: string }[];
}> {
  const notices: { type: 'error' | 'warning' | 'info'; text: string }[] = [];
  const uniqueProductIds = [...new Set(items.map((i) => i.product_id))];
  const productsById = new Map<string, Product>();
  try {
    const settled = await Promise.allSettled(
      uniqueProductIds.map((id) => productService.get(id, true))
    );
    settled.forEach((res, idx) => {
      if (res.status === 'fulfilled') {
        productsById.set(uniqueProductIds[idx], res.value);
      }
    });
  } catch {
    // No refresh at all → fall back to snapshot below.
  }
  const liveKnown = productsById.size > 0;
  const resolved: HeldOrderItem[] = [];
  for (const it of items) {
    const product = productsById.get(it.product_id);
    const sku = product?.skus?.find((s) => s.id === it.sku_id);
    if (liveKnown && (!product || !sku)) {
      // The product/SKU no longer exists → drop the line (can't be paid).
      notices.push({ type: 'error', text: t('posHold.itemUnavailable', { name: it.name }) });
      continue;
    }
    if (product && sku) {
      if (!product.is_active || !sku.is_active || Number(sku.stock) <= 0) {
        notices.push({ type: 'error', text: t('posHold.itemUnavailable', { name: it.name }) });
        continue;
      }
      const liveStock = Number(sku.stock) || 0;
      if (it.quantity > liveStock) {
        notices.push({
          type: 'warning',
          text: t('posHold.stockAdjusted', {
            name: it.name,
            old: it.quantity,
            new: liveStock,
          }),
        });
        resolved.push({ ...it, quantity: liveStock, stock: liveStock, original_stock: liveStock });
      } else {
        resolved.push({ ...it, stock: liveStock, original_stock: liveStock });
      }
      continue;
    }
    // No live data (offline) → restore from snapshot as-is.
    resolved.push({ ...it });
  }
  return { items: resolved, notices };
}

function pushItemsToCart(items: HeldOrderItem[]) {
  for (const it of items) {
    const existing = cart.value.find((i) => i.sku_id === it.sku_id);
    if (existing) {
      existing.quantity += it.quantity;
    } else {
      cart.value.push({
        product_id: it.product_id,
        sku_id: it.sku_id,
        name: it.name,
        sku_code: it.sku_code,
        unit_price: it.unit_price,
        quantity: it.quantity,
        stock: it.stock,
        original_stock: it.original_stock,
      });
    }
  }
}

async function performRestore(h: HeldOrder) {
  restoring.value = true;
  try {
    const order = heldOrdersApi.restoreHold(h.id);
    if (!order) return; // already restored/removed in another tab
    const { items, notices } = await resolveRestoredItems(order.items);
    for (const n of notices) ui.addToast(n.type, n.text);
    pushItemsToCart(items);
    discount.value = order.discount || 0;
    taxRate.value = order.tax_rate;
    selectedCustomerId.value = order.customer_id || '';
    ui.addToast('success', t('posHold.toastRestored'));
  } finally {
    restoring.value = false;
  }
}

function openPayment() {
  if (cart.value.length === 0) return;
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

async function submitSale(payload: {
  customer_id: string;
  payment_method: string;
  amount_paid: number;
}) {
  // Gate: never send a sale while the shift is not OPEN (server enforces it
  // too — this keeps the cart intact and lets the cashier re-try after
  // opening/re-opening the shift).
  if (!(await ensureShiftOpen())) {
    ui.addToast('error', t('posShift.notOpened'));
    return;
  }
  if (payload.amount_paid < grandTotal.value) {
    ui.addToast('error', t('admin.pos.insufficient'));
    return;
  }
  const storeId = authStore.currentUser?.store_id;
  if (!storeId) {
    ui.addToast('error', 'No store assigned');
    return;
  }
  // Snapshot the sale line for the struk before the cart is cleared.
  const saleLines = cart.value.map(i => ({
    name: i.name,
    sku_code: i.sku_code,
    quantity: i.quantity,
    unit_price: i.unit_price,
  }));
  const customer = customers.value.find(c => c.id === payload.customer_id);
  const paidAmount = payload.amount_paid;

  processing.value = true;
  try {
    const detail = await saleService.create({
      store_id: storeId,
      customer_id: payload.customer_id && payload.customer_id !== NO_CUSTOMER ? payload.customer_id : undefined,
      items: cart.value.map(i => ({
        product_id: i.product_id,
        sku_id: i.sku_id,
        quantity: i.quantity,
        unit_price: i.unit_price,
      })),
      discount: discount.value,
      tax_rate: taxRate.value,
      payment_method: payload.payment_method,
      amount_paid: paidAmount,
    });
    ui.addToast('success', t('admin.pos.toastSaleCreated'));
    paymentOpen.value = false;

    // Server numbers are authoritative; fall back to local math when missing.
    const subtotalN = Number(detail.sale.subtotal) || subtotal.value;
    const discountN = detail.sale.discount !== undefined && Number(detail.sale.discount) ? Number(detail.sale.discount) : discount.value;
    const taxN = detail.sale.tax !== undefined && Number(detail.sale.tax) ? Number(detail.sale.tax) : taxAmount.value;
    const totalN = Number(detail.sale.grand_total) || grandTotal.value;
    const methodLabelKey = PAY_METHOD_KEYS[payload.payment_method] ?? 'other';
    const storeAny = authStore.currentUser as any;

    lastReceipt.value = {
      store_name: storeAny?.store_name || storeAny?.store?.name || '',
      store_address: storeAny?.store?.address || '',
      store_phone: storeAny?.store?.phone || '',
      invoice_number: detail.sale.invoice_number || '',
      created_at: detail.sale.created_at || new Date().toISOString(),
      cashier_name: authStore.userName,
      customer_name: payload.customer_id && payload.customer_id !== NO_CUSTOMER
        ? (customer?.name || '')
        : '',
      tax_rate: taxRate.value,
      payment_method_label: t(`admin.payments.${methodLabelKey}`),
      items: saleLines,
      subtotal: subtotalN,
      discount: discountN,
      tax: taxN,
      total: totalN,
      paid: paidAmount,
      change: Math.max(0, paidAmount - totalN),
    };

    clearCart();
    searchProducts(); // Refresh stock levels shown in the product list
    loadSummary(); // Live sync: totals in the shift status bar
    receiptOpen.value = true;
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
      await loadStoreSettings();
      // Fresh carts adopt the store's default tax rate (settings).
      if (cart.value.length === 0) taxRate.value = defaultTaxRate.value;
      await refreshLive();
      heldOrdersApi.reload();
      if (currentShift.value) searchProducts();
    } finally {
      checkingShift.value = false;
    }
    productBrowserRef.value?.focusSearch();
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
        <Button class="mt-6" @click="shiftDialogsRef?.openOpenShiftDialog()">{{ t('posShift.openShift') }}</Button>
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
          <Button variant="destructive" @click="shiftDialogsRef?.openCloseDialog()">{{ t('posShift.closeShift') }}</Button>
        </CardContent>
      </Card>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 flex-1">
        <!-- Left: Search & Product List (PosProductBrowser.vue) -->
        <div class="lg:col-span-2 flex flex-col gap-4">
          <PosProductBrowser
            ref="productBrowserRef"
            :products="searchResults"
            :searching="searching"
            :cart-quantities="cartQuantities"
            v-model:search-query="searchQuery"
            v-model:barcode-input="barcodeInput"
            @search="searchProducts"
            @lookup="lookupBarcode"
            @add="addToCart"
          />
        </div>

        <!-- Right: Cart -->
        <div class="flex flex-col gap-4">
          <Card class="flex-1 flex flex-col">
            <CardHeader class="flex flex-row items-center justify-between gap-2 pb-2">
              <CardTitle class="text-lg">{{ t('admin.pos.cart') }}</CardTitle>
              <div class="flex gap-1.5">
                <Button
                  variant="outline"
                  size="sm"
                  class="h-8 px-2.5"
                  :disabled="cart.length === 0"
                  @click="holdCurrentCart()"
                >{{ t('admin.pos.holdCart') }}</Button>
                <Button
                  variant="outline"
                  size="sm"
                  class="h-8 px-2.5"
                  :disabled="cart.length === 0"
                  @click="clearCart"
                >{{ t('admin.pos.clearCart') }}</Button>
              </div>
            </CardHeader>
            <!-- Held orders (docs/pos-hold-order.md) — UI lives in PosHeldOrders.vue -->
            <PosHeldOrders
              :orders="holds"
              :restoring="restoring"
              :cart-count="cart.length"
              @restore="handleRestoreHeld"
              @hold-current-and-restore="handleHoldCurrentAndRestore"
              @replace-and-restore="handleReplaceAndRestore"
              @delete="handleDeleteHeld"
            />
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

    <!-- Cashier shift dialogs (PosShiftDialogs.vue) -->
    <PosShiftDialogs
      ref="shiftDialogsRef"
      :current-shift="currentShift"
      :summary="summary"
      :summary-loading="summaryLoading"
      :opening="opening"
      :closing="closing"
      :open-shift-action="handleOpenShift"
      :close-shift-action="handleCloseShift"
      :load-summary-action="loadSummary"
      :fetch-shift-items-action="handleFetchShiftItems"
    />

    <!-- Payment dialog (PosPaymentDialog.vue) -->
    <PosPaymentDialog
      :open="paymentOpen"
      :customers="customers"
      :grand-total="grandTotal"
      :processing="processing"
      :customer-id="selectedCustomerId"
      @update:open="paymentOpen = $event"
      @update:customer-id="selectedCustomerId = $event"
      @cancel="paymentOpen = false"
      @submit="submitSale"
    />

    <!-- Receipt/struk dialog after a successful payment (PosReceiptDialog.vue) -->
    <PosReceiptDialog
      :open="receiptOpen"
      :settings="settings"
      :data="lastReceipt"
      @update:open="receiptOpen = $event"
      @new-sale="receiptOpen = false"
    />
  </DashboardShell>
</template>
