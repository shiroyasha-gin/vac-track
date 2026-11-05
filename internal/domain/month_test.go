package domain

import (
	"testing"
	"time"
)

func TestParseMonthCanonical(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		ok    bool
		year  int
		month time.Month
	}{
		{"2026-11", true, 2026, time.November},
		{"2024-02", true, 2024, time.February},
		{"2026-1", false, 0, 0},
		{"2026-13", false, 0, 0},
		{"2026-00", false, 0, 0},
		{"0000-01", false, 0, 0},
		{"", false, 0, 0},
		{"2026-01-01", false, 0, 0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			m, err := ParseMonth(tt.input)
			if tt.ok {
				if err != nil {
					t.Fatalf("ParseMonth(%q) returned error: %v", tt.input, err)
				}
				if m.Year != tt.year || m.Month != tt.month {
					t.Fatalf("ParseMonth(%q) = %+v, want %d-%v", tt.input, m, tt.year, tt.month)
				}
				if m.String() != tt.input {
					t.Fatalf("round trip = %q, want %q", m.String(), tt.input)
				}
				return
			}
			if err == nil {
				t.Fatalf("ParseMonth(%q) unexpectedly succeeded", tt.input)
			}
			if !IsCode(err, CodeInvalidMonth) {
				t.Fatalf("error code = %v, want %v", err, CodeInvalidMonth)
			}
		})
	}
}

func TestMonthDayCountsAndNavigation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		month    string
		days     int
		previous string
		next     string
	}{
		{"2024-02", 29, "2024-01", "2024-03"},
		{"2025-02", 28, "2025-01", "2025-03"},
		{"2026-04", 30, "2026-03", "2026-05"},
		{"2026-12", 31, "2026-11", "2027-01"},
		{"2026-01", 31, "2025-12", "2026-02"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.month, func(t *testing.T) {
			t.Parallel()
			m, err := ParseMonth(tt.month)
			if err != nil {
				t.Fatal(err)
			}
			if got := m.Days(); got != tt.days {
				t.Errorf("Days() = %d, want %d", got, tt.days)
			}
			if got := m.Previous().String(); got != tt.previous {
				t.Errorf("Previous() = %s, want %s", got, tt.previous)
			}
			if got := m.Next().String(); got != tt.next {
				t.Errorf("Next() = %s, want %s", got, tt.next)
			}
			if got := m.FirstDay().Day(); got != 1 {
				t.Errorf("FirstDay().Day() = %d, want 1", got)
			}
		})
	}
}

func TestMonthOfUsesUTC(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("offset", 13*60*60)
	// 2026-03-01 00:30 +13:00 is still February in UTC.
	tm := time.Date(2026, time.March, 1, 0, 30, 0, 0, loc)
	if got := MonthOf(tm).String(); got != "2026-02" {
		t.Fatalf("MonthOf() = %s, want 2026-02", got)
	}
}

func TestDateInMonth(t *testing.T) {
	t.Parallel()
	feb, _ := ParseMonth("2024-02")
	if err := DateInMonth(feb, "2024-02-29"); err != nil {
		t.Fatalf("leap day rejected: %v", err)
	}
	if err := DateInMonth(feb, "2024-02-30"); err == nil {
		t.Fatal("invalid calendar date accepted")
	}
	if err := DateInMonth(feb, "2024-03-01"); !IsCode(err, CodeDateOutsideMonth) {
		t.Fatalf("outside-month date error = %v, want %v", err, CodeDateOutsideMonth)
	}
	if err := DateInMonth(feb, "2024-2-03"); !IsCode(err, CodeInvalidDate) {
		t.Fatalf("non-canonical date error = %v, want %v", err, CodeInvalidDate)
	}
}
