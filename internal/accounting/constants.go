package accounting

import (
	"github.com/shopspring/decimal"
)

// Account codes. Never hardcode raw strings elsewhere — use these constants.
const (
	CodeCash              = "1100"
	CodeBank              = "1110"
	CodeInputVAT          = "1120"
	CodeReceivables       = "1130"
	CodeInventory         = "1200"
	CodePayables          = "2100"
	CodeTaxPayable        = "2110"
	CodeOwnerCapital      = "3100"
	CodeRetainedEarnings  = "3110"
	CodeSalesRevenue      = "4100"
	CodeInventoryGain     = "4200"
	CodeCOGS              = "5100"
	CodeInventoryShrinkage = "5200"
	CodeOperatingExpenses = "6100"
)

// AccountType / NormalBalance constants.
const (
	AccountTypeAsset      = "ASSET"
	AccountTypeLiability  = "LIABILITY"
	AccountTypeEquity     = "EQUITY"
	AccountTypeRevenue    = "REVENUE"
	AccountTypeExpense    = "EXPENSE"

	NormalBalanceDebit  = "DEBIT"
	NormalBalanceCredit = "CREDIT"
)

// Journal status / entry type constants.
const (
	JournalStatusDraft    = "DRAFT"
	JournalStatusPosted   = "POSTED"
	JournalStatusReversed = "REVERSED"

	EntryTypeOpening    = "OPENING"
	EntryTypeSale       = "SALE"
	EntryTypePayment    = "PAYMENT"
	EntryTypePurchase   = "PURCHASE"
	EntryTypeAdjustment = "ADJUSTMENT"
	EntryTypeReversal   = "REVERSAL"
	EntryTypeManual     = "MANUAL"
)

// Period status.
const (
	PeriodStatusOpen   = "OPEN"
	PeriodStatusClosed = "CLOSED"
)

// Source types.
const (
	SourceTypeSale      = "SALE"
	SourceTypePayment   = "PAYMENT"
	SourceTypeInventory = "INVENTORY_TX"
	SourceTypeManual    = "MANUAL"
)

// DefaultCOA is the per-store chart of accounts seeded on store registration
// and backfilled for existing stores.
var DefaultCOA = []DefaultAccount{
	{Code: CodeCash, Name: "Cash", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, IsCashEquiv: true},
	{Code: CodeBank, Name: "Bank", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, IsCashEquiv: true},
	{Code: CodeInputVAT, Name: "Input VAT (recoverable)", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit},
	{Code: CodeReceivables, Name: "Accounts Receivable", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit},
	{Code: CodeInventory, Name: "Inventory", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit},
	{Code: CodePayables, Name: "Accounts Payable", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit},
	{Code: CodeTaxPayable, Name: "Tax Payable (PPN)", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit},
	{Code: CodeOwnerCapital, Name: "Owner Capital", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceCredit},
	{Code: CodeRetainedEarnings, Name: "Retained Earnings", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceCredit},
	{Code: CodeSalesRevenue, Name: "Sales Revenue", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit},
	{Code: CodeInventoryGain, Name: "Inventory Gain", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit},
	{Code: CodeCOGS, Name: "Cost of Goods Sold", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit},
	{Code: CodeInventoryShrinkage, Name: "Inventory Shrinkage", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit},
	{Code: CodeOperatingExpenses, Name: "Operating Expenses", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit},
}

// DefaultAccount is one entry of the default chart of accounts.
type DefaultAccount struct {
	Code          string
	Name          string
	AccountType   string
	NormalBalance string
	IsCashEquiv   bool
}

// LineInput is one journal line to post.
type LineInput struct {
	AccountCode string
	Debit       decimal.Decimal
	Credit      decimal.Decimal
	Memo        string
}
