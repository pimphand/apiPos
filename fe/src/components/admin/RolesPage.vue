<script setup lang="ts">
import { onMounted, ref, computed } from 'vue';
import { roleService } from '@/services/role';
import { permissionService } from '@/services/permission';
import { useUiStore } from '@/stores/ui';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { Role, Permission } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import Checkbox from '@/components/ui/checkbox/Checkbox.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed } = usePageGuard('roles.read');
const ui = useUiStore();

const DEFAULT_ROLES = new Set(['admin', 'manager', 'cashier']);

const roles = ref<Role[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);

// Role form dialog
const formOpen = ref(false);
const editing = ref<Role | null>(null);
const saving = ref(false);
const form = ref({ name: '', description: '' });

// Permissions dialog
const permOpen = ref(false);
const permRole = ref<Role | null>(null);
const allPermissions = ref<Permission[]>([]);
const selectedPerms = ref<Set<string>>(new Set());
const permLoading = ref(false);
const permSaving = ref(false);

const groupedPermissions = computed(() => {
  const map = new Map<string, Permission[]>();
  for (const p of allPermissions.value) {
    const key = p.resource;
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(p);
  }
  return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]));
});

async function load() {
  loading.value = true;
  try {
    const res = await roleService.list({ page: page.value, per_page: perPage });
    roles.value = res.roles;
    total.value = res.total_count;
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editing.value = null;
  form.value = { name: '', description: '' };
  formOpen.value = true;
}

function openEdit(r: Role) {
  editing.value = r;
  form.value = { name: r.name, description: r.description || '' };
  formOpen.value = true;
}

async function save() {
  saving.value = true;
  try {
    if (editing.value) {
      await roleService.update(editing.value.id, form.value);
      ui.addToast('success', 'Role updated');
    } else {
      await roleService.create(form.value);
      ui.addToast('success', 'Role created');
    }
    formOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function removeRole(r: Role) {
  try {
    await roleService.delete(r.id);
    ui.addToast('success', 'Role deleted');
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function openPermissions(r: Role) {
  permRole.value = r;
  permOpen.value = true;
  permLoading.value = true;
  selectedPerms.value = new Set();
  try {
    const [perms, rolePerms] = await Promise.all([
      permissionService.list({ page: 1, per_page: 100 }),
      roleService.getPermissions(r.id),
    ]);
    allPermissions.value = perms.permissions;
    selectedPerms.value = new Set(rolePerms.map((p) => p.id));
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    permLoading.value = false;
  }
}

function togglePerm(id: string, checked: boolean) {
  if (checked) selectedPerms.value.add(id);
  else selectedPerms.value.delete(id);
  // force reactivity on Set
  selectedPerms.value = new Set(selectedPerms.value);
}

async function savePermissions() {
  if (!permRole.value) return;
  permSaving.value = true;
  try {
    await roleService.assignPermissions(permRole.value.id, Array.from(selectedPerms.value));
    ui.addToast('success', 'Permissions updated');
    permOpen.value = false;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    permSaving.value = false;
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
    <p class="text-muted-foreground mt-2 max-w-sm">You don't have permission to manage roles.</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">Roles</h2>
        <p class="text-muted-foreground mt-1">Define roles and their permissions. Default roles cannot be deleted.</p>
      </div>
      <Button @click="openCreate">+ New Role</Button>
    </div>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Permissions</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="4" class="text-center py-8 text-muted-foreground">Loading...</TableCell>
            </TableRow>
            <TableEmpty v-else-if="roles.length === 0" :colspan="4">No roles found</TableEmpty>
            <TableRow v-for="r in roles" v-else :key="r.id">
              <TableCell>
                <div class="flex items-center gap-2">
                  <span class="font-medium">{{ r.name }}</span>
                  <Badge v-if="DEFAULT_ROLES.has(r.name)" variant="secondary">default</Badge>
                </div>
              </TableCell>
              <TableCell class="text-muted-foreground">{{ r.description || '—' }}</TableCell>
              <TableCell>
                <Button variant="outline" size="sm" @click="openPermissions(r)">Assign Permissions</Button>
              </TableCell>
              <TableCell class="text-right">
                <div class="flex justify-end gap-2">
                  <Button variant="outline" size="sm" @click="openEdit(r)">Edit</Button>
                  <AlertDialog v-if="!DEFAULT_ROLES.has(r.name)">
                    <AlertDialogTrigger as-child>
                      <Button variant="destructive" size="sm">Delete</Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete role?</AlertDialogTitle>
                        <AlertDialogDescription>
                          This will permanently delete the role "{{ r.name }}".
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction @click="removeRole(r)">Delete</AlertDialogAction>
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

    <!-- Role create/edit dialog -->
    <Dialog v-model:open="formOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit Role' : 'Create Role' }}</DialogTitle>
          <DialogDescription>Enter the role name and an optional description.</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>Name</Label>
            <Input v-model="form.name" placeholder="e.g. supervisor" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>Description</Label>
            <Textarea v-model="form.description" placeholder="Role description" :rows="3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">Cancel</Button>
          <Button :disabled="saving" @click="save">{{ saving ? 'Saving...' : 'Save' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Permissions dialog -->
    <Dialog v-model:open="permOpen">
      <DialogContent class="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Permissions — {{ permRole?.name }}</DialogTitle>
          <DialogDescription>Select the permissions this role should have.</DialogDescription>
        </DialogHeader>
        <div v-if="permLoading" class="py-8 text-center text-muted-foreground">Loading permissions...</div>
        <div v-else class="space-y-4">
          <div v-for="[resource, items] in groupedPermissions" :key="resource">
            <h4 class="font-medium capitalize mb-2">{{ resource }}</h4>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <label
                v-for="p in items"
                :key="p.id"
                class="flex items-center gap-2 rounded-md border px-3 py-2 text-sm cursor-pointer hover:bg-muted"
              >
                <Checkbox
                  :model-value="selectedPerms.has(p.id)"
                  @update:model-value="(v) => togglePerm(p.id, v === true)"
                />
                <span class="font-mono text-xs">{{ p.name }}</span>
              </label>
            </div>
          </div>
          <p v-if="allPermissions.length === 0" class="text-muted-foreground">No permissions available.</p>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="permOpen = false">Cancel</Button>
          <Button :disabled="permSaving" @click="savePermissions">{{ permSaving ? 'Saving...' : 'Save' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
