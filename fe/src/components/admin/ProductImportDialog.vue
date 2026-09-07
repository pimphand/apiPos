<script setup lang="ts">
import { ref, computed } from 'vue';
import { productService } from '@/services/product';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { apiMessage } from '@/lib/apiError';
import type { ImportResult } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ 'update:open': [value: boolean] }>();

const ui = useUiStore();
const t = useLocaleStore().t;

const file = ref<File | null>(null);
const uploading = ref(false);
const result = ref<ImportResult | null>(null);

const isValid = computed(() => !!file.value);

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  file.value = input.files?.[0] ?? null;
  result.value = null;
}

function downloadSample() {
  productService.downloadSample();
}

async function doImport() {
  if (!file.value) return;
  uploading.value = true;
  result.value = null;
  try {
    const res = await productService.importProducts(file.value);
    result.value = res;
    if (res.failed === 0 && res.imported > 0) {
      ui.addToast('success', t('admin.products.importSuccess', { count: res.imported }));
    } else if (res.imported > 0) {
      ui.addToast('warning', t('admin.products.importPartial', { imported: res.imported, failed: res.failed }));
    } else {
      ui.addToast('error', t('admin.products.importFailed'));
    }
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    uploading.value = false;
  }
}

function close() {
  file.value = null;
  result.value = null;
  emit('update:open', false);
}

function onDrop(e: DragEvent) {
  const f = e.dataTransfer?.files?.[0];
  if (f) {
    file.value = f;
    result.value = null;
  }
}
</script>

<template>
  <Dialog :open="props.open" @update:open="close">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t('admin.products.importTitle') }}</DialogTitle>
        <DialogDescription>{{ t('admin.products.importDescription') }}</DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <!-- Download sample -->
        <div class="flex items-center gap-2 text-sm">
          <span class="text-muted-foreground">{{ t('admin.products.importTemplateHint') }}</span>
          <Button variant="link" class="h-auto p-0 text-sm" @click="downloadSample">
            {{ t('admin.products.importDownloadSample') }}
          </Button>
        </div>

        <!-- Drop zone / file input -->
        <div
          class="relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 text-center transition-colors hover:border-primary/50"
          :class="file ? 'border-primary/30 bg-primary/5' : 'border-muted-foreground/25'"
          @dragover.prevent
          @drop.prevent="onDrop"
        >
          <input
            type="file"
            accept=".csv,.xlsx,.xls"
            class="absolute inset-0 cursor-pointer opacity-0"
            @change="onFileChange"
          />
          <div v-if="file" class="space-y-1">
            <p class="font-medium text-sm">{{ file.name }}</p>
            <p class="text-xs text-muted-foreground">{{ (file.size / 1024).toFixed(1) }} KB</p>
          </div>
          <div v-else class="space-y-1 text-muted-foreground">
            <p class="text-sm font-medium">{{ t('admin.products.importDragDrop') }}</p>
            <p class="text-xs">{{ t('admin.products.importDragDropHint') }}</p>
          </div>
        </div>

        <!-- Results -->
        <div v-if="result" class="rounded-lg border bg-muted/30 p-4 space-y-3">
          <div class="flex items-center gap-4 text-sm">
            <span>{{ t('admin.products.importTotal') }}: <strong>{{ result.total }}</strong></span>
            <span class="text-green-600">{{ t('admin.products.imported') }}: <strong>{{ result.imported }}</strong></span>
            <span v-if="result.failed > 0" class="text-destructive">{{ t('admin.products.importFailedCount') }}: <strong>{{ result.failed }}</strong></span>
          </div>

          <div v-if="result.errors?.length" class="max-h-48 overflow-y-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead class="w-16">{{ t('admin.products.importRow') }}</TableHead>
                  <TableHead>{{ t('admin.products.importError') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="err in result.errors" :key="err.row">
                  <TableCell class="tabular-nums">{{ err.row }}</TableCell>
                  <TableCell class="text-destructive text-sm">{{ err.msg }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="close">{{ t('admin.shared.cancel') }}</Button>
        <Button :disabled="!isValid || uploading" @click="doImport">
          {{ uploading ? t('admin.shared.saving') : t('admin.products.importButton') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
