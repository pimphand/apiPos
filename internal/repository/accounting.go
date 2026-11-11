package repository

import (
	"context"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// AccountingRepository defines the interface for accounting operations.
type AccountingRepository interface {
	// Accounts
	GetAccount(ctx context.Context, id string) (*gormdb.ChartOfAccount, error)
	ListAccounts(ctx context.Context, accountType, isActive string, page, perPage int32) ([]gormdb.ChartOfAccount, int64, error)
	CreateAccount(ctx context.Context, acc *gormdb.ChartOfAccount) error
	UpdateAccount(ctx context.Context, acc *gormdb.ChartOfAccount) error

	// Journals
	GetJournal(ctx context.Context, id string) (*gormdb.JournalEntry, error)
	ListJournals(ctx context.Context, sourceType, status, startDate, endDate string, page, perPage int32) ([]gormdb.JournalEntry, int64, error)
	GetJournalLines(ctx context.Context, entryID string) ([]gormdb.JournalLine, error)

	// Periods
	GetPeriod(ctx context.Context, id string) (*gormdb.AccountingPeriod, error)
	ListPeriods(ctx context.Context) ([]gormdb.AccountingPeriod, error)
	CreatePeriod(ctx context.Context, p *gormdb.AccountingPeriod) error
	ClosePeriod(ctx context.Context, p *gormdb.AccountingPeriod) error

	// Ledger aggregation (POSTED only, over general_ledger view)
	AccountBalances(ctx context.Context, startDate, endDate string) ([]AccountBalance, error)
	AccountBalancesAsOf(ctx context.Context, asOf string) ([]AccountBalance, error)
	LedgerLines(ctx context.Context, accountID, startDate, endDate string, page, perPage int32) ([]gormdb.GeneralLedgerEntry, int64, error)
}

// AccountBalance is the aggregated balance of one account over a period.
type AccountBalance struct {
	AccountID     string
	Code          string
	Name          string
	AccountType   string
	NormalBalance string
	TotalDebit    string
	TotalCredit   string
}

// Repo implements AccountingRepository with GORM.
type Repo struct {
	db *gorm.DB
}

// NewAccountingRepository creates a new accounting Repo.
func NewAccountingRepository(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) GetAccount(ctx context.Context, id string) (*gormdb.ChartOfAccount, error) {
	var acc gormdb.ChartOfAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&acc).Error
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repo) ListAccounts(ctx context.Context, accountType, isActive string, page, perPage int32) ([]gormdb.ChartOfAccount, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	var total int64
	query := r.db.WithContext(ctx).Model(&gormdb.ChartOfAccount{})
	if accountType != "" {
		query = query.Where("chart_of_accounts.account_type = ?", accountType)
	}
	if isActive == "true" {
		query = query.Where("chart_of_accounts.is_active = ?", true)
	} else if isActive == "false" {
		query = query.Where("chart_of_accounts.is_active = ?", false)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var accounts []gormdb.ChartOfAccount
	offset := (page - 1) * perPage
	err := query.Order("chart_of_accounts.code ASC").Offset(int(offset)).Limit(int(perPage)).Find(&accounts).Error
	return accounts, total, err
}

func (r *Repo) CreateAccount(ctx context.Context, acc *gormdb.ChartOfAccount) error {
	return r.db.WithContext(ctx).Create(acc).Error
}

func (r *Repo) UpdateAccount(ctx context.Context, acc *gormdb.ChartOfAccount) error {
	return r.db.WithContext(ctx).Save(acc).Error
}

func (r *Repo) GetJournal(ctx context.Context, id string) (*gormdb.JournalEntry, error) {
	var entry gormdb.JournalEntry
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *Repo) ListJournals(ctx context.Context, sourceType, status, startDate, endDate string, page, perPage int32) ([]gormdb.JournalEntry, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	var total int64
	query := r.db.WithContext(ctx).Model(&gormdb.JournalEntry{})
	if sourceType != "" {
		query = query.Where("journal_entries.source_type = ?", sourceType)
	}
	if status != "" {
		query = query.Where("journal_entries.status = ?", status)
	}
	if startDate != "" {
		query = query.Where("journal_entries.entry_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("journal_entries.entry_date <= ?", endDate)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var entries []gormdb.JournalEntry
	offset := (page - 1) * perPage
	err := query.Order("journal_entries.entry_date DESC, journal_entries.entry_number DESC").
		Offset(int(offset)).Limit(int(perPage)).Find(&entries).Error
	return entries, total, err
}

func (r *Repo) GetJournalLines(ctx context.Context, entryID string) ([]gormdb.JournalLine, error) {
	var lines []gormdb.JournalLine
	err := r.db.WithContext(ctx).Where("entry_id = ?", entryID).Find(&lines).Error
	return lines, err
}

func (r *Repo) GetPeriod(ctx context.Context, id string) (*gormdb.AccountingPeriod, error) {
	var p gormdb.AccountingPeriod
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) ListPeriods(ctx context.Context) ([]gormdb.AccountingPeriod, error) {
	var periods []gormdb.AccountingPeriod
	err := r.db.WithContext(ctx).Order("period_name DESC").Find(&periods).Error
	return periods, err
}

func (r *Repo) CreatePeriod(ctx context.Context, p *gormdb.AccountingPeriod) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *Repo) ClosePeriod(ctx context.Context, p *gormdb.AccountingPeriod) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *Repo) AccountBalances(ctx context.Context, startDate, endDate string) ([]AccountBalance, error) {
	var results []AccountBalance
	query := r.db.WithContext(ctx).
		Table("general_ledger gl").
		Select("gl.account_id, a.code, a.name, a.account_type, a.normal_balance, COALESCE(SUM(gl.debit),0)::text AS total_debit, COALESCE(SUM(gl.credit),0)::text AS total_credit").
		Joins("JOIN chart_of_accounts a ON a.id = gl.account_id")
	if startDate != "" {
		query = query.Where("gl.entry_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("gl.entry_date <= ?", endDate)
	}
	err := query.Group("gl.account_id, a.code, a.name, a.account_type, a.normal_balance").
		Order("a.code ASC").Scan(&results).Error
	return results, err
}

func (r *Repo) AccountBalancesAsOf(ctx context.Context, asOf string) ([]AccountBalance, error) {
	if asOf == "" {
		return nil, nil
	}
	return r.AccountBalances(ctx, "", asOf)
}

func (r *Repo) LedgerLines(ctx context.Context, accountID, startDate, endDate string, page, perPage int32) ([]gormdb.GeneralLedgerEntry, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	var total int64
	query := r.db.WithContext(ctx).Model(&gormdb.GeneralLedgerEntry{}).Where("account_id = ?", accountID)
	if startDate != "" {
		query = query.Where("entry_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("entry_date <= ?", endDate)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var lines []gormdb.GeneralLedgerEntry
	offset := (page - 1) * perPage
	err := query.Order("entry_date ASC, entry_number ASC").
		Offset(int(offset)).Limit(int(perPage)).Find(&lines).Error
	return lines, total, err
}
