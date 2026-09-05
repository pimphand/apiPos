<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { activityLogService } from '@/services/activitylog';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatDateTime } from '@/lib/utils';
import type { ActivityLog } from '@/types';
import Button from '@/components/ui/button/Button.vue';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Badge from '@/components/ui/badge/Badge.vue';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import PaginationBar from '@/components/ui/PaginationBar.vue';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent } from '@/components/ui/card';

const { isReady, allowed, authStore } = usePageGuard('activity_logs.read');
// The store column is only meaningful for system-wide (superadmin) views;
// tenant users only ever see their own store's logs, so the column is hidden.
const showStoreColumn = computed(() => (authStore.currentUser as any)?.role_name === 'superadmin');
const ui = useUiStore();
const t = useLocaleStore().t;

const ACTIONS = ['LOGIN', 'LOGOUT', 'CREATE', 'UPDATE', 'DELETE', 'VIEW'];

const logs = ref<ActivityLog[]>([]);
const loading = ref(false);
const page = ref(1);
const perPage = 20;
const total = ref(0);

const filters = ref({
  username: '',
  action: '',
  resource: '',
  start_date: '',
  end_date: '',
});

const detail = ref<ActivityLog | null>(null);
const detailOpen = ref(false);

async function load() {
  loading.value = true;
  try {
    const res = await activityLogService.list({
      user_id: undefined,
      action: filters.value.action !== 'all' ? filters.value.action || undefined : undefined,
      resource: filters.value.resource || undefined,
      start_date: filters.value.start_date || undefined,
      end_date: filters.value.end_date || undefined,
      page: page.value,
      per_page: perPage,
    });
    logs.value = res.activity_logs;
    total.value = res.total_count;
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

function applyFilters() {
  page.value = 1;
  load();
}

function resetFilters() {
  filters.value = { username: '', action: '', resource: '', start_date: '', end_date: '' };
  applyFilters();
}

function openDetail(log: ActivityLog) {
  detail.value = log;
  detailOpen.value = true;
}

function actionVariant(action: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (action) {
    case 'DELETE': return 'destructive';
    case 'CREATE': return 'default';
    case 'UPDATE': return 'secondary';
    default: return 'outline';
  }
}

function parseMetadata(raw: string): Record<string, unknown> {
  if (!raw) return {};
  try {
    return JSON.parse(raw) as Record<string, unknown>;
  } catch {
    return {};
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
    <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.activityLogs.noPermissionAction') }) }}</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div>
      <h2 class="text-2xl font-bold">{{ t('admin.activityLogs.title') }}</h2>
      <p class="text-muted-foreground mt-1">{{ t('admin.activityLogs.description') }}</p>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>{{ t('admin.activityLogs.action') }}</Label>
          <Select v-model="filters.action">
            <SelectTrigger class="w-40"><SelectValue :placeholder="t('admin.activityLogs.allActions')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t('admin.activityLogs.allActions') }}</SelectItem>
              <SelectItem v-for="a in ACTIONS" :key="a" :value="a">{{ a }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.activityLogs.resource') }}</Label>
          <Input v-model="filters.resource" :placeholder="t('admin.activityLogs.resourcePlaceholder')" class="w-44" @keyup.enter="applyFilters" />
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.activityLogs.startDate') }}</Label>
          <Input v-model="filters.start_date" type="date" class="w-40" />
        </div>
        <div class="space-y-1">
          <Label>{{ t('admin.activityLogs.endDate') }}</Label>
          <Input v-model="filters.end_date" type="date" class="w-40" />
        </div>
        <Button @click="applyFilters">{{ t('admin.shared.filter') }}</Button>
        <Button variant="ghost" @click="resetFilters">{{ t('admin.shared.reset') }}</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('admin.activityLogs.user') }}</TableHead>
              <TableHead v-if="showStoreColumn">{{ t('admin.activityLogs.store') }}</TableHead>
              <TableHead>{{ t('admin.activityLogs.action') }}</TableHead>
              <TableHead>{{ t('admin.activityLogs.resource') }}</TableHead>
              <TableHead>{{ t('admin.shared.description') }}</TableHead>
              <TableHead>{{ t('admin.activityLogs.when') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="showStoreColumn ? 6 : 5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="logs.length === 0">
                <TableCell :colspan="showStoreColumn ? 6 : 5" class="text-center py-8 text-muted-foreground">{{ t('admin.activityLogs.noData') }}</TableCell>
              </TableRow>
              <TableRow v-for="log in logs" :key="log.id" class="cursor-pointer hover:bg-muted/50" @click="openDetail(log)">
                <TableCell>
                  <div class="font-medium">{{ log.username || '—' }}</div>
                </TableCell>
                <TableCell v-if="showStoreColumn">{{ log.store_name || '—' }}</TableCell>
                <TableCell><Badge :variant="actionVariant(log.action)">{{ log.action }}</Badge></TableCell>
                <TableCell><span class="font-mono text-xs">{{ log.resource }}</span></TableCell>
                <TableCell class="max-w-[280px] truncate text-muted-foreground">{{ log.description }}</TableCell>
                <TableCell class="text-muted-foreground text-sm whitespace-nowrap">{{ formatDateTime(log.created_at) }}</TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
        <div class="flex justify-center py-4">
          <PaginationBar :page="page" :per-page="perPage" :total="total" @update:page="page = $event; load()" />
        </div>
      </CardContent>
    </Card>

    <Dialog v-model:open="detailOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ t('admin.activityLogs.detailTitle') }}</DialogTitle>
          <DialogDescription>{{ t('admin.activityLogs.detailDescription') }}</DialogDescription>
        </DialogHeader>
        <div v-if="detail" class="grid gap-3 text-sm">
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.activityLogs.user') }}</span><span class="col-span-2">{{ detail.username || '—' }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.activityLogs.action') }}</span><span class="col-span-2"><Badge :variant="actionVariant(detail.action)">{{ detail.action }}</Badge></span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.activityLogs.resource') }}</span><span class="col-span-2 font-mono text-xs">{{ detail.resource }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.activityLogs.resourceId') }}</span><span class="col-span-2 font-mono text-xs break-all">{{ detail.resource_id || '—' }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.shared.description') }}</span><span class="col-span-2">{{ detail.description }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.activityLogs.ip') }}</span><span class="col-span-2">{{ detail.ip_address || '—' }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">{{ t('admin.activityLogs.timestamp') }}</span><span class="col-span-2">{{ formatDateTime(detail.created_at) }}</span></div>
          <div v-if="detail.metadata" class="grid grid-cols-3 gap-2">
            <span class="text-muted-foreground">{{ t('admin.activityLogs.metadata') }}</span>
            <pre class="col-span-2 overflow-auto rounded-md bg-muted p-2 text-xs">{{ parseMetadata(detail.metadata) }}</pre>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
