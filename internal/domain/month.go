package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var monthPattern = regexp.MustCompile(`^([0-9]{4})-(0[1-9]|1[0-2])$`)

// Month is a canonical calendar month. It is represented as YYYY-MM in Go and
// SQLite. All calculations use UTC and never floating point.
type Month struct {
	Year  int
	Month time.Month
}

// ParseMonth parses a canonical YYYY-MM month key.
func ParseMonth(value string) (Month, error) {
	m := monthPattern.FindStringSubmatch(value)
	if m == nil {
		return Month{}, ErrInvalidMonth
	}
	year, err := strconv.Atoi(m[1])
	if err != nil || year < 1 {
		return Month{}, ErrInvalidMonth
	}
	month, _ := strconv.Atoi(m[2])
	return Month{Year: year, Month: time.Month(month)}, nil
}

// NewMonth constructs a month from a year and month number.
func NewMonth(year int, month time.Month) (Month, error) {
	if year < 1 || year > 9999 || month < time.January || month > time.December {
		return Month{}, ErrInvalidMonth
	}
	return Month{Year: year, Month: month}, nil
}

// MonthOf returns the calendar month containing t, evaluated in UTC.
func MonthOf(t time.Time) Month {
	u := t.UTC()
	return Month{Year: u.Year(), Month: u.Month()}
}

// String returns the canonical YYYY-MM representation.
func (m Month) String() string {
	if m.Year == 0 || m.Month < time.January || m.Month > time.December {
		return ""
	}
	return fmt.Sprintf("%04d-%02d", m.Year, int(m.Month))
}

// FirstDay returns the first instant of the month in UTC.
func (m Month) FirstDay() time.Time {
	return time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, time.UTC)
}

// Days returns the number of days in the month (28-31, leap years included).
func (m Month) Days() int {
	return time.Date(m.Year, m.Month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// Previous returns the calendar month immediately before m.
func (m Month) Previous() Month {
	t := m.FirstDay().AddDate(0, -1, 0)
	return Month{Year: t.Year(), Month: t.Month()}
}

// Next returns the calendar month immediately after m.
func (m Month) Next() Month {
	t := m.FirstDay().AddDate(0, 1, 0)
	return Month{Year: t.Year(), Month: t.Month()}
}

// IsZero reports whether m has not been initialized.
func (m Month) IsZero() bool {
	return m.Year == 0 || m.Month < time.January || m.Month > time.December
}

// Compare orders m against other: -1, 0 or 1.
func (m Month) Compare(other Month) int {
	if m.Year != other.Year {
		if m.Year < other.Year {
			return -1
		}
		return 1
	}
	if m.Month < other.Month {
		return -1
	}
	if m.Month > other.Month {
		return 1
	}
	return 0
}

// ContainsDate reports whether the canonical YYYY-MM-DD date belongs to m.
func (m Month) ContainsDate(date string) bool {
	d, err := ParseDate(date)
	if err != nil {
		return false
	}
	return d.Year() == m.Year && d.Month() == m.Month
}

// ParseDate parses a canonical YYYY-MM-DD calendar date in UTC.
func ParseDate(value string) (time.Time, error) {
	if len(value) != 10 {
		return time.Time{}, NewError(CodeInvalidDate, "date", "استخدم تاريخاً صالحاً بصيغة YYYY-MM-DD.")
	}
	t, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil || t.Format("2006-01-02") != value {
		return time.Time{}, NewError(CodeInvalidDate, "date", "استخدم تاريخاً صالحاً بصيغة YYYY-MM-DD.")
	}
	return t, nil
}

// FormatDate formats a time as a canonical UTC calendar date.
func FormatDate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// DateInMonth validates that date is canonical and belongs to m.
func DateInMonth(m Month, date string) error {
	t, err := ParseDate(date)
	if err != nil {
		return err
	}
	if t.Year() != m.Year || t.Month() != m.Month {
		return NewError(CodeDateOutsideMonth, "date", "يجب أن يقع تاريخ الإعطاء داخل شهر المخزون المحدد.")
	}
	return nil
}
