<script setup lang="ts">
import { ref, watch } from 'vue';
import { categoryService } from '@/services/category';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { Category } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import Switch from '@/components/ui/switch/Switch.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed } = usePageGuard('categories.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const categories = ref<Category[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const activeFilter = ref('');

const formOpen = ref(false);
const editing = ref<Category | null>(null);
const saving = ref(false);
const form = ref({ name: '', description: '' });

async function load() {
  loading.value = true;
  try {
    const res = await categoryService.list({
      is_active: ['true', 'false'].includes(activeFilter.value) ? activeFilter.value : undefined,
      page: page.value,
      per_page: perPage,
    });
    categories.value = res.categories ?? [];
    total.value = res.total_count ?? 0;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

function applyFilter() {
  page.value = 1;
  load();
}

function openCreate() {
  editing.value = null;
  form.value = { name: '', description: '' };
  formOpen.value = true;
}

function openEdit(c: Category) {
  editing.value = c;
  form.value = { name: c.name, description: c.description || '' };
  formOpen.value = true;
}

async function save() {
  saving.value = true;
  try {
    if (editing.value) {
      await categoryService.update(editing.value.id, form.value);
      ui.addToast('success', t('admin.categories.toastUpdated'));
    } else {
      await categoryService.create(form.value);
      ui.addToast('success', t('admin.categories.toastCreated'));
    }
    formOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function toggleActive(c: Category) {
  try {
    if (c.is_active) {
      await categoryService.deactivate(c.id);
      ui.addToast('success', t('admin.categories.toastDeactivated'));
    } else {
      await categoryService.update(c.id, { is_active: true });
      ui.addToast('success', t('admin.categories.toastActivated'));
    }
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function removeCategory(c: Category) {
  try {
    await categoryService.delete(c.id);
    ui.addToast('success', t('admin.categories.toastDeleted'));
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
  <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
  </div>

  <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.categories.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.categories.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.categories.description') }}</p>
      </div>
      <Button @click="openCreate">{{ t('admin.categories.newCategory') }}</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>{{ t('admin.shared.status') }}</Label>
          <Select v-model="activeFilter">
            <SelectTrigger class="w-44"><SelectValue :placeholder="t('admin.shared.all')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.shared.all') }}</SelectItem>
              <SelectItem value="true">{{ t('admin.shared.active') }}</SelectItem>
              <SelectItem value="false">{{ t('admin.shared.inactive') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" @click="applyFilter">{{ t('admin.shared.filter') }}</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('admin.shared.name') }}</TableHead>
              <TableHead>{{ t('admin.shared.description') }}</TableHead>
              <TableHead>{{ t('admin.shared.status') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="categories.length === 0">
                <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">{{ t('admin.categories.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="c in categories" :key="c.id">
                <TableCell class="font-medium">{{ c.name }}</TableCell>
                <TableCell class="text-muted-foreground">{{ c.description || '—' }}</TableCell>
                <TableCell>
                  <Badge :variant="c.is_active ? 'default' : 'secondary'">{{ c.is_active ? t('admin.shared.active') : t('admin.shared.inactive') }}</Badge>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end items-center gap-3">
                    <div class="flex items-center gap-2 text-xs text-muted-foreground">
                      <Switch :model-value="c.is_active" @update:model-value="toggleActive(c)" />
                      {{ c.is_active ? t('admin.shared.active') : t('admin.shared.inactive') }}
                    </div>
                    <Button variant="outline" size="sm" @click="openEdit(c)">{{ t('admin.shared.edit') }}</Button>
                    <AlertDialog>
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">{{ t('admin.shared.delete') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.categories.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('admin.categories.deleteDescription', { name: c.name }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="removeCategory(c)">{{ t('admin.shared.delete') }}</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
        <div class="flex justify-center py-4">
          <PaginationBar :page="page" :per-page="perPage" :total="total" @update:page="page = $event; load()" />
        </div>
      </CardContent>
    </Card>

    <Dialog v-model:open="formOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('admin.categories.editTitle') : t('admin.categories.createTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.categories.dialogDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>{{ t('admin.shared.name') }}</Label>
            <Input v-model="form.name" :placeholder="t('admin.categories.namePlaceholder')" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>{{ t('admin.shared.description') }}</Label>
            <Textarea v-model="form.description" :placeholder="t('admin.categories.descriptionPlaceholder')" :rows="3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="saving" @click="save">{{ saving ? t('admin.shared.saving') : t('admin.shared.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
