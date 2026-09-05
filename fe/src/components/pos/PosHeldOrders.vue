<script setup lang="ts">
import { ref } from 'vue';
import Button from '@/components/ui/button/Button.vue';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { useLocaleStore } from '@/stores/locale';
import { formatCurrency, formatDateTime } from '@/lib/utils';
import type { HeldOrder } from '@/types';

/**
 * Presentational UI for POS "held orders" (docs/pos-hold-order.md).
 *
 * The parent owns the data (useHeldOrders) and all actions; this component
 * only renders the entry button and the dialogs, and asks the parent to act:
 *
 * - `restore`                — restore order (current cart is empty)
 * - `hold-current-and-restore` — user chose to hold the current cart first
 * - `replace-and-restore`    — user chose to replace the current cart
 * - `delete`                 — user confirmed deleting a held order
 */
const props = defineProps<{
  orders: HeldOrder[];
  /** True while the parent is applying a restore (disables the actions). */
  restoring?: boolean;
  /** Number of lines currently in the cart (drives the replace flow). */
  cartCount: number;
}>();

const emit = defineEmits<{
  restore: [order: HeldOrder];
  holdCurrentAndRestore: [order: HeldOrder];
  replaceAndRestore: [order: HeldOrder];
  delete: [order: HeldOrder];
}>();

const t = useLocaleStore().t;

const listOpen = ref(false);
const deleteOpen = ref(false);
const deleteTarget = ref<HeldOrder | null>(null);
const replaceOpen = ref(false);
const replaceTarget = ref<HeldOrder | null>(null);

function heldSubtotal(h: HeldOrder): number {
  return (h.items ?? []).reduce((sum, i) => sum + i.unit_price * i.quantity, 0);
}

/** Display total of a held order (subtotal + tax − discount). */
function heldTotal(h: HeldOrder): number {
  const sub = heldSubtotal(h);
  const tax = Math.round(sub * ((h.tax_rate || 0) / 100));
  return sub + tax - (h.discount || 0);
}

function askRestore(h: HeldOrder) {
  if (props.cartCount === 0) {
    listOpen.value = false;
    emit('restore', h);
    return;
  }
  // Current cart is not empty → ask how to proceed (R5).
  listOpen.value = false;
  replaceTarget.value = h;
  replaceOpen.value = true;
}

function confirmHoldAndRestore() {
  const h = replaceTarget.value;
  if (!h) return;
  replaceOpen.value = false;
  replaceTarget.value = null;
  emit('holdCurrentAndRestore', h);
}

function confirmReplace() {
  const h = replaceTarget.value;
  if (!h) return;
  replaceOpen.value = false;
  replaceTarget.value = null;
  emit('replaceAndRestore', h);
}

function cancelReplace() {
  replaceOpen.value = false;
  replaceTarget.value = null;
}

function askDelete(h: HeldOrder) {
  deleteTarget.value = h;
  deleteOpen.value = true;
}

function cancelDelete() {
  deleteOpen.value = false;
  deleteTarget.value = null;
}

function confirmDelete() {
  const h = deleteTarget.value;
  if (!h) return;
  deleteOpen.value = false;
  deleteTarget.value = null;
  emit('delete', h);
}
</script>

<template>
  <!-- Only rendered by the parent inside the cart card; keep root lean. -->
  <div v-if="orders.length > 0" class="px-3 pb-2">
    <Button
      variant="secondary"
      size="sm"
      class="w-full h-8"
      :disabled="restoring"
      @click="listOpen = true"
    >
      {{ t('posHold.restoreHold') }} ({{ orders.length }})
    </Button>
  </div>

  <!-- Held Orders list -->
  <Dialog v-model:open="listOpen">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t('posHold.title') }}</DialogTitle>
        <DialogDescription>{{ t('posHold.description') }}</DialogDescription>
      </DialogHeader>

      <div v-if="orders.length === 0" class="text-center py-10 text-muted-foreground text-sm">
        {{ t('posHold.empty') }}
      </div>
      <div v-else class="space-y-2 max-h-[50vh] overflow-y-auto pr-1">
        <div
          v-for="h in orders"
          :key="h.id"
          class="flex items-center gap-3 rounded-lg border p-3"
        >
          <div class="flex-1 min-w-0">
            <div class="font-medium text-sm truncate">{{ h.label }}</div>
            <div class="text-xs text-muted-foreground mt-0.5">
              {{ t('posHold.itemCount', { count: h.items.length }) }} ·
              {{ t('posHold.heldAt', { time: formatDateTime(h.created_at) }) }}
            </div>
          </div>
          <div class="shrink-0 text-right">
            <div class="text-sm font-semibold tabular-nums">{{ formatCurrency(heldTotal(h)) }}</div>
            <div class="flex justify-end gap-1 mt-1">
              <Button
                size="sm"
                variant="outline"
                class="h-7 px-2 text-xs"
                :disabled="restoring"
                @click="askRestore(h)"
              >
                {{ t('posHold.restore') }}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                class="h-7 px-2 text-xs text-destructive"
                :disabled="restoring"
                @click="askDelete(h)"
              >
                {{ t('posHold.delete') }}
              </Button>
            </div>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="listOpen = false">
          {{ t('admin.shared.cancel') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Delete held order confirm (R9) -->
  <Dialog v-model:open="deleteOpen">
    <DialogContent class="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ t('posHold.confirmDeleteTitle') }}</DialogTitle>
        <DialogDescription>
          {{ t('posHold.confirmDeleteDesc', { label: deleteTarget?.label ?? '' }) }}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter>
        <Button variant="outline" @click="cancelDelete">{{ t('admin.shared.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('posHold.delete') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Restore over a non-empty cart (R5) -->
  <Dialog v-model:open="replaceOpen">
    <DialogContent class="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ t('posHold.confirmReplaceTitle') }}</DialogTitle>
        <DialogDescription>
          {{ t('posHold.confirmReplaceDesc', { count: cartCount }) }}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter class="flex-col sm:flex-row">
        <Button variant="outline" @click="cancelReplace">{{ t('admin.shared.cancel') }}</Button>
        <Button :disabled="restoring" @click="confirmHoldAndRestore">
          {{ t('posHold.actionHoldAndRestore') }}
        </Button>
        <Button variant="destructive" :disabled="restoring" @click="confirmReplace">
          {{ t('posHold.actionReplace') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
