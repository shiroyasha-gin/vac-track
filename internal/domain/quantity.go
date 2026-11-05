package domain

import (
	"math"
	"strconv"
	"strings"
)

// Quantity is a whole number of vaccine doses. The application deliberately
// does not use floating point for inventory.
type Quantity int64

// NewDoses validates a positive dose quantity.
func NewDoses(n int64) (Quantity, error) {
	if n <= 0 {
		return 0, NewError(CodeInvalidQuantity, "quantity", "يجب أن تكون كمية الجرعات عدداً صحيحاً أكبر من صفر.")
	}
	return Quantity(n), nil
}

// NewNonNegativeDoses validates a zero-or-greater dose quantity.
func NewNonNegativeDoses(n int64) (Quantity, error) {
	if n < 0 {
		return 0, NewError(CodeInvalidQuantity, "quantity", "لا يمكن أن تكون كمية الجرعات سالبة.")
	}
	return Quantity(n), nil
}

// ParseDoses parses a whole-number dose quantity from user input.
func ParseDoses(value string) (Quantity, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, NewError(CodeInvalidQuantity, "quantity", "يجب أن تكون كمية الجرعات عدداً صحيحاً.")
	}
	return NewDoses(n)
}

// Int64 returns the quantity as an int64 for storage.
func (q Quantity) Int64() int64 {
	return int64(q)
}

// CheckedAdd adds non-negative quantities, rejecting overflow.
func CheckedAdd(a, b int64) (int64, error) {
	if a < 0 || b < 0 {
		return 0, NewError(CodeInvalidQuantity, "", "لا يمكن أن تكون الكميات سالبة.")
	}
	if a > math.MaxInt64-b {
		return 0, NewError(CodeInvalidQuantity, "", "الكمية كبيرة جداً.")
	}
	return a + b, nil
}

// ComputeBalance applies opening + received - active usage. A negative result
// is rejected; usage must never make stock negative through the normal path.
func ComputeBalance(opening, received, usage int64) (int64, error) {
	if opening < 0 {
		return 0, NewError(CodeInvalidQuantity, "opening", "لا يمكن أن يكون الرصيد الافتتاحي سالباً.")
	}
	if received < 0 {
		return 0, NewError(CodeInvalidQuantity, "stock", "لا يمكن أن يكون المخزون الوارد سالباً.")
	}
	if usage < 0 {
		return 0, NewError(CodeInvalidQuantity, "usage", "لا يمكن أن يكون الاستخدام سالباً.")
	}
	available, err := CheckedAdd(opening, received)
	if err != nil {
		return 0, err
	}
	if usage > available {
		return 0, NewError(CodeInsufficientStock, "quantity", "ستتجاوز هذه الكمية الجرعات المتاحة.")
	}
	return available - usage, nil
}

// EnsureAvailable validates a requested positive dose count against available
// stock. It returns the remaining quantity after the requested usage.
func EnsureAvailable(opening, received, usage, requested int64) (int64, error) {
	if _, err := NewDoses(requested); err != nil {
		return 0, err
	}
	remaining, err := ComputeBalance(opening, received, usage)
	if err != nil {
		return 0, err
	}
	if requested > remaining {
		return remaining, NewError(CodeInsufficientStock, "quantity", "لا يوجد ما يكفي من اللقاح المتبقي لهذا القيد.")
	}
	return remaining - requested, nil
}

// IsLowStock is a presentation helper. Threshold zero disables the alert.
func IsLowStock(remaining, threshold int64) bool {
	return threshold > 0 && remaining <= threshold
}
