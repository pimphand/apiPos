<script setup lang="ts">
import { computed, ref } from 'vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import type { Category, Product, ProductSKU } from '@/types';

/**
 * Left column of the POS: unified search (name / barcode / SKU), category
 * filter, and the product list rendered as per-SKU cards with live stock badges.
 *
 * Pure view: the parent owns search/filter state and side effects; "Add" clicks
 * are emitted upwards with the product + SKU so the parent stays the single
 * owner of the cart.
 */
const props = defineProps<{
  products: Product[];
  categories: Category[];
  searching: boolean;
  /** total quantity of each SKU already in the cart (for live badges) */
  cartQuantities: Record<string, number>;
  searchQuery: string;
  categoryFilter: string;
}>();

const emit = defineEmits<{
  'update:searchQuery': [value: string];
  'update:categoryFilter': [value: string];
  search: [];
  add: [product: Product, sku: ProductSKU];
}>();

const t = useLocaleStore().t;

const searchQuery = computed({
  get: () => props.searchQuery,
  set: (v: string) => emit('update:searchQuery', v),
});
const categoryFilter = computed({
  get: () => props.categoryFilter,
  set: (v: string) => emit('update:categoryFilter', v),
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
  <div class="min-w-0 flex flex-col gap-3 sm:gap-4">
    <div class="flex min-w-0 flex-col gap-2 sm:flex-row">
      <div class="min-w-0 flex-1">
        <Input
          ref="searchInputRef"
          v-model="searchQuery"
          :placeholder="t('admin.pos.searchPlaceholder')"
          @keyup.enter="submitSearch"
          class="text-base h-11"
        />
      </div>
      <Select v-model="categoryFilter" @update:model-value="submitSearch">
        <SelectTrigger class="h-11 w-full sm:w-48">
          <SelectValue :placeholder="t('admin.shared.all')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{{ t('admin.shared.all') }}</SelectItem>
          <SelectItem v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
        </SelectContent>
      </Select>
      <Button @click="submitSearch" class="h-11 shrink-0 px-3 sm:px-6">{{ t('admin.shared.search') }}</Button>
    </div>

    <Card class="min-w-0">
      <CardContent class="p-3 sm:p-4">
        <div v-if="searching" class="text-center py-8 text-muted-foreground">
          {{ t('admin.shared.loading') }}
        </div>

        <div v-else-if="products.filter(p => p.skus && p.skus.length > 0).length === 0" class="text-center py-8 text-muted-foreground">
          {{ t('admin.pos.noProducts') }}
        </div>

        <div v-else class="grid max-h-[26rem] grid-cols-1 gap-3 overflow-y-auto pr-1 sm:max-h-[30rem] sm:grid-cols-2 xl:max-h-[34rem] xl:grid-cols-3">
          <template v-for="p in products.filter(p => p.skus && p.skus.length > 0)" :key="p.id">
            <div
              v-for="sku in (p.skus ?? [])"
              :key="sku.id"
              class="flex h-full flex-col rounded-lg border bg-card p-3 shadow-sm"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="min-w-0">
                  <div class="truncate font-semibold">{{ p.name }}</div>
                  <div class="mt-0.5 flex flex-wrap items-center gap-x-1 text-xs text-muted-foreground">
                    <span class="font-mono">{{ sku.sku }}</span>
                    <span v-if="sku.variant">· {{ sku.variant }}</span>
                    <span v-if="sku.size">· {{ sku.size }}</span>
                  </div>
                </div>
                <Badge :variant="isAvailable(p, sku) ? 'default' : 'destructive'" class="shrink-0">
                  {{ t('admin.pos.stock') }}: {{ remainingStock(sku) }}
                </Badge>
              </div>

              <div class="mt-3 flex items-center justify-between text-sm">
                <span class="text-muted-foreground">{{ t('admin.pos.price') }}</span>
                <span class="font-semibold tabular-nums">{{ formatCurrency(sku.selling_price) }}</span>
              </div>

              <Button
                size="sm"
                class="mt-3 w-full"
                :disabled="!isAvailable(p, sku)"
                @click="addToCart(p, sku)"
              >
                {{ t('admin.pos.addToCart') }}
              </Button>
            </div>
          </template>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
