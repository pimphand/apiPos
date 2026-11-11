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
      <div v-else class="flex-1 overflow-y-auto max-h-[50vh] lg:max-h-none min-h-0">
        <div class="overflow-x-auto">
        <Table class="min-w-[480px]">
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
                <div class="text-xs text-muted-foreground">{{ item.sku_code }} <br> {{ formatCurrency(item.unit_price) }}</div>
              </TableCell>
              <TableCell>
                <div class="flex items-center justify-center gap-1">
                  <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="emit('updateQty', item, -1)">−</Button>
                  <span class="w-8 text-center text-sm tabular-nums">{{ item.quantity }}</span>
                  <Button variant="outline" size="sm" class="h-7 w-7 p-0" @click="emit('updateQty', item, 1)" :disabled="item.quantity >= item.stock">+</Button>
                </div>
              </TableCell>
              <TableCell class="text-right text-sm tabular-nums">{{ formatCurrency(item.unit_price * item.quantity) }}</TableCell>
              <TableCell>
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
