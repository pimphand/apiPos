<script setup lang="ts">
/**
 * Public self-order flow (docs/pos-self-order.md §8.1). Runs unauthenticated:
 * resolve table → menu → cart → submit (PENDING) → pay (PAID) → status.
 */
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { selfOrderService } from '@/services/selforder';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import type { PublicMenu, PublicProduct, PublicVariant, SelfOrder } from '@/types';

const props = defineProps<{ tableToken?: string }>();

const t = useLocaleStore().t;

// ---- State ---------------------------------------------------------------
type Stage = 'loading' | 'error' | 'menu' | 'paying' | 'processing' | 'success';

const stage = ref<Stage>('loading');
/** In-flight guards (button disabled states + labels). */
const submitting = ref(false);
const paying = ref(false);
const errorMessage = ref('');
const tableInfo = ref<{ code: string; name: string; store_name: string } | null>(null);
const menu = ref<PublicMenu | null>(null);
const activeCategory = ref<string>('all');

interface CartLine {
  product_id: string;
  product_name: string;
  sku_id: string;
  sku_code: string;
  variant_label: string;
  unit_price: number;
  quantity: number;
}

const cart = ref<CartLine[]>([]);
const customerName = ref('');
const notes = ref('');

const paymentMethod = ref('');
const paymentReference = ref('');
const order = ref<SelfOrder | null>(null);
const orderError = ref('');

const PAY_METHODS = ['EWALLET', 'CARD', 'BANK_TRANSFER', 'OTHER'] as const;
const methodKey = (m: string) =>
  ({ EWALLET: 'methodEwallet', CARD: 'methodCard', BANK_TRANSFER: 'methodBankTransfer', OTHER: 'methodOther' })[m] ?? 'methodOther';

// ---- Computed ------------------------------------------------------------
const subtotal = computed(() => cart.value.reduce((s, i) => s + i.unit_price * i.quantity, 0));
const tax = computed(() => Math.round(subtotal.value * 0.11));
const total = computed(() => subtotal.value + tax.value);

const filteredProducts = computed(() => {
  const list = menu.value?.products ?? [];
  if (activeCategory.value === 'all') return list;
  return list.filter((p) => p.category_id === activeCategory.value);
});

/** Variant label (variant / color / size). */
function variantLabel(v: PublicVariant): string {
  return [v.variant, v.color, v.size].filter(Boolean).join(' / ');
}

function addToCart(product: PublicProduct, variant?: PublicVariant) {
  // A product without variants is sold at product level (sku empty).
  const line: CartLine = {
    product_id: product.id,
    product_name: product.name,
    sku_id: variant?.sku_id ?? '',
    sku_code: variant?.sku_code ?? '',
    variant_label: variant ? variantLabel(variant) : '',
    unit_price: variant?.selling_price ?? 0,
    quantity: 1,
  };
  const existing = cart.value.find((i) => i.sku_id === line.sku_id && i.product_id === line.product_id);
  if (existing) {
    existing.quantity++;
  } else {
    cart.value.push(line);
  }
}

function changeQty(line: CartLine, delta: number) {
  const q = line.quantity + delta;
  if (q <= 0) {
    cart.value = cart.value.filter((i) => i !== line);
  } else {
    line.quantity = q;
  }
}

function clearCart() {
  cart.value = [];
  customerName.value = '';
  notes.value = '';
}

// ---- Flow ----------------------------------------------------------------
/** Resolve the table token from prop → query string → pretty path (/o/TOKEN). */
function resolveToken(): string {
  if (props.tableToken) return props.tableToken;
  if (typeof window !== 'undefined') {
    const q = new URLSearchParams(window.location.search).get('table');
    if (q) return q;
    const m = window.location.pathname.match(/^\/o\/([^/]+)/);
    if (m) return decodeURIComponent(m[1]);
  }
  return '';
}

async function boot() {
  const token = resolveToken();
  if (!token) {
    stage.value = 'error';
    errorMessage.value = t('selfOrder.errorLoading');
    return;
  }
  stage.value = 'loading';
  try {
    const info = await selfOrderService.resolveTable(token);
    if (!info.self_order_enabled) {
      stage.value = 'error';
      errorMessage.value = t('selfOrder.tableDisabled');
      return;
    }
    tableInfo.value = { code: info.code, name: info.name, store_name: info.store_name };
    const m = await selfOrderService.getMenu(token);
    menu.value = m;
    stage.value = 'menu';
  } catch {
    stage.value = 'error';
    errorMessage.value = t('selfOrder.errorLoading');
  }
}

async function submitOrder() {
  if (cart.value.length === 0) {
    orderError.value = t('selfOrder.requireItem');
    return;
  }
  orderError.value = '';
  submitting.value = true;
  try {
    order.value = await selfOrderService.createOrder({
      table_token: props.tableToken ?? '',
      items: cart.value.map((i) => ({
        product_id: i.product_id,
        product_sku_id: i.sku_id,
        quantity: i.quantity,
      })),
      customer_name: customerName.value.trim() || undefined,
      notes: notes.value.trim() || undefined,
    });
    stage.value = 'paying';
  } catch (err: any) {
    const msg = String(err?.response?.data?.message ?? err?.message ?? '');
    orderError.value = /stok|stock/i.test(msg) ? t('selfOrder.orderTooLarge') : msg || t('selfOrder.orderFailed');
  } finally {
    submitting.value = false;
  }
}

async function payOrder() {
  if (!paymentMethod.value) {
    orderError.value = t('selfOrder.requireMethod');
    return;
  }
  if (!order.value) {
    orderError.value = t('selfOrder.submitRequired');
    return;
  }
  orderError.value = '';
  paying.value = true;
  try {
    order.value = await selfOrderService.payOrder({
      order_token: order.value.order_token,
      payment_method: paymentMethod.value,
      reference: paymentReference.value.trim() || undefined,
    });
    stage.value = 'success';
    startStatusPolling();
  } catch (err: any) {
    const msg = String(err?.response?.data?.message ?? err?.message ?? '');
    orderError.value = msg || t('selfOrder.orderFailed');
  } finally {
    paying.value = false;
  }
}

// ---- Status polling (after payment) ---------------------------------------
let statusTimer: ReturnType<typeof setInterval> | null = null;

function startStatusPolling() {
  stopStatusPolling();
  statusTimer = setInterval(async () => {
    if (!order.value) return;
    try {
      order.value = await selfOrderService.getOrderStatus(order.value.order_token);
    } catch {
      // Keep last known status; polling continues.
    }
  }, 5000);
}

function stopStatusPolling() {
  if (statusTimer) {
    clearInterval(statusTimer);
    statusTimer = null;
  }
}

const orderStatusKey = computed(() => {
  const s = order.value?.order_status ?? 'ORDERED';
  return `selfOrder.orderStatus${s.charAt(0) + s.slice(1).toLowerCase()}`;
});

onMounted(boot);
onUnmounted(stopStatusPolling);
</script>

<template>
  <div class="min-h-screen bg-muted/40 pb-24">
    <!-- Header -->
    <header class="sticky top-0 z-10 border-b bg-background/95 backdrop-blur">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
        <div>
          <p class="text-xs text-muted-foreground">{{ tableInfo?.store_name || '' }}</p>
          <h1 class="text-lg font-bold leading-tight">
            {{ t('selfOrder.tableLabel', { code: tableInfo?.code || '' }) }}
          </h1>
        </div>
        <span v-if="menu?.table_name" class="text-sm text-muted-foreground">{{ menu.table_name }}</span>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="stage === 'loading'" class="flex items-center justify-center py-32">
      <div class="animate-spin rounded-full h-10 w-10 border-b-2 border-primary"></div>
    </div>

    <!-- Error -->
    <div v-else-if="stage === 'error'" class="mx-auto max-w-md px-4 py-24 text-center">
      <div class="text-4xl">📵</div>
      <p class="mt-4 font-semibold">{{ errorMessage }}</p>
      <Button class="mt-6" @click="boot">⟳ {{ t('common.loading') }}</Button>
    </div>

    <!-- Menu + cart -->
    <div v-else-if="stage === 'menu'" class="mx-auto grid max-w-5xl grid-cols-1 gap-4 px-4 pt-4 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <!-- Category filter -->
        <div class="mb-3 flex flex-wrap gap-2">
          <Button
            v-for="cat in menu?.categories"
            :key="cat.id"
            size="sm"
            :variant="activeCategory === cat.id ? 'default' : 'outline'"
            @click="activeCategory = cat.id"
          >
            {{ cat.name }}
          </Button>
          <Button
            size="sm"
            :variant="activeCategory === 'all' ? 'default' : 'outline'"
            @click="activeCategory = 'all'"
          >
            {{ t('admin.shared.all') }}
          </Button>
        </div>

        <!-- Products -->
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div v-for="p in filteredProducts" :key="p.id" class="rounded-xl border bg-card p-3">
            <div class="flex items-start justify-between gap-2">
              <div>
                <h3 class="font-semibold leading-snug">{{ p.name }}</h3>
                <p v-if="p.description" class="mt-1 text-xs text-muted-foreground line-clamp-2">{{ p.description }}</p>
              </div>
              <span class="text-sm font-semibold tabular-nums shrink-0">
                {{ p.variants.length > 0 ? '' : '' }}
              </span>
            </div>
            <!-- Variants -->
            <div v-if="p.variants.length > 0" class="mt-2 space-y-1.5">
              <div
                v-for="v in p.variants"
                :key="v.sku_id"
                class="flex items-center justify-between gap-2 rounded-lg border px-2 py-1.5"
                :class="v.available ? 'border-border' : 'opacity-50'"
              >
                <div class="min-w-0">
                  <p class="truncate text-sm">
                    {{ variantLabel(v) || v.sku_code }}
                  </p>
                  <p class="text-xs font-semibold tabular-nums">{{ formatCurrency(v.selling_price) }}</p>
                </div>
                <Button
                  size="sm"
                  :disabled="!v.available"
                  @click="addToCart(p, v)"
                >
                  {{ v.available ? t('selfOrder.add') : t('selfOrder.soldOut') }}
                </Button>
              </div>
            </div>
            <!-- No variants: product-level sale -->
            <div v-else class="mt-2 flex items-center justify-between">
              <span class="text-sm font-semibold tabular-nums">{{ formatCurrency(p.variants[0]?.selling_price ?? 0) }}</span>
              <Button size="sm" @click="addToCart(p)">{{ t('selfOrder.add') }}</Button>
            </div>
          </div>
          <div v-if="filteredProducts.length === 0" class="col-span-full py-10 text-center text-muted-foreground text-sm">
            {{ t('admin.pos.noProducts') }}
          </div>
        </div>
      </div>

      <!-- Cart -->
      <aside class="lg:sticky lg:top-20 self-start rounded-xl border bg-card p-4">
        <h2 class="font-bold">{{ t('selfOrder.cart') }}</h2>
        <div v-if="cart.length === 0" class="py-8 text-center text-sm text-muted-foreground">
          {{ t('selfOrder.emptyCart') }}
        </div>
        <div v-else class="mt-3 space-y-2">
          <div v-for="line in cart" :key="line.sku_id || line.product_id" class="flex items-center justify-between gap-2 text-sm">
            <div class="min-w-0">
              <p class="truncate font-medium">{{ line.product_name }}</p>
              <p v-if="line.variant_label" class="truncate text-xs text-muted-foreground">{{ line.variant_label }}</p>
              <p class="text-xs tabular-nums">{{ formatCurrency(line.unit_price) }} × {{ line.quantity }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-1">
              <Button size="sm" variant="outline" class="h-7 w-7 px-0" @click="changeQty(line, -1)">−</Button>
              <span class="w-6 text-center tabular-nums">{{ line.quantity }}</span>
              <Button size="sm" variant="outline" class="h-7 w-7 px-0" @click="changeQty(line, 1)">+</Button>
            </div>
          </div>

          <div class="space-y-1 border-t pt-2 text-sm">
            <div class="flex justify-between"><span class="text-muted-foreground">{{ t('selfOrder.subtotal') }}</span><span class="tabular-nums">{{ formatCurrency(subtotal) }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">{{ t('selfOrder.tax') }}</span><span class="tabular-nums">{{ formatCurrency(tax) }}</span></div>
            <div class="flex justify-between font-bold"><span>{{ t('selfOrder.total') }}</span><span class="tabular-nums">{{ formatCurrency(total) }}</span></div>
          </div>

          <div class="space-y-2 pt-1">
            <Input v-model="customerName" :placeholder="t('selfOrder.customerNamePlaceholder')" class="h-9" />
            <Input v-model="notes" :placeholder="t('selfOrder.notesPlaceholder')" class="h-9" />
          </div>

          <p v-if="orderError" class="text-xs text-destructive">{{ orderError }}</p>

          <Button class="w-full" :disabled="cart.length === 0 || submitting" @click="submitOrder">
            {{ submitting ? t('selfOrder.submitting') : t('selfOrder.submitOrder') }} · {{ formatCurrency(total) }}
          </Button>
        </div>
      </aside>
    </div>

    <!-- Payment -->
    <div v-else-if="stage === 'paying'" class="mx-auto max-w-md px-4 pt-8">
      <div class="rounded-xl border bg-card p-5">
        <h2 class="text-lg font-bold">{{ t('selfOrder.payNow') }}</h2>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('selfOrder.payDescription') }}</p>

        <div class="mt-4 space-y-1 text-sm">
          <div class="flex justify-between"><span class="text-muted-foreground">{{ t('admin.sales.invoice') }}</span><span class="tabular-nums">{{ order?.invoice_number }}</span></div>
          <div class="flex justify-between"><span class="text-muted-foreground">{{ t('selfOrder.queueNumber') }}</span><span class="tabular-nums">{{ order?.invoice_number }}</span></div>
          <div class="flex justify-between font-bold text-base"><span>{{ t('selfOrder.total') }}</span><span class="tabular-nums">{{ formatCurrency(order?.grand_total ?? 0) }}</span></div>
        </div>

        <div class="mt-4">
          <p class="mb-2 text-sm font-medium">{{ t('selfOrder.paymentMethod') }}</p>
          <div class="grid grid-cols-1 gap-2">
            <button
              v-for="m in PAY_METHODS"
              :key="m"
              type="button"
              class="rounded-lg border px-3 py-2.5 text-left text-sm transition"
              :class="paymentMethod === m ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'border-border'"
              @click="paymentMethod = m"
            >
              {{ t(`selfOrder.${methodKey(m)}`) }}
            </button>
          </div>
          <Input
            v-model="paymentReference"
            class="mt-3 h-9"
            :placeholder="t('selfOrder.paymentReferencePlaceholder')"
          />
        </div>

        <p v-if="orderError" class="mt-3 text-xs text-destructive">{{ orderError }}</p>

        <div class="mt-4 flex gap-2">
          <Button variant="outline" class="flex-1" @click="stage = 'menu'">{{ t('admin.shared.cancel') }}</Button>
          <Button class="flex-1" :disabled="paying" @click="payOrder">
            {{ paying ? t('selfOrder.paying') : `${t('selfOrder.payNow')} ${formatCurrency(order?.grand_total ?? 0)}` }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Success -->
    <div v-else-if="stage === 'success' && order" class="mx-auto max-w-md px-4 pt-12 text-center">
      <div class="rounded-full bg-green-100 text-green-700 mx-auto flex h-20 w-20 items-center justify-center text-4xl dark:bg-green-900/40 dark:text-green-300">
        ✓
      </div>
      <h2 class="mt-4 text-2xl font-bold">{{ t('selfOrder.paymentSuccess') }}</h2>
      <p class="mt-1 text-sm text-muted-foreground">{{ t('selfOrder.paymentSuccessDesc') }}</p>

      <div class="mt-5 space-y-1 rounded-xl border bg-card p-4 text-sm">
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ t('selfOrder.queueNumber') }}</span>
          <span class="font-semibold tabular-nums">{{ order.invoice_number }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ t('selfOrder.tableLabel', { code: tableInfo?.code || '' }) }}</span>
          <span class="tabular-nums">{{ formatCurrency(order.grand_total) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ t('selfOrder.orderStatus') }}</span>
          <span class="font-medium">{{ t(orderStatusKey) }}</span>
        </div>
      </div>

      <Button class="mt-6 w-full" variant="outline" @click="boot">{{ t('selfOrder.backToMenu') }}</Button>
    </div>
  </div>
</template>