<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { userService } from '@/services/user';
import { roleService } from '@/services/role';
import { useUiStore } from '@/stores/ui';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import type { User } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import Switch from '@/components/ui/switch/Switch.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed } = usePageGuard('users.read');
const ui = useUiStore();

interface Role {
  id: string;
  name: string;
}

const users = ref<User[]>([]);
const roles = ref<Role[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const roleFilter = ref('');
const search = ref('');

// Dialog state
const dialogOpen = ref(false);
const editing = ref<User | null>(null);
const saving = ref(false);

const form = ref({
  username: '',
  email: '',
  full_name: '',
  password: '',
  role_id: '',
  store_id: '',
});

const filteredTotal = computed(() => total.value);

async function loadRoles() {
  try {
    const res = await roleService.list({ page: 1, per_page: 100 });
    roles.value = res.roles.map((r) => ({ id: r.id, name: r.name }));
  } catch {
    roles.value = [];
  }
}

async function load() {
  loading.value = true;
  try {
    const res = await userService.list({
      page: page.value,
      per_page: perPage,
      role_id: roleFilter.value !== 'all' ? roleFilter.value || undefined : undefined,
      is_active: undefined,
    });
    users.value = res.users;
    total.value = res.total_count;
  } finally {
    loading.value = false;
  }
}

function applyFilters() {
  page.value = 1;
  load();
}

function openCreate() {
  editing.value = null;
  form.value = { username: '', email: '', full_name: '', password: '', role_id: roles.value[0]?.id || '', store_id: '' };
  dialogOpen.value = true;
}

function openEdit(u: User) {
  editing.value = u;
  form.value = {
    username: u.username,
    email: u.email,
    full_name: u.full_name,
    password: '',
    role_id: u.role_id || '',
    store_id: u.store_id || '',
  };
  dialogOpen.value = true;
}

async function save() {
  saving.value = true;
  try {
    if (editing.value) {
      await userService.update(editing.value.id, {
        username: form.value.username,
        email: form.value.email,
        full_name: form.value.full_name,
      });
      if (form.value.role_id && form.value.role_id !== editing.value.role_id) {
        await userService.updateRole(editing.value.id, form.value.role_id);
      }
      ui.addToast('success', 'User updated');
    } else {
      await userService.create({
        username: form.value.username,
        email: form.value.email,
        full_name: form.value.full_name,
        password: form.value.password,
        role_id: form.value.role_id,
        store_id: form.value.store_id || undefined,
      });
      ui.addToast('success', 'User created');
    }
    dialogOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    saving.value = false;
  }
}

async function toggleActive(u: User) {
  try {
    await userService.update(u.id, { is_active: !u.is_active });
    ui.addToast('success', u.is_active ? 'User deactivated' : 'User activated');
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function removeUser(u: User) {
  try {
    await userService.delete(u.id);
    ui.addToast('success', 'User deleted');
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

watch(isReady, async (r) => {
  if (r) {
    await loadRoles();
    await load();
  }
}, { immediate: true });
</script>

<template>
  <DashboardShell>
  <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
  </div>

  <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">Access restricted</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">
      You don't have permission to view users. Contact an administrator if you believe this is a mistake.
    </p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">Users</h2>
        <p class="text-muted-foreground mt-1">Manage user accounts, roles, and access.</p>
      </div>
      <Button @click="openCreate">+ New User</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>Search</Label>
          <Input v-model="search" placeholder="Search by name or username" class="w-64" @keyup.enter="applyFilters" />
        </div>
        <div class="space-y-1">
          <Label>Role</Label>
          <Select v-model="roleFilter">
            <SelectTrigger class="w-48"><SelectValue placeholder="All roles" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All roles</SelectItem>
              <SelectItem v-for="r in roles" :key="r.id" :value="r.id">{{ r.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" size="icon" title="Search" @click="applyFilters">🔍</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>User</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Status</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="5" class="text-center py-8 text-muted-foreground">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="users.length === 0">
                <TableCell colspan="5" class="text-center py-8 text-muted-foreground">No users found</TableCell>
              </TableRow>
              <TableRow v-for="u in users" :key="u.id">
                <TableCell>
                  <div class="font-medium">{{ u.full_name || u.username }}</div>
                  <div class="text-xs text-muted-foreground">@{{ u.username }}</div>
                </TableCell>
                <TableCell>{{ u.email }}</TableCell>
                <TableCell>
                  <Badge variant="outline">{{ u.role_name || u.role_id || '—' }}</Badge>
                </TableCell>
                <TableCell>
                  <div class="flex items-center gap-2">
                    <Switch :model-value="u.is_active" size="sm" @update:model-value="toggleActive(u)" />
                    <span class="text-xs text-muted-foreground">{{ u.is_active ? 'Active' : 'Inactive' }}</span>
                  </div>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end gap-2">
                    <Button variant="outline" size="sm" @click="openEdit(u)">Edit</Button>
                    <AlertDialog>
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">Delete</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete user?</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will permanently delete {{ u.username }}. This action cannot be undone.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction @click="removeUser(u)">Delete</AlertDialogAction>
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
          <PaginationBar :page="page" :per-page="perPage" :total="filteredTotal" @update:page="page = $event; load()" />
        </div>
      </CardContent>
    </Card>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit User' : 'Create User' }}</DialogTitle>
          <DialogDescription>
            {{ editing ? 'Update the user account details.' : 'Create a new user account.' }}
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>Full Name</Label>
            <Input v-model="form.full_name" placeholder="Full name" />
          </div>
          <div class="space-y-2">
            <Label>Username</Label>
            <Input v-model="form.username" placeholder="Username" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>Email</Label>
            <Input v-model="form.email" type="email" placeholder="Email" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>Password</Label>
            <Input v-model="form.password" type="password" placeholder="Password" autocomplete="new-password" />
          </div>
          <div class="space-y-2">
            <Label>Role</Label>
            <Select v-model="form.role_id">
              <SelectTrigger><SelectValue placeholder="Select role" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="r in roles" :key="r.id" :value="r.id">{{ r.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="dialogOpen = false">Cancel</Button>
          <Button :disabled="saving" @click="save">{{ saving ? 'Saving...' : 'Save' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
