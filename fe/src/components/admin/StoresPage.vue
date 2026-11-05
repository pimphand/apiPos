<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { storeService } from '@/services/store';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatDateTime } from '@/lib/utils';
import type { Store } from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

// Store management is superadmin-only (enforced in the service layer).
// The auth-only guard handles session restore; `allowed` gates the page UI.
const { isReady, authStore } = usePageGuard();
const ui = useUiStore();
const t = useLocaleStore().t;

const stores = ref<Store[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);
const statusFilter = ref('');
const verifiedFilter = ref('');

const detailOpen = ref(false);
const selected = ref<Store | null>(null);
const actionBusy = ref(false);

const isSuperadmin = computed(() => authStore.currentUser?.role_name === 'superadmin');
const allowed = computed(() => isSuperadmin.value);
const canVerify = computed(() => isSuperadmin.value);

async function load() {
  loading.value = true;
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      per_page: perPage,
    };
    if (['true', 'false'].includes(verifiedFilter.value)) {
      params.is_verified = verifiedFilter.value;
    }
    if (['true', 'false'].includes(statusFilter.value)) {
      params.is_active = statusFilter.value;
    }
    const res = await storeService.list(params);
    stores.value = res.stores ?? [];
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

function openDetail(s: Store) {
  selected.value = s;
  detailOpen.value = true;
}

async function verifyStore() {
  if (!selected.value) return;
  actionBusy.value = true;
  try {
    await storeService.verify(selected.value.id);
    ui.addToast('success', t('admin.stores.toastVerified'));
    detailOpen.value = false;
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    actionBusy.value = false;
  }
}

watch(isReady, (r) => { if (r) load(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
  <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
    <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.stores.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ t('admin.stores.title') }}</h2>
        <p class="text-muted-foreground mt-1">{{ t('admin.stores.description') }}</p>
      </div>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>{{ t('admin.stores.verified') }}</Label>
          <Select v-model="verifiedFilter">
            <SelectTrigger class="w-44"><SelectValue :placeholder="t('admin.shared.all')"/></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.shared.all') }}</SelectItem>
              <SelectItem value="true">{{ t('admin.stores.verifiedLabel') }}</SelectItem>
              <SelectItem value="false">{{ t('admin.stores.pendingLabel') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.shared.status') }}</Label>
          <Select v-model="statusFilter">
            <SelectTrigger class="w-44"><SelectValue :placeholder="t('admin.shared.all')"/></SelectTrigger>
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
              <TableHead>{{ t('admin.stores.name') }}</TableHead>
              <TableHead>{{ t('admin.stores.address') }}</TableHead>
              <TableHead>{{ t('admin.stores.phone') }}</TableHead>
              <TableHead>{{ t('admin.stores.verified') }}</TableHead>
              <TableHead>{{ t('admin.shared.status') }}</TableHead>
              <TableHead>{{ t('admin.stores.createdAt') }}</TableHead>
              <TableHead class="text-right">{{ t('admin.shared.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="7" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="stores.length === 0">
                <TableCell :colspan="7" class="text-center py-8 text-muted-foreground">{{ t('admin.stores.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="s in stores" :key="s.id">
                <TableCell class="font-medium">{{ s.name }}</TableCell>
                <TableCell class="text-muted-foreground">{{ s.address || '—' }}</TableCell>
                <TableCell class="text-muted-foreground">{{ s.phone || '—' }}</TableCell>
                <TableCell>
                  <Badge :variant="s.is_verified === 'true' ? 'default' : 'secondary'">
                    {{ s.is_verified === 'true' ? t('admin.stores.verifiedLabel') : t('admin.stores.pendingLabel') }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge :variant="s.is_active === 'true' ? 'default' : 'secondary'">
                    {{ s.is_active === 'true' ? t('admin.shared.active') : t('admin.shared.inactive') }}
                  </Badge>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ formatDateTime(s.created_at) }}</TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end gap-2">
                    <Button variant="outline" size="sm" @click="openDetail(s)">{{ t('admin.stores.viewDetail') }}</Button>
                    <AlertDialog v-if="canVerify && s.is_verified !== 'true'">
                      <AlertDialogTrigger as-child>
                        <Button size="sm">{{ t('admin.stores.verify') }}</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('admin.stores.verifyTitle') }}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {{ t('admin.stores.verifyDescription', { name: s.name }) }}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('admin.shared.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="verifyStore">{{ t('admin.stores.verify') }}</AlertDialogAction>
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

    <!-- Detail dialog -->
    <Dialog v-model:open="detailOpen">
      <DialogContent class="sm:max-w-lg max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('admin.stores.detailTitle') }}</DialogTitle>
          <DialogDescription v-if="selected">{{ selected.name }}</DialogDescription>
        </DialogHeader>
        <div v-if="selected" class="grid gap-3 text-sm">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <div class="text-muted-foreground">{{ t('admin.stores.name') }}</div>
              <div class="font-medium">{{ selected.name }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.stores.phone') }}</div>
              <div class="font-medium">{{ selected.phone || '—' }}</div>
            </div>
          </div>
          <div>
            <div class="text-muted-foreground">{{ t('admin.stores.address') }}</div>
            <div class="font-medium">{{ selected.address || '—' }}</div>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <div class="text-muted-foreground">{{ t('admin.stores.verified') }}</div>
              <Badge :variant="selected.is_verified === 'true' ? 'default' : 'secondary'">
                {{ selected.is_verified === 'true' ? t('admin.stores.verifiedLabel') : t('admin.stores.pendingLabel') }}
              </Badge>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.shared.status') }}</div>
              <Badge :variant="selected.is_active === 'true' ? 'default' : 'secondary'">
                {{ selected.is_active === 'true' ? t('admin.shared.active') : t('admin.shared.inactive') }}
              </Badge>
            </div>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <div class="text-muted-foreground">{{ t('admin.stores.createdAt') }}</div>
              <div class="font-medium">{{ formatDateTime(selected.created_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground">{{ t('admin.stores.updatedAt') }}</div>
              <div class="font-medium">{{ formatDateTime(selected.updated_at) }}</div>
            </div>
          </div>
        </div>
        <DialogFooter v-if="canVerify && selected && selected.is_verified !== 'true'" class="flex justify-end gap-2">
          <Button :disabled="actionBusy" @click="verifyStore">{{ actionBusy ? t('admin.shared.saving') : t('admin.stores.verify') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
