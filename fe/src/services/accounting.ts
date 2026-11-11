import http from '@/lib/http';
import type {
  AccountingStatementResponse,
  AccountingLedgerResponse,
  AccountingJournalListResponse,
  AccountingAccountListResponse,
  AccountingPeriod,
  AccountingValidateResponse,
} from '@/types';

export interface ReportParams {
  start_date?: string;
  end_date?: string;
  store_id?: string;
}

export const accountingService = {
  async trialBalance(params: ReportParams): Promise<AccountingStatementResponse> {
    const r = await http.get<AccountingStatementResponse>('/v1/accounting/reports/trial-balance', { params });
    return r.data;
  },
  async incomeStatement(params: ReportParams): Promise<AccountingStatementResponse> {
    const r = await http.get<AccountingStatementResponse>('/v1/accounting/reports/income-statement', { params });
    return r.data;
  },
  async balanceSheet(params: ReportParams): Promise<AccountingStatementResponse> {
    const r = await http.get<AccountingStatementResponse>('/v1/accounting/reports/balance-sheet', { params });
    return r.data;
  },
  async changesInEquity(params: ReportParams): Promise<AccountingStatementResponse> {
    const r = await http.get<AccountingStatementResponse>('/v1/accounting/reports/changes-in-equity', { params });
    return r.data;
  },
  async cashFlow(params: ReportParams): Promise<AccountingStatementResponse> {
    const r = await http.get<AccountingStatementResponse>('/v1/accounting/reports/cash-flow', { params });
    return r.data;
  },
  async generalLedger(params: { account_id: string; start_date?: string; end_date?: string; page?: number; per_page?: number }): Promise<AccountingLedgerResponse> {
    const r = await http.get<AccountingLedgerResponse>('/v1/accounting/reports/general-ledger', { params });
    return r.data;
  },
  async generalJournal(params: { source_type?: string; status?: string; start_date?: string; end_date?: string; page?: number; per_page?: number }): Promise<AccountingJournalListResponse> {
    const r = await http.get<AccountingJournalListResponse>('/v1/accounting/reports/general-journal', { params });
    return r.data;
  },
  async listJournals(params: { source_type?: string; status?: string; start_date?: string; end_date?: string; page?: number; per_page?: number }): Promise<AccountingJournalListResponse> {
    const r = await http.get<AccountingJournalListResponse>('/v1/accounting/journals', { params });
    return r.data;
  },
  async listAccounts(params: { account_type?: string; is_active?: string; page?: number; per_page?: number }): Promise<AccountingAccountListResponse> {
    const r = await http.get<AccountingAccountListResponse>('/v1/accounting/accounts', { params });
    return r.data;
  },
  async listPeriods(): Promise<{ periods: AccountingPeriod[]; total_count: number }> {
    const r = await http.get('/v1/accounting/periods');
    return r.data;
  },
  async validate(params: ReportParams): Promise<AccountingValidateResponse> {
    const r = await http.get<AccountingValidateResponse>('/v1/accounting/reports/validate', { params });
    return r.data;
  },
  async exportStatement(report: string, params: { format: 'pdf' | 'xlsx'; start_date?: string; end_date?: string; store_id?: string }): Promise<Blob> {
    const r = await http.get(`/v1/accounting/export/${report}`, {
      params,
      responseType: 'blob',
    });
    return r.data as Blob;
  },
};
