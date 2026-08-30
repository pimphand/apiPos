package timeutil

import (
	"testing"
	"time"
)

func TestParseFlexible(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want time.Time
	}{
		{"date only", "2026-08-30", time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)},
		{"date only with space", " 2026-08-30 ", time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)},
		{"rfc3339", "2026-08-30T00:00:00Z", time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)},
		{"rfc3339 offset", "2026-08-30T15:04:05+07:00", time.Date(2026, 8, 30, 15, 4, 5, 0, time.FixedZone("", 7*3600))},
		{"space datetime", "2026-08-30 15:04:05", time.Date(2026, 8, 30, 15, 4, 5, 0, time.UTC)},
		{"dd-mm-yyyy", "30-08-2026", time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFlexible(tc.in)
			if err != nil {
				t.Fatalf("ParseFlexible(%q) unexpected error: %v", tc.in, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("ParseFlexible(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseFlexibleEmpty(t *testing.T) {
	got, err := ParseFlexible("")
	if err != nil {
		t.Fatalf("ParseFlexible(\"\") unexpected error: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("ParseFlexible(\"\") = %v, want zero time", got)
	}
}

func TestParseFlexibleInvalid(t *testing.T) {
	if _, err := ParseFlexible("not-a-date"); err == nil {
		t.Error("ParseFlexible(\"not-a-date\") expected error, got nil")
	}
}
