package domain

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable machine-readable error code shared by the service
// layer, Wails DTOs and the frontend.
type ErrorCode string

const (
	CodeInvalidInput      ErrorCode = "invalid_input"
	CodeInvalidMonth      ErrorCode = "invalid_month"
	CodeInvalidDate       ErrorCode = "invalid_date"
	CodeDateOutsideMonth  ErrorCode = "date_outside_month"
	CodeInvalidQuantity   ErrorCode = "invalid_quantity"
	CodeNotInitialized    ErrorCode = "inventory_not_initialized"
	CodeMonthClosed       ErrorCode = "month_closed"
	CodeInsufficientStock ErrorCode = "insufficient_stock"
	CodeReasonRequired    ErrorCode = "reason_required"
	CodeArchived          ErrorCode = "archived"
	CodeNotFound          ErrorCode = "not_found"
	CodeConflict          ErrorCode = "conflict"
	CodeUnsafeDelete      ErrorCode = "unsafe_delete"
	CodeNotClosed         ErrorCode = "not_closed"
	CodeAlreadyClosed     ErrorCode = "already_closed"
	CodeForbidden         ErrorCode = "forbidden"
)

// Error is a user-facing, structured validation or business-rule error.
type Error struct {
	Code    ErrorCode
	Field   string
	Message string
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// NewError builds a structured domain error.
func NewError(code ErrorCode, field, message string) *Error {
	return &Error{Code: code, Field: field, Message: message}
}

// IsCode reports whether err is or wraps a domain error with the given code.
func IsCode(err error, code ErrorCode) bool {
	var de *Error
	if errors.As(err, &de) {
		return de.Code == code
	}
	return false
}

// AsError returns the structured domain error carried by err, if any.
func AsError(err error) (*Error, bool) {
	var de *Error
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

var (
	ErrInvalidMonth = NewError(CodeInvalidMonth, "month", "اختر شهراً صالحاً بصيغة YYYY-MM.")
)

// entityLabels maps internal entity names to Arabic user-facing labels.
var entityLabels = map[string]string{
	"Vaccine":           "اللقاح",
	"Patient":           "المريض",
	"Usage entry":       "قيد الاستخدام",
	"Monthly inventory": "المخزون الشهري",
}

// NotFound builds a localized not-found error for an internal entity name.
func NotFound(entity string) *Error {
	label, ok := entityLabels[entity]
	if !ok || label == "" {
		label = entity
	}
	return NewError(CodeNotFound, "", "لم يتم العثور على "+label+".")
}

func notFound(entity string) *Error {
	return NotFound(entity)
}
