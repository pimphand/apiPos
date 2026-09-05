<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { roleService } from '@/services/role';
import { permissionService } from '@/services/permission';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { can } from '@/lib/rbac';
import type { Role, Permission } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Textarea from '@/components/ui/textarea/Textarea.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import Checkbox from '@/components/ui/checkbox/Checkbox.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed, authStore } = usePageGuard('roles.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const DEFAULT_ROLES = new Set(['superadmin', 'admin', 'manager', 'cashier']);

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
      ui.addToast('success', t('admin.roles.toastUpdated'));
    } else {
      await roleService.create(form.value);
      ui.addToast('success', t('admin.roles.toastCreated'));
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
    ui.addToast('success', t('admin.roles.toastDeleted'));
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
    ui.addToast('success', t('admin.roles.toastPerms'));
    permOpen.value = false;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    permSaving.value = false;
  }
}

const canCreate = computed(() => can(authStore.currentUser, 'roles.create'));
const canUpdate = computed(() => can(authStore.currentUser, 'roles.update'));
const canDelete = computed(() => can(authStore.currentUser, 'roles.delete'));
const canAssign = computed(() => can(authStore.currentUser, 'permissions.assign'));
// Hide the Actions column entirely when the user has no mutating permissions.
const showActions = computed(() => canCreate.value || canUpdate.value || canDelete.value || canAssign.value);

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

 <template>
  <DashboardShell>
  <div v-if="!isReady" class="flex items-center justify-center h-[60vh]">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
  </div>

  <div v-else-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.roles.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.roles.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.roles.description') }}</p>
      </div>
      <Button v-if="canCreate" @click="openCreate">+ {{ t('admin.roles.newRole') }}</Button>
    </div>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('admin.shared.name') }}</TableHead>
              <TableHead>{{ t('admin.shared.description') }}</TableHead>
              <TableHead>{{ t('admin.roles.permissions') }}</TableHead>
              <TableHead v-if="showActions" class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="showActions ? 4 : 3" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="roles.length === 0">
                <TableCell :colspan="showActions ? 4 : 3" class="text-center py-8 text-muted-foreground">{{ t('admin.roles.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="r in roles" :key="r.id">
                <TableCell>
                  <div class="flex items-center gap-2">
                    <span class="font-medium">{{ r.name }}</span>
                    <Badge v-if="DEFAULT_ROLES.has(r.name)" variant="secondary">{{ t('admin.roles.default') }}</Badge>
                  </div>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ r.description || '—' }}</TableCell>
                <TableCell>
                  <Button v-if="canAssign" variant="outline" size="sm" @click="openPermissions(r)">{{ t('admin.roles.assignPermissions') }}</Button>
                </TableCell>
                <TableCell v-if="showActions" class="text-right">
                  <div class="flex justify-end gap-2">
                    <Button v-if="canUpdate" variant="outline" size="sm" @click="openEdit(r)">{{ t('admin.shared.edit') }}</Button>
                    <AlertDialog v-if="canDelete && !DEFAULT_ROLES.has(r.name)">
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">{{ t('admin.shared.delete') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.roles.deleteTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('admin.roles.deleteDescription', { name: r.name }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="removeRole(r)">{{ t('admin.shared.delete') }}</AlertDialogAction>
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

    <!-- Role create/edit dialog -->
    <Dialog v-model:open="formOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('admin.roles.editTitle') : t('admin.roles.createTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.roles.dialogDescription') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>{{ t('admin.shared.name') }}</Label>
            <Input v-model="form.name" :placeholder="t('admin.roles.namePlaceholder')" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>{{ t('admin.shared.description') }}</Label>
            <Textarea v-model="form.description" :placeholder="t('admin.roles.descriptionPlaceholder')" :rows="3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="saving" @click="save">{{ saving ? t('admin.shared.saving') : t('admin.shared.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Permissions dialog -->
    <Dialog v-model:open="permOpen">
      <DialogContent class="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('admin.roles.permTitle', { name: permRole?.name ?? '' }) }}</DialogTitle>
          <DialogDescription>{{ t('admin.roles.permDescription') }}</DialogDescription>
        </DialogHeader>
        <div v-if="permLoading" class="py-8 text-center text-muted-foreground">{{ t('admin.roles.loadingPermissions') }}</div>
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
          <p v-if="allPermissions.length === 0" class="text-muted-foreground">{{ t('admin.roles.noPermissions') }}</p>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="permOpen = false">{{ t('admin.shared.cancel') }}</Button>
          <Button :disabled="permSaving" @click="savePermissions">{{ permSaving ? t('admin.shared.saving') : t('admin.shared.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
