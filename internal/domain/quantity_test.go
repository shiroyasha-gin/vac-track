package domain

import "testing"

func TestComputeBalance(t *testing.T) {
	t.Parallel()
	got, err := ComputeBalance(8, 12, 5)
	if err != nil {
		t.Fatalf("ComputeBalance() error = %v", err)
	}
	if got != 15 {
		t.Fatalf("ComputeBalance(8,12,5) = %d, want 15", got)
	}

	got, err = ComputeBalance(0, 0, 0)
	if err != nil || got != 0 {
		t.Fatalf("zero inventory = %d, %v; want 0, nil", got, err)
	}

	if _, err := ComputeBalance(-1, 0, 0); !IsCode(err, CodeInvalidQuantity) {
		t.Fatalf("negative opening error = %v", err)
	}
	if _, err := ComputeBalance(0, 0, 1); !IsCode(err, CodeInsufficientStock) {
		t.Fatalf("usage above stock error = %v", err)
	}
}

func TestEnsureAvailable(t *testing.T) {
	t.Parallel()
	remaining, err := EnsureAvailable(8, 12, 0, 15)
	if err != nil {
		t.Fatalf("15 doses from 20 available should succeed: %v", err)
	}
	if remaining != 5 {
		t.Fatalf("remaining = %d, want 5", remaining)
	}

	if _, err := EnsureAvailable(8, 12, 5, 16); !IsCode(err, CodeInsufficientStock) {
		t.Fatalf("16 doses from 15 available error = %v, want %v", err, CodeInsufficientStock)
	}
	if _, err := EnsureAvailable(8, 12, 5, 0); !IsCode(err, CodeInvalidQuantity) {
		t.Fatalf("zero quantity error = %v", err)
	}
	if _, err := EnsureAvailable(8, 12, 5, -1); !IsCode(err, CodeInvalidQuantity) {
		t.Fatalf("negative quantity error = %v", err)
	}
}

func TestVoidingIncreasesRemaining(t *testing.T) {
	t.Parallel()
	opening, received := int64(8), int64(12)
	activeUsage := int64(5)
	remaining, _ := ComputeBalance(opening, received, activeUsage)
	voided := int64(3)
	afterVoid, err := ComputeBalance(opening, received, activeUsage-voided)
	if err != nil {
		t.Fatal(err)
	}
	if afterVoid != remaining+voided {
		t.Fatalf("remaining after void = %d, want %d", afterVoid, remaining+voided)
	}
}

func TestDosesRejectFractionalAndNegative(t *testing.T) {
	t.Parallel()
	if _, err := ParseDoses("2"); err != nil {
		t.Fatalf("whole dose rejected: %v", err)
	}
	if _, err := ParseDoses("2.5"); !IsCode(err, CodeInvalidQuantity) {
		t.Fatalf("fractional quantity error = %v", err)
	}
	if _, err := ParseDoses("-1"); !IsCode(err, CodeInvalidQuantity) {
		t.Fatalf("negative quantity error = %v", err)
	}
	if _, err := NewNonNegativeDoses(-1); !IsCode(err, CodeInvalidQuantity) {
		t.Fatalf("negative stock error = %v", err)
	}
	if _, err := NewNonNegativeDoses(0); err != nil {
		t.Fatalf("explicit zero stock should be valid: %v", err)
	}
}

func TestRequireReason(t *testing.T) {
	t.Parallel()
	if _, err := RequireReason("  ", "reason", "Reason"); !IsCode(err, CodeReasonRequired) {
		t.Fatalf("empty reason error = %v", err)
	}
	if got, err := RequireReason("  correction ", "reason", "Reason"); err != nil || got != "correction" {
		t.Fatalf("reason = %q, %v", got, err)
	}
}

func TestRequireOpen(t *testing.T) {
	t.Parallel()
	if err := RequireOpen(StatusOpen); err != nil {
		t.Fatalf("open status rejected: %v", err)
	}
	if err := RequireOpen(StatusClosed); !IsCode(err, CodeMonthClosed) {
		t.Fatalf("closed status error = %v", err)
	}
}
