<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { accountingService } from '@/services/accounting';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { usePageGuard } from '@/composables/usePageGuard';
import { apiMessage } from '@/lib/apiError';
import { formatCurrency } from '@/lib/utils';
import { saveExportFile } from '@/lib/platform';
import type {
  AccountingStatementResponse,
  AccountingLedgerResponse,
  AccountingJournalListResponse,
  AccountingAccountListResponse,
  AccountingValidateResponse,
} from '@/types';
import DashboardShell from '@/components/layout/DashboardShell.vue';
import ReportFilterPanel from '@/components/reports/ReportFilterPanel.vue';
import Button from '@/components/ui/button/Button.vue';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Download, FileText, AlertTriangle, CheckCircle2 } from '@lucide/vue';

const { isReady, allowed } = usePageGuard('accounting.read');
const ui = useUiStore();
const t = useLocaleStore().t;

type TabKey = 'trial-balance' | 'income-statement' | 'balance-sheet' | 'changes-in-equity' | 'cash-flow' | 'general-journal' | 'general-ledger' | 'validate';

const activeTab = ref<TabKey>('trial-balance');
const fromDate = ref('');
const toDate = ref('');
const loading = ref(false);
const exporting = ref<string | null>(null);

const stmt = ref<AccountingStatementResponse | null>(null);
const ledger = ref<AccountingLedgerResponse | null>(null);
const journal = ref<AccountingJournalListResponse | null>(null);
const accountsCache = ref<AccountingAccountListResponse | null>(null);
const validateRes = ref<AccountingValidateResponse | null>(null);
const selectedAccount = ref('');

const tabs: { key: TabKey; labelKey: string; reportName: string }[] = [
  { key: 'trial-balance', labelKey: 'admin.accounting.tabTrialBalance', reportName: 'trial-balance' },
  { key: 'income-statement', labelKey: 'admin.accounting.tabIncomeStatement', reportName: 'income-statement' },
  { key: 'balance-sheet', labelKey: 'admin.accounting.tabBalanceSheet', reportName: 'balance-sheet' },
  { key: 'changes-in-equity', labelKey: 'admin.accounting.tabChangesInEquity', reportName: 'changes-in-equity' },
  { key: 'cash-flow', labelKey: 'admin.accounting.tabCashFlow', reportName: 'cash-flow' },
  { key: 'general-journal', labelKey: 'admin.accounting.tabGeneralJournal', reportName: 'general-journal' },
  { key: 'general-ledger', labelKey: 'admin.accounting.tabGeneralLedger', reportName: 'general-ledger' },
  { key: 'validate', labelKey: 'admin.accounting.tabValidate', reportName: 'validate' },
];

const errorWarnings = computed(() =>
  (stmt.value?.warnings ?? []).filter((w) => w.severity === 'ERROR')
);

function dateParams() {
  return { start_date: fromDate.value || undefined, end_date: toDate.value || undefined };
}

function safeLines<T>(arr: T[] | null | undefined): T[] {
  return Array.isArray(arr) ? arr : [];
}

function clearData() {
  stmt.value = null;
  ledger.value = null;
  journal.value = null;
  validateRes.value = null;
}

async function loadTab() {
  loading.value = true;
  clearData();
  try {
    switch (activeTab.value) {
      case 'trial-balance':
        stmt.value = await accountingService.trialBalance(dateParams());
        break;
      case 'income-statement':
        stmt.value = await accountingService.incomeStatement(dateParams());
        break;
      case 'balance-sheet':
        stmt.value = await accountingService.balanceSheet(dateParams());
        break;
      case 'changes-in-equity':
        stmt.value = await accountingService.changesInEquity(dateParams());
        break;
      case 'cash-flow':
        stmt.value = await accountingService.cashFlow(dateParams());
        break;
      case 'general-journal':
        journal.value = await accountingService.generalJournal({ ...dateParams(), page: 1, per_page: 20 });
        break;
      case 'general-ledger':
        await ensureAccounts();
        if (selectedAccount.value) {
          ledger.value = await accountingService.generalLedger({ account_id: selectedAccount.value, ...dateParams(), page: 1, per_page: 20 });
        }
        break;
      case 'validate':
        validateRes.value = await accountingService.validate(dateParams());
        break;
    }
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

async function ensureAccounts() {
  if (accountsCache.value) return;
  accountsCache.value = await accountingService.listAccounts({ page: 1, per_page: 100 });
  if (!selectedAccount.value && accountsCache.value.accounts.length > 0) {
    selectedAccount.value = accountsCache.value.accounts[0].id;
  }
}

async function onLedgerAccountChange() {
  if (!selectedAccount.value) return;
  loading.value = true;
  try {
    ledger.value = await accountingService.generalLedger({ account_id: selectedAccount.value, ...dateParams(), page: 1, per_page: 20 });
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    loading.value = false;
  }
}

function switchTab(key: TabKey) {
  activeTab.value = key;
  loadTab();
}

function applyFilter() {
  loadTab();
}

async function exportReport(format: 'pdf' | 'xlsx') {
  const meta = tabs.find((x) => x.key === activeTab.value);
  if (!meta) return;
  exporting.value = format;
  try {
    const blob = await accountingService.exportStatement(meta.reportName, { format, ...dateParams() });
    const filename = `${meta.reportName}_${new Date().toISOString().slice(0, 10)}.${format}`;
    const saved = await saveExportFile(blob, filename);
    if (saved === 'unsupported') {
      // Browser build (or a native failure): keep the anchor download.
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    }
    if (saved !== 'cancelled') ui.addToast('success', t('admin.accounting.exportSuccess'));
  } catch (err) {
    ui.addToast('error', apiMessage(err));
  } finally {
    exporting.value = null;
  }
}

watch(isReady, (r) => { if (r) loadTab(); }, { immediate: true });
</script>

<template>
  <DashboardShell>
    <div v-if="!allowed" class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border">
      <h3 class="text-xl font-semibold">{{ t('admin.shared.accessRestricted') }}</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">{{ t('admin.shared.noPermission', { action: t('admin.accounting.noPermissionAction') }) }}</p>
    </div>

    <div v-else class="flex flex-col gap-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="text-xl font-bold sm:text-2xl">{{ t('admin.accounting.title') }}</h2>
          <p class="text-muted-foreground mt-1 text-sm sm:text-base">{{ t('admin.accounting.description') }}</p>
        </div>
        <div class="flex shrink-0 gap-2">
          <Button variant="outline" size="sm" :disabled="exporting !== null" @click="exportReport('pdf')">
            <Download class="mr-1.5 h-4 w-4" /> PDF
          </Button>
          <Button variant="outline" size="sm" :disabled="exporting !== null" @click="exportReport('xlsx')">
            <Download class="mr-1.5 h-4 w-4" /> Excel
          </Button>
        </div>
      </div>

      <ReportFilterPanel v-model:from-date="fromDate" v-model:to-date="toDate" @apply="applyFilter" />

      <div v-if="errorWarnings.length > 0" class="flex items-start gap-2 rounded-xl border border-amber-500/40 bg-amber-50 p-3 text-sm dark:bg-amber-950/30">
        <AlertTriangle class="h-4 w-4 mt-0.5 shrink-0 text-amber-600" />
        <div class="min-w-0">
          <p class="font-semibold">{{ t('admin.accounting.validationErrors') }}</p>
          <ul class="mt-1 list-disc pl-5">
            <li v-for="w in errorWarnings" :key="w.code">{{ w.message }} ({{ w.expected }} vs {{ w.actual }})</li>
          </ul>
        </div>
      </div>

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

      <!-- Statement table -->
      <Card v-if="stmt">
        <CardHeader>
          <CardTitle class="text-base">{{ t(tabs.find((x) => x.key === activeTab)?.labelKey ?? '') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[640px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.accounting.colCode') }}</TableHead>
                <TableHead>{{ t('admin.accounting.colAccount') }}</TableHead>
                <TableHead class="hidden sm:table-cell">{{ t('admin.accounting.colType') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colDebit') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colCredit') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colBalance') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="6" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="(stmt?.lines ?? []).length === 0">
                  <TableCell :colspan="6" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow
                  v-for="(l, i) in (stmt?.lines ?? [])"
                  :key="i"
                  :class="l.is_total ? 'font-bold bg-muted/40' : l.is_subtotal ? 'font-semibold' : ''"
                >
                  <TableCell>{{ l.account_code }}</TableCell>
                  <TableCell>{{ l.account_name }}</TableCell>
                  <TableCell class="hidden sm:table-cell text-muted-foreground">{{ l.account_type }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ l.debit !== '0.00' ? formatCurrency(Number(l.debit)) : '' }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ l.credit !== '0.00' ? formatCurrency(Number(l.credit)) : '' }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(Number(l.balance)) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
          <div v-if="(stmt?.lines ?? []).length > 0" class="flex flex-wrap justify-end gap-x-4 gap-y-2 p-4 border-t text-sm font-semibold">
            <span>{{ t('admin.accounting.colDebit') }}: {{ formatCurrency(Number(stmt?.total_debit ?? 0)) }}</span>
            <span>{{ t('admin.accounting.colCredit') }}: {{ formatCurrency(Number(stmt?.total_credit ?? 0)) }}</span>
            <span>{{ t('admin.accounting.colBalance') }}: {{ formatCurrency(Number(stmt?.total_balance ?? 0)) }}</span>
          </div>
        </CardContent>
      </Card>

      <!-- General journal -->
      <Card v-if="activeTab === 'general-journal'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.accounting.tabGeneralJournal') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[720px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.accounting.colEntry') }}</TableHead>
                <TableHead>{{ t('admin.accounting.colDate') }}</TableHead>
                <TableHead class="hidden md:table-cell">{{ t('admin.accounting.colType') }}</TableHead>
                <TableHead class="hidden sm:table-cell">{{ t('admin.accounting.colStatus') }}</TableHead>
                <TableHead>{{ t('admin.accounting.colAccount') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colDebit') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colCredit') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="7" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="!journal || (journal.journals ?? []).length === 0">
                  <TableCell :colspan="7" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <template v-for="j in (journal?.journals ?? [])" :key="j.id">
                  <TableRow v-for="(l, idx) in (j.lines ?? [])" :key="j.id + idx">
                    <TableCell v-if="idx === 0" :rowspan="(j.lines ?? []).length" class="align-top font-medium whitespace-nowrap">{{ j.entry_number }}</TableCell>
                    <TableCell v-if="idx === 0" :rowspan="(j.lines ?? []).length" class="align-top whitespace-nowrap">{{ j.entry_date }}</TableCell>
                    <TableCell v-if="idx === 0" :rowspan="(j.lines ?? []).length" class="hidden md:table-cell align-top">{{ j.entry_type }}</TableCell>
                    <TableCell v-if="idx === 0" :rowspan="(j.lines ?? []).length" class="hidden sm:table-cell align-top">
                      <span class="inline-flex items-center gap-1">
                        <FileText class="h-3.5 w-3.5" /> {{ j.status }}
                      </span>
                    </TableCell>
                    <TableCell>{{ l.account_code }} — {{ l.account_name }}</TableCell>
                    <TableCell class="text-right tabular-nums">{{ l.debit !== '0.00' ? formatCurrency(Number(l.debit)) : '' }}</TableCell>
                    <TableCell class="text-right tabular-nums">{{ l.credit !== '0.00' ? formatCurrency(Number(l.credit)) : '' }}</TableCell>
                  </TableRow>
                </template>
              </template>
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>

      <!-- General ledger -->
      <Card v-if="activeTab === 'general-ledger'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.accounting.tabGeneralLedger') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="flex flex-wrap items-center gap-3 p-4 border-b">
            <label class="text-sm font-medium">{{ t('admin.accounting.colAccount') }}</label>
            <select
              v-model="selectedAccount"
              class="h-9 w-full min-w-0 max-w-[calc(100vw-2rem)] rounded-md border bg-background px-3 text-sm sm:w-auto sm:min-w-56"
              @change="onLedgerAccountChange"
            >
              <option v-if="accountsCache" v-for="a in accountsCache.accounts" :key="a.id" :value="a.id">
                {{ a.code }} — {{ a.name }}
              </option>
            </select>
          </div>
          <div class="overflow-x-auto">
          <Table class="min-w-[640px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.accounting.colDate') }}</TableHead>
                <TableHead>{{ t('admin.accounting.colEntry') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colDebit') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colCredit') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colBalance') }}</TableHead>
                <TableHead class="hidden md:table-cell">{{ t('admin.accounting.colAccount') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="6" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="!ledger || (ledger.lines ?? []).length === 0">
                  <TableCell :colspan="6" class="text-center py-8 text-muted-foreground">{{ t('admin.reports.noData') }}</TableCell>
                </TableRow>
                <TableRow v-for="l in (ledger?.lines ?? [])" :key="l.entry_id + l.entry_number">
                  <TableCell class="whitespace-nowrap">{{ l.entry_date }}</TableCell>
                  <TableCell class="whitespace-nowrap">{{ l.entry_number }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ l.debit !== '0.00' ? formatCurrency(Number(l.debit)) : '' }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ l.credit !== '0.00' ? formatCurrency(Number(l.credit)) : '' }}</TableCell>
                  <TableCell class="text-right font-medium tabular-nums">{{ formatCurrency(Number(l.balance)) }}</TableCell>
                  <TableCell class="hidden md:table-cell text-muted-foreground">{{ l.memo }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
          <div v-if="ledger" class="flex flex-wrap justify-end gap-x-4 gap-y-2 p-4 border-t text-sm font-semibold">
            <span>{{ t('admin.accounting.colBalance') }}: {{ formatCurrency(Number(ledger.running_balance)) }}</span>
          </div>
        </CardContent>
      </Card>

      <!-- Validate -->
      <Card v-if="activeTab === 'validate'">
        <CardHeader>
          <CardTitle class="text-base">{{ t('admin.accounting.tabValidate') }}</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <div class="overflow-x-auto">
          <Table class="min-w-[640px]">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('admin.accounting.colCode') }}</TableHead>
                <TableHead class="hidden sm:table-cell">{{ t('admin.accounting.colSeverity') }}</TableHead>
                <TableHead>{{ t('admin.accounting.colMessage') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colExpected') }}</TableHead>
                <TableHead class="text-right">{{ t('admin.accounting.colActual') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="loading">
                <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">{{ t('admin.shared.loading') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-if="!validateRes || (validateRes.warnings ?? []).length === 0">
                  <TableCell :colspan="5" class="text-center py-8 text-muted-foreground">
                    <CheckCircle2 class="inline h-4 w-4 mr-1.5 text-emerald-600" /> {{ t('admin.accounting.allValid') }}
                  </TableCell>
                </TableRow>
                <TableRow v-for="(w, i) in (validateRes?.warnings ?? [])" :key="i">
                  <TableCell class="font-mono text-xs">{{ w.code }}</TableCell>
                  <TableCell class="hidden sm:table-cell">
                    <span :class="w.severity === 'ERROR' ? 'text-destructive' : 'text-amber-600'">{{ w.severity }}</span>
                  </TableCell>
                  <TableCell>{{ w.message }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ formatCurrency(Number(w.expected)) }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ formatCurrency(Number(w.actual)) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  </DashboardShell>
</template>
