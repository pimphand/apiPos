package accounting

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// AdminService handles manual journals, account CRUD, and period management.
type AdminService struct {
	repos *repository.Repositories
	post  *PostingService
}

// NewAdminService creates a new AdminService.
func NewAdminService(repos *repository.Repositories) *AdminService {
	return &AdminService{repos: repos, post: NewPostingService(repos.DB())}
}

// CreateAccount creates a new chart-of-accounts entry.
func (s *AdminService) CreateAccount(ctx context.Context, input CreateAccountInput) (*gormdb.ChartOfAccount, error) {
	storeID := input.StoreID
	if storeID == uuid.Nil {
		// Store scope comes from JWT — but GORM callback will set store_id.
		// For direct calls (tests) the caller passes storeID explicitly.
		storeID = input.StoreID
	}
	acc := &gormdb.ChartOfAccount{
		StoreID: storeID, Code: input.Code, Name: input.Name,
		AccountType: input.AccountType, NormalBalance: input.NormalBalance,
		IsCashEquiv: input.IsCashEquiv, IsActive: input.IsActive,
	}
	if input.ParentID != nil && *input.ParentID != uuid.Nil {
		acc.ParentID = input.ParentID
	}
	if err := s.repos.Acct.CreateAccount(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

// CreateAccountInput holds parameters for CreateAccount.
type CreateAccountInput struct {
	StoreID       uuid.UUID
	Code          string
	Name          string
	AccountType   string
	NormalBalance string
	ParentID      *uuid.UUID
	IsCashEquiv   bool
	IsActive      bool
}

// CreateManualJournal creates a DRAFT manual journal.
func (s *AdminService) CreateManualJournal(ctx context.Context, storeID uuid.UUID, date time.Time, description string, lines []LineInput, createdBy uuid.UUID) (*gormdb.JournalEntry, error) {
	period, err := s.post.ResolvePeriod(ctx, s.repos.DB().WithContext(ctx), storeID, date)
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.post.NextEntryNumber(ctx, s.repos.DB().WithContext(ctx), storeID, date)
	if err != nil {
		return nil, err
	}
	entry := &gormdb.JournalEntry{
		StoreID: storeID, PeriodID: period.ID, EntryNumber: entryNumber,
		EntryDate: date, EntryType: EntryTypeManual, SourceType: SourceTypeManual,
		IdempotencyKey: "MANUAL:" + uuid.NewString(),
		Description:    description, Status: JournalStatusDraft,
		CreatedBy: &createdBy,
	}
	if err := s.repos.DB().WithContext(ctx).Create(entry).Error; err != nil {
		return nil, err
	}
	for _, l := range lines {
		accountID, err := GetAccountID(ctx, s.repos.DB().WithContext(ctx), storeID, l.AccountCode)
		if err != nil {
			return nil, err
		}
		if err := s.repos.DB().WithContext(ctx).Create(&gormdb.JournalLine{
			StoreID: storeID, EntryID: entry.ID, AccountID: accountID,
			Debit: l.Debit.Round(2), Credit: l.Credit.Round(2), Memo: l.Memo,
		}).Error; err != nil {
			return nil, err
		}
	}
	return entry, nil
}

// PostJournal transitions a DRAFT journal to POSTED (balance is enforced by
// the DB trigger and re-checked service-side).
func (s *AdminService) PostJournal(ctx context.Context, entryID uuid.UUID) (*gormdb.JournalEntry, error) {
	entry, err := s.repos.Acct.GetJournal(ctx, entryID.String())
	if err != nil {
		return nil, err
	}
	if entry.Status == JournalStatusPosted {
		return entry, nil
	}
	if entry.Status != JournalStatusDraft {
		return nil, gorm.ErrRecordNotFound
	}
	// Re-check balance before posting.
	var lines []gormdb.JournalLine
	if err := s.repos.DB().WithContext(ctx).Where("entry_id = ?", entryID).Find(&lines).Error; err != nil {
		return nil, err
	}
	var dr, cr decimal.Decimal
	for _, l := range lines {
		dr = dr.Add(l.Debit)
		cr = cr.Add(l.Credit)
	}
	if !dr.Equal(cr) {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now()
	entry.Status = JournalStatusPosted
	entry.PostedAt = &now
	if err := s.repos.DB().WithContext(ctx).Save(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

// ReverseJournal reverses a POSTED journal (idempotent).
func (s *AdminService) ReverseJournal(ctx context.Context, entryID uuid.UUID, reason string, date time.Time, createdBy uuid.UUID) (*gormdb.JournalEntry, error) {
	orig, err := s.repos.Acct.GetJournal(ctx, entryID.String())
	if err != nil {
		return nil, err
	}
	if orig.Status != JournalStatusPosted {
		return nil, gorm.ErrRecordNotFound
	}
	return s.post.PostReversal(ctx, s.repos.DB().WithContext(ctx), orig, reason, date)
}

// CreatePeriod creates an accounting period (or returns the existing one).
func (s *AdminService) CreatePeriod(ctx context.Context, storeID uuid.UUID, periodName, startDate, endDate string) (*gormdb.AccountingPeriod, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, err
	}
	var existing gormdb.AccountingPeriod
	if err := s.repos.DB().WithContext(ctx).
		Where("store_id = ? AND period_name = ?", storeID, periodName).
		First(&existing).Error; err == nil {
		return &existing, nil
	}
	p := &gormdb.AccountingPeriod{
		StoreID: storeID, PeriodName: periodName,
		StartDate: start, EndDate: end, Status: PeriodStatusOpen,
	}
	if err := s.repos.Acct.CreatePeriod(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ClosePeriod closes a period (idempotent).
func (s *AdminService) ClosePeriod(ctx context.Context, periodID uuid.UUID, closedBy uuid.UUID) (*gormdb.AccountingPeriod, error) {
	p, err := s.repos.Acct.GetPeriod(ctx, periodID.String())
	if err != nil {
		return nil, err
	}
	if p.Status == PeriodStatusClosed {
		return p, nil
	}
	now := time.Now()
	p.Status = PeriodStatusClosed
	p.ClosedAt = &now
	p.ClosedBy = &closedBy
	if err := s.repos.Acct.ClosePeriod(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// BackfillOpeningBalance ensures an OPENING journal exists for a store.
// Safe to call repeatedly (idempotent via the opening key).
func (s *AdminService) BackfillOpeningBalance(ctx context.Context, storeID uuid.UUID, asOf time.Time) (*gormdb.JournalEntry, error) {
	// Inventory opening = SUM(products.stock * cost_price) at seed time.
	var invValue decimal.Decimal
	err := s.repos.DB().WithContext(ctx).Raw(
		"SELECT COALESCE(SUM(stock::numeric * cost_price::numeric), 0) FROM products WHERE store_id = ?",
		storeID).Scan(&invValue).Error
	if err != nil {
		return nil, err
	}
	if invValue.IsZero() {
		return nil, nil
	}
	key := "OPENING:" + storeID.String() + ":" + asOf.Format("2006-01-02")
	return s.post.PostJournal(ctx, s.repos.DB().WithContext(ctx), PostParams{
		StoreID: storeID, Date: asOf,
		EntryType:      EntryTypeOpening,
		SourceType:     SourceTypeManual,
		IdempotencyKey: key,
		Description:    "Opening balance for " + storeID.String(),
		Lines: []LineInput{
			{AccountCode: CodeInventory, Debit: invValue.Round(2), Memo: "opening inventory"},
			{AccountCode: CodeOwnerCapital, Credit: invValue.Round(2), Memo: "opening capital"},
		},
	})
}
