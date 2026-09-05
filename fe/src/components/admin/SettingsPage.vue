<script setup lang="ts">
/**
 * /admin/settings — per-store settings: default tax + nota/struk format
 * (docs/pos-store-settings.md). Read for all store users; editable only by
 * admin & manager (backend enforces the same guard).
 */
import { ref, computed, watch } from 'vue';
import { settingsService } from '@/services/settings';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { StoreSettings } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import PosReceiptPaper, { type ReceiptData } from '@/components/pos/PosReceiptPaper.vue';

const { isReady, authStore } = usePageGuard();
const ui = useUiStore();
const t = useLocaleStore().t;

const allowed = computed(() => {
  const role = (authStore.currentUser as any)?.role_name;
  return role === 'admin' || role === 'manager';
});

/** Full editable form state (mirrors StoreSettings booleans). */
function defaults(): StoreSettings {
  return {
    store_id: '',
    tax_rate: 11,
    receipt_header: '',
    receipt_footer: '',
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
    updated_by: '',
    updated_at: '',
  };
}

const form = ref<StoreSettings>(defaults());
const loading = ref(true);
const saving = ref(false);

type BooleanSettingKey = { [K in keyof StoreSettings]: StoreSettings[K] extends boolean ? K : never }[keyof StoreSettings];
const rowOptions: { key: BooleanSettingKey; label: string }[] = [
  { key: 'show_store_name', label: 'showStoreName' },
  { key: 'show_store_contact', label: 'showStoreContact' },
  { key: 'show_cashier', label: 'showCashier' },
  { key: 'show_customer', label: 'showCustomer' },
  { key: 'show_item_sku', label: 'showItemSku' },
  { key: 'show_item_barcode', label: 'showItemBarcode' },
  { key: 'show_discount', label: 'showDiscount' },
  { key: 'show_tax', label: 'showTax' },
  { key: 'show_payment', label: 'showPayment' },
  { key: 'show_change', label: 'showChange' },
];

function onToggle(key: BooleanSettingKey, value: boolean | 'indeterminate') {
  form.value[key] = value === true;
}

async function load() {
  loading.value = true;
  try {
    const saved = await settingsService.get();
    form.value = { ...defaults(), ...saved };
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    const saved = await settingsService.update({
      tax_rate: Number(form.value.tax_rate) || 0,
      receipt_header: form.value.receipt_header,
      receipt_footer: form.value.receipt_footer,
      show_store_name: !!form.value.show_store_name,
      show_store_contact: !!form.value.show_store_contact,
      show_cashier: !!form.value.show_cashier,
      show_customer: !!form.value.show_customer,
      show_item_sku: !!form.value.show_item_sku,
      show_item_barcode: !!form.value.show_item_barcode,
      show_discount: !!form.value.show_discount,
      show_tax: !!form.value.show_tax,
      show_payment: !!form.value.show_payment,
      show_change: !!form.value.show_change,
    });
    form.value = { ...defaults(), ...saved };
    ui.addToast('success', t('admin.settings.toastSaved'));
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

// ---- Live preview with sample data --------------------------------------
const taxPct = computed(() => Math.max(0, Math.min(100, Number(form.value.tax_rate) || 0)));

const previewData = computed<ReceiptData>(() => {
  const subtotal = 65000;
  const tax = Math.round(subtotal * (taxPct.value / 100));
  const total = subtotal + tax;
  const storeAny = authStore.currentUser as any;
  return {
    store_name: storeAny?.store_name || storeAny?.store?.name || t('sidebar.appName'),
    store_address: storeAny?.store?.address || 'Jl. Contoh No. 1',
    store_phone: storeAny?.store?.phone || '0812-3456-7890',
    invoice_number: 'INV-DEMO-0001',
    created_at: new Date().toISOString().slice(0, 16).replace('T', ' '),
    cashier_name: authStore.userName || 'Kasir',
    customer_name: '',
    tax_rate: taxPct.value,
    payment_method_label: t('admin.payments.cash'),
    items: [
      { name: 'Es Kopi Susu', quantity: 2, unit_price: 25000 },
      { name: 'Croissant', sku_code: 'SKU-CRO-01', quantity: 1, unit_price: 15000 },
    ],
    subtotal,
    discount: 0,
    tax,
    total,
    paid: total + 10000,
    change: 10000,
  };
});

watch(isReady, (r) => { if (r) void load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
    <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
    </div>

    <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
      <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.settings.accessRestrictedAction') }) }}</p>
    </div>

    <div v-else class="flex flex-col gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.settings.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.settings.description') }}</p>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 items-start">
        <!-- Left column: form -->
        <div class="flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle>{{ t('admin.settings.taxCard') }}</CardTitle>
            </CardHeader>
            <CardContent class="space-y-2">
              <div class="flex items-center gap-3">
                <Label class="w-32 shrink-0">{{ t('admin.settings.taxRate') }}</Label>
                <Input v-model.number="form.tax_rate" type="number" min="0" max="100" step="0.5" class="w-28 h-9 text-right tabular-nums" />
              </div>
              <p class="text-xs text-muted-foreground">{{ t('admin.settings.taxHint') }}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{{ t('admin.settings.receiptCard') }}</CardTitle>
            </CardHeader>
            <CardContent class="space-y-4">
              <div class="space-y-1.5">
                <Label>{{ t('admin.settings.receiptHeader') }}</Label>
                <Textarea v-model="form.receipt_header" rows="2" :placeholder="t('admin.settings.receiptHeaderHint')" />
              </div>
              <div class="space-y-1.5">
                <Label>{{ t('admin.settings.receiptFooter') }}</Label>
                <Textarea v-model="form.receipt_footer" rows="2" :placeholder="t('admin.settings.receiptFooterHint')" />
              </div>
              <div class="space-y-2">
                <Label>{{ t('admin.settings.rowOptions') }}</Label>
                <div class="grid grid-cols-2 gap-x-4 gap-y-2">
                  <label v-for="opt in rowOptions" :key="opt.key" class="flex items-center gap-2 text-sm cursor-pointer">
                    <Checkbox
                      :checked="Boolean(form[opt.key])"
                      @update:checked="onToggle(opt.key, $event)"
                    />
                    {{ t(`admin.settings.${opt.label}`) }}
                  </label>
                </div>
              </div>
            </CardContent>
          </Card>

          <div class="flex justify-end">
            <Button :disabled="saving" @click="save">
              {{ saving ? t('admin.settings.saving') : t('admin.settings.save') }}
            </Button>
          </div>
        </div>

        <!-- Right column: live preview -->
        <Card class="lg:sticky lg:top-4">
          <CardHeader>
            <CardTitle>{{ t('admin.settings.previewCard') }}</CardTitle>
            <CardDescription>{{ t('admin.settings.previewExample') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <div v-if="loading" class="flex justify-center py-10 text-muted-foreground text-sm">{{ t('admin.shared.loading') }}</div>
            <div v-else class="border rounded-lg p-3 bg-muted/40">
              <PosReceiptPaper :settings="form" :data="previewData" />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  </DashboardShell>
</template>
