package auth

import (
	"strings"
	"sync"
	"time"

	"apiPos/internal/apperror"
)

// Failed-login lockout defaults. A username is locked after
// thresholdFailedLogins failures within failureWindow; the lock lifts
// automatically after lockoutDuration and a successful login clears the
// counter immediately.
const (
	thresholdFailedLogins = 5
	failureWindow         = 15 * time.Minute
	lockoutDuration       = 15 * time.Minute

	// maxLockoutEntries bounds memory: entries older than failureWindow (and
	// not locked) are pruned once the map reaches this size.
	maxLockoutEntries = 10_000
)

type lockoutEntry struct {
	failures     int
	firstFailure time.Time
	lockedUntil  time.Time
}

// LoginLockout keeps per-username failed-login state in memory.
//
// It complements the per-IP rate limiter in middleware: the limiter stops a
// single host from spraying passwords, while the lockout stops slow,
// distributed guessing against one account. Like the rate limiter, the state is
// process-local — a restart clears counters and multiple backend instances
// count independently.
type LoginLockout struct {
	mu        sync.Mutex
	entries   map[string]*lockoutEntry
	threshold int
	window    time.Duration
	duration  time.Duration
	now       func() time.Time
}

// NewLoginLockout creates a lockout tracker with the production defaults.
func NewLoginLockout() *LoginLockout {
	return newLoginLockout(time.Now, thresholdFailedLogins, failureWindow, lockoutDuration)
}

func newLoginLockout(now func() time.Time, threshold int, window, duration time.Duration) *LoginLockout {
	return &LoginLockout{
		entries:   map[string]*lockoutEntry{},
		threshold: threshold,
		window:    window,
		duration:  duration,
		now:       now,
	}
}

// Check returns apperror.ErrTooManyAttempts while the username is locked out.
func (l *LoginLockout) Check(username string) error {
	key := loginKey(username)
	if key == "" {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return nil
	}
	if l.now().Before(entry.lockedUntil) {
		return apperror.ErrTooManyAttempts
	}
	return nil
}

// RecordFailure counts one failed login for the username and locks it once the
// threshold is reached inside the current window. Unknown usernames are counted
// too, so lockout behaviour cannot be used to tell existing accounts apart.
func (l *LoginLockout) RecordFailure(username string) {
	key := loginKey(username)
	if key == "" {
		return
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) >= maxLockoutEntries {
		l.pruneLocked(now)
	}

	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.firstFailure) > l.window {
		entry = &lockoutEntry{firstFailure: now}
		l.entries[key] = entry
	}

	entry.failures++
	if entry.failures >= l.threshold {
		entry.lockedUntil = now.Add(l.duration)
	}
}

// Reset clears the failure counter for the username (successful login).
func (l *LoginLockout) Reset(username string) {
	key := loginKey(username)
	if key == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// pruneLocked drops entries that are neither locked nor inside the failure
// window. The caller must hold l.mu.
func (l *LoginLockout) pruneLocked(now time.Time) {
	for key, entry := range l.entries {
		unlocked := !now.Before(entry.lockedUntil)
		if unlocked && now.Sub(entry.firstFailure) > l.window {
			delete(l.entries, key)
		}
	}
}

// loginKey normalises a username so "Admin", "admin " and "ADMIN" share one
// counter.
func loginKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
