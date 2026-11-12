<script setup lang="ts">
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import type { HeldOrder, HeldOrderItem, Product, ProductSKU } from '@/types';

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
import Button from '@/components/ui/button/Button.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import Table from '@/components/ui/table/Table.vue';
import {
  TableHead,
  TableHeader,
  TableBody,
  TableCell,
  TableRow,
} from '@/components/ui/table';
import PosHeldOrders from '@/components/pos/PosHeldOrders.vue';
import { Trash } from '@lucide/vue';

const props = defineProps<{
  cart: CartItem[];
  holds: HeldOrder[];
  restoring: boolean;
  cartCount: number;
}>();

const emit = defineEmits<{
  addToCart: [product: Product, sku: ProductSKU];
  updateQty: [item: CartItem, delta: number];
  removeItem: [item: CartItem];
  clearCart: [];
  holdCart: [];
  restore: [order: HeldOrder];
  holdAndRestore: [order: HeldOrder];
  replaceAndRestore: [order: HeldOrder];
  deleteHold: [order: HeldOrder];
}>();

const t = useLocaleStore().t;
</script>

<template>
  <Card class="min-w-0 flex-1 flex flex-col">
    <CardHeader class="flex flex-col items-start justify-between gap-2 pb-2 sm:flex-row sm:items-center">
      <CardTitle class="text-base sm:text-lg">{{ t('admin.pos.cart') }}</CardTitle>
      <div class="flex w-full gap-1.5 sm:w-auto">
        <Button
          variant="outline"
          size="sm"
          class="h-9 flex-1 px-2.5 text-xs sm:h-8 sm:flex-none sm:text-sm"
          :disabled="cart.length === 0"
          @click="emit('holdCart')"
        >{{ t('admin.pos.holdCart') }}</Button>
        <Button
          variant="outline"
          size="sm"
          class="h-9 flex-1 px-2.5 text-xs sm:h-8 sm:flex-none sm:text-sm"
          :disabled="cart.length === 0"
          @click="emit('clearCart')"
        >{{ t('admin.pos.clearCart') }}</Button>
      </div>
    </CardHeader>

    <PosHeldOrders
      :orders="holds"
      :restoring="restoring"
      :cart-count="cartCount"
      @restore="emit('restore', $event)"
      @hold-current-and-restore="emit('holdAndRestore', $event)"
      @replace-and-restore="emit('replaceAndRestore', $event)"
      @delete="emit('deleteHold', $event)"
    />

    <CardContent class="flex-1 flex flex-col p-0 min-h-0">
      <div v-if="cart.length === 0" class="flex items-center justify-center flex-1 py-12 text-muted-foreground text-sm">
        {{ t('admin.pos.emptyCart') }}
      </div>
      <div v-else class="min-h-0 flex-1 overflow-y-auto max-h-[40vh] lg:max-h-[24rem]">
        <div class="overflow-x-auto">
        <Table class="w-full min-w-[340px]">
          <TableHeader>
            <TableRow>
              <TableHead class="px-2 sm:px-3">{{ t('admin.pos.productName') }}</TableHead>
              <TableHead class="w-[96px] px-2 text-center sm:px-3">{{ t('admin.pos.qty') }}</TableHead>
              <TableHead class="w-[96px] px-2 text-right sm:px-3">{{ t('admin.pos.subtotal') }}</TableHead>
              <TableHead class="w-9 px-1"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="item in cart" :key="item.sku_id">
              <TableCell class="px-2 py-2 sm:px-3">
                <div class="font-medium text-sm leading-snug">{{ item.name }}</div>
                <div class="text-xs text-muted-foreground leading-snug">{{ item.sku_code }} <br> {{ formatCurrency(item.unit_price) }}</div>
              </TableCell>
              <TableCell class="px-1 py-2 sm:px-2">
                <div class="flex items-center justify-center gap-0.5">
                  <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="emit('updateQty', item, -1)">−</Button>
                  <span class="w-7 text-center text-sm tabular-nums">{{ item.quantity }}</span>
                  <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="emit('updateQty', item, 1)" :disabled="item.quantity >= item.stock">+</Button>
                </div>
              </TableCell>
              <TableCell class="px-2 py-2 text-right text-sm tabular-nums sm:px-3">{{ formatCurrency(item.unit_price * item.quantity) }}</TableCell>
              <TableCell class="px-1 py-2">
                <Button variant="ghost" size="sm" class="h-7 w-7 p-0 text-destructive" @click="emit('removeItem', item)">
                  <Trash></Trash>
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
