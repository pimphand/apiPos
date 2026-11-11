package handler

import (
	"context"
	"time"

	accountingv1 "apiPos/gen/accounting/v1"
	"apiPos/internal/accounting"
	accoutningreport "apiPos/internal/accounting/report"
	"apiPos/internal/accounting/export"
	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountingHandler implements the AccountingService gRPC server.
type AccountingHandler struct {
	accountingv1.UnimplementedAccountingServiceServer
	report  *accoutningreport.Service
	admin   *accounting.AdminService
	export  *export.Service
	posting *accounting.PostingService
	repos   *repository.Repositories
}

// NewAccountingHandler creates a new accounting Handler.
func NewAccountingHandler(r *accoutningreport.Service, a *accounting.AdminService, e *export.Service, p *accounting.PostingService, repos *repository.Repositories) *AccountingHandler {
	return &AccountingHandler{report: r, admin: a, export: e, posting: p, repos: repos}
}

func decStr(d decimal.Decimal) string { return d.StringFixed(2) }

func toStatementResponse(r *accoutningreport.StatementResult) *accountingv1.StatementResponse {
	lines := make([]*accountingv1.StatementLine, 0, len(r.Lines))
	for _, l := range r.Lines {
		lines = append(lines, &accountingv1.StatementLine{
			AccountCode:  l.AccountCode,
			AccountName:  l.AccountName,
			AccountType:  l.AccountType,
			Debit:        decStr(l.Debit),
			Credit:       decStr(l.Credit),
			Balance:      decStr(l.Balance),
			PriorBalance: decStr(l.PriorBalance),
			Delta:        decStr(l.Delta),
			IsSubtotal:   l.IsSubtotal,
			IsTotal:      l.IsTotal,
		})
	}
	warnings := make([]*accountingv1.ValidationWarning, 0, len(r.Warnings))
	for _, w := range r.Warnings {
		warnings = append(warnings, &accountingv1.ValidationWarning{
			Code:        w.Code,
			Severity:    w.Severity,
			Message:     w.Message,
			AccountCode: w.AccountCode,
			Expected:    w.Expected,
			Actual:      w.Actual,
		})
	}
	return &accountingv1.StatementResponse{
		Title:        r.Title,
		StartDate:    r.StartDate,
		EndDate:      r.EndDate,
		Lines:        lines,
		TotalDebit:   decStr(r.TotalDebit),
		TotalCredit:  decStr(r.TotalCredit),
		TotalBalance: decStr(r.TotalBalance),
		Warnings:     warnings,
		GeneratedAt:  r.GeneratedAt,
	}
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}

// ---- Accounts ----

func (h *AccountingHandler) CreateAccount(ctx context.Context, req *accountingv1.CreateAccountRequest) (*accountingv1.AccountResponse, error) {
	acc, err := h.admin.CreateAccount(ctx, accounting.CreateAccountInput{
		StoreID: storeIDFromCtx(ctx, h.repos),
		Code: req.Code, Name: req.Name, AccountType: req.AccountType,
		NormalBalance: req.NormalBalance, IsCashEquiv: req.IsCashEquiv, IsActive: req.IsActive,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toAccountResponse(acc), nil
}

func (h *AccountingHandler) GetAccount(ctx context.Context, req *accountingv1.GetAccountRequest) (*accountingv1.AccountResponse, error) {
	acc, err := h.repos.Acct.GetAccount(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toAccountResponse(acc), nil
}

func (h *AccountingHandler) ListAccounts(ctx context.Context, req *accountingv1.ListAccountsRequest) (*accountingv1.ListAccountsResponse, error) {
	accounts, total, err := h.repos.Acct.ListAccounts(ctx, req.AccountType, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	perPage := req.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	out := make([]*accountingv1.AccountResponse, 0, len(accounts))
	for i := range accounts {
		out = append(out, toAccountResponse(&accounts[i]))
	}
	return &accountingv1.ListAccountsResponse{Accounts: out, TotalCount: total, Page: page, PerPage: perPage}, nil
}

func (h *AccountingHandler) UpdateAccount(ctx context.Context, req *accountingv1.UpdateAccountRequest) (*accountingv1.AccountResponse, error) {
	acc, err := h.repos.Acct.GetAccount(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	if req.Name != "" {
		acc.Name = req.Name
	}
	if req.IsActive != "" {
		acc.IsActive = req.IsActive == "true"
	}
	if err := h.repos.Acct.UpdateAccount(ctx, acc); err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toAccountResponse(acc), nil
}

func toAccountResponse(a *gormdb.ChartOfAccount) *accountingv1.AccountResponse {
	parentID := ""
	if a.ParentID != nil {
		parentID = a.ParentID.String()
	}
	return &accountingv1.AccountResponse{
		Id:            a.ID.String(),
		Code:          a.Code,
		Name:          a.Name,
		AccountType:   a.AccountType,
		NormalBalance: a.NormalBalance,
		ParentId:      parentID,
		IsCashEquiv:   a.IsCashEquiv,
		IsActive:      a.IsActive,
		CreatedAt:     a.CreatedAt.Format(time.RFC3339),
	}
}

// ---- Journals ----

func (h *AccountingHandler) CreateManualJournal(ctx context.Context, req *accountingv1.CreateManualJournalRequest) (*accountingv1.JournalResponse, error) {
	date, err := parseDate(req.EntryDate)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "invalid entry_date", err).GRPCStatus().Err()
	}
	lines := make([]accounting.LineInput, 0, len(req.Lines))
	for _, l := range req.Lines {
		dr, _ := decimal.NewFromString(l.Debit)
		cr, _ := decimal.NewFromString(l.Credit)
		lines = append(lines, accounting.LineInput{AccountCode: l.AccountCode, Debit: dr, Credit: cr, Memo: l.Memo})
	}
	entry, err := h.admin.CreateManualJournal(ctx, storeIDFromCtx(ctx, h.repos), date, req.Description, lines, uuid.Nil)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return h.toJournalResponse(ctx, entry)
}

func (h *AccountingHandler) PostJournal(ctx context.Context, req *accountingv1.PostJournalRequest) (*accountingv1.JournalResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "invalid id", err).GRPCStatus().Err()
	}
	entry, err := h.admin.PostJournal(ctx, id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return h.toJournalResponse(ctx, entry)
}

func (h *AccountingHandler) ReverseJournal(ctx context.Context, req *accountingv1.ReverseJournalRequest) (*accountingv1.JournalResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "invalid id", err).GRPCStatus().Err()
	}
	entry, err := h.admin.ReverseJournal(ctx, id, req.Reason, time.Now(), uuid.Nil)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return h.toJournalResponse(ctx, entry)
}

func (h *AccountingHandler) ListJournals(ctx context.Context, req *accountingv1.ListJournalsRequest) (*accountingv1.ListJournalsResponse, error) {
	entries, total, err := h.repos.Acct.ListJournals(ctx, req.SourceType, req.Status, req.StartDate, req.EndDate, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	perPage := req.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	out := make([]*accountingv1.JournalResponse, 0, len(entries))
	for i := range entries {
		jr, _ := h.toJournalResponse(ctx, &entries[i])
		out = append(out, jr)
	}
	return &accountingv1.ListJournalsResponse{Journals: out, TotalCount: total, Page: page, PerPage: perPage}, nil
}

func (h *AccountingHandler) GetJournal(ctx context.Context, req *accountingv1.GetJournalRequest) (*accountingv1.JournalResponse, error) {
	entry, err := h.repos.Acct.GetJournal(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return h.toJournalResponse(ctx, entry)
}

func (h *AccountingHandler) toJournalResponse(ctx context.Context, entry *gormdb.JournalEntry) (*accountingv1.JournalResponse, error) {
	lines, _ := h.repos.Acct.GetJournalLines(ctx, entry.ID.String())
	respLines := make([]*accountingv1.JournalLineResponse, 0, len(lines))
	for _, l := range lines {
		var acc gormdb.ChartOfAccount
		code, name := "", ""
		if err := h.repos.DB().WithContext(ctx).Where("id = ?", l.AccountID).First(&acc).Error; err == nil {
			code, name = acc.Code, acc.Name
		}
		respLines = append(respLines, &accountingv1.JournalLineResponse{
			Id: l.ID.String(), AccountId: l.AccountID.String(),
			AccountCode: code, AccountName: name,
			Debit: decStr(l.Debit), Credit: decStr(l.Credit), Memo: l.Memo,
		})
	}
	srcID := ""
	if entry.SourceID != nil {
		srcID = entry.SourceID.String()
	}
	revOf, revBy := "", ""
	if entry.ReversalOfID != nil {
		revOf = entry.ReversalOfID.String()
	}
	if entry.ReversedByID != nil {
		revBy = entry.ReversedByID.String()
	}
	return &accountingv1.JournalResponse{
		Id:            entry.ID.String(),
		EntryNumber:   entry.EntryNumber,
		EntryDate:     entry.EntryDate.Format("2006-01-02"),
		EntryType:     entry.EntryType,
		SourceType:    entry.SourceType,
		SourceId:      srcID,
		Description:   entry.Description,
		Status:        entry.Status,
		ReversalOfId:  revOf,
		ReversedById:  revBy,
		CreatedAt:     entry.CreatedAt.Format(time.RFC3339),
		Lines:         respLines,
	}, nil
}

// ---- Periods ----

func (h *AccountingHandler) CreatePeriod(ctx context.Context, req *accountingv1.CreatePeriodRequest) (*accountingv1.PeriodResponse, error) {
	p, err := h.admin.CreatePeriod(ctx, storeIDFromCtx(ctx, h.repos), req.PeriodName, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toPeriodResponse(p), nil
}

func (h *AccountingHandler) ClosePeriod(ctx context.Context, req *accountingv1.ClosePeriodRequest) (*accountingv1.PeriodResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "invalid id", err).GRPCStatus().Err()
	}
	p, err := h.admin.ClosePeriod(ctx, id, uuid.Nil)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toPeriodResponse(p), nil
}

func (h *AccountingHandler) ListPeriods(ctx context.Context, req *accountingv1.ListPeriodsRequest) (*accountingv1.ListPeriodsResponse, error) {
	periods, err := h.repos.Acct.ListPeriods(ctx)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	out := make([]*accountingv1.PeriodResponse, 0, len(periods))
	for i := range periods {
		out = append(out, toPeriodResponse(&periods[i]))
	}
	return &accountingv1.ListPeriodsResponse{Periods: out, TotalCount: int64(len(periods))}, nil
}

func toPeriodResponse(p *gormdb.AccountingPeriod) *accountingv1.PeriodResponse {
	closedAt := ""
	if p.ClosedAt != nil {
		closedAt = p.ClosedAt.Format(time.RFC3339)
	}
	return &accountingv1.PeriodResponse{
		Id:         p.ID.String(),
		PeriodName: p.PeriodName,
		StartDate:  p.StartDate.Format("2006-01-02"),
		EndDate:    p.EndDate.Format("2006-01-02"),
		Status:     p.Status,
		ClosedAt:   closedAt,
	}
}

// ---- Statements ----

func (h *AccountingHandler) GetTrialBalance(ctx context.Context, req *accountingv1.ReportRequest) (*accountingv1.StatementResponse, error) {
	r, err := h.report.TrialBalance(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toStatementResponse(r), nil
}

func (h *AccountingHandler) GetIncomeStatement(ctx context.Context, req *accountingv1.ReportRequest) (*accountingv1.StatementResponse, error) {
	r, err := h.report.IncomeStatement(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toStatementResponse(r), nil
}

func (h *AccountingHandler) GetBalanceSheet(ctx context.Context, req *accountingv1.ReportRequest) (*accountingv1.StatementResponse, error) {
	r, err := h.report.BalanceSheet(ctx, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toStatementResponse(r), nil
}

func (h *AccountingHandler) GetChangesInEquity(ctx context.Context, req *accountingv1.ReportRequest) (*accountingv1.StatementResponse, error) {
	r, err := h.report.ChangesInEquity(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toStatementResponse(r), nil
}

func (h *AccountingHandler) GetCashFlow(ctx context.Context, req *accountingv1.ReportRequest) (*accountingv1.StatementResponse, error) {
	r, err := h.report.CashFlow(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return toStatementResponse(r), nil
}

func (h *AccountingHandler) GetGeneralLedger(ctx context.Context, req *accountingv1.GeneralLedgerRequest) (*accountingv1.GeneralLedgerResponse, error) {
	r, err := h.report.GeneralLedger(ctx, req.AccountId, req.StartDate, req.EndDate, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	lines := make([]*accountingv1.LedgerLine, 0, len(r.Lines))
	for _, l := range r.Lines {
		lines = append(lines, &accountingv1.LedgerLine{
			EntryDate: l.EntryDate, EntryNumber: l.EntryNumber, SourceType: l.SourceType,
			SourceId: l.SourceID, Debit: decStr(l.Debit), Credit: decStr(l.Credit),
			Balance: decStr(l.Balance), Memo: l.Memo, EntryId: l.EntryID,
		})
	}
	page := r.Page
	if page <= 0 {
		page = 1
	}
	perPage := r.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	return &accountingv1.GeneralLedgerResponse{
		Lines: lines, TotalCount: r.TotalCount, Page: page, PerPage: perPage,
		RunningBalance: decStr(r.RunningBalance),
	}, nil
}

func (h *AccountingHandler) GetGeneralJournal(ctx context.Context, req *accountingv1.ListJournalsRequest) (*accountingv1.ListJournalsResponse, error) {
	return h.ListJournals(ctx, req)
}

func (h *AccountingHandler) ValidateBooks(ctx context.Context, req *accountingv1.ReportRequest) (*accountingv1.ValidateResponse, error) {
	warnings, err := h.report.Validate(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	out := make([]*accountingv1.ValidationWarning, 0, len(warnings))
	for _, w := range warnings {
		out = append(out, &accountingv1.ValidationWarning{
			Code: w.Code, Severity: w.Severity, Message: w.Message,
			AccountCode: w.AccountCode, Expected: w.Expected, Actual: w.Actual,
		})
	}
	return &accountingv1.ValidateResponse{Warnings: out, GeneratedAt: time.Now().Format(time.RFC3339)}, nil
}

// ---- Export ----

func (h *AccountingHandler) ExportStatement(ctx context.Context, req *accountingv1.ExportRequest) (*accountingv1.ExportResponse, error) {
	data, filename, contentType, err := h.export.Export(ctx, req.Report, req.Format, req.StartDate, req.EndDate)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}
	return &accountingv1.ExportResponse{FileBytes: data, Filename: filename, ContentType: contentType}, nil
}

func storeIDFromCtx(ctx context.Context, repos *repository.Repositories) uuid.UUID {
	// Derive the tenant from JWT claims (superadmin may explicitly scope via
	// query params at the service layer; GORM callbacks handle the rest).
	if claims, ok := auth.ClaimsFromContext(ctx); ok && claims != nil && claims.StoreID != "" {
		if id, err := uuid.Parse(claims.StoreID); err == nil {
			return id
		}
	}
	if id, err := uuid.Parse(repository.DefaultStoreID); err == nil {
		return id
	}
	return uuid.Nil
}
