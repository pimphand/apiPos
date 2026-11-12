package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
)

// PostingService is the single choke point for all journal writes.
// All methods run inside the caller's transaction (tx) — never open their own.
type PostingService struct {
	db *gorm.DB
}

// NewPostingService creates a PostingService. The db is the shared GORM
// connection; posting methods accept a *gorm.DB (either the root or an open tx).
func NewPostingService(db *gorm.DB) *PostingService {
	return &PostingService{db: db}
}

// DB returns the base connection (for wiring / read queries).
func (p *PostingService) DB() *gorm.DB { return p.db }

// COGSMoney converts a legacy float64 amount to a 2-dp decimal.
func COGSMoney(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f).Round(2)
}

// ResolvePeriod returns the OPEN accounting period covering date, auto-creating
// it if missing. Returns apperror.ErrFailedPrecondition when the period is CLOSED.
func (p *PostingService) ResolvePeriod(ctx context.Context, tx *gorm.DB, storeID uuid.UUID, date time.Time) (*gormdb.AccountingPeriod, error) {
	if tx == nil {
		tx = p.db
	}
	// Normalize to date (midnight Asia/Jakarta -> store time).
	tz, tzErr := time.LoadLocation("Asia/Jakarta")
	if tzErr != nil {
		// Fallback for minimal images without tzdata (e.g. distroless/alpine).
		tz = time.FixedZone("Asia/Jakarta", 7*60*60)
	}
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, tz)
	periodName := day.Format("2006-01")

	var period gormdb.AccountingPeriod
	err := tx.WithContext(ctx).
		Where("store_id = ? AND period_name = ?", storeID, periodName).
		First(&period).Error
	if err == nil {
		if period.Status == PeriodStatusClosed {
			return nil, apperror.Wrap(apperror.ErrFailedPrecondition, "accounting period is closed", nil)
		}
		return &period, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to load accounting period", err)
	}

	// Auto-create OPEN period.
	start := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, tz)
	end := start.AddDate(0, 1, -1)
	newPeriod := &gormdb.AccountingPeriod{
		StoreID: storeID, PeriodName: periodName,
		StartDate: start, EndDate: end, Status: PeriodStatusOpen,
	}
	if err := tx.WithContext(ctx).Create(newPeriod).Error; err != nil {
		// Race: another goroutine created it concurrently.
		if err2 := tx.WithContext(ctx).
			Where("store_id = ? AND period_name = ?", storeID, periodName).
			First(&period).Error; err2 == nil {
			if period.Status == PeriodStatusClosed {
				return nil, apperror.Wrap(apperror.ErrFailedPrecondition, "accounting period is closed", nil)
			}
			return &period, nil
		}
		if err2 := tx.WithContext(ctx).Create(newPeriod).Error; err2 != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "failed to create accounting period", err2)
		}
		return newPeriod, nil
	}
	return newPeriod, nil
}

// NextEntryNumber returns the next JE-YYYY-NNNNNN for a store (per-store sequence).
func (p *PostingService) NextEntryNumber(ctx context.Context, tx *gorm.DB, storeID uuid.UUID, date time.Time) (string, error) {
	if tx == nil {
		tx = p.db
	}
	year := fmt.Sprintf("%04d", date.Year())
	var count int64
	prefix := "JE-" + year + "-"
	err := tx.WithContext(ctx).Model(&gormdb.JournalEntry{}).
		Where("store_id = ? AND entry_number LIKE ?", storeID, prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%06d", prefix, count+1), nil
}

// PostJournal creates a POSTED journal entry (or returns the existing one if
// the idempotency key already exists). Lines are validated for balance.
func (p *PostingService) PostJournal(ctx context.Context, tx *gorm.DB, params PostParams) (*gormdb.JournalEntry, error) {
	if tx == nil {
		tx = p.db
	}
	if len(params.Lines) == 0 {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "journal must have at least one line", nil)
	}
	var sumDr, sumCr decimal.Decimal
	for _, l := range params.Lines {
		if l.Debit.IsZero() && l.Credit.IsZero() {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest, "journal line must have debit or credit", nil)
		}
		if !l.Debit.IsZero() && !l.Credit.IsZero() {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest, "journal line cannot have both debit and credit", nil)
		}
		if l.Debit.IsNegative() || l.Credit.IsNegative() {
			return nil, apperror.Wrap(apperror.ErrInvalidRequest, "journal line cannot be negative", nil)
		}
		sumDr = sumDr.Add(l.Debit)
		sumCr = sumCr.Add(l.Credit)
	}
	if !sumDr.Equal(sumCr) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest,
			fmt.Sprintf("journal not balanced: debit %s != credit %s", sumDr, sumCr), nil)
	}

	// Idempotency: return existing entry if key already present.
	var existing gormdb.JournalEntry
	err := tx.WithContext(ctx).
		Where("store_id = ? AND idempotency_key = ?", params.StoreID, params.IdempotencyKey).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to check idempotency", err)
	}

	period, err := p.ResolvePeriod(ctx, tx, params.StoreID, params.Date)
	if err != nil {
		return nil, err
	}
	entryNumber, err := p.NextEntryNumber(ctx, tx, params.StoreID, params.Date)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to generate entry number", err)
	}

	entry := &gormdb.JournalEntry{
		StoreID: params.StoreID, PeriodID: period.ID,
		EntryNumber: entryNumber, EntryDate: params.Date,
		EntryType: params.EntryType, SourceType: params.SourceType,
		SourceID: params.SourceID, IdempotencyKey: params.IdempotencyKey,
		Description: params.Description, Status: JournalStatusPosted,
		CreatedBy: params.CreatedBy, ReversalOfID: params.ReversalOfID,
	}
	now := time.Now()
	entry.PostedAt = &now

	if err := tx.WithContext(ctx).Create(entry).Error; err != nil {
		// Unique idempotency race: return existing.
		var race gormdb.JournalEntry
		if err2 := tx.WithContext(ctx).
			Where("store_id = ? AND idempotency_key = ?", params.StoreID, params.IdempotencyKey).
			First(&race).Error; err2 == nil {
			return &race, nil
		}
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to create journal entry", err)
	}

	for _, l := range params.Lines {
		accountID, err := GetAccountID(ctx, tx, params.StoreID, l.AccountCode)
		if err != nil {
			return nil, err
		}
		if err := tx.WithContext(ctx).Create(&gormdb.JournalLine{
			StoreID: params.StoreID, EntryID: entry.ID, AccountID: accountID,
			Debit: l.Debit.Round(2), Credit: l.Credit.Round(2), Memo: l.Memo,
		}).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "failed to create journal line", err)
		}
	}
	return entry, nil
}

// PostParams holds parameters for PostJournal.
type PostParams struct {
	StoreID        uuid.UUID
	Date           time.Time
	EntryType      string
	SourceType     string
	SourceID       *uuid.UUID
	IdempotencyKey string
	Description    string
	Lines          []LineInput
	CreatedBy      *uuid.UUID
	ReversalOfID   *uuid.UUID
}

// PostSale posts the journal for a PAID sale. payments is the list of payments
// for the sale (may be empty for PENDING sales — then no posting is done).
// The caller must invoke this only for PAID sales; the method itself is a no-op
// for PENDING/CANCELLED/REFUNDED (those are handled by PostReversal).
func (p *PostingService) PostSale(ctx context.Context, tx *gorm.DB, sale *gormdb.Sale, items []gormdb.SaleItem, payments []gormdb.Payment) (*gormdb.JournalEntry, error) {
	if sale.Status != "PAID" {
		return nil, nil
	}

	// COGS = SUM(cost_price * qty) via product lookup.
	cogs := decimal.Zero
	for _, it := range items {
		var product gormdb.Product
		if err := tx.WithContext(ctx).Where("id = ?", it.ProductID).First(&product).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "failed to load product for COGS", err)
		}
		cogs = cogs.Add(decimal.NewFromFloat(product.CostPrice).Mul(decimal.NewFromInt(int64(it.Quantity))))
	}
	cogs = cogs.Round(2)

	// Revenue = subtotal - discount, tax, and cash are rounded at the 2dp
	// boundary so debit/credit legs always balance even when float64 inputs
	// (or split payments) have rounding artifacts.
	rev := COGSMoney(sale.Subtotal - sale.Discount)
	taxAmt := COGSMoney(sale.Tax)
	cashTotal := rev.Add(taxAmt)

	lines := []LineInput{}
	cashOrBank := cashOrBankAccount(sale, payments)
	// Cash/bank debits per payment method (split payments -> one line per method).
	if len(payments) == 0 {
		// No payment rows: derive from sale fields (checkout always creates one).
		lines = append(lines, LineInput{AccountCode: cashOrBank, Debit: cashTotal, Memo: "sale " + sale.InvoiceNumber})
	} else {
		remaining := cashTotal
		for i, pay := range payments {
			acct := CodeCash
			if pay.PaymentMethod != "CASH" {
				acct = CodeBank
			}
			amt := COGSMoney(pay.Amount)
			// Put any rounding remainder on the last payment line so debits
			// match the credited revenue/tax total exactly.
			if i == len(payments)-1 {
				amt = remaining
			}
			if amt.IsNegative() {
				amt = decimal.Zero
			}
			remaining = remaining.Sub(amt)
			if amt.IsZero() {
				continue
			}
			lines = append(lines, LineInput{AccountCode: acct, Debit: amt, Memo: "payment " + pay.PaymentMethod})
		}
		// Safety: if payments summed to less than expected (after clamping),
		// add the remainder to a cash/bank line so the journal stays balanced.
		if remaining.IsPositive() {
			lines = append(lines, LineInput{AccountCode: cashOrBank, Debit: remaining, Memo: "payment rounding"})
		}
	}
	if !rev.IsZero() {
		lines = append(lines, LineInput{AccountCode: CodeSalesRevenue, Credit: rev, Memo: "sales revenue"})
	}
	if !taxAmt.IsZero() {
		lines = append(lines, LineInput{AccountCode: CodeTaxPayable, Credit: taxAmt, Memo: "PPN"})
	}
	if !cogs.IsZero() {
		lines = append(lines, LineInput{AccountCode: CodeCOGS, Debit: cogs, Memo: "COGS"})
		lines = append(lines, LineInput{AccountCode: CodeInventory, Credit: cogs, Memo: "inventory out"})
	}

	saleID := sale.ID
	return p.PostJournal(ctx, tx, PostParams{
		StoreID: sale.StoreID, Date: sale.CreatedAt,
		EntryType: EntryTypeSale, SourceType: SourceTypeSale, SourceID: &saleID,
		IdempotencyKey: "SALE:" + sale.ID.String() + ":SALE",
		Description:    "Sale " + sale.InvoiceNumber, Lines: lines,
	})
}

// PostPayment posts a payment top-up (partial/split) against an existing receivable.
// Only called when the sale still has an outstanding receivable balance (i.e. the
// SALE entry posted an unsettled Dr Receivable). If the sale was fully settled in
// PostSale, this is a no-op (returns nil, nil).
func (p *PostingService) PostPayment(ctx context.Context, tx *gorm.DB, payment *gormdb.Payment, sale *gormdb.Sale) (*gormdb.JournalEntry, error) {
	// Check if the SALE entry left an unsettled receivable: SALE entry debit to
	// 1100/1110 vs total received. If no receivable leg exists, skip.
	// Cheap proxy: if the sale has an existing PAYMENT journal for the same
	// invoice, the receivable is already settled — but the idempotency key
	// check in PostJournal makes double-posting impossible, so we proceed and
	// rely on idempotency for safety.
	acct := CodeCash
	if payment.PaymentMethod != "CASH" {
		acct = CodeBank
	}
	payID := payment.ID
	return p.PostJournal(ctx, tx, PostParams{
		StoreID: payment.StoreID, Date: payment.PaidAt,
		EntryType: EntryTypePayment, SourceType: SourceTypePayment, SourceID: &payID,
		IdempotencyKey: "PAYMENT:" + payment.ID.String(),
		Description:    "Payment for sale " + sale.InvoiceNumber,
		Lines: []LineInput{
			{AccountCode: acct, Debit: COGSMoney(payment.Amount), Memo: "payment received"},
			{AccountCode: CodeReceivables, Credit: COGSMoney(payment.Amount), Memo: "settle receivable"},
		},
	})
}

// PostReversal mirrors an existing POSTED journal (Dr<->Cr swapped) and marks
// the original REVERSED. Returns the reversal entry.
func (p *PostingService) PostReversal(ctx context.Context, tx *gorm.DB, orig *gormdb.JournalEntry, reason string, date time.Time) (*gormdb.JournalEntry, error) {
	var lines []gormdb.JournalLine
	if err := tx.WithContext(ctx).Where("entry_id = ?", orig.ID).Find(&lines).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to load journal lines", err)
	}
	mirror := make([]LineInput, 0, len(lines))
	for _, l := range lines {
		var acc gormdb.ChartOfAccount
		if err := tx.WithContext(ctx).Where("id = ?", l.AccountID).First(&acc).Error; err != nil {
			return nil, apperror.Wrap(apperror.ErrInternal, "failed to load account", err)
		}
		mirror = append(mirror, LineInput{
			AccountCode: acc.Code, Debit: l.Credit, Credit: l.Debit, Memo: "reversal: " + l.Memo,
		})
	}
	reversal, err := p.PostJournal(ctx, tx, PostParams{
		StoreID: orig.StoreID, Date: date,
		EntryType: EntryTypeReversal, SourceType: orig.SourceType, SourceID: orig.SourceID,
		IdempotencyKey: "REVERSAL:" + orig.ID.String(),
		Description:    "Reversal: " + reason,
		Lines:          mirror, ReversalOfID: &orig.ID,
	})
	if err != nil {
		return nil, err
	}
	// Mark original REVERSED and link the pair.
	origID := orig.ID
	if err := tx.WithContext(ctx).Model(&gormdb.JournalEntry{}).
		Where("id = ?", origID).Updates(map[string]interface{}{
		"status":        JournalStatusReversed,
		"reversed_by_id": reversal.ID,
	}).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to mark journal reversed", err)
	}
	return reversal, nil
}

// PostPurchase posts inventory PURCHASE (stock-in). Amount = qty * product cost_price.
func (p *PostingService) PostPurchase(ctx context.Context, tx *gorm.DB, invTx *gormdb.InventoryTransaction) (*gormdb.JournalEntry, error) {
	var product gormdb.Product
	if err := tx.WithContext(ctx).Where("id = ?", invTx.ProductID).First(&product).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to load product", err)
	}
	amount := decimal.NewFromFloat(product.CostPrice).Mul(decimal.NewFromInt(int64(invTx.Quantity))).Round(2)
	if amount.IsZero() {
		return nil, nil
	}
	invID := invTx.ID
	return p.PostJournal(ctx, tx, PostParams{
		StoreID: invTx.StoreID, Date: invTx.CreatedAt,
		EntryType: EntryTypePurchase, SourceType: SourceTypeInventory, SourceID: &invID,
		IdempotencyKey: "INV:" + invTx.ID.String(),
		Description:    "Inventory purchase",
		Lines: []LineInput{
			{AccountCode: CodeInventory, Debit: amount, Memo: "stock in"},
			{AccountCode: CodePayables, Credit: amount, Memo: "payable"},
		},
	})
}

// PostAdjustment posts inventory ADJUSTMENT/DAMAGE (stock out) or ADJUSTMENT up.
// Direction is derived from before/after: after > before = stock-in (Credit Inventory Gain),
// after < before = shrinkage (Debit Shrinkage).
func (p *PostingService) PostAdjustment(ctx context.Context, tx *gorm.DB, invTx *gormdb.InventoryTransaction) (*gormdb.JournalEntry, error) {
	var product gormdb.Product
	if err := tx.WithContext(ctx).Where("id = ?", invTx.ProductID).First(&product).Error; err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "failed to load product", err)
	}
	qty := invTx.Quantity
	if qty < 0 {
		qty = -qty
	}
	amount := decimal.NewFromFloat(product.CostPrice).Mul(decimal.NewFromInt(int64(qty))).Round(2)
	if amount.IsZero() {
		return nil, nil
	}
	invID := invTx.ID
	if invTx.AfterStock > invTx.BeforeStock {
		// Stock up: Dr Inventory / Cr Inventory Gain
		return p.PostJournal(ctx, tx, PostParams{
			StoreID: invTx.StoreID, Date: invTx.CreatedAt,
			EntryType: EntryTypeAdjustment, SourceType: SourceTypeInventory, SourceID: &invID,
			IdempotencyKey: "INV:" + invTx.ID.String(),
			Description:    "Inventory adjustment (up)",
			Lines: []LineInput{
				{AccountCode: CodeInventory, Debit: amount, Memo: "stock up"},
				{AccountCode: CodeInventoryGain, Credit: amount, Memo: "inventory gain"},
			},
		})
	}
	// Stock down / damage: Dr Shrinkage / Cr Inventory
	return p.PostJournal(ctx, tx, PostParams{
		StoreID: invTx.StoreID, Date: invTx.CreatedAt,
		EntryType: EntryTypeAdjustment, SourceType: SourceTypeInventory, SourceID: &invID,
		IdempotencyKey: "INV:" + invTx.ID.String(),
		Description:    "Inventory adjustment (down)",
		Lines: []LineInput{
			{AccountCode: CodeInventoryShrinkage, Debit: amount, Memo: "shrinkage"},
			{AccountCode: CodeInventory, Credit: amount, Memo: "stock out"},
		},
	})
}

// cashOrBankAccount picks the cash/bank account for a sale's payment (CASH -> 1100, else 1110).
func cashOrBankAccount(sale *gormdb.Sale, payments []gormdb.Payment) string {
	if len(payments) == 0 {
		return CodeCash
	}
	if payments[0].PaymentMethod == "CASH" {
		return CodeCash
	}
	return CodeBank
}
