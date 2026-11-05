<script setup lang="ts">
import { computed, ref } from 'vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import type { Product, ProductSKU } from '@/types';

/**
 * Left column of the POS checkout: search box, barcode/SKU lookup and the
 * product grid (product header rows + SKU rows with live stock badges).
 *
 * Pure view: typed text is two-way bound to the parent, the parent performs
 * search/lookup side effects, and every "Add" click is emitted upwards with
 * the product + SKU so the parent stays the single owner of the cart.
 */
const props = defineProps<{
  products: Product[];
  searching: boolean;
  /** total quantity of each SKU already in the cart (for live badges) */
  cartQuantities: Record<string, number>;
  searchQuery: string;
  barcodeInput: string;
}>();

const emit = defineEmits<{
  'update:searchQuery': [value: string];
  'update:barcodeInput': [value: string];
  search: [];
  lookup: [code: string];
  add: [product: Product, sku: ProductSKU];
}>();

const t = useLocaleStore().t;

const searchQuery = computed({
  get: () => props.searchQuery,
  set: (v: string) => emit('update:searchQuery', v),
});
const barcodeInput = computed({
  get: () => props.barcodeInput,
  set: (v: string) => emit('update:barcodeInput', v),
});

const searchInputRef = ref<HTMLInputElement | null>(null);

/** Live remaining stock for a SKU (original stock minus what's in the cart). */
function remainingStock(sku: ProductSKU): number {
  return sku.stock - (props.cartQuantities[sku.id] ?? 0);
}

/** Product + SKU can be sold right now (active and in stock). */
function isAvailable(product: Product, sku: ProductSKU): boolean {
  return product.is_active && sku.is_active && remainingStock(sku) > 0;
}

function submitSearch() {
  emit('search');
}

function submitLookup() {
  const code = barcodeInput.value.trim();
  if (!code) return;
  emit('lookup', code);
}

function addToCart(product: Product, sku: ProductSKU) {
  emit('add', product, sku);
}

defineExpose({
  focusSearch() {
    searchInputRef.value?.focus();
  },
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex min-w-0 gap-2">
      <div class="min-w-0 flex-1">
        <Input
          ref="searchInputRef"
          v-model="searchQuery"
          :placeholder="t('admin.pos.searchPlaceholder')"
          @keyup.enter="submitSearch"
          class="text-base h-11"
        />
      </div>
      <Button @click="submitSearch" class="h-11 shrink-0 px-3 sm:px-6">{{ t('admin.shared.search') }}</Button>
    </div>

    <div class="flex min-w-0 gap-2">
      <Input
        v-model="barcodeInput"
        placeholder="Barcode / SKU"
        class="flex-1"
        @keyup.enter="submitLookup"
      />
      <Button variant="outline" class="shrink-0 px-2 sm:px-4" @click="submitLookup">{{ t('admin.products.scanBarcode') }}</Button>
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
              <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">
                {{ t('admin.shared.loading') }}
              </TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="products.filter(p => p.skus && p.skus.length > 0).length === 0">
                <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">
                  {{ t('admin.pos.noProducts') }}
                </TableCell>
              </TableRow>
              <template v-for="p in products.filter(p => p.skus && p.skus.length > 0)" :key="p.id">
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
                  <TableCell class="text-right tabular-nums text-sm">
                    {{ formatCurrency(sku.selling_price) }}
                  </TableCell>
                  <TableCell class="text-right tabular-nums">
                    <Badge :variant="isAvailable(p, sku) ? 'default' : 'destructive'">
                      {{ remainingStock(sku) }}
                    </Badge>
                  </TableCell>
                  <TableCell class="text-right">
                    <Button
                      size="sm"
                      :disabled="!isAvailable(p, sku)"
                      @click="addToCart(p, sku)"
                    >{{ t('admin.pos.addToCart') }}</Button>
                  </TableCell>
                </TableRow>
                <!-- No SKUs fallback -->
                <TableRow v-if="!p.skus?.length">
                  <TableCell colspan="5" class="pl-8 text-sm text-muted-foreground">
                    No SKUs
                  </TableCell>
                </TableRow>
              </template>
            </template>
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  </div>
</template>
