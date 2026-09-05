<script setup lang="ts">
/**
 * Thermal-style receipt paper. Presentational: renders a struk exactly as it
 * should print, driven by the store settings toggles (docs/pos-store-settings.md).
 * Shared by the settings preview (SettingsPage) and the POS receipt dialog.
 */
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import type { StoreSettings } from '@/types';

export interface ReceiptLine {
  name: string;
  sku_code?: string;
  barcode?: string;
  quantity: number;
  unit_price: number;
}

export interface ReceiptData {
  store_name?: string;
  store_address?: string;
  store_phone?: string;
  invoice_number?: string;
  created_at?: string;
  cashier_name?: string;
  customer_name?: string;
  tax_rate?: number;
  payment_method_label?: string;
  items: ReceiptLine[];
  subtotal: number;
  discount: number;
  tax: number;
  total: number;
  paid: number;
  change: number;
}

const props = withDefaults(defineProps<{
  settings?: StoreSettings | null;
  data: ReceiptData;
}>(), { settings: null });

const t = useLocaleStore().t;

const s = () =>
  props.settings ?? {
    show_store_name: true,
    show_store_contact: true,
    show_cashier: true,
    show_customer: true,
    show_item_sku: false,
    show_item_barcode: false,
    show_discount: true,
    show_tax: true,
    show_payment: true,
    show_change: true,
  } as Partial<StoreSettings>;
</script>

<template>
  <div class="bg-white text-neutral-900 font-mono text-[11px] leading-[1.45] w-[280px] max-w-full mx-auto px-2 py-2 select-none shadow-sm rounded-sm">
    <!-- Store identity -->
    <template v-if="s().show_store_name && data.store_name">
      <div class="text-center font-bold text-[13px] uppercase">{{ data.store_name }}</div>
    </template>
    <template v-if="s().show_store_contact && (data.store_address || data.store_phone)">
      <div v-if="data.store_address" class="text-center">{{ data.store_address }}</div>
      <div v-if="data.store_phone" class="text-center">Telp: {{ data.store_phone }}</div>
    </template>

    <!-- Free header -->
    <template v-if="settings?.receipt_header">
      <div class="mt-1 whitespace-pre-line text-center">{{ settings.receipt_header }}</div>
    </template>

    <div class="my-1 border-t border-dashed border-neutral-400"></div>

    <!-- Meta rows -->
    <div v-if="data.invoice_number" class="flex justify-between"><span>{{ t('posReceipt.invoice') }}</span><span>{{ data.invoice_number }}</span></div>
    <div v-if="data.created_at" class="flex justify-between"><span>{{ t('posReceipt.date') }}</span><span>{{ data.created_at }}</span></div>
    <div v-if="s().show_cashier && data.cashier_name" class="flex justify-between"><span>{{ t('posReceipt.cashier') }}</span><span>{{ data.cashier_name }}</span></div>
    <div v-if="s().show_customer" class="flex justify-between">
      <span>{{ t('posReceipt.customer') }}</span>
      <span>{{ data.customer_name || t('posReceipt.walkIn') }}</span>
    </div>

    <div class="my-1 border-t border-dashed border-neutral-400"></div>

    <!-- Item lines -->
    <div v-for="(line, i) in data.items" :key="i" class="mb-0.5">
      <div class="flex justify-between gap-2">
        <span class="truncate">{{ line.name }}</span>
        <span class="whitespace-nowrap tabular-nums">{{ line.quantity }} × {{ formatCurrency(line.unit_price) }}</span>
      </div>
      <div v-if="s().show_item_sku && line.sku_code" class="text-neutral-500">{{ line.sku_code }}</div>
      <div v-if="s().show_item_barcode && line.barcode" class="text-neutral-500">{{ line.barcode }}</div>
      <div class="flex justify-between font-semibold">
        <span></span>
        <span class="tabular-nums">{{ formatCurrency(line.quantity * line.unit_price) }}</span>
      </div>
    </div>

    <div class="my-1 border-t border-dashed border-neutral-400"></div>

    <!-- Totals -->
    <div class="flex justify-between"><span>{{ t('posReceipt.subtotal') }}</span><span class="tabular-nums">{{ formatCurrency(data.subtotal) }}</span></div>
    <div v-if="s().show_discount && data.discount > 0" class="flex justify-between">
      <span>{{ t('posReceipt.discount') }}</span>
      <span class="tabular-nums">-{{ formatCurrency(data.discount) }}</span>
    </div>
    <div v-if="s().show_tax && data.tax > 0" class="flex justify-between">
      <span>{{ t('posReceipt.tax') }}{{ data.tax_rate ? ` (${data.tax_rate}%)` : '' }}</span>
      <span class="tabular-nums">{{ formatCurrency(data.tax) }}</span>
    </div>
    <div class="flex justify-between font-bold text-[13px]">
      <span>{{ t('posReceipt.total') }}</span>
      <span class="tabular-nums">{{ formatCurrency(data.total) }}</span>
    </div>

    <!-- Payment -->
    <template v-if="s().show_payment">
      <div class="my-1 border-t border-dashed border-neutral-400"></div>
      <div class="flex justify-between">
        <span>{{ data.payment_method_label || t('posReceipt.paid') }}</span>
        <span class="tabular-nums">{{ formatCurrency(data.paid) }}</span>
      </div>
      <div v-if="s().show_change && data.change > 0" class="flex justify-between">
        <span>{{ t('posReceipt.change') }}</span>
        <span class="tabular-nums">{{ formatCurrency(data.change) }}</span>
      </div>
    </template>

    <!-- Free footer -->
    <template v-if="settings?.receipt_footer">
      <div class="my-1 border-t border-dashed border-neutral-400"></div>
      <div class="whitespace-pre-line text-center">{{ settings.receipt_footer }}</div>
    </template>
  </div>
</template>
