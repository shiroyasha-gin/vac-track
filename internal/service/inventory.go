package service

import (
	"context"
	"database/sql"
	"errors"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

// OpeningSuggestion explains the recommended carryover for a vaccine/month.
type OpeningSuggestion struct {
	Month       string
	Amount      int64
	Source      string
	PriorMonth  string
	PriorExists bool
	PriorClosed bool
}

// InitializeInventoryInput explicitly initializes one vaccine/month record.
type InitializeInventoryInput struct {
	VaccineID      int64
	Month          string
	OpeningBalance int64
	StockReceived  int64
	OverrideReason string
}

// UpdateInventoryInput changes opening and received stock on an open month.
type UpdateInventoryInput struct {
	VaccineID      int64
	Month          string
	OpeningBalance int64
	StockReceived  int64
	OverrideReason string
}

// AdjustmentInput is an audited correction to received stock.
type AdjustmentInput struct {
	VaccineID int64
	Month     string
	Amount    int64
	Reason    string
}

// AdjustmentResult reports the new state after an adjustment.
type AdjustmentResult struct {
	Inventory  generated.MonthlyInventory
	Adjustment generated.InventoryAdjustment
	Balance    int64
}

// InventoryService owns monthly stock initialization, stock edits, closing,
// reopening and audited adjustments.
type InventoryService struct {
	base
}

// Suggestion computes the carryover proposal from the immediately previous
// calendar month. A previous open month uses its derived remaining quantity;
// a closed month uses its stored closing balance.
func (s *InventoryService) Suggestion(ctx context.Context, vaccineID int64, monthStr string) (OpeningSuggestion, error) {
	month, err := domain.ParseMonth(monthStr)
	if err != nil {
		return OpeningSuggestion{}, err
	}
	if vaccineID <= 0 {
		return OpeningSuggestion{}, notFound("Vaccine")
	}
	return s.suggestion(ctx, s.store.Queries(), vaccineID, month)
}

func (s *InventoryService) suggestion(ctx context.Context, q generated.Querier, vaccineID int64, month domain.Month) (OpeningSuggestion, error) {
	prev := month.Previous()
	out := OpeningSuggestion{Month: month.String(), PriorMonth: prev.String(), Source: "none", Amount: 0}
	if prev.IsZero() {
		return out, nil
	}
	inv, err := q.GetPreviousMonthlyInventory(ctx, generated.GetPreviousMonthlyInventoryParams{
		VaccineID: vaccineID,
		Month:     prev.String(),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.PriorExists = true
	out.PriorClosed = inv.Status == string(domain.StatusClosed)
	amount := inv.ClosingBalance.Int64
	if !out.PriorClosed || !inv.ClosingBalance.Valid {
		usage, err := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{
			VaccineID: vaccineID,
			Month:     prev.String(),
		})
		if err != nil {
			return out, err
		}
		amount, err = domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, usage)
		if err != nil {
			return out, err
		}
	}
	out.Amount = amount
	if out.PriorClosed {
		out.Source = "prior_month_closing"
	} else {
		out.Source = "prior_month_balance"
	}
	return out, nil
}

// Initialize creates the explicit monthly inventory row. Explicit zero stock
// is valid. If a previous month exists and the operator changes its carryover,
// a reason is required and audited.
func (s *InventoryService) Initialize(ctx context.Context, input InitializeInventoryInput) (generated.MonthlyInventory, error) {
	month, err := domain.ParseMonth(input.Month)
	if err != nil {
		return generated.MonthlyInventory{}, err
	}
	if input.VaccineID <= 0 {
		return generated.MonthlyInventory{}, notFound("Vaccine")
	}
	if _, err := domain.NewNonNegativeDoses(input.OpeningBalance); err != nil {
		return generated.MonthlyInventory{}, err
	}
	if _, err := domain.NewNonNegativeDoses(input.StockReceived); err != nil {
		return generated.MonthlyInventory{}, err
	}

	var created generated.MonthlyInventory
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		vaccine, err := q.GetVaccine(ctx, input.VaccineID)
		if err != nil {
			return requireFound(err, "Vaccine")
		}
		if vaccine.IsArchived == 1 {
			return domain.NewError(domain.CodeArchived, "", "لا يمكن تهيئة مخزون شهري للقاحات المؤرشفة.")
		}
		closed, err := q.CountClosedMonthlyInventoryForMonth(ctx, month.String())
		if err != nil {
			return err
		}
		if closed > 0 {
			return domain.NewError(domain.CodeMonthClosed, "", "هذا الشهر مغلق. أعد فتحه مع ذكر سبب قبل تهيئة مخزون إضافي.")
		}
		if _, err := q.GetMonthlyInventory(ctx, generated.GetMonthlyInventoryParams{
			VaccineID: input.VaccineID,
			Month:     month.String(),
		}); err == nil {
			return domainConflict("تمت تهيئة المخزون الشهري لهذا اللقاح وهذا الشهر بالفعل.")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		suggestion, err := s.suggestion(ctx, q, input.VaccineID, month)
		if err != nil {
			return err
		}
		overridden := suggestion.PriorExists && input.OpeningBalance != suggestion.Amount
		reason := ""
		if overridden {
			reason, err = domain.RequireReason(input.OverrideReason, "override_reason", "سبب تجاوز الترحيل")
			if err != nil {
				return err
			}
		}
		now := s.timestamp()
		created, err = q.CreateMonthlyInventory(ctx, generated.CreateMonthlyInventoryParams{
			VaccineID:           input.VaccineID,
			Month:               month.String(),
			OpeningBalance:      input.OpeningBalance,
			StockReceived:       input.StockReceived,
			CarryoverOverridden: boolInt(overridden),
			CarryoverReason:     reason,
			CreatedAt:           now,
			UpdatedAt:           now,
		})
		if err != nil {
			return mapPersistenceError(err, "تمت تهيئة المخزون الشهري لهذا اللقاح وهذا الشهر بالفعل.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventInventoryInitialized,
			EntityType: "monthly_inventory",
			EntityID:   idString(created.ID),
			Month:      month.String(),
			Reason:     reason,
			Metadata: map[string]any{
				"vaccine_id":           input.VaccineID,
				"opening_balance":      input.OpeningBalance,
				"stock_received":       input.StockReceived,
				"carryover_overridden": overridden,
				"suggested_opening":    suggestion.Amount,
			},
		})
	})
	if err != nil {
		return generated.MonthlyInventory{}, err
	}
	return created, nil
}

// UpdateStock edits opening/received stock on an open month and revalidates
// that usage cannot exceed available doses.
func (s *InventoryService) UpdateStock(ctx context.Context, input UpdateInventoryInput) (generated.MonthlyInventory, error) {
	month, err := domain.ParseMonth(input.Month)
	if err != nil {
		return generated.MonthlyInventory{}, err
	}
	if input.VaccineID <= 0 {
		return generated.MonthlyInventory{}, notFound("Vaccine")
	}
	if _, err := domain.NewNonNegativeDoses(input.OpeningBalance); err != nil {
		return generated.MonthlyInventory{}, err
	}
	if _, err := domain.NewNonNegativeDoses(input.StockReceived); err != nil {
		return generated.MonthlyInventory{}, err
	}

	var updated generated.MonthlyInventory
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		inv, err := q.GetMonthlyInventory(ctx, generated.GetMonthlyInventoryParams{
			VaccineID: input.VaccineID,
			Month:     month.String(),
		})
		if err != nil {
			return requireFound(err, "Monthly inventory")
		}
		if err := domain.RequireOpen(domain.InventoryStatus(inv.Status)); err != nil {
			return err
		}
		usage, err := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{
			VaccineID: input.VaccineID,
			Month:     month.String(),
		})
		if err != nil {
			return err
		}
		if _, err := domain.ComputeBalance(input.OpeningBalance, input.StockReceived, usage); err != nil {
			return err
		}

		suggestion, err := s.suggestion(ctx, q, input.VaccineID, month)
		if err != nil {
			return err
		}
		overridden := suggestion.PriorExists && input.OpeningBalance != suggestion.Amount
		reason := ""
		if overridden {
			reason, err = domain.RequireReason(input.OverrideReason, "override_reason", "سبب تجاوز الترحيل")
			if err != nil {
				return err
			}
		}
		updated, err = q.UpdateMonthlyInventory(ctx, generated.UpdateMonthlyInventoryParams{
			OpeningBalance:      input.OpeningBalance,
			StockReceived:       input.StockReceived,
			CarryoverOverridden: boolInt(overridden),
			CarryoverReason:     reason,
			UpdatedAt:           s.timestamp(),
			ID:                  inv.ID,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر تحديث المخزون الشهري.")
		}
		eventType := audit.EventInventoryStockUpdated
		if overridden {
			eventType = audit.EventCarryoverOverridden
		}
		return s.record(ctx, q, audit.Event{
			Type:       eventType,
			EntityType: "monthly_inventory",
			EntityID:   idString(inv.ID),
			Month:      month.String(),
			Reason:     reason,
			Metadata: map[string]any{
				"vaccine_id": input.VaccineID,
				"before": map[string]any{
					"opening_balance": inv.OpeningBalance,
					"stock_received":  inv.StockReceived,
				},
				"after": map[string]any{
					"opening_balance": updated.OpeningBalance,
					"stock_received":  updated.StockReceived,
				},
				"carryover_overridden": overridden,
				"suggested_opening":    suggestion.Amount,
				"active_usage":         usage,
			},
		})
	})
	if err != nil {
		return generated.MonthlyInventory{}, err
	}
	return updated, nil
}

// Adjust applies an audited correction to received stock. The adjustment may
// be negative as long as the resulting balance never becomes negative.
func (s *InventoryService) Adjust(ctx context.Context, input AdjustmentInput) (AdjustmentResult, error) {
	month, err := domain.ParseMonth(input.Month)
	if err != nil {
		return AdjustmentResult{}, err
	}
	if input.VaccineID <= 0 {
		return AdjustmentResult{}, notFound("Vaccine")
	}
	if input.Amount == 0 {
		return AdjustmentResult{}, domain.NewError(domain.CodeInvalidQuantity, "amount", "يجب ألا يكون مقدار التسوية صفراً.")
	}
	reason, err := domain.RequireReason(input.Reason, "reason", "سبب التسوية")
	if err != nil {
		return AdjustmentResult{}, err
	}

	var result AdjustmentResult
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		inv, err := q.GetMonthlyInventory(ctx, generated.GetMonthlyInventoryParams{
			VaccineID: input.VaccineID,
			Month:     month.String(),
		})
		if err != nil {
			return requireFound(err, "Monthly inventory")
		}
		if err := domain.RequireOpen(domain.InventoryStatus(inv.Status)); err != nil {
			return err
		}
		usage, err := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{
			VaccineID: input.VaccineID,
			Month:     month.String(),
		})
		if err != nil {
			return err
		}
		before, err := domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, usage)
		if err != nil {
			return err
		}
		newReceived := inv.StockReceived + input.Amount
		if newReceived < 0 {
			return domain.NewError(domain.CodeInsufficientStock, "amount", "ستجعل التسوية المخزون الوارد سالباً.")
		}
		after, err := domain.ComputeBalance(inv.OpeningBalance, newReceived, usage)
		if err != nil {
			return err
		}
		updated, err := q.UpdateMonthlyInventory(ctx, generated.UpdateMonthlyInventoryParams{
			OpeningBalance:      inv.OpeningBalance,
			StockReceived:       newReceived,
			CarryoverOverridden: inv.CarryoverOverridden,
			CarryoverReason:     inv.CarryoverReason,
			UpdatedAt:           s.timestamp(),
			ID:                  inv.ID,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر حفظ التسوية.")
		}
		adjustment, err := q.CreateInventoryAdjustment(ctx, generated.CreateInventoryAdjustmentParams{
			VaccineID:     input.VaccineID,
			Month:         month.String(),
			Amount:        input.Amount,
			BalanceBefore: before,
			BalanceAfter:  after,
			Reason:        reason,
			CreatedAt:     s.timestamp(),
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر تسجيل التسوية.")
		}
		result = AdjustmentResult{Inventory: updated, Adjustment: adjustment, Balance: after}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventInventoryAdjusted,
			EntityType: "monthly_inventory",
			EntityID:   idString(inv.ID),
			Month:      month.String(),
			Reason:     reason,
			Metadata: map[string]any{
				"vaccine_id":     input.VaccineID,
				"amount":         input.Amount,
				"balance_before": before,
				"balance_after":  after,
			},
		})
	})
	if err != nil {
		return AdjustmentResult{}, err
	}
	return result, nil
}

// CloseMonth stores a final balance for every open vaccine inventory record in
// the month. Routine stock and usage edits are then rejected until a reasoned
// reopen.
func (s *InventoryService) CloseMonth(ctx context.Context, monthStr string) (int, error) {
	month, err := domain.ParseMonth(monthStr)
	if err != nil {
		return 0, err
	}
	count := 0
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		open, err := q.ListOpenMonthlyInventoryForMonth(ctx, month.String())
		if err != nil {
			return err
		}
		if len(open) == 0 {
			closed, err := q.CountClosedMonthlyInventoryForMonth(ctx, month.String())
			if err != nil {
				return err
			}
			if closed > 0 {
				return domain.NewError(domain.CodeAlreadyClosed, "", "هذا الشهر مغلق بالفعل.")
			}
			return domain.NewError(domain.CodeNotInitialized, "", "لم تتم تهيئة أي مخزون شهري لهذا الشهر.")
		}
		var closingTotal int64
		now := s.timestamp()
		for _, inv := range open {
			usage, err := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{
				VaccineID: inv.VaccineID,
				Month:     month.String(),
			})
			if err != nil {
				return err
			}
			balance, err := domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, usage)
			if err != nil {
				return domain.NewError(domain.CodeInsufficientStock, "", "لا يمكن إغلاق الشهر مع وجود رصيد لقاح سالب.")
			}
			if _, err := q.CloseMonthlyInventory(ctx, generated.CloseMonthlyInventoryParams{
				ClosingBalance: sql.NullInt64{Int64: balance, Valid: true},
				ClosedAt:       sql.NullString{String: now, Valid: true},
				UpdatedAt:      now,
				ID:             inv.ID,
			}); err != nil {
				return mapPersistenceError(err, "تعذر إغلاق الشهر.")
			}
			closingTotal += balance
			count++
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventMonthClosed,
			EntityType: "month",
			EntityID:   month.String(),
			Month:      month.String(),
			Metadata: map[string]any{
				"vaccine_count":         count,
				"closing_balance_total": closingTotal,
			},
		})
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ReopenMonth requires a reason and restores every closed vaccine record to
// open so corrections can be made deliberately.
func (s *InventoryService) ReopenMonth(ctx context.Context, monthStr, reason string) (int, error) {
	month, err := domain.ParseMonth(monthStr)
	if err != nil {
		return 0, err
	}
	reason, err = domain.RequireReason(reason, "reason", "سبب إعادة الفتح")
	if err != nil {
		return 0, err
	}
	count := 0
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		closed, err := q.ListClosedMonthlyInventoryForMonth(ctx, month.String())
		if err != nil {
			return err
		}
		if len(closed) == 0 {
			return domain.NewError(domain.CodeNotClosed, "", "هذا الشهر غير مغلق.")
		}
		now := s.timestamp()
		for _, inv := range closed {
			if _, err := q.ReopenMonthlyInventory(ctx, generated.ReopenMonthlyInventoryParams{
				ReopenedAt: sql.NullString{String: now, Valid: true},
				UpdatedAt:  now,
				ID:         inv.ID,
			}); err != nil {
				return mapPersistenceError(err, "تعذر إعادة فتح الشهر.")
			}
			count++
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventMonthReopened,
			EntityType: "month",
			EntityID:   month.String(),
			Month:      month.String(),
			Reason:     reason,
			Metadata:   map[string]any{"vaccine_count": count},
		})
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Adjustments lists applied corrections for a month.
func (s *InventoryService) Adjustments(ctx context.Context, month string) ([]generated.ListInventoryAdjustmentsForMonthRow, error) {
	if _, err := domain.ParseMonth(month); err != nil {
		return nil, err
	}
	return s.store.Queries().ListInventoryAdjustmentsForMonth(ctx, month)
}

func requireInitializedOpen(ctx context.Context, q generated.Querier, vaccineID int64, month string) (generated.MonthlyInventory, error) {
	inv, err := q.GetMonthlyInventory(ctx, generated.GetMonthlyInventoryParams{
		VaccineID: vaccineID,
		Month:     month,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return generated.MonthlyInventory{}, domain.NewError(domain.CodeNotInitialized, "", "هيّئ المخزون الشهري لهذا اللقاح وهذا الشهر قبل تسجيل الاستخدام.")
	}
	if err != nil {
		return generated.MonthlyInventory{}, err
	}
	if err := domain.ValidateInventoryStatus(inv.Status); err != nil {
		return generated.MonthlyInventory{}, err
	}
	if err := domain.RequireOpen(domain.InventoryStatus(inv.Status)); err != nil {
		return generated.MonthlyInventory{}, err
	}
	return inv, nil
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
