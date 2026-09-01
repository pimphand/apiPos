<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue';
import { productService } from '@/services/product';
import { customerService } from '@/services/customer';
import { saleService } from '@/services/sale';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency } from '@/lib/utils';
import type { Product, ProductSKU, Customer } from '@/types';
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

interface CartItem {
  product_id: string;
  sku_id: string;
  name: string;
  sku_code: string;
  unit_price: number;
  quantity: number;
  stock: number;
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
const customers = ref<Customer[]>([]);
const customersLoading = ref(false);

const paymentOpen = ref(false);
const paymentMethod = ref('CASH');
const amountPaid = ref(0);
const processing = ref(false);

const subtotal = computed(() => cart.value.reduce((sum, item) => sum + item.unit_price * item.quantity, 0));
const taxAmount = computed(() => Math.round(subtotal.value * (taxRate.value / 100)));
const grandTotal = computed(() => subtotal.value + taxAmount.value - discount.value);
const change = computed(() => Math.max(0, amountPaid.value - grandTotal.value));

async function searchProducts() {
  const q = searchQuery.value.trim();
  if (!q) { searchResults.value = []; return; }
  searching.value = true;
  try {
    const res = await productService.list({ is_active: 'true', page: 1, per_page: 20, get_sku: true });
    searchResults.value = (res.products ?? []).filter(p =>
      p.name.toLowerCase().includes(q.toLowerCase()) ||
      p.skus?.some(sku => sku.barcode === q || sku.sku.toLowerCase().includes(q.toLowerCase()))
    );
  } catch (err) {
    ui.addToast('error', apiMessage(err));
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

  const existing = cart.value.find(i => i.sku_id === skuData.id);
  if (existing) {
    if (existing.quantity >= skuData.stock) {
      ui.addToast('error', t('modules.pos.toastStockIssue', { name: product.name }));
      return;
    }
    existing.quantity++;
  } else {
    if (skuData.stock <= 0) {
      ui.addToast('error', t('modules.pos.toastStockIssue', { name: product.name }));
      return;
    }
    cart.value.push({
      product_id: product.id,
      sku_id: skuData.id,
      name: product.name,
      sku_code: skuData.sku,
      unit_price: skuData.selling_price,
      quantity: 1,
      stock: skuData.stock,
    });
  }
  ui.addToast('success', t('modules.pos.toastAdded'));
  searchResults.value = [];
  searchQuery.value = '';
}

function updateQty(item: CartItem, delta: number) {
  const newQty = item.quantity + delta;
  if (newQty <= 0) {
    cart.value = cart.value.filter(i => i.sku_id !== item.sku_id);
  } else if (newQty <= item.stock) {
    item.quantity = newQty;
  } else {
    ui.addToast('error', t('modules.pos.toastStockIssue', { name: item.name }));
  }
}

function removeItem(item: CartItem) {
  cart.value = cart.value.filter(i => i.sku_id !== item.sku_id);
  ui.addToast('success', t('modules.pos.toastRemoved'));
}

function clearCart() {
  cart.value = [];
  discount.value = 0;
  taxRate.value = 11;
  selectedCustomerId.value = '';
}

function openPayment() {
  if (cart.value.length === 0) return;
  amountPaid.value = grandTotal.value;
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
  if (amountPaid.value < grandTotal.value) {
    ui.addToast('error', t('modules.pos.insufficient'));
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
      customer_id: selectedCustomerId.value || undefined,
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
    ui.addToast('success', t('modules.pos.toastSaleCreated'));
    paymentOpen.value = false;
    clearCart();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    processing.value = false;
  }
}

watch(isReady, (r) => { if (r) searchInputRef.value?.focus(); }, { immediate: true });
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

    <div v-else class="grid grid-cols-1 lg:grid-cols-3 gap-4 min-h-[calc(100vh-8rem)]">
      <!-- Left: Search & Product List -->
      <div class="lg:col-span-2 flex flex-col gap-4">
        <div class="flex gap-2">
          <div class="flex-1">
            <Input
              ref="searchInputRef"
              v-model="searchQuery"
              :placeholder="t('modules.pos.searchPlaceholder')"
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
                  <TableHead>{{ t('modules.pos.productName') }}</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead class="text-right">{{ t('modules.pos.price') }}</TableHead>
                  <TableHead class="text-right">{{ t('modules.pos.stock') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="searching">
                  <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
                </TableRow>
                <template v-else>
                  <TableRow v-if="searchResults.length === 0 && searchQuery">
                    <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">{{ t('modules.pos.noProducts') }}</TableCell>
                  </TableRow>
                  <TableRow v-for="p in searchResults" :key="p.id">
                    <TableCell class="font-medium">{{ p.name }}</TableCell>
                    <TableCell class="text-muted-foreground text-xs">
                      <span v-if="p.skus?.length">{{ p.skus[0].sku }}</span>
                      <span v-else>—</span>
                    </TableCell>
                    <TableCell class="text-right tabular-nums">{{ formatCurrency(p.selling_price) }}</TableCell>
                    <TableCell class="text-right tabular-nums">
                      <Badge :variant="p.stock > 0 ? 'default' : 'destructive'">{{ p.stock }}</Badge>
                    </TableCell>
                    <TableCell class="text-right">
                      <Button
                        size="sm"
                        :disabled="!p.skus?.length || p.stock <= 0"
                        @click="addToCart(p, p.skus?.[0])"
                      >{{ t('modules.pos.addToCart') }}</Button>
                    </TableCell>
                  </TableRow>
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
            <CardTitle class="text-lg">{{ t('modules.pos.cart') }}</CardTitle>
            <div class="flex gap-2">
              <Button variant="outline" size="sm" @click="clearCart" :disabled="cart.length === 0">{{ t('modules.pos.clearCart') }}</Button>
            </div>
          </CardHeader>
          <CardContent class="flex-1 flex flex-col p-0">
            <div v-if="cart.length === 0" class="flex items-center justify-center flex-1 py-12 text-muted-foreground text-sm">
              {{ t('modules.pos.emptyCart') }}
            </div>
            <div v-else class="flex-1 overflow-y-auto max-h-[40vh]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('modules.pos.productName') }}</TableHead>
                    <TableHead class="text-center">{{ t('modules.pos.qty') }}</TableHead>
                    <TableHead class="text-right">{{ t('modules.pos.subtotal') }}</TableHead>
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
                        <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="updateQty(item, 1)">+</Button>
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
                <span class="text-muted-foreground">{{ t('modules.pos.subtotal') }}</span>
                <span class="tabular-nums">{{ formatCurrency(subtotal) }}</span>
              </div>
              <div class="flex items-center justify-between text-sm">
                <span class="text-muted-foreground">{{ t('modules.pos.discount') }}</span>
                <Input v-model.number="discount" type="number" min="0" class="w-28 h-8 text-right text-sm tabular-nums" />
              </div>
              <div class="flex items-center justify-between text-sm">
                <span class="text-muted-foreground">{{ t('modules.pos.taxRate') }}</span>
                <Input v-model.number="taxRate" type="number" min="0" max="100" class="w-20 h-8 text-right text-sm tabular-nums" />
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-muted-foreground">{{ t('modules.pos.taxRate') }}</span>
                <span class="tabular-nums">{{ formatCurrency(taxAmount) }}</span>
              </div>
              <div class="border-t pt-2 flex justify-between font-semibold">
                <span>{{ t('modules.pos.total') }}</span>
                <span class="tabular-nums text-lg">{{ formatCurrency(grandTotal) }}</span>
              </div>
            </div>

            <Button class="w-full h-11 text-base" :disabled="cart.length === 0" @click="openPayment">
              {{ t('modules.pos.pay') }} {{ formatCurrency(grandTotal) }}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>

    <!-- Payment Dialog -->
    <Dialog v-model:open="paymentOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('modules.pos.paymentTitle') }}</DialogTitle>
          <DialogDescription>{{ t('modules.pos.paymentDescription') }}</DialogDescription>
        </DialogHeader>

        <div class="space-y-4">
          <div class="space-y-2">
            <Label>{{ t('modules.pos.customer') }}</Label>
            <Select v-model="selectedCustomerId">
              <SelectTrigger class="w-full"><SelectValue :placeholder="t('modules.pos.noCustomer')" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="">{{ t('modules.pos.noCustomer') }}</SelectItem>
                <SelectItem v-for="c in customers" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label>{{ t('modules.pos.paymentMethod') }}</Label>
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
            <div class="text-sm text-muted-foreground">{{ t('modules.pos.grandTotal') }}</div>
            <div class="text-2xl font-bold tabular-nums">{{ formatCurrency(grandTotal) }}</div>
          </div>

          <div class="space-y-2">
            <Label>{{ t('modules.pos.amountPaid') }}</Label>
            <Input v-model.number="amountPaid" type="number" min="0" class="text-lg h-11 tabular-nums" />
          </div>

          <div v-if="amountPaid >= grandTotal" class="text-center text-emerald-600 font-medium">
            {{ t('modules.pos.change') }}: {{ formatCurrency(change) }}
          </div>
          <div v-else-if="amountPaid > 0" class="text-center text-destructive font-medium">
            {{ t('modules.pos.insufficient') }}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="paymentOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="processing || amountPaid < grandTotal" @click="submitSale">
            {{ processing ? t('modules.pos.processing') : t('modules.pos.pay') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </DashboardShell>
</template>
