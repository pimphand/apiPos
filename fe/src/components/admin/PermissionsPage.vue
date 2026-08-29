<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { permissionService } from '@/services/permission';
import { useUiStore } from '@/stores/ui';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { Permission } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed } = usePageGuard('permissions.read');
const ui = useUiStore();

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
    const res = resourceFilter.value
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
      ui.addToast('success', 'Permission updated');
    } else {
      await permissionService.create(form.value);
      ui.addToast('success', 'Permission created');
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
    ui.addToast('success', 'Permission deleted');
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

onMounted(load);
</script>

<template>
  <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
  </div>

  <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">Access restricted</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">You don't have permission to manage permissions.</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">Permissions</h2>
        <p class="text-muted-foreground mt-1">Manage granular permission definitions keyed by resource and action.</p>
      </div>
      <Button @click="openCreate">+ New Permission</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>Resource</Label>
          <Select v-model="resourceFilter">
            <SelectTrigger class="w-48"><SelectValue placeholder="All resources" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">All resources</SelectItem>
              <SelectItem v-for="r in RESOURCES" :key="r" :value="r">{{ r }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" @click="applyResourceFilter">Filter</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Resource</TableHead>
              <TableHead>Action</TableHead>
              <TableHead>Description</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="5" class="text-center py-8 text-muted-foreground">Loading...</TableCell>
            </TableRow>
            <TableEmpty v-else-if="permissions.length === 0" :colspan="5">No permissions found</TableEmpty>
            <TableRow v-for="p in permissions" v-else :key="p.id">
              <TableCell><span class="font-mono text-sm font-medium">{{ p.name }}</span></TableCell>
              <TableCell><Badge variant="outline">{{ p.resource }}</Badge></TableCell>
              <TableCell><Badge variant="secondary">{{ p.action }}</Badge></TableCell>
              <TableCell class="text-muted-foreground">{{ p.description || '—' }}</TableCell>
              <TableCell class="text-right">
                <div class="flex justify-end gap-2">
                  <Button variant="outline" size="sm" @click="openEdit(p)">Edit</Button>
                  <AlertDialog>
                    <AlertDialogTrigger as-child>
                      <Button variant="destructive" size="sm">Delete</Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete permission?</AlertDialogTitle>
                        <AlertDialogDescription>
                          This will permanently delete "{{ p.name }}".
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction @click="removePermission(p)">Delete</AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </TableCell>
            </TableRow>
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
          <DialogTitle>{{ editing ? 'Edit Permission' : 'Create Permission' }}</DialogTitle>
          <DialogDescription>Permissions follow the {resource}.{action} format.</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>Name</Label>
            <Input v-model="form.name" placeholder="e.g. products.create" autocomplete="off" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2">
              <Label>Resource</Label>
              <Select v-model="form.resource">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="r in RESOURCES" :key="r" :value="r">{{ r }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2">
              <Label>Action</Label>
              <Select v-model="form.action">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="a in ACTIONS" :key="a" :value="a">{{ a }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div class="space-y-2">
            <Label>Description</Label>
            <Textarea v-model="form.description" placeholder="Permission description" :rows="3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">Cancel</Button>
          <Button :disabled="saving" @click="save">{{ saving ? 'Saving...' : 'Save' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
