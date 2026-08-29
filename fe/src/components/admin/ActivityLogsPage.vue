<script setup lang="ts">
import { ref, watch } from 'vue';
import { activityLogService } from '@/services/activitylog';
import { useUiStore } from '@/stores/ui';
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

const { isReady, allowed } = usePageGuard('activity_logs.read');
const ui = useUiStore();

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
    <h3 class="text-xl font-semibold">Access restricted</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">You don't have permission to view activity logs.</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div>
      <h2 class="text-2xl font-bold">Activity Logs</h2>
      <p class="text-muted-foreground mt-1">Audit trail of all user actions across the system.</p>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>Action</Label>
          <Select v-model="filters.action">
            <SelectTrigger class="w-40"><SelectValue placeholder="All actions" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All actions</SelectItem>
              <SelectItem v-for="a in ACTIONS" :key="a" :value="a">{{ a }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1">
          <Label>Resource</Label>
          <Input v-model="filters.resource" placeholder="e.g. products" class="w-44" @keyup.enter="applyFilters" />
        </div>
        <div class="space-y-1">
          <Label>Start date</Label>
          <Input v-model="filters.start_date" type="date" class="w-40" />
        </div>
        <div class="space-y-1">
          <Label>End date</Label>
          <Input v-model="filters.end_date" type="date" class="w-40" />
        </div>
        <Button @click="applyFilters">Filter</Button>
        <Button variant="ghost" @click="resetFilters">Reset</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>User</TableHead>
              <TableHead>Action</TableHead>
              <TableHead>Resource</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>When</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="5" class="text-center py-8 text-muted-foreground">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="logs.length === 0">
                <TableCell colspan="5" class="text-center py-8 text-muted-foreground">No activity logs found</TableCell>
              </TableRow>
              <TableRow v-for="log in logs" :key="log.id" class="cursor-pointer hover:bg-muted/50" @click="openDetail(log)">
                <TableCell>
                  <div class="font-medium">{{ log.username || '—' }}</div>
                </TableCell>
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
          <DialogTitle>Activity Log Detail</DialogTitle>
          <DialogDescription>Full record for this activity.</DialogDescription>
        </DialogHeader>
        <div v-if="detail" class="grid gap-3 text-sm">
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">User</span><span class="col-span-2">{{ detail.username || '—' }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">Action</span><span class="col-span-2"><Badge :variant="actionVariant(detail.action)">{{ detail.action }}</Badge></span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">Resource</span><span class="col-span-2 font-mono text-xs">{{ detail.resource }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">Resource ID</span><span class="col-span-2 font-mono text-xs break-all">{{ detail.resource_id || '—' }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">Description</span><span class="col-span-2">{{ detail.description }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">IP</span><span class="col-span-2">{{ detail.ip_address || '—' }}</span></div>
          <div class="grid grid-cols-3 gap-2"><span class="text-muted-foreground">Timestamp</span><span class="col-span-2">{{ formatDateTime(detail.created_at) }}</span></div>
          <div v-if="detail.metadata" class="grid grid-cols-3 gap-2">
            <span class="text-muted-foreground">Metadata</span>
            <pre class="col-span-2 overflow-auto rounded-md bg-muted p-2 text-xs">{{ parseMetadata(detail.metadata) }}</pre>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
