package selforder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"apiPos/internal/apperror"
)

func TestValidatePaymentMethod(t *testing.T) {
	t.Run("cash rejected", func(t *testing.T) {
		err := validatePaymentMethod("CASH")
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, apperror.MapError(err).GRPCStatus().Code())
	})

	t.Run("empty rejected", func(t *testing.T) {
		err := validatePaymentMethod("")
		assert.Error(t, err)
	})

	t.Run("unknown rejected", func(t *testing.T) {
		err := validatePaymentMethod("BITCOIN")
		assert.Error(t, err)
	})

	t.Run("non-cash accepted", func(t *testing.T) {
		for _, m := range []string{"CARD", "BANK_TRANSFER", "EWALLET", "OTHER"} {
			assert.NoError(t, validatePaymentMethod(m), "method %s must be accepted", m)
		}
	})
}

func TestComputeOrderTotals(t *testing.T) {
	t.Run("11 percent tax", func(t *testing.T) {
		tax, grand := computeOrderTotals(65000, 11)
		assert.Equal(t, 7150.0, tax)
		assert.Equal(t, 72150.0, grand)
	})

	t.Run("zero tax", func(t *testing.T) {
		tax, grand := computeOrderTotals(10000, 0)
		assert.Equal(t, 0.0, tax)
		assert.Equal(t, 10000.0, grand)
	})

	t.Run("negative tax clamped", func(t *testing.T) {
		tax, grand := computeOrderTotals(10000, -5)
		assert.Equal(t, 0.0, tax)
		assert.Equal(t, 10000.0, grand)
	})
}

func TestGenerateInvoiceNumber(t *testing.T) {
	a := generateInvoiceNumber()
	b := generateInvoiceNumber()
	assert.NotEmpty(t, a)
	assert.NotEqual(t, a, b)
	assert.Contains(t, a, "INV-")
}

func TestPaymentTimeoutConstant(t *testing.T) {
	// Guard: the timeout must stay positive and sane (docs default 15 minutes).
	assert.Greater(t, SelfOrderPaymentTimeout, int64(0))
}