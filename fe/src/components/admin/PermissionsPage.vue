<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { permissionService } from '@/services/permission';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { can } from '@/lib/rbac';
import type { Permission } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed, authStore } = usePageGuard('permissions.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const RESOURCES = [
  'users', 'roles', 'permissions', 'categories', 'products', 'inventory',
  'customers', 'sales', 'payments', 'cashier', 'reports', 'activity_logs',
];
const ACTIONS = ['create', 'read', 'update', 'delete', 'assign', 'adjust', 'cancel', 'refund', 'open_shift', 'close_shift', 'view_shift'];

const permissions = ref<Permission[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const resourceFilter = ref('');

const formOpen = ref(false);
const editing = ref<Permission | null>(null);
const saving = ref(false);
const form = ref({ name: '', resource: RESOURCES[0], action: 'read', description: '' });

async function load() {
  loading.value = true;
  try {
    const res = resourceFilter.value && resourceFilter.value !== 'all'
      ? await permissionService.listByResource(resourceFilter.value, { page: page.value, per_page: perPage })
      : await permissionService.list({ page: page.value, per_page: perPage });
    permissions.value = res.permissions;
    total.value = res.total_count;
  } finally {
    loading.value = false;
  }
}

function applyResourceFilter() {
  page.value = 1;
  load();
}

function openCreate() {
  editing.value = null;
  form.value = { name: '', resource: RESOURCES[0], action: 'read', description: '' };
  formOpen.value = true;
}

function openEdit(p: Permission) {
  editing.value = p;
  form.value = {
    name: p.name,
    resource: p.resource,
    action: p.action,
    description: p.description || '',
  };
  formOpen.value = true;
}

async function save() {
  saving.value = true;
  try {
    if (editing.value) {
      await permissionService.update(editing.value.id, form.value);
      ui.addToast('success', t('admin.permissions.toastUpdated'));
    } else {
      await permissionService.create(form.value);
      ui.addToast('success', t('admin.permissions.toastCreated'));
    }
    formOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function removePermission(p: Permission) {
  try {
    await permissionService.delete(p.id);
    ui.addToast('success', t('admin.permissions.toastDeleted'));
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

const canCreate = computed(() => can(authStore.currentUser, 'permissions.create'));
const canUpdate = computed(() => can(authStore.currentUser, 'permissions.update'));
const canDelete = computed(() => can(authStore.currentUser, 'permissions.delete'));
// Hide the Actions column entirely when the user has no mutating permissions.
const showActions = computed(() => canCreate.value || canUpdate.value || canDelete.value);

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
  <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.permissions.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.permissions.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.permissions.description') }}</p>
      </div>
      <Button v-if="canCreate" @click="openCreate">{{ t('admin.permissions.newPermission') }}</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>{{ t('admin.permissions.resource') }}</Label>
          <Select v-model="resourceFilter">
            <SelectTrigger class="w-48"><SelectValue :placeholder="t('admin.permissions.allResources')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.permissions.allResources') }}</SelectItem>
              <SelectItem v-for="r in RESOURCES" :key="r" :value="r">{{ r }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" @click="applyResourceFilter">{{ t('admin.shared.filter') }}</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('admin.shared.name') }}</TableHead>
              <TableHead>{{ t('admin.permissions.resource') }}</TableHead>
              <TableHead>{{ t('admin.permissions.action') }}</TableHead>
              <TableHead>{{ t('admin.shared.description') }}</TableHead>
              <TableHead v-if="showActions" class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="showActions ? 5 : 4" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="permissions.length === 0">
                <TableCell :colspan="showActions ? 5 : 4" class="text-center py-8 text-muted-foreground">{{ t('admin.permissions.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="p in permissions" :key="p.id">
                <TableCell><span class="font-mono text-sm font-medium">{{ p.name }}</span></TableCell>
                <TableCell><Badge variant="outline">{{ p.resource }}</Badge></TableCell>
                <TableCell><Badge variant="secondary">{{ p.action }}</Badge></TableCell>
                <TableCell class="text-muted-foreground">{{ p.description || '—' }}</TableCell>
                <TableCell v-if="showActions" class="text-right">
                  <div class="flex justify-end gap-2">
                    <Button v-if="canUpdate" variant="outline" size="sm" @click="openEdit(p)">{{ t('admin.shared.edit') }}</Button>
                    <AlertDialog v-if="canDelete">
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">{{ t('admin.shared.delete') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.permissions.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('admin.permissions.deleteDescription', { name: p.name }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="removePermission(p)">{{ t('admin.shared.delete') }}</AlertDialogAction>
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
          <DialogTitle>{{ editing ? t('admin.permissions.editTitle') : t('admin.permissions.createTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.permissions.dialogDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>{{ t('admin.shared.name') }}</Label>
            <Input v-model="form.name" :placeholder="t('admin.permissions.namePlaceholder')" autocomplete="off" />
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-2">
              <Label>{{ t('admin.permissions.resource') }}</Label>
              <Select v-model="form.resource">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="r in RESOURCES" :key="r" :value="r">{{ r }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2">
              <Label>{{ t('admin.permissions.action') }}</Label>
              <Select v-model="form.action">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="a in ACTIONS" :key="a" :value="a">{{ a }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div class="space-y-2">
            <Label>{{ t('admin.shared.description') }}</Label>
            <Textarea v-model="form.description" :placeholder="t('admin.permissions.descriptionPlaceholder')" :rows="3" />
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
