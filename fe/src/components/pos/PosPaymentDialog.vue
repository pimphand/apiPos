<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency } from '@/lib/utils';
import type { Customer } from '@/types';

/**
 * Payment dialog for the POS checkout. Collects customer / payment method /
 * amount paid and lets the parent handle the actual sale (the parent owns the
 * cart, shift state and services). Local input state resets each time the
 * dialog opens; `grandTotal` comes from the parent so the change/insufficient
 * preview always matches the cart.
 */
const props = defineProps<{
  open: boolean;
  customers: Customer[];
  grandTotal: number;
  processing?: boolean;
  customerId: string; // v-model — value 'none' means walk-in
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'update:customerId': [value: string];
  cancel: [];
  submit: [
    payload: { customer_id: string; payment_method: string; amount_paid: number }
  ];
}>();

const t = useLocaleStore().t;

const NO_CUSTOMER = 'none'; // reka-ui disallows empty-string SelectItem values

const paymentMethod = ref('CASH');
const amountPaid = ref(0);

// Local view-model for the customer select (props are read-only).
const customerId = computed({
  get: () => props.customerId,
  set: (v: string) => emit('update:customerId', v),
});

const change = computed(() => Math.max(0, amountPaid.value - props.grandTotal));

// Reset inputs every time the dialog opens.
watch(
  () => props.open,
  (open) => {
    if (open) {
      paymentMethod.value = 'CASH';
      amountPaid.value = 0;
    }
  }
);

// Ensure amountPaid is always a number (Input component may not propagate type="number")
function setAmountPaid(value: string | number) {
  amountPaid.value = Number(value) || 0;
}

function cancel() {
  emit('cancel');
}

function submit() {
  if (props.processing || amountPaid.value < props.grandTotal) return;
  emit('submit', {
    customer_id: customerId.value,
    payment_method: paymentMethod.value,
    amount_paid: amountPaid.value,
  });
}
</script>

<template>
  <Dialog :open="open" @update:open="(v) => emit('update:open', v)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t('admin.pos.paymentTitle') }}</DialogTitle>
        <DialogDescription>{{ t('admin.pos.paymentDescription') }}</DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <div class="space-y-2">
          <Label>{{ t('admin.pos.customer') }}</Label>
          <Select v-model="customerId">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="t('admin.pos.noCustomer')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="NO_CUSTOMER">{{ t('admin.pos.noCustomer') }}</SelectItem>
              <SelectItem v-for="c in customers" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="space-y-2">
          <Label>{{ t('admin.pos.paymentMethod') }}</Label>
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
          <div class="text-sm text-muted-foreground">{{ t('admin.pos.grandTotal') }}</div>
          <div class="text-2xl font-bold tabular-nums">{{ formatCurrency(grandTotal) }}</div>
        </div>

        <div class="space-y-2">
          <Label>{{ t('admin.pos.amountPaid') }}</Label>
          <Input
            v-model="amountPaid"
            type="number"
            min="0"
            placeholder="0"
            class="text-lg h-11 tabular-nums"
            @update:modelValue="setAmountPaid"
          />
        </div>

        <div
          v-if="amountPaid > 0 && amountPaid < grandTotal"
          class="text-center text-destructive font-medium"
        >
          {{ t('admin.pos.insufficient') }}
        </div>
        <div v-else-if="amountPaid === 0" class="text-center text-muted-foreground text-sm">
          {{ t('admin.pos.inputAmountToSeeChange') }}
        </div>
        <div v-else class="text-center text-emerald-600 font-medium">
          {{ t('admin.pos.change') }}: {{ formatCurrency(change) }}
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="cancel">{{ t('admin.shared.cancel') }}</Button>
        <Button :disabled="processing || amountPaid < grandTotal" @click="submit">
          {{ processing ? t('admin.pos.processing') : t('admin.pos.pay') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
