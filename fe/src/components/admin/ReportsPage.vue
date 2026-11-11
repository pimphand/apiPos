<script setup lang="ts">
import { ref, watch } from 'vue';
import { reportService } from '@/services/report';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency } from '@/lib/utils';
import type {
  DailySalesReport,
  CashierSalesReport,
  ProductSalesReport,
  CategorySalesReport,
  PaymentMethodReport,
  InventoryReport,
  LowStockReport,
} from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

const { isReady, allowed } = usePageGuard('reports.read');
const ui = useUiStore();
const t = useLocaleStore().t;

const activeTab = ref('daily-sales');
const fromDate = ref('');
const toDate = ref('');
const loading = ref(false);

const dailySales = ref<DailySalesReport[]>([]);
const cashierReports = ref<CashierSalesReport[]>([]);
const productReports = ref<ProductSalesReport[]>([]);
const categoryReports = ref<CategorySalesReport[]>([]);
const paymentReports = ref<PaymentMethodReport[]>([]);
const inventoryReports = ref<InventoryReport[]>([]);
const lowStockReports = ref<LowStockReport[]>([]);

const summaryTotals = ref({
  total_sales: 0,
  total_amount: 0,
  total_products: 0,
  total_stock: 0,
  total_value: 0,
});

const tabs = [
  { key: 'daily-sales', labelKey: 'admin.reports.tabDailySales' },
  { key: 'by-cashier', labelKey: 'admin.reports.tabByCashier' },
  { key: 'by-product', labelKey: 'admin.reports.tabByProduct' },
  { key: 'by-category', labelKey: 'admin.reports.tabByCategory' },
  { key: 'payment-summary', labelKey: 'admin.reports.tabPaymentSummary' },
  { key: 'inventory', labelKey: 'admin.reports.tabInventory' },
];

function dateParams() {
  return {
    start_date: fromDate.value || undefined,
    end_date: toDate.value || undefined,
  };
}

function clearData() {
  dailySales.value = [];
  cashierReports.value = [];
  productReports.value = [];
  categoryReports.value = [];
  paymentReports.value = [];
  inventoryReports.value = [];
  lowStockReports.value = [];
  summaryTotals.value = { total_sales: 0, total_amount: 0, total_products: 0, total_stock: 0, total_value: 0 };
}

async function loadTab() {
  loading.value = true;
  clearData();
  try {
    switch (activeTab.value) {
      case 'daily-sales': await loadDailySales(); break;
      case 'by-cashier': await loadByCashier(); break;
      case 'by-product': await loadByProduct(); break;
      case 'by-category': await loadByCategory(); break;
      case 'payment-summary': await loadPaymentSummary(); break;
      case 'inventory': await loadInventory(); break;
    }
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

async function loadDailySales() {
  const res = await reportService.getDailySales(dateParams());
  dailySales.value = res.reports ?? [];
  summaryTotals.value.total_sales = res.total_sales;
  summaryTotals.value.total_amount = res.total_amount;
}

async function loadByCashier() {
  const res = await reportService.getSalesByCashier(dateParams());
  cashierReports.value = res.reports ?? [];
}

async function loadByProduct() {
  const res = await reportService.getSalesByProduct({ ...dateParams(), limit: 50 });
  productReports.value = res.reports ?? [];
}

async function loadByCategory() {
  const res = await reportService.getSalesByCategory(dateParams());
  categoryReports.value = res.reports ?? [];
}

async function loadPaymentSummary() {
  const res = await reportService.getPaymentSummary(dateParams());
  paymentReports.value = res.reports ?? [];
  summaryTotals.value.total_amount = res.total_amount;
}

async function loadInventory() {
  const [invRes, lowRes] = await Promise.all([
    reportService.getInventorySummary(),
    reportService.getLowStock({ limit: 20 }),
  ]);
  inventoryReports.value = invRes.reports ?? [];
  lowStockReports.value = lowRes.products ?? [];
  summaryTotals.value.total_products = invRes.total_products;
  summaryTotals.value.total_stock = invRes.total_stock;
  summaryTotals.value.total_value = invRes.total_value;
}

function applyFilter() {
  loadTab();
}

function switchTab(key: string) {
  activeTab.value = key;
  loadTab();
}

watch(isReady, (r) => { if (r) loadTab(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
    <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
      <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.reports.noPermissionAction') }) }}</p>
    </div>

    <div v-else class="flex flex-col gap-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="text-xl font-bold sm:text-2xl">{{ t('admin.reports.title') }}</h2>
          <p class="text-muted-foreground mt-1 text-sm sm:text-base">{{ t('admin.reports.description') }}</p>
        </div>
        <a
          href="/reports/accounting"
          class="inline-flex h-9 shrink-0 items-center rounded-md border bg-background px-4 text-sm font-medium hover:bg-muted transition-colors"
        >
          {{ t('admin.accounting.title') }} →
        </a>
      </div>

      <Card>
        <CardContent class="grid grid-cols-1 gap-3 pt-4 sm:grid-cols-2 md:grid-cols-[auto_auto_1fr] md:items-end">
          <div class="space-y-1">
            <Label>{{ t('admin.reports.fromDate') }}</Label>
            <Input v-model="fromDate" type="date" class="w-full sm:w-40" />
          </div>
          <div class="space-y-1">
            <Label>{{ t('admin.reports.toDate') }}</Label>
            <Input v-model="toDate" type="date" class="w-full sm:w-40" />
          </div>
          <div class="flex sm:justify-start md:justify-end">
            <Button variant="outline" class="w-full sm:w-auto" @click="applyFilter">{{ t('admin.reports.apply') }}</Button>
          </div>
        </CardContent>
      </Card>

      <div class="flex gap-2 overflow-x-auto pb-1 -mx-1 px-1 snap-x snap-mandatory sm:flex-wrap sm:snap-none sm:pb-0">
        <Button
          v-for="tab in tabs"
          :key="tab.key"
          :variant="activeTab === tab.key ? 'default' : 'outline'"
          size="sm"
          class="shrink-0 snap-start"
          @click="switchTab(tab.key)"
        >
          {{ t(tab.labelKey) }}
        </Button>
      </div>

      <!-- Daily Sales -->
      <Card v-if="activeTab === 'daily-sales'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.reports.tabDailySales') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[480px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.reports.date') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.totalSales') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.totalAmount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="dailySales.length === 0">
                  <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="r in dailySales" :key="r.date">
                  <TableCell class="whitespace-nowrap">{{ r.date }}</TableCell>
                  <TableCell class="text-right">{{ r.total_sales }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(r.total_amount) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
          <div v-if="dailySales.length > 0" class="flex flex-wrap justify-end gap-x-4 gap-y-2 p-4 border-t text-sm font-semibold">
            <span>{{ t('admin.reports.totalSales') }}: {{ summaryTotals.total_sales }}</span>
            <span>{{ t('admin.reports.totalAmount') }}: {{ formatCurrency(summaryTotals.total_amount) }}</span>
          </div>
        </CardContent>
      </Card>

      <!-- By Cashier -->
      <Card v-if="activeTab === 'by-cashier'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.reports.tabByCashier') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[480px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.reports.cashier') }}</TableHead>
                <TableHead class="hidden sm:table-cell text-right">{{ t('admin.reports.transactions') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.totalAmount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="cashierReports.length === 0">
                  <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="r in cashierReports" :key="r.cashier_id">
                  <TableCell>{{ r.cashier_name }}</TableCell>
                  <TableCell class="hidden sm:table-cell text-right">{{ r.total_transactions }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(r.total_amount) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>

      <!-- By Product -->
      <Card v-if="activeTab === 'by-product'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.reports.tabByProduct') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[480px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.reports.product') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.quantitySold') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.totalAmount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="productReports.length === 0">
                  <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="r in productReports" :key="r.product_id">
                  <TableCell>{{ r.product_name }}</TableCell>
                  <TableCell class="text-right">{{ r.quantity_sold }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(r.total_amount) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>

      <!-- By Category -->
      <Card v-if="activeTab === 'by-category'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.reports.tabByCategory') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[560px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.reports.category') }}</TableHead>
                <TableHead class="hidden sm:table-cell text-right">{{ t('admin.reports.products') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.quantitySold') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.totalAmount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="categoryReports.length === 0">
                  <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="r in categoryReports" :key="r.category_id">
                  <TableCell>{{ r.category_name }}</TableCell>
                  <TableCell class="hidden sm:table-cell text-right">{{ r.product_count }}</TableCell>
                  <TableCell class="text-right">{{ r.quantity_sold }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(r.total_amount) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>

      <!-- Payment Summary -->
      <Card v-if="activeTab === 'payment-summary'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.reports.tabPaymentSummary') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[480px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.reports.paymentMethod') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.count') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.reports.totalAmount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="paymentReports.length === 0">
                  <TableCell :colspan="3" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="r in paymentReports" :key="r.payment_method">
                  <TableCell>{{ r.payment_method }}</TableCell>
                  <TableCell class="text-right">{{ r.count }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(r.total_amount) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
          <div v-if="paymentReports.length > 0" class="flex flex-wrap justify-end p-4 border-t text-sm font-semibold">
            <span>{{ t('admin.reports.totalAmount') }}: {{ formatCurrency(summaryTotals.total_amount) }}</span>
          </div>
        </CardContent>
      </Card>

      <!-- Inventory -->
      <div v-if="activeTab === 'inventory'" class="flex flex-col gap-4">
        <Card>
          <CardHeader>
            <CardTitle class="text-base">{{ t('admin.reports.tabInventory') }}</CardTitle>
          </CardHeader>
          <CardContent class="p-0">
            <div class="overflow-x-auto">
            <Table class="min-w-[560px]">
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('admin.reports.category') }}</TableHead>
                  <TableHead class="hidden sm:table-cell text-right">{{ t('admin.reports.products') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.reports.totalStock') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.reports.totalValue') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="loading">
                  <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
                </TableRow>
                <template v-else>
                  <TableRow v-if="inventoryReports.length === 0">
                    <TableCell :colspan="4" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                  </TableRow>
                  <TableRow v-for="r in inventoryReports" :key="r.category_name">
                    <TableCell>{{ r.category_name }}</TableCell>
                    <TableCell class="hidden sm:table-cell text-right">{{ r.product_count }}</TableCell>
                    <TableCell class="text-right">{{ r.total_stock }}</TableCell>
                    <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(r.total_value) }}</TableCell>
                  </TableRow>
                </template>
              </TableBody>
            </Table>
            </div>
            <div v-if="inventoryReports.length > 0" class="flex flex-wrap justify-end gap-x-4 gap-y-2 p-4 border-t text-sm font-semibold">
              <span>{{ t('admin.reports.products') }}: {{ summaryTotals.total_products }}</span>
              <span>{{ t('admin.reports.totalStock') }}: {{ summaryTotals.total_stock }}</span>
              <span>{{ t('admin.reports.totalValue') }}: {{ formatCurrency(summaryTotals.total_value) }}</span>
            </div>
          </CardContent>
        </Card>

        <Card v-if="lowStockReports.length > 0">
          <CardHeader>
            <CardTitle class="text-base">{{ t('admin.reports.lowStock') }}</CardTitle>
          </CardHeader>
          <CardContent class="p-0">
            <div class="overflow-x-auto">
            <Table class="min-w-[640px]">
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('admin.reports.product') }}</TableHead>
                  <TableHead class="hidden sm:table-cell">{{ t('admin.reports.category') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.reports.totalStock') }}</TableHead>
                  <TableHead class="hidden md:table-cell text-right">{{ t('admin.reports.lowStock') }}</TableHead>
                  <TableHead class="text-right">{{ t('admin.reports.deficit') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="r in lowStockReports" :key="r.product_id">
                  <TableCell>{{ r.product_name }}</TableCell>
                  <TableCell class="hidden sm:table-cell text-muted-foreground">{{ r.category_name }}</TableCell>
                  <TableCell class="text-right">{{ r.current_stock }}</TableCell>
                  <TableCell class="hidden md:table-cell text-right">{{ r.minimum_stock }}</TableCell>
                  <TableCell class="text-right font-medium text-destructive">{{ r.deficit }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  </DashboardShell>
</template>
