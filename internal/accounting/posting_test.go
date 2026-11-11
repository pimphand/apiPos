package accounting

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCOGSMoney_rounding verifies the POS float64 -> decimal boundary rounds
// to 2dp, preventing float artifacts from entering the official books.
func TestCOGSMoney_rounding(t *testing.T) {
	assert.True(t, COGSMoney(10000.005).Equal(decimal.NewFromFloat(10000.01).Round(2)) || COGSMoney(10000.005).Equal(decimal.NewFromFloat(10000.00).Round(2)))
	assert.Equal(t, "12345.68", COGSMoney(12345.678).StringFixed(2))
	assert.Equal(t, "0.30", COGSMoney(0.1+0.2).StringFixed(2))
}

// TestDefaultCOA_codes verifies every code the posting map depends on exists
// in the default COA, with the correct account type / normal balance.
func TestDefaultCOA_codes(t *testing.T) {
	byCode := map[string]DefaultAccount{}
	for _, a := range DefaultCOA {
		byCode[a.Code] = a
	}
	for _, code := range []string{CodeCash, CodeBank, CodeInputVAT, CodeReceivables, CodeInventory,
		CodePayables, CodeTaxPayable, CodeOwnerCapital, CodeRetainedEarnings,
		CodeSalesRevenue, CodeInventoryGain, CodeCOGS, CodeInventoryShrinkage, CodeOperatingExpenses} {
		_, ok := byCode[code]
		require.True(t, ok, "missing COA code %s", code)
	}
	// Spot-check types.
	assert.Equal(t, AccountTypeAsset, byCode[CodeCash].AccountType)
	assert.Equal(t, NormalBalanceDebit, byCode[CodeCash].NormalBalance)
	assert.True(t, byCode[CodeCash].IsCashEquiv)
	assert.Equal(t, AccountTypeLiability, byCode[CodeTaxPayable].AccountType)
	assert.Equal(t, AccountTypeRevenue, byCode[CodeSalesRevenue].AccountType)
	assert.Equal(t, AccountTypeExpense, byCode[CodeCOGS].AccountType)
}

// TestPostJournal_balanceValidation verifies unbalanced journals are rejected
// before any DB write (pure Go logic).
func TestPostJournal_balanceValidation(t *testing.T) {
	// A balanced pair passes the balance check stage but requires a DB to
	// proceed; we only test the two paths that reject before touching the DB.
	svc := NewPostingService(nil)

	// Unbalanced lines are rejected immediately.
	_, err := svc.PostJournal(context.Background(), nil, PostParams{
		StoreID: uuid.New(), Lines: []LineInput{
			{AccountCode: CodeCash, Debit: decimal.NewFromInt(100)},
			{AccountCode: CodeSalesRevenue, Credit: decimal.NewFromInt(99)},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not balanced")
}

// TestPostJournal_lineValidation verifies empty and multi-sided lines are rejected.
func TestPostJournal_lineValidation(t *testing.T) {
	svc := NewPostingService(nil)
	_, err := svc.PostJournal(context.Background(), nil, PostParams{
		StoreID: uuid.New(),
		Lines:   []LineInput{},
	})
	require.Error(t, err)

	_, err = svc.PostJournal(context.Background(), nil, PostParams{
		StoreID: uuid.New(), Lines: []LineInput{
			{AccountCode: CodeCash, Debit: decimal.NewFromInt(100), Credit: decimal.NewFromInt(100)},
		},
	})
	require.Error(t, err)
}
