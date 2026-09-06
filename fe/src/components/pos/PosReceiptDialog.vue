<script setup lang="ts">
/** Dialog struk (receipt) shown after a successful payment (docs/pos-store-settings.md). */
import { computed } from 'vue';
import { useLocaleStore } from '@/stores/locale';
import Button from '@/components/ui/button/Button.vue';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog';
import PosReceiptPaper, { type ReceiptData } from '@/components/pos/PosReceiptPaper.vue';
import {
  ORDER_CHANNEL_LABELS,
  type StoreSettings,
} from '@/types';

const props = defineProps<{
  open: boolean;
  settings?: StoreSettings | null;
  data: ReceiptData | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'new-sale': [];
}>();

const t = useLocaleStore().t;

const paperSettings = computed(() => props.settings);

function esc(v: string): string {
  return String(v ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

/** Renders a standalone thermal-style HTML doc for the browser print dialog. */
function buildPrintHtml(): string {
  const d = props.data;
  if (!d) return '';
  const s = (props.settings ?? {}) as Partial<StoreSettings>;
  const fmt = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 });
  const lines: string[] = [];
  const row = (l: string, r = '') => `  <div style="display:flex;justify-content:space-between;gap:8px">${l ? `<span>${l}</span>` : '<span></span>'}<span>${r}</span></div>\n`;
  const dash = '<div style="border-top:1px dashed #666;margin:4px 0"></div>\n';

  if (s.show_store_name !== false && d.store_name) {
    lines.push(`  <div style="text-align:center;font-weight:bold;text-transform:uppercase;font-size:14px">${esc(d.store_name)}</div>\n`);
  }
  if (s.show_store_contact !== false) {
    if (d.store_address) lines.push(`  <div style="text-align:center">${esc(d.store_address)}</div>\n`);
    if (d.store_phone) lines.push(`  <div style="text-align:center">Telp: ${esc(d.store_phone)}</div>\n`);
  }
  if (s.receipt_header) {
    lines.push(`  <div style="white-space:pre-line;text-align:center;margin-top:4px">${esc(s.receipt_header)}</div>\n`);
  }
  lines.push(dash);
  if (d.invoice_number) lines.push(row(esc(t('posReceipt.invoice')), esc(d.invoice_number)));
  if (d.created_at) lines.push(row(esc(t('posReceipt.date')), esc(d.created_at)));
  if (s.show_cashier !== false && d.cashier_name) lines.push(row(esc(t('posReceipt.cashier')), esc(d.cashier_name)));
  if (s.show_customer !== false) {
    lines.push(row(esc(t('posReceipt.customer')), esc(d.customer_name || t('posReceipt.walkIn'))));
  }
  if (d.orderChannel) {
    lines.push(row(esc(t('orderChannel.title')), esc(ORDER_CHANNEL_LABELS[d.orderChannel])));
  }
  lines.push(dash);

  for (const item of d.items) {
    lines.push(`  <div style="display:flex;justify-content:space-between;gap:8px"><span>${esc(item.name)}</span><span>${item.quantity} × ${fmt.format(item.unit_price)}</span></div>\n`);
    if (s.show_item_sku !== false && item.sku_code) lines.push(`  <div style="color:#555">${esc(item.sku_code)}</div>\n`);
    if (s.show_item_barcode !== false && item.barcode) lines.push(`  <div style="color:#555">${esc(item.barcode)}</div>\n`);
    lines.push(`  <div style="display:flex;justify-content:space-between;font-weight:bold"><span></span><span>${fmt.format(item.quantity * item.unit_price)}</span></div>\n`);
  }
  lines.push(dash);
  lines.push(row(esc(t('posReceipt.subtotal')), fmt.format(d.subtotal)));
  if (s.show_discount !== false && d.discount > 0) lines.push(row(esc(t('posReceipt.discount')), `-${fmt.format(d.discount)}`));
  if (s.show_tax !== false && d.tax > 0) lines.push(row(esc(t('posReceipt.tax')) + (d.tax_rate ? ` (${d.tax_rate}%)` : ''), fmt.format(d.tax)));
  lines.push(`  <div style="display:flex;justify-content:space-between;font-weight:bold;font-size:14px"><span>${esc(t('posReceipt.total'))}</span><span>${fmt.format(d.total)}</span></div>\n`);
  if (s.show_payment !== false) {
    lines.push(dash);
    lines.push(row(esc(d.payment_method_label || t('posReceipt.paid')), fmt.format(d.paid)));
    if (s.show_change !== false && d.change > 0) lines.push(row(esc(t('posReceipt.change')), fmt.format(d.change)));
  }
  // Free footer: rendered once at the end, text aligned per setting.
  if (s.receipt_footer) {
    const footerAlign = String(s.receipt_footer_align || 'center').toLowerCase();
    const align = footerAlign === 'left' ? 'left' : footerAlign === 'right' ? 'right' : 'center';
    lines.push(dash);
    lines.push(`  <div style="white-space:pre-line;text-align:${align}">${esc(s.receipt_footer)}</div>\n`);
  }

  return `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>${esc(t('posReceipt.title'))}</title>
<style>
  @page { size: 80mm auto; margin: 4mm; }
  body { font-family: 'Courier New', Courier, monospace; font-size: 12px; color: #000; width: 72mm; margin: 0 auto; }
</style></head>
<body>
${lines.join('')}
<script>window.onload = function(){ window.print(); }<\/script>
</body></html>`;
}

function printReceipt() {
  const w = window.open('', '_blank', 'width=400,height=600');
  if (!w) return;
  w.document.write(buildPrintHtml());
  w.document.close();
}

/** Builds a kitchen-only receipt: items, quantities, variant — no prices. */
function buildKitchenHtml(): string {
  const d = props.data;
  if (!d) return '';
  const fmt = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 });
  const lines: string[] = [];
  const dash = '<div style="border-top:1px dashed #666;margin:4px 0"></div>\n';

  lines.push(`  <div style="text-align:center;font-weight:bold;text-transform:uppercase;font-size:14px">${esc(t('posReceipt.kitchenTitle'))}</div>\n`);
  lines.push(dash);
  if (d.invoice_number) lines.push(`  <div>Invoice: ${esc(d.invoice_number)}</div>\n`);
  if (d.created_at) lines.push(`  <div>${esc(d.created_at)}</div>\n`);
  if (d.orderChannel) lines.push(`  <div>${esc(t('orderChannel.title'))}: ${esc(ORDER_CHANNEL_LABELS[d.orderChannel])}</div>\n`);
  lines.push(dash);

  for (const item of d.items) {
    lines.push(`  <div style="font-weight:bold">${item.quantity}× ${esc(item.name)}</div>\n`);
    if (item.sku_code) lines.push(`  <div style="color:#555">${esc(item.sku_code)}</div>\n`);
  }

  if (d.notes) {
    lines.push(dash);
    lines.push(`  <div style="font-weight:bold">Notes:</div>\n`);
    lines.push(`  <div style="white-space:pre-line">${esc(d.notes)}</div>\n`);
  }

  return `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>${esc(t('posReceipt.kitchenReceipt'))}</title>
<style>
  @page { size: 80mm auto; margin: 4mm; }
  body { font-family: 'Courier New', Courier, monospace; font-size: 12px; color: #000; width: 72mm; margin: 0 auto; }
</style></head>
<body>
${lines.join('')}
<script>window.onload = function(){ window.print(); }<\/script>
</body></html>`;
}

function printKitchen() {
  const w = window.open('', '_blank', 'width=400,height=600');
  if (!w) return;
  w.document.write(buildKitchenHtml());
  w.document.close();
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle class="text-center">{{ t('posReceipt.title') }}</DialogTitle>
      </DialogHeader>
      <div class="flex justify-center py-2">
        <PosReceiptPaper v-if="data" :settings="paperSettings" :data="data" />
      </div>
      <DialogFooter class="flex sm:justify-between gap-2">
        <div class="flex gap-2">
          <Button variant="outline" @click="printReceipt">{{ t('posReceipt.print') }}</Button>
          <Button
            v-if="settings?.show_kitchen_print && data?.items?.length"
            variant="outline"
            @click="printKitchen"
          >
            {{ t('posReceipt.kitchenReceipt') }}
          </Button>
        </div>
        <Button @click="emit('new-sale')">{{ t('posReceipt.newSale') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
