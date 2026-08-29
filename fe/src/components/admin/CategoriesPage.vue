<script setup lang="ts">
import { ref, watch } from 'vue';
import { categoryService } from '@/services/category';
import { useUiStore } from '@/stores/ui';
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
      ui.addToast('success', 'Category updated');
    } else {
      await categoryService.create(form.value);
      ui.addToast('success', 'Category created');
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
      ui.addToast('success', 'Category deactivated');
    } else {
      await categoryService.update(c.id, { is_active: true });
      ui.addToast('success', 'Category activated');
    }
    load();
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  }
}

async function removeCategory(c: Category) {
  try {
    await categoryService.delete(c.id);
    ui.addToast('success', 'Category deleted');
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
    <h3 class="text-xl font-semibold">Access restricted</h3>
    <p class="text-muted-foreground mt-2 max-w-sm">You don't have permission to manage categories.</p>
  </div>

  <div v-else class="flex flex-col gap-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">Categories</h2>
        <p class="text-muted-foreground mt-1">Organize products into categories for easier management and reporting.</p>
      </div>
      <Button @click="openCreate">+ New Category</Button>
    </div>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3 pt-4">
        <div class="space-y-1">
          <Label>Status</Label>
          <Select v-model="activeFilter">
            <SelectTrigger class="w-44"><SelectValue placeholder="All" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="true">Active</SelectItem>
              <SelectItem value="false">Inactive</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" @click="applyFilter">Filter</Button>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Status</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableRow v-if="categories.length === 0">
                <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">No categories found</TableCell>
              </TableRow>
              <TableRow v-for="c in categories" :key="c.id">
                <TableCell class="font-medium">{{ c.name }}</TableCell>
                <TableCell class="text-muted-foreground">{{ c.description || '—' }}</TableCell>
                <TableCell>
                  <Badge :variant="c.is_active ? 'default' : 'secondary'">{{ c.is_active ? 'Active' : 'Inactive' }}</Badge>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end items-center gap-3">
                    <div class="flex items-center gap-2 text-xs text-muted-foreground">
                      <Switch :model-value="c.is_active" @update:model-value="toggleActive(c)" />
                      {{ c.is_active ? 'Active' : 'Inactive' }}
                    </div>
                    <Button variant="outline" size="sm" @click="openEdit(c)">Edit</Button>
                    <AlertDialog>
                      <AlertDialogTrigger as-child>
                        <Button variant="destructive" size="sm">Delete</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete category?</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will permanently delete "{{ c.name }}". Products in this category may be affected.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction @click="removeCategory(c)">Delete</AlertDialogAction>
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
          <DialogTitle>{{ editing ? 'Edit Category' : 'Create Category' }}</DialogTitle>
          <DialogDescription>Enter the category name and an optional description.</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4">
          <div class="space-y-2">
            <Label>Name</Label>
            <Input v-model="form.name" placeholder="e.g. Beverages" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>Description</Label>
            <Textarea v-model="form.description" placeholder="Category description" :rows="3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="formOpen = false">Cancel</Button>
          <Button :disabled="saving" @click="save">{{ saving ? 'Saving...' : 'Save' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
  </DashboardShell>
</template>
