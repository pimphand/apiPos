package timeutil

import (
	"strings"
	"time"
)

// Layouts accepted when parsing flexible user-supplied timestamps/date strings.
// Ordered from most specific to least specific so that precedence is deterministic.
var flexibleLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
	"02-01-2006",
}

// ParseFlexible parses a time string that may be a date-only value (e.g.
// "2026-08-30") or a full timestamp (RFC3339, "2006-01-02 15:04:05", etc.).
// It returns (time, nil) on success, or (time.Time{}, error) when the value
// cannot be parsed by any supported layout.
func ParseFlexible(s string) (time.Time, error) {
	v := strings.TrimSpace(s)
	if v == "" {
		return time.Time{}, nil
	}

	for _, layout := range flexibleLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &ErrInvalidTime{Value: s}
}

// ErrInvalidTime is returned when a flexible time string cannot be parsed.
type ErrInvalidTime struct {
	Value string
}

func (e *ErrInvalidTime) Error() string {
	return "invalid time value: " + e.Value
}
