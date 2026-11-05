<script setup lang="ts">
/**
 * /admin/tables — per-store dine-in tables for self ordering
 * (docs/pos-self-order.md §7). CRUD + QR print + token rotation.
 */
import { ref, onMounted, watch } from 'vue';
import QRCode from 'qrcode';
import { tableService } from '@/services/table';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { Table } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import { Label } from '@/components/ui/label';
import Badge from '@/components/ui/badge/Badge.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';

const { isReady } = usePageGuard();
const ui = useUiStore();
const t = useLocaleStore().t;

const tables = ref<Table[]>([]);
const totalCount = ref(0);
const page = ref(1);
const perPage = 20;
const loading = ref(false);
const saving = ref(false);

const dialogOpen = ref(false);
const editing = ref<Table | null>(null);
const form = ref({ code: '', name: '' });

// QR dialog
const qrOpen = ref(false);
const qrTable = ref<Table | null>(null);
const qrDataUrl = ref('');

async function load() {
  loading.value = true;
  try {
    const res = await tableService.list({ page: page.value, per_page: perPage });
    tables.value = res.tables ?? [];
    totalCount.value = res.total_count ?? 0;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editing.value = null;
  form.value = { code: '', name: '' };
  dialogOpen.value = true;
}

function openEdit(t: Table) {
  editing.value = t;
  form.value = { code: t.code, name: t.name };
  dialogOpen.value = true;
}

async function save() {
  if (!form.value.code.trim()) {
    ui.addToast('error', t('admin.shared.save') + ' — ' + t('admin.tables.code'));
    return;
  }
  saving.value = true;
  try {
    if (editing.value) {
      await tableService.update(editing.value.id, { code: form.value.code, name: form.value.name });
      ui.addToast('success', t('admin.tables.toastUpdated'));
    } else {
      await tableService.create({ code: form.value.code, name: form.value.name });
      ui.addToast('success', t('admin.tables.toastCreated'));
    }
    dialogOpen.value = false;
    await load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function toggleActive(tbl: Table) {
  try {
    await tableService.update(tbl.id, { is_active: !tbl.is_active });
    ui.addToast('success', t('admin.tables.' + (tbl.is_active ? 'toastDeactivated' : 'toastActivated')));
    await load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function remove(tbl: Table) {
  try {
    await tableService.delete(tbl.id);
    ui.addToast('success', t('admin.tables.toastDeleted'));
    await load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function rotate(tbl: Table) {
  try {
    await tableService.rotateToken(tbl.id);
    ui.addToast('success', t('admin.tables.toastRotated'));
    await load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function openQr(tbl: Table) {
  qrTable.value = tbl;
  qrDataUrl.value = '';
  qrOpen.value = true;
  try {
    const absolute = `${window.location.origin}${tbl.qr_url}`;
    qrDataUrl.value = await QRCode.toDataURL(absolute, { width: 320, margin: 1, errorCorrectionLevel: 'M' });
  } catch {
    qrDataUrl.value = '';
  }
}

function printQr() {
  const w = window.open('', '_blank', 'width=400,height=520');
  if (!w) return;
  w.document.write(`<html><head><title>${qrTable.value?.code ?? ''}</title></head>`);
  w.document.write('<body style="font-family:sans-serif;text-align:center;padding:24px;">');
  w.document.write(`<h2>${t('selfOrder.tableLabel', { code: qrTable.value?.code ?? '' })}</h2>`);
  w.document.write(`<p>${t('admin.tables.printQrDesc')}</p>`);
  if (qrDataUrl.value) w.document.write(`<img src="${qrDataUrl.value}" style="width:280px;height:280px;" />`);
  w.document.write('<p style="color:#666;">' + t('sidebar.appName') + '</p>');
  w.document.write('</body></html>');
  w.document.close();
  w.focus();
  w.print();
}

watch(isReady, (r) => { if (r) void load(); }, { immediate: true });
onMounted(() => { void load(); });
</script>

<template>
  <DashboardShell>
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-2xl font-bold">{{ t('admin.tables.title') }}</h2>
          <p class="text-muted-foreground mt-1">{{ t('admin.tables.description') }}</p>
        </div>
        <Button @click="openCreate">{{ t('admin.tables.newTable') }}</Button>
      </div>

      <Card>
        <CardContent class="p-0">
          <div v-if="loading" class="flex justify-center py-12 text-sm text-muted-foreground">{{ t('admin.shared.loading') }}</div>
          <div v-else-if="tables.length === 0" class="py-12 text-center text-sm text-muted-foreground">
            {{ t('admin.tables.noData') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b text-left text-muted-foreground">
                  <th class="px-4 py-2.5 font-medium">{{ t('admin.tables.code') }}</th>
                  <th class="px-4 py-2.5 font-medium">{{ t('admin.tables.name') }}</th>
                  <th class="px-4 py-2.5 font-medium">{{ t('admin.shared.status') }}</th>
                  <th class="px-4 py-2.5 font-medium">{{ t('admin.tables.qr') }}</th>
                  <th class="px-4 py-2.5 text-right font-medium">{{ t('admin.shared.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="tbl in tables" :key="tbl.id" class="border-b last:border-0">
                  <td class="px-4 py-2.5 font-semibold">{{ tbl.code }}</td>
                  <td class="px-4 py-2.5">{{ tbl.name || '—' }}</td>
                  <td class="px-4 py-2.5">
                    <Badge :variant="tbl.is_active ? 'default' : 'secondary'">
                      {{ tbl.is_active ? t('admin.shared.active') : t('admin.shared.inactive') }}
                    </Badge>
                  </td>
                  <td class="px-4 py-2.5">
                    <Button size="sm" variant="outline" @click="openQr(tbl)">{{ t('admin.tables.printQr') }}</Button>
                  </td>
                  <td class="px-4 py-2.5">
                    <div class="flex justify-end gap-1.5">
                      <Button size="sm" variant="outline" @click="openEdit(tbl)">{{ t('admin.shared.edit') }}</Button>
                      <Button size="sm" variant="outline" @click="toggleActive(tbl)">
                        {{ tbl.is_active ? t('admin.tables.deactivate') : t('admin.tables.activate') }}
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger as-child>
                          <Button size="sm" variant="outline">{{ t('admin.tables.rotateToken') }}</Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>{{ t('admin.tables.rotateTokenTitle') }}</AlertDialogTitle>
                            <AlertDialogDescription>{{ t('admin.tables.rotateTokenDescription') }}</AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                            <AlertDialogAction @click="rotate(tbl)">{{ t('admin.tables.rotate') }}</AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                      <AlertDialog>
                        <AlertDialogTrigger as-child>
                          <Button size="sm" variant="destructive">{{ t('admin.shared.delete') }}</Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>{{ t('admin.tables.deleteTitle') }}</AlertDialogTitle>
                            <AlertDialogDescription>{{ t('admin.tables.deleteDescription', { code: tbl.code }) }}</AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                            <AlertDialogAction @click="remove(tbl)">{{ t('admin.shared.delete') }}</AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      <!-- Create / edit dialog -->
      <Dialog v-model:open="dialogOpen">
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{{ editing ? t('admin.tables.editTitle') : t('admin.tables.createTitle') }}</DialogTitle>
            <DialogDescription>{{ t('admin.tables.dialogDescription') }}</DialogDescription>
          </DialogHeader>
          <div class="space-y-3">
            <div class="space-y-1.5">
              <Label>{{ t('admin.tables.code') }}</Label>
              <Input v-model="form.code" :placeholder="t('admin.tables.codePlaceholder')" class="h-9" />
            </div>
            <div class="space-y-1.5">
              <Label>{{ t('admin.tables.name') }}</Label>
              <Input v-model="form.name" :placeholder="t('admin.tables.namePlaceholder')" class="h-9" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="dialogOpen = false">{{ t('admin.shared.cancel') }}</Button>
            <Button :disabled="saving" @click="save">
              {{ saving ? t('admin.shared.saving') : t('admin.shared.save') }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- QR dialog -->
      <Dialog v-model:open="qrOpen">
        <DialogContent class="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{{ t('admin.tables.printQrTitle', { code: qrTable?.code ?? '' }) }}</DialogTitle>
            <DialogDescription>{{ t('admin.tables.printQrDesc') }}</DialogDescription>
          </DialogHeader>
          <div class="flex flex-col items-center gap-3 py-2">
            <div v-if="qrDataUrl" class="rounded-lg border p-2">
              <img :src="qrDataUrl" alt="QR" class="h-56 w-56" />
            </div>
            <div v-else class="flex h-56 w-56 items-center justify-center text-sm text-muted-foreground">
              {{ t('admin.shared.loading') }}
            </div>
            <p class="text-sm font-medium">{{ t('selfOrder.tableLabel', { code: qrTable?.code ?? '' }) }}</p>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="qrOpen = false">{{ t('admin.tables.close') }}</Button>
            <Button :disabled="!qrDataUrl" @click="printQr">{{ t('admin.tables.print') }}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  </DashboardShell>
</template>