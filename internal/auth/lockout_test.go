package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"apiPos/internal/apperror"
)

func testLockout(now *time.Time) *LoginLockout {
	return newLoginLockout(func() time.Time { return *now }, 3, 10*time.Minute, 15*time.Minute)
}

func TestLoginLockoutLocksAfterThreshold(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	lockout := testLockout(&now)

	for i := 0; i < 3; i++ {
		require.NoError(t, lockout.Check("admin"), "attempt %d must not be locked yet", i)
		lockout.RecordFailure("admin")
	}

	err := lockout.Check("admin")
	require.ErrorIs(t, err, apperror.ErrTooManyAttempts)
	assert.Equal(t, codes.ResourceExhausted, apperror.MapError(err).Code)
}

func TestLoginLockoutExpires(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	lockout := testLockout(&now)

	for i := 0; i < 3; i++ {
		lockout.RecordFailure("admin")
	}
	require.ErrorIs(t, lockout.Check("admin"), apperror.ErrTooManyAttempts)

	now = now.Add(15*time.Minute + time.Second)
	assert.NoError(t, lockout.Check("admin"))
}

func TestLoginLockoutResetsOnSuccess(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	lockout := testLockout(&now)

	lockout.RecordFailure("admin")
	lockout.RecordFailure("admin")
	lockout.Reset("admin")

	lockout.RecordFailure("admin")
	lockout.RecordFailure("admin")
	assert.NoError(t, lockout.Check("admin"), "counter must restart after a successful login")
}

func TestLoginLockoutIgnoresFailuresOutsideWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	lockout := testLockout(&now)

	lockout.RecordFailure("admin")
	lockout.RecordFailure("admin")

	now = now.Add(11 * time.Minute)
	lockout.RecordFailure("admin")
	assert.NoError(t, lockout.Check("admin"), "stale failures must not count toward the threshold")
}

func TestLoginLockoutNormalisesUsername(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	lockout := testLockout(&now)

	lockout.RecordFailure(" Admin ")
	lockout.RecordFailure("ADMIN")
	lockout.RecordFailure("admin")

	assert.ErrorIs(t, lockout.Check("  admin"), apperror.ErrTooManyAttempts)
	assert.NoError(t, lockout.Check("cashier"), "other usernames are unaffected")
}

func TestLoginLockoutBoundsEntries(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	lockout := testLockout(&now)

	for i := 0; i < maxLockoutEntries; i++ {
		lockout.RecordFailure(string(rune('a'+i%26)) + "-user-" + itoa(i))
	}

	// Move past the window so the next failure triggers pruning of stale entries.
	now = now.Add(time.Hour)
	lockout.RecordFailure("late-user")

	assert.Less(t, len(lockout.entries), maxLockoutEntries, "stale entries must be pruned")
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := make([]byte, 0, 8)
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	return string(digits)
}
