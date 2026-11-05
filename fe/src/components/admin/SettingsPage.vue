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
import type { StoreSettings, StoreInfo } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import PosReceiptPaper, { type ReceiptData } from '@/components/pos/PosReceiptPaper.vue';

const { isReady, authStore } = usePageGuard();
const ui = useUiStore();
const t = useLocaleStore().t;

const allowed = computed(() => {
  const role = authStore.currentUser?.role_name;
  return role === 'admin' || role === 'manager';
});

/**
 * Neutral blank state. Real values always come from the server via
 * settingsService.normalizeSettings, which returns every key (proto3 JSON omits
 * zero fields, and an omitted field means the stored value is its zero value —
 * false/0). Never fabricate `true`/`11` here: that would silently turn unchecked
 * rows back on and reset a saved 0% tax to 11% on the next save.
 */
function defaults(): StoreSettings {
  return {
    store_id: '',
    tax_rate: 0,
    receipt_header: '',
    receipt_footer: '',
    receipt_footer_align: 'center',
    show_store_name: false,
    show_store_contact: false,
    show_cashier: false,
    show_customer: false,
    show_item_sku: false,
    show_item_barcode: false,
    show_discount: false,
    show_tax: false,
    show_payment: false,
    show_change: false,
    show_kitchen_print: false,
    self_order_enabled: false,
    updated_by: '',
    updated_at: '',
  };
}

const form = ref<StoreSettings>(defaults());
const storeInfo = ref<StoreInfo>({ name: '', address: '', phone: '' });
const loading = ref(true);
const saving = ref(false);
const savingInfo = ref(false);

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
  { key: 'show_kitchen_print', label: 'showKitchenPrint' },
];

async function load() {
  loading.value = true;
  try {
    const res = await settingsService.get();
    // settings is fully normalized (every key present), so it overrides the
    // blank defaults completely — no phantom `true`/11% values can leak in.
    form.value = { ...defaults(), ...res.settings };
    storeInfo.value = res.store_info;
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
      receipt_footer_align: form.value.receipt_footer_align || 'center',
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
      show_kitchen_print: !!form.value.show_kitchen_print,
      self_order_enabled: !!form.value.self_order_enabled,
    });
    form.value = { ...defaults(), ...saved.settings };
    if (saved.store_info.name) storeInfo.value = saved.store_info;
    ui.addToast('success', t('admin.settings.toastSaved'));
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function saveStoreInfo() {
  savingInfo.value = true;
  try {
    storeInfo.value = await settingsService.updateStoreInfo({
      address: storeInfo.value.address,
      phone: storeInfo.value.phone,
    });
    ui.addToast('success', t('admin.settings.toastSaved'));
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    savingInfo.value = false;
  }
}

// ---- Live preview with sample data --------------------------------------
const taxPct = computed(() => Math.max(0, Math.min(100, Number(form.value.tax_rate) || 0)));

const previewData = computed<ReceiptData>(() => {
  const subtotal = 65000;
  const tax = Math.round(subtotal * (taxPct.value / 100));
  const total = subtotal + tax;
  const currentUser = authStore.currentUser as any;
  return {
    // Prefer the store info edited on this page so the preview shows exactly
    // what will print; fall back to the session user, then placeholders.
    store_name: storeInfo.value.name || currentUser?.store_name || currentUser?.store?.name || t('sidebar.appName'),
    store_address: storeInfo.value.address || currentUser?.store?.address || '',
    store_phone: storeInfo.value.phone || currentUser?.store?.phone || '',
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
    <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
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
          <!-- Store identity: address & phone (saved to the stores table) -->
          <Card>
            <CardHeader>
              <CardTitle>{{ t('admin.settings.storeInfoCard') }}</CardTitle>
              <CardDescription>{{ t('admin.settings.storeInfoDesc') }}</CardDescription>
            </CardHeader>
            <CardContent class="space-y-3">
              <div class="flex items-center gap-3">
                <Label class="w-32 shrink-0">{{ t('admin.settings.storeName') }}</Label>
                <Input :model-value="storeInfo.name" disabled class="flex-1 h-9 bg-muted" />
              </div>
              <div class="flex items-start gap-3">
                <Label class="w-32 shrink-0 pt-2">{{ t('admin.settings.storeAddress') }}</Label>
                <Textarea v-model="storeInfo.address" rows="2" class="flex-1" :placeholder="t('admin.settings.storeAddressHint')" />
              </div>
              <div class="flex items-center gap-3">
                <Label class="w-32 shrink-0">{{ t('admin.settings.storePhone') }}</Label>
                <Input v-model="storeInfo.phone" type="tel" class="flex-1 h-9" :placeholder="t('admin.settings.storePhoneHint')" />
              </div>
              <div class="flex justify-end">
                <Button variant="outline" :disabled="savingInfo" @click="saveStoreInfo">
                  {{ savingInfo ? t('admin.settings.saving') : t('admin.settings.save') }}
                </Button>
              </div>
            </CardContent>
          </Card>

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
              <div class="flex items-center gap-3">
                <Label class="w-32 shrink-0">{{ t('admin.settings.footerAlign') }}</Label>
                <Select v-model="form.receipt_footer_align">
                  <SelectTrigger class="w-40 h-9">
                    <SelectValue :placeholder="t('admin.settings.alignCenter')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="left">{{ t('admin.settings.alignLeft') }}</SelectItem>
                    <SelectItem value="center">{{ t('admin.settings.alignCenter') }}</SelectItem>
                    <SelectItem value="right">{{ t('admin.settings.alignRight') }}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="space-y-2">
                <Label>{{ t('admin.settings.rowOptions') }}</Label>
                <div class="grid grid-cols-2 gap-x-4 gap-y-2">
                  <label v-for="opt in rowOptions" :key="opt.key" class="flex items-center gap-2 text-sm cursor-pointer">
                    <!-- reka-ui Checkbox uses modelValue (NOT checked) — same
                         pattern as RolesPage.vue. A wrong binding here makes
                         checkboxes look tappable but never update the form,
                         so saved values silently stay stale. -->
                    <Checkbox
                      :model-value="form[opt.key] === true"
                      @update:model-value="(v: boolean | 'indeterminate') => (form[opt.key] = v === true)"
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
             <Card class="mt-3">
            <CardHeader>
              <CardTitle>{{ t('admin.settings.selfOrderCard') }}</CardTitle>
              <CardDescription>{{ t('admin.settings.selfOrderDesc') }}</CardDescription>
            </CardHeader>
            <CardContent class="space-y-2">
              <label class="flex items-center gap-2 text-sm cursor-pointer">
                <Checkbox
                  :model-value="form.self_order_enabled === true"
                  @update:model-value="(v: boolean | 'indeterminate') => (form.self_order_enabled = v === true)"
                />
                {{ t('admin.settings.selfOrderEnabled') }}
              </label>
              <p class="text-xs text-muted-foreground">{{ t('admin.settings.selfOrderHint') }}</p>
            </CardContent>
          </Card>

          </CardContent>
          
        </Card>
        
      </div>
    </div>
  </DashboardShell>
</template>
