package report

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"apiPos/internal/accounting"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// ValidationWarning is returned when a reconciliation check fails.
type ValidationWarning struct {
	Code        string
	Severity    string
	Message     string
	AccountCode string
	Expected    string
	Actual      string
}

// decOrZero parses a decimal string, returning zero on error.
func decOrZero(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// StatementLine is one row of a financial statement.
type StatementLine struct {
	AccountCode  string
	AccountName  string
	AccountType  string
	Debit        decimal.Decimal
	Credit       decimal.Decimal
	Balance      decimal.Decimal
	PriorBalance decimal.Decimal
	Delta        decimal.Decimal
	IsSubtotal   bool
	IsTotal      bool
}

// StatementResult holds a financial statement.
type StatementResult struct {
	Title        string
	StartDate    string
	EndDate      string
	Lines        []StatementLine
	TotalDebit   decimal.Decimal
	TotalCredit  decimal.Decimal
	TotalBalance decimal.Decimal
	Warnings     []ValidationWarning
	GeneratedAt  string
}

// LedgerResult holds a paginated general-ledger view for one account.
type LedgerResult struct {
	Lines []LedgerLine
	TotalCount int64
	Page int32
	PerPage int32
	RunningBalance decimal.Decimal
}

// LedgerLine is one general-ledger entry with running balance.
type LedgerLine struct {
	EntryDate   string
	EntryNumber string
	SourceType  string
	SourceID    string
	Debit       decimal.Decimal
	Credit      decimal.Decimal
	Balance     decimal.Decimal
	Memo        string
	EntryID     string
}

// Service implements the 7 financial statements over general_ledger.
type Service struct {
	repos *repository.Repositories
	posting *accounting.PostingService
}

// NewService creates a new accounting statement Service.
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos, posting: accounting.NewPostingService(repos.DB())}
}

func (s *Service) db() *repository.Repositories { return s.repos }

// resolveStoreID returns the store to report on: explicit request value wins,
// otherwise the tenant scope from JWT claims (superadmin sees all).
func (s *Service) resolveStoreID(ctx context.Context, requested string) string {
	if requested != "" {
		return requested
	}
	// Tenant scope is applied by GORM callbacks when store_id is empty; for
	// aggregation we deliberately pass "" and let the callback scope it.
	return ""
}

// TrialBalance returns the trial balance for a period.
func (s *Service) TrialBalance(ctx context.Context, startDate, endDate string) (*StatementResult, error) {
	balances, err := s.repos.Acct.AccountBalances(ctx, startDate, endDate)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to get trial balance", err)
	}
	result := &StatementResult{Title: "Trial Balance", StartDate: startDate, EndDate: endDate, GeneratedAt: time.Now().Format(time.RFC3339)}
	totalDr := decimal.Zero
	totalCr := decimal.Zero
	for _, b := range balances {
		dr := decOrZero(b.TotalDebit)
		cr := decOrZero(b.TotalCredit)
		var bal decimal.Decimal
		if b.NormalBalance == accounting.NormalBalanceDebit {
			bal = dr.Sub(cr)
		} else {
			bal = cr.Sub(dr)
		}
		if bal.IsZero() {
			continue
		}
		result.Lines = append(result.Lines, StatementLine{
			AccountCode: b.Code, AccountName: b.Name, AccountType: b.AccountType,
			Debit: dr, Credit: cr, Balance: bal,
		})
		totalDr = totalDr.Add(dr)
		totalCr = totalCr.Add(cr)
	}
	result.TotalDebit = totalDr
	result.TotalCredit = totalCr
	result.TotalBalance = totalDr.Sub(totalCr)
	result.Warnings = s.validateBalances(balances)
	return result, nil
}

// GeneralLedger returns paginated ledger lines for one account with a running balance.
func (s *Service) GeneralLedger(ctx context.Context, accountID, startDate, endDate string, page, perPage int32) (*LedgerResult, error) {
	entries, total, err := s.repos.Acct.LedgerLines(ctx, accountID, startDate, endDate, page, perPage)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to get general ledger", err)
	}
	var acc gormdb.ChartOfAccount
	if err := s.repos.DB().WithContext(ctx).Where("id = ?", accountID).First(&acc).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "account not found", err)
	}
	running := decimal.Zero
	result := &LedgerResult{TotalCount: total, Page: page, PerPage: perPage}
	for _, e := range entries {
		if acc.NormalBalance == accounting.NormalBalanceDebit {
			running = running.Add(e.Debit).Sub(e.Credit)
		} else {
			running = running.Add(e.Credit).Sub(e.Debit)
		}
		srcID := ""
		if e.SourceID != nil {
			srcID = e.SourceID.String()
		}
		result.Lines = append(result.Lines, LedgerLine{
			EntryDate: e.EntryDate.Format("2006-01-02"), EntryNumber: e.EntryNumber,
			SourceType: e.SourceType, SourceID: srcID, Debit: e.Debit, Credit: e.Credit,
			Balance: running, Memo: e.Memo, EntryID: e.EntryID.String(),
		})
	}
	result.RunningBalance = running
	return result, nil
}

// IncomeStatement returns the profit-and-loss statement for a period.
func (s *Service) IncomeStatement(ctx context.Context, startDate, endDate string) (*StatementResult, error) {
	balances, err := s.repos.Acct.AccountBalances(ctx, startDate, endDate)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to get income statement", err)
	}
	result := &StatementResult{Title: "Income Statement", StartDate: startDate, EndDate: endDate, GeneratedAt: time.Now().Format(time.RFC3339)}
	revTotal := decimal.Zero
	expTotal := decimal.Zero
	for _, b := range balances {
		if b.AccountType != accounting.AccountTypeRevenue && b.AccountType != accounting.AccountTypeExpense {
			continue
		}
		dr := decOrZero(b.TotalDebit)
		cr := decOrZero(b.TotalCredit)
		var bal decimal.Decimal
		if b.NormalBalance == accounting.NormalBalanceDebit {
			bal = dr.Sub(cr)
		} else {
			bal = cr.Sub(dr)
		}
		if bal.IsZero() {
			continue
		}
		isSub := false
		result.Lines = append(result.Lines, StatementLine{
			AccountCode: b.Code, AccountName: b.Name, AccountType: b.AccountType,
			Debit: dr, Credit: cr, Balance: bal, IsSubtotal: isSub,
		})
		if b.AccountType == accounting.AccountTypeRevenue {
			revTotal = revTotal.Add(bal)
		} else {
			expTotal = expTotal.Add(bal)
		}
	}
	net := revTotal.Sub(expTotal)
	result.TotalDebit = revTotal
	result.TotalCredit = expTotal
	result.TotalBalance = net
	// Net profit summary line.
	result.Lines = append(result.Lines, StatementLine{
		AccountName: "Net Profit", Balance: net, IsTotal: true,
	})
	result.Warnings = s.validateBalances(balances)
	return result, nil
}

// BalanceSheet returns the statement of financial position as-of a date.
func (s *Service) BalanceSheet(ctx context.Context, asOf string) (*StatementResult, error) {
	balances, err := s.repos.Acct.AccountBalancesAsOf(ctx, asOf)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to get balance sheet", err)
	}
	result := &StatementResult{Title: "Balance Sheet", EndDate: asOf, GeneratedAt: time.Now().Format(time.RFC3339)}
	assets := decimal.Zero
	liab := decimal.Zero
	equity := decimal.Zero
	for _, b := range balances {
		if b.AccountType != accounting.AccountTypeAsset && b.AccountType != accounting.AccountTypeLiability && b.AccountType != accounting.AccountTypeEquity {
			continue
		}
		dr := decOrZero(b.TotalDebit)
		cr := decOrZero(b.TotalCredit)
		var bal decimal.Decimal
		if b.NormalBalance == accounting.NormalBalanceDebit {
			bal = dr.Sub(cr)
		} else {
			bal = cr.Sub(dr)
		}
		if bal.IsZero() {
			continue
		}
		result.Lines = append(result.Lines, StatementLine{
			AccountCode: b.Code, AccountName: b.Name, AccountType: b.AccountType,
			Debit: dr, Credit: cr, Balance: bal,
		})
		switch b.AccountType {
		case accounting.AccountTypeAsset:
			assets = assets.Add(bal)
		case accounting.AccountTypeLiability:
			liab = liab.Add(bal)
		case accounting.AccountTypeEquity:
			equity = equity.Add(bal)
		}
	}
	// Current-year profit = revenue - expense up to asOf (P&L is not closed
	// into equity until period close, so show it explicitly).
	plBalances, err := s.repos.Acct.AccountBalancesAsOf(ctx, asOf)
	if err == nil {
		rev := decimal.Zero
		exp := decimal.Zero
		for _, b := range plBalances {
			dr := decOrZero(b.TotalDebit)
			cr := decOrZero(b.TotalCredit)
			var bal decimal.Decimal
			if b.NormalBalance == accounting.NormalBalanceDebit {
				bal = dr.Sub(cr)
			} else {
				bal = cr.Sub(dr)
			}
			if b.AccountType == accounting.AccountTypeRevenue {
				rev = rev.Add(bal)
			} else if b.AccountType == accounting.AccountTypeExpense {
				exp = exp.Add(bal)
			}
		}
		profit := rev.Sub(exp)
		if !profit.IsZero() {
			result.Lines = append(result.Lines, StatementLine{
				AccountName: "Current Period Profit", Balance: profit, IsSubtotal: true,
			})
			equity = equity.Add(profit)
		}
	}
	result.TotalDebit = assets
	result.TotalCredit = liab.Add(equity)
	result.TotalBalance = assets.Sub(liab.Add(equity))
	if !result.TotalBalance.IsZero() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code: "BS_EQUATION", Severity: "ERROR",
			Message: "Balance Sheet: Assets != Liabilities + Equity",
			Expected: result.TotalDebit.StringFixed(2),
			Actual:   result.TotalCredit.StringFixed(2),
		})
	}
	return result, nil
}

// ChangesInEquity returns the statement of changes in equity for a period.
func (s *Service) ChangesInEquity(ctx context.Context, startDate, endDate string) (*StatementResult, error) {
	balances, err := s.repos.Acct.AccountBalances(ctx, startDate, endDate)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to get changes in equity", err)
	}
	result := &StatementResult{Title: "Changes in Equity", StartDate: startDate, EndDate: endDate, GeneratedAt: time.Now().Format(time.RFC3339)}
	var beginning, profit, ending decimal.Decimal
	// Opening equity = all EQUITY balances before startDate.
	var opening []repository.AccountBalance
	if startDate != "" {
		prev := addDay(startDate, -1)
		opening, _ = s.repos.Acct.AccountBalancesAsOf(ctx, prev)
	}
	for _, b := range opening {
		if b.AccountType != accounting.AccountTypeEquity {
			continue
		}
		dr := decOrZero(b.TotalDebit)
		cr := decOrZero(b.TotalCredit)
		var bal decimal.Decimal
		if b.NormalBalance == accounting.NormalBalanceDebit {
			bal = dr.Sub(cr)
		} else {
			bal = cr.Sub(dr)
		}
		beginning = beginning.Add(bal)
	}
	for _, b := range balances {
		if b.AccountType == accounting.AccountTypeEquity {
			dr := decOrZero(b.TotalDebit)
			cr := decOrZero(b.TotalCredit)
			var bal decimal.Decimal
			if b.NormalBalance == accounting.NormalBalanceDebit {
				bal = dr.Sub(cr)
			} else {
				bal = cr.Sub(dr)
			}
			result.Lines = append(result.Lines, StatementLine{
				AccountCode: b.Code, AccountName: b.Name, AccountType: b.AccountType,
				Debit: dr, Credit: cr, Balance: bal,
			})
		} else if b.AccountType == accounting.AccountTypeRevenue || b.AccountType == accounting.AccountTypeExpense {
			dr := decOrZero(b.TotalDebit)
			cr := decOrZero(b.TotalCredit)
			var bal decimal.Decimal
			if b.NormalBalance == accounting.NormalBalanceDebit {
				bal = dr.Sub(cr)
			} else {
				bal = cr.Sub(dr)
			}
			if b.AccountType == accounting.AccountTypeRevenue {
				profit = profit.Add(bal)
			} else {
				profit = profit.Sub(bal)
			}
		}
	}
	ending = beginning.Add(profit)
	// Recompute ending from all equity balances as-of endDate.
	if all, err := s.repos.Acct.AccountBalancesAsOf(ctx, endDate); err == nil {
		eq := decimal.Zero
		for _, b := range all {
			if b.AccountType != accounting.AccountTypeEquity {
				continue
			}
			dr := decOrZero(b.TotalDebit)
			cr := decOrZero(b.TotalCredit)
			var bal decimal.Decimal
			if b.NormalBalance == accounting.NormalBalanceDebit {
				bal = dr.Sub(cr)
			} else {
				bal = cr.Sub(dr)
			}
			eq = eq.Add(bal)
		}
		ending = eq
	}
	result.TotalBalance = ending
	result.Lines = append(result.Lines,
		StatementLine{AccountName: "Beginning Equity", Balance: beginning, IsSubtotal: true},
		StatementLine{AccountName: "Net Profit / (Loss)", Balance: profit, IsSubtotal: true},
		StatementLine{AccountName: "Ending Equity", Balance: ending, IsTotal: true},
	)
	return result, nil
}

// CashFlow returns the indirect cash-flow statement for a period.
func (s *Service) CashFlow(ctx context.Context, startDate, endDate string) (*StatementResult, error) {
	// Net profit from the income statement.
	is, err := s.IncomeStatement(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}
	netProfit := is.TotalBalance

	// Working-capital deltas: sum movements on relevant accounts in the period.
	delta := func(code string) decimal.Decimal {
		balances, err := s.repos.Acct.AccountBalances(ctx, startDate, endDate)
		if err != nil {
			return decimal.Zero
		}
		for _, b := range balances {
			if b.Code == code {
				dr := decOrZero(b.TotalDebit)
				cr := decOrZero(b.TotalCredit)
				if b.NormalBalance == accounting.NormalBalanceDebit {
					return dr.Sub(cr)
				}
				return cr.Sub(dr)
			}
		}
		return decimal.Zero
	}

	recvDelta := delta(accounting.CodeReceivables)  // increase = cash outflow
	invDelta := delta(accounting.CodeInventory)     // increase = cash outflow
	payDelta := delta(accounting.CodePayables)      // increase = cash inflow
	taxDelta := delta(accounting.CodeTaxPayable)    // increase = cash inflow

	operating := netProfit.Sub(recvDelta).Sub(invDelta).Add(payDelta).Add(taxDelta)

	result := &StatementResult{Title: "Cash Flow Statement", StartDate: startDate, EndDate: endDate, GeneratedAt: time.Now().Format(time.RFC3339)}
	result.Lines = []StatementLine{
		{AccountName: "Net Profit Before Tax", Balance: netProfit, IsSubtotal: true},
		{AccountName: "Change in Receivables", Balance: recvDelta.Neg()},
		{AccountName: "Change in Inventory", Balance: invDelta.Neg()},
		{AccountName: "Change in Payables", Balance: payDelta},
		{AccountName: "Change in Tax Payable", Balance: taxDelta},
		{AccountName: "Net Cash from Operating", Balance: operating, IsTotal: true},
		{AccountName: "Net Cash from Investing", Balance: decimal.Zero},
		{AccountName: "Net Cash from Financing", Balance: decimal.Zero},
		{AccountName: "Net Change in Cash", Balance: operating, IsTotal: true},
	}
	result.TotalBalance = operating
	return result, nil
}

func (s *Service) validateBalances(balances []repository.AccountBalance) []ValidationWarning {
	var totalDr, totalCr decimal.Decimal
	for _, b := range balances {
		dr := decOrZero(b.TotalDebit)
		cr := decOrZero(b.TotalCredit)
		bal := dr.Sub(cr)
		if bal.IsZero() {
			continue
		}
		totalDr = totalDr.Add(dr)
		totalCr = totalCr.Add(cr)
	}
	if !totalDr.Equal(totalCr) {
		return []ValidationWarning{{
			Code: "TB_IMBALANCE", Severity: "ERROR",
			Message: "Trial Balance: total debit != total credit",
			Expected: totalDr.StringFixed(2), Actual: totalCr.StringFixed(2),
		}}
	}
	return nil
}

func addDay(dateStr string, days int) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
