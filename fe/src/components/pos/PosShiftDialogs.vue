<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useLocaleStore } from '@/stores/locale';
import { useUiStore } from '@/stores/ui';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency } from '@/lib/utils';
import type { CashierShift, ShiftItem, ShiftSummary } from '@/types';

/**
 * All cashier-shift dialogs of the POS page (open / close / diff-confirm /
 * closed-result) plus their local transient state and the OUT/IN items fetch.
 *
 * The parent owns the `useCashierShift` composable (single source of truth)
 * and provides the state via props and every side-effectful action as a
 * function prop. Open/close are triggered through the exposed methods below.
 */
const props = defineProps<{
  currentShift: CashierShift | null;
  summary: ShiftSummary | null;
  summaryLoading: boolean;
  opening: boolean;
  closing: boolean;
  /** Opens the shift for `openingCash`; returns true on success. */
  openShiftAction: (openingCash: number) => Promise<boolean>;
  /** Closes the shift (idempotent) and returns the final summary. */
  closeShiftAction: (
    shiftId: string,
    actualCash: number,
    closingCash: number
  ) => Promise<ShiftSummary>;
  /** Loads a shift summary into the parent state; returns it (or null). */
  loadSummaryAction: (shiftId?: string) => Promise<ShiftSummary | null>;
  /** Fetches OUT/IN item ledger rows for a shift. */
  fetchShiftItemsAction: (
    shiftId: string
  ) => Promise<{ out: ShiftItem[]; in: ShiftItem[] }>;
}>();

const t = useLocaleStore().t;
const ui = useUiStore();

// Open shift dialog
const openDialog = ref(false);
const openForm = reactive({ opening_cash: 0 });

// Close shift dialog
const closeDialog = ref(false);
const closeActual = ref(0);
const closeClosing = ref(0);
const diffConfirm = ref(false);
const closedResult = ref<ShiftSummary | null>(null);
const resultDialog = ref(false);

// OUT/IN item ledger shown inside the close dialog
const itemsDirection = ref<'OUT' | 'IN'>('OUT');
const itemsOut = ref<ShiftItem[]>([]);
const itemsIn = ref<ShiftItem[]>([]);
const itemsLoading = ref(false);

const differencePreview = computed(
  () => (Number(closeActual.value) || 0) - (props.summary?.expected_cash ?? 0)
);

function openOpenShiftDialog() {
  openForm.opening_cash = 0;
  openDialog.value = true;
}

async function submitOpenShift() {
  if (props.opening) return;
  const ok = await props.openShiftAction(openForm.opening_cash);
  if (ok) openDialog.value = false;
}

async function loadShiftItems(shiftId: string) {
  itemsLoading.value = true;
  try {
    const res = await props.fetchShiftItemsAction(shiftId);
    itemsOut.value = res.out;
    itemsIn.value = res.in;
  } finally {
    itemsLoading.value = false;
  }
}

async function openCloseDialog() {
  const shift = props.currentShift;
  if (!shift) return;
  itemsDirection.value = 'OUT';
  closeActual.value = 0;
  closeClosing.value = 0;
  diffConfirm.value = false;
  closeDialog.value = true;
  const s = await props.loadSummaryAction();
  if (s && closeActual.value === 0) {
    // Suggest expected cash; the cashier may edit before confirming.
    closeActual.value = s.expected_cash ?? 0;
    closeClosing.value = closeActual.value;
  }
  await loadShiftItems(shift.id);
}

function askClose() {
  // Require an explicit confirmation when the counted cash differs.
  if (differencePreview.value !== 0) {
    diffConfirm.value = true;
  } else {
    void performClose();
  }
}

async function performClose() {
  const id = props.currentShift?.id;
  if (!id || props.closing) return;
  try {
    // Inputs may come back as strings from the number input when cleared.
    const actual = Number(closeActual.value) || 0;
    const closingCash = Number(closeClosing.value) || 0;
    const finalSummary = await props.closeShiftAction(id, actual, closingCash);
    ui.addToast('success', t('posShift.toastClosed'));
    closeDialog.value = false;
    diffConfirm.value = false;
    closedResult.value = finalSummary;
    resultDialog.value = true;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

defineExpose({
  openOpenShiftDialog,
  openCloseDialog,
});
</script>

<template>
  <!-- Open Shift Dialog -->
  <Dialog v-model:open="openDialog">
    <DialogContent class="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ t('posShift.openShiftTitle') }}</DialogTitle>
        <DialogDescription>{{ t('posShift.openShiftDesc') }}</DialogDescription>
      </DialogHeader>
      <div class="space-y-2">
        <Label>{{ t('posShift.openingCashLabel') }}</Label>
        <Input
          v-model.number="openForm.opening_cash"
          type="number"
          min="0"
          :placeholder="t('posShift.placeholderCash')"
        />
      </div>
      <DialogFooter>
        <Button variant="outline" @click="openDialog = false">{{ t('admin.shared.cancel') }}</Button>
        <Button :disabled="opening" @click="submitOpenShift">
          {{ opening ? t('posShift.opening') : t('posShift.openShift') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Close Shift Dialog -->
  <Dialog v-model:open="closeDialog">
    <DialogContent class="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ t('posShift.closeShiftTitle') }}</DialogTitle>
        <DialogDescription>{{ t('posShift.closeShiftDesc') }}</DialogDescription>
      </DialogHeader>

      <div v-if="summaryLoading" class="text-center py-10 text-muted-foreground">
        {{ t('admin.shared.loading') }}
      </div>
      <template v-else-if="summary">
        <!-- Reconciliation (server numbers) -->
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 text-sm rounded-lg bg-muted/40 p-3">
          <div>
            <div class="text-muted-foreground">{{ t('posShift.openingCashLabel') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(summary.opening_cash) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.cashSales') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(summary.cash_sales) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.cashRefund') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(summary.cash_refund) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.totalTransactions') }}</div>
            <div class="font-medium tabular-nums">{{ summary.total_transactions }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.totalSalesAmount') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(summary.total_sales_amount) }}</div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('posShift.expectedCash') }}</div>
            <div class="font-medium tabular-nums">{{ formatCurrency(summary.expected_cash) }}</div>
          </div>
        </div>

        <!-- Cashier inputs -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4">
          <div class="space-y-1">
            <Label>{{ t('posShift.actualCashLabel') }}</Label>
            <Input
              v-model.number="closeActual"
              type="number"
              min="0"
              class="text-lg h-11 tabular-nums"
              :placeholder="t('posShift.actualPlaceholder')"
            />
          </div>
          <div class="space-y-1">
            <Label>{{ t('posShift.closingCashLabel') }}</Label>
            <Input
              v-model.number="closeClosing"
              type="number"
              min="0"
              class="text-lg h-11 tabular-nums"
              :placeholder="t('posShift.placeholderCash')"
            />
          </div>
        </div>

        <!-- Difference preview -->
        <div class="rounded-lg border p-3 mt-3 flex items-center justify-between">
          <div>
            <div class="text-sm font-medium">{{ t('posShift.cashDifferenceLabel') }}</div>
            <div class="text-xs text-muted-foreground">{{ t('posShift.differenceHint') }}</div>
          </div>
          <span
            class="text-xl font-bold tabular-nums"
            :class="differencePreview === 0 ? '' : (differencePreview > 0 ? 'text-emerald-600' : 'text-destructive')"
          >{{ differencePreview > 0 ? '+' : '' }}{{ formatCurrency(differencePreview) }}</span>
        </div>

        <!-- Items moved (goods OUT/IN) -->
        <div class="mt-4">
          <div class="flex items-center justify-between mb-2">
            <h4 class="font-semibold text-sm">{{ t('posShift.itemDetail') }}</h4>
            <div class="flex gap-1">
              <Button
                variant="outline"
                size="sm"
                :class="itemsDirection === 'OUT' ? 'bg-accent' : ''"
                @click="itemsDirection = 'OUT'"
              >
                {{ t('posShift.outItems') }} ({{ summary.items_out_total ?? 0 }})
              </Button>
              <Button
                variant="outline"
                size="sm"
                :class="itemsDirection === 'IN' ? 'bg-accent' : ''"
                @click="itemsDirection = 'IN'"
              >
                {{ t('posShift.inItems') }} ({{ summary.items_in_total ?? 0 }})
              </Button>
            </div>
          </div>
          <div v-if="itemsLoading" class="text-center py-6 text-muted-foreground text-sm">
            {{ t('admin.shared.loading') }}
          </div>
          <div
            v-else-if="(itemsDirection === 'OUT' ? itemsOut : itemsIn).length === 0"
            class="text-center py-6 text-muted-foreground text-sm"
          >
            {{ t('posShift.noItems') }}
          </div>
          <Table v-else>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('posShift.product') }}</TableHead>
                <TableHead class="text-right">{{ t('posShift.qty') }}</TableHead>
                <TableHead class="text-right">{{ t('posShift.amount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow
                v-for="it in (itemsDirection === 'OUT' ? itemsOut : itemsIn)"
                :key="it.id"
              >
                <TableCell>
                  <div class="text-sm font-medium">{{ it.product_name }}</div>
                  <div v-if="it.sku_code || it.variant_label" class="text-xs text-muted-foreground">
                    <span v-if="it.sku_code">{{ it.sku_code }}</span>
                    <span v-if="it.variant_label"> · {{ it.variant_label }}</span>
                  </div>
                  <div v-if="it.reference_no" class="text-xs text-muted-foreground font-mono">
                    {{ it.reference_no }}
                  </div>
                </TableCell>
                <TableCell class="text-right tabular-nums">{{ it.quantity }}</TableCell>
                <TableCell class="text-right tabular-nums">{{ formatCurrency(it.line_amount) }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </template>

      <DialogFooter class="mt-4">
        <Button variant="outline" @click="closeDialog = false">{{ t('admin.shared.cancel') }}</Button>
        <Button variant="destructive" :disabled="closing || !summary" @click="askClose">
          {{ closing ? t('posShift.closing') : t('posShift.close') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Confirm close with difference -->
  <Dialog v-model:open="diffConfirm">
    <DialogContent class="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ t('posShift.confirmTitle') }}</DialogTitle>
        <DialogDescription>
          {{ t('posShift.confirmDescription', { amount: formatCurrency(differencePreview) }) }}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter>
        <Button variant="outline" @click="diffConfirm = false">{{ t('admin.shared.cancel') }}</Button>
        <Button variant="destructive" :disabled="closing" @click="performClose">
          {{ closing ? t('posShift.closing') : t('posShift.close') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Closed result summary -->
  <Dialog v-model:open="resultDialog">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t('posShift.resultTitle') }}</DialogTitle>
        <DialogDescription>{{ t('posShift.summaryTitle') }}</DialogDescription>
      </DialogHeader>
      <div v-if="closedResult" class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <div class="text-muted-foreground">{{ t('posShift.totalTransactions') }}</div>
          <div class="font-medium tabular-nums">{{ closedResult.total_transactions }}</div>
        </div>
        <div>
          <div class="text-muted-foreground">{{ t('posShift.totalSalesAmount') }}</div>
          <div class="font-medium tabular-nums">{{ formatCurrency(closedResult.total_sales_amount) }}</div>
        </div>
        <div>
          <div class="text-muted-foreground">{{ t('posShift.expectedCash') }}</div>
          <div class="font-medium tabular-nums">{{ formatCurrency(closedResult.expected_cash) }}</div>
        </div>
        <div>
          <div class="text-muted-foreground">{{ t('posShift.actualCashLabel') }}</div>
          <div class="font-medium tabular-nums">{{ formatCurrency(closedResult.actual_cash) }}</div>
        </div>
        <div>
          <div class="text-muted-foreground">{{ t('posShift.cashDifferenceLabel') }}</div>
          <div
            class="font-medium tabular-nums"
            :class="closedResult.cash_difference !== 0 ? (closedResult.cash_difference > 0 ? 'text-emerald-600' : 'text-destructive') : ''"
          >
            {{ closedResult.cash_difference > 0 ? '+' : '' }}{{ formatCurrency(closedResult.cash_difference) }}
          </div>
        </div>
        <div>
          <div class="text-muted-foreground">{{ t('posShift.itemsOut') }}</div>
          <div class="font-medium tabular-nums">{{ closedResult.items_out_total ?? 0 }}</div>
        </div>
      </div>
      <DialogFooter>
        <Button @click="resultDialog = false">{{ t('posShift.closeResult') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
