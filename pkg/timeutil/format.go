package timeutil

import (
	"strings"
	"time"
)

// Default layouts used when no explicit format is configured. Neither contains
// the "T"/"Z" markers of RFC3339, because API consumers render these values
// as-is (e.g. the report table prints the daily-sales date verbatim).
const (
	DefaultDateLayout     = "2006-01-02"
	DefaultDateTimeLayout = "2006-01-02 15:04:05"
)

var (
	dateLayout     = DefaultDateLayout
	dateTimeLayout = DefaultDateTimeLayout
)

// Configure sets the layouts used for date and timestamp output. Empty values
// keep their defaults, so an unset environment variable cannot blank out the
// format. Call it once at startup with the values from config.
func Configure(date, dateTime string) {
	if date != "" {
		dateLayout = date
	}
	if dateTime != "" {
		dateTimeLayout = dateTime
	}
}

// FormatDate renders t using the configured date layout.
func FormatDate(t time.Time) string { return t.Format(dateLayout) }

// FormatDateTime renders t using the configured timestamp layout.
func FormatDateTime(t time.Time) string { return t.Format(dateTimeLayout) }

// NormalizeDate turns a value that represents a single day into the configured
// date layout. Database drivers hand a DATE column back in several shapes
// ("2026-11-17T00:00:00Z" through database/sql, "2026-11-17 00:00:00" or
// "2026-11-17"), and returning them unchanged is what leaked the RFC3339 form
// into responses. Values that cannot be parsed are returned untouched.
func NormalizeDate(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	t, err := ParseFlexible(v)
	if err != nil || t.IsZero() {
		return v
	}

	return FormatDate(t)
}
