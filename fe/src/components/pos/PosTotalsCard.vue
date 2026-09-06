<script setup lang="ts">
import { computed } from 'vue';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import type { OrderChannel } from '@/types';
import { ORDER_CHANNEL_LABELS } from '@/types';
import PosOrderChannelSelector from '@/components/pos/PosOrderChannelSelector.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import Badge from '@/components/ui/badge/Badge.vue';

const props = defineProps<{
  subtotal: number;
  discount: number;
  discountEditable?: boolean;
  taxRate: number;
  taxRateEditable?: boolean;
  taxAmount: number;
  grandTotal: number;
  orderChannel: OrderChannel;
  cartCount: number;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  'update:discount': [value: number];
  'update:taxRate': [value: number];
  'update:orderChannel': [value: OrderChannel];
  clearCart: [];
  pay: [];
}>();

const t = useLocaleStore().t;

const taxLabel = computed(() => {
  const v = props.taxRate;
  return v > 0 ? `${t('admin.pos.taxRate')} (${v})` : t('admin.pos.taxRate');
});
</script>

<template>
  <Card>
    <CardHeader class="pb-2">
      <CardTitle class="text-base">{{ t('admin.pos.totals') }}</CardTitle>
    </CardHeader>
    <CardContent class="p-4 space-y-3">
      <!-- Itemized totals -->
      <div class="space-y-2">
        <div class="flex justify-between text-sm">
          <span class="text-muted-foreground">{{ t('admin.pos.subtotal') }}</span>
          <span class="tabular-nums">{{ formatCurrency(subtotal) }}</span>
        </div>

        <div class="flex items-center justify-between text-sm gap-2">
          <span class="text-muted-foreground">{{ t('admin.pos.discount') }}</span>
          <Input
            v-if="discountEditable !== false"
            :model-value="discount"
            type="number"
            min="0"
            class="w-20 h-8 text-right text-sm tabular-nums"
            @update:model-value="emit('update:discount', Number($event) || 0)"
          />
          <span v-else class="tabular-nums">{{ formatCurrency(discount) }}</span>
          <span class="tabular-nums text-muted-foreground">({{ formatCurrency(discount) }})</span>
        </div>

        <div class="flex items-center justify-between text-sm gap-2">
          <span class="text-muted-foreground">{{ taxLabel }}</span>
          <div class="flex items-center gap-1">
            <Input
              v-if="taxRateEditable !== false"
              :model-value="taxRate"
              type="number"
              min="0"
              max="100"
              class="w-16 h-8 text-right text-sm tabular-nums"
              @update:model-value="emit('update:taxRate', Number($event) || 0)"
            />
            <span v-else class="tabular-nums">{{ taxRate }}%</span>
            <span class="tabular-nums text-muted-foreground">→</span>
            <span class="tabular-nums">{{ formatCurrency(taxAmount) }}</span>
          </div>
        </div>
      </div>

      <!-- Grand total row -->
      <div class="border-t pt-3 space-y-2">
        <div class="flex items-center justify-between">
          <PosOrderChannelSelector
            :model-value="orderChannel"
            :disabled="disabled"
            @update:model-value="emit('update:orderChannel', $event)"
          />
          <div class="text-right">
            <div class="text-xs text-muted-foreground">{{ t('admin.pos.total') }}</div>
            <div class="font-semibold text-lg tabular-nums">{{ formatCurrency(grandTotal) }}</div>
          </div>
          <div class="flex items-center gap-2">
            <Button
              v-if="cartCount > 0"
              variant="outline"
              size="sm"
              class="h-8 px-3 text-sm"
              :disabled="disabled"
              @click="emit('clearCart')"
            >
              {{ t('admin.pos.clearCart') }}
            </Button>
            <Button
              class="h-9 min-w-0 px-4 text-sm font-medium"
              :disabled="cartCount === 0 || disabled"
              @click="emit('pay')"
            >
              {{ t('admin.pos.pay') }} {{ formatCurrency(grandTotal) }}
            </Button>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
