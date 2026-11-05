package domain

import (
	"strings"
	"unicode/utf8"
)

// InventoryStatus is the state of one vaccine/month inventory record.
type InventoryStatus string

const (
	StatusOpen    InventoryStatus = "open"
	StatusClosed  InventoryStatus = "closed"
	StatusMissing InventoryStatus = "not_initialized"
)

// MonthState describes a month across all initialized vaccine records.
type MonthState string

const (
	MonthNotInitialized MonthState = "not_initialized"
	MonthOpen           MonthState = "open"
	MonthClosed         MonthState = "closed"
)

// RequiredText trims and validates a required single-line text field.
func RequiredText(value, field, label string, maxLen int) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", NewError(CodeInvalidInput, field, label+" مطلوب.")
	}
	if utf8.RuneCountInString(v) > maxLen {
		return "", NewError(CodeInvalidInput, field, label+" طويل جداً.")
	}
	return v, nil
}

// OptionalText trims and validates an optional single-line text field.
func OptionalText(value, field, label string, maxLen int) (string, error) {
	v := strings.TrimSpace(value)
	if utf8.RuneCountInString(v) > maxLen {
		return "", NewError(CodeInvalidInput, field, label+" طويل جداً.")
	}
	return v, nil
}

// RequireReason validates a reason attached to an audited correction.
func RequireReason(value, field, label string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", NewError(CodeReasonRequired, field, label+" مطلوب لهذا التغيير.")
	}
	if utf8.RuneCountInString(v) > 500 {
		return "", NewError(CodeInvalidInput, field, label+" طويل جداً.")
	}
	return v, nil
}

// RequireOpen rejects routine edits to closed inventory.
func RequireOpen(status InventoryStatus) error {
	if status == StatusClosed {
		return NewError(CodeMonthClosed, "", "هذا الشهر مغلق. أعد فتحه مع ذكر سبب قبل تغيير المخزون أو الاستخدام.")
	}
	return nil
}

// ValidateInventoryStatus rejects unknown persisted values.
func ValidateInventoryStatus(status string) error {
	if status != string(StatusOpen) && status != string(StatusClosed) {
		return NewError(CodeInvalidInput, "", "حالة مخزون غير معروفة.")
	}
	return nil
}
