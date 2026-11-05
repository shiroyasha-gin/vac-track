package service

import (
	"context"
	"database/sql"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

// CreateUsageInput records a dose administration.
type CreateUsageInput struct {
	PatientID      int64
	VaccineID      int64
	AdministeredOn string
	Quantity       int64
	Note           string
}

// UpdateUsageInput edits an existing usage entry. The vaccine cannot be
// changed; void and re-enter if the vaccine itself was wrong. The original and
// target inventory months must both be open.
type UpdateUsageInput struct {
	ID             int64
	PatientID      int64
	AdministeredOn string
	Quantity       int64
	Note           string
	Reason         string
}

// VoidUsageInput cancels an entry without deleting history.
type VoidUsageInput struct {
	ID     int64
	Reason string
}

// UsageResult reports the entry and the recalculated stock state.
type UsageResult struct {
	Entry      generated.UsageEntry
	Inventory  generated.MonthlyInventory
	UsageTotal int64
	Balance    int64
}

// UsageService owns patient-level usage and the negative-stock guarantee.
type UsageService struct {
	base
}

func normalizeUsageNote(note string) (string, error) {
	return domain.OptionalText(note, "note", "ملاحظة", 300)
}

// Create atomically validates setup, month, stock and patient state, inserts
// the usage entry, and writes the audit event.
func (s *UsageService) Create(ctx context.Context, input CreateUsageInput) (UsageResult, error) {
	if input.PatientID <= 0 {
		return UsageResult{}, notFound("Patient")
	}
	if input.VaccineID <= 0 {
		return UsageResult{}, notFound("Vaccine")
	}
	if _, err := domain.NewDoses(input.Quantity); err != nil {
		return UsageResult{}, err
	}
	date, err := domain.ParseDate(input.AdministeredOn)
	if err != nil {
		return UsageResult{}, err
	}
	month := domain.MonthOf(date)
	note, err := normalizeUsageNote(input.Note)
	if err != nil {
		return UsageResult{}, err
	}

	var result UsageResult
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		patient, err := q.GetPatient(ctx, input.PatientID)
		if err != nil {
			return requireFound(err, "Patient")
		}
		if patient.IsArchived == 1 {
			return domain.NewError(domain.CodeArchived, "patient", "لا يمكن تسجيل استخدامات اعتيادية جديدة للمرضى المؤرشفين.")
		}
		vaccine, err := q.GetVaccine(ctx, input.VaccineID)
		if err != nil {
			return requireFound(err, "Vaccine")
		}
		if vaccine.IsArchived == 1 {
			return domain.NewError(domain.CodeArchived, "vaccine", "لا تُعرض اللقاحات المؤرشفة للاستخدامات الاعتيادية الجديدة.")
		}
		if err := domain.DateInMonth(month, input.AdministeredOn); err != nil {
			return err
		}
		inv, err := requireInitializedOpen(ctx, q, input.VaccineID, month.String())
		if err != nil {
			return err
		}
		usage, err := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{
			VaccineID: input.VaccineID,
			Month:     month.String(),
		})
		if err != nil {
			return err
		}
		if _, err := domain.EnsureAvailable(inv.OpeningBalance, inv.StockReceived, usage, input.Quantity); err != nil {
			return err
		}
		now := s.timestamp()
		entry, err := q.CreateUsageEntry(ctx, generated.CreateUsageEntryParams{
			PatientID:      input.PatientID,
			VaccineID:      input.VaccineID,
			AdministeredOn: input.AdministeredOn,
			Month:          month.String(),
			Quantity:       input.Quantity,
			Note:           note,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر حفظ قيد الاستخدام.")
		}
		newUsage := usage + input.Quantity
		balance, err := domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, newUsage)
		if err != nil {
			return err
		}
		result = UsageResult{Entry: entry, Inventory: inv, UsageTotal: newUsage, Balance: balance}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventUsageCreated,
			EntityType: "usage_entry",
			EntityID:   idString(entry.ID),
			Month:      month.String(),
			Metadata: map[string]any{
				"patient_id":      input.PatientID,
				"vaccine_id":      input.VaccineID,
				"quantity":        input.Quantity,
				"administered_on": input.AdministeredOn,
			},
		})
	})
	if err != nil {
		return UsageResult{}, err
	}
	return result, nil
}

// Update revalidates both the source and target inventory before committing
// the edit and its audit event.
func (s *UsageService) Update(ctx context.Context, input UpdateUsageInput) (UsageResult, error) {
	if input.ID <= 0 {
		return UsageResult{}, notFound("Usage entry")
	}
	if input.PatientID <= 0 {
		return UsageResult{}, notFound("Patient")
	}
	if _, err := domain.NewDoses(input.Quantity); err != nil {
		return UsageResult{}, err
	}
	date, err := domain.ParseDate(input.AdministeredOn)
	if err != nil {
		return UsageResult{}, err
	}
	targetMonth := domain.MonthOf(date)
	note, err := normalizeUsageNote(input.Note)
	if err != nil {
		return UsageResult{}, err
	}
	reason, err := domain.OptionalText(input.Reason, "reason", "سبب التعديل", 500)
	if err != nil {
		return UsageResult{}, err
	}

	var result UsageResult
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		original, err := q.GetUsageEntry(ctx, input.ID)
		if err != nil {
			return requireFound(err, "Usage entry")
		}
		if original.VoidedAt.Valid {
			return domain.NewError(domain.CodeInvalidInput, "", "لا يمكن تعديل استخدام ملغى. سجّل قيداً جديداً بدلاً من ذلك.")
		}
		// The source month must still be open: editing removes the original
		// contribution from that month's balance.
		if _, err := requireInitializedOpen(ctx, q, original.VaccineID, original.Month); err != nil {
			return err
		}
		patient, err := q.GetPatient(ctx, input.PatientID)
		if err != nil {
			return requireFound(err, "Patient")
		}
		if patient.IsArchived == 1 {
			return domain.NewError(domain.CodeArchived, "patient", "لا يمكن تسجيل استخدامات اعتيادية للمرضى المؤرشفين.")
		}
		inv, err := requireInitializedOpen(ctx, q, original.VaccineID, targetMonth.String())
		if err != nil {
			return err
		}
		usage, err := q.SumActiveUsageForVaccineMonthExcluding(ctx, generated.SumActiveUsageForVaccineMonthExcludingParams{
			VaccineID: original.VaccineID,
			Month:     targetMonth.String(),
			ID:        original.ID,
		})
		if err != nil {
			return err
		}
		if _, err := domain.EnsureAvailable(inv.OpeningBalance, inv.StockReceived, usage, input.Quantity); err != nil {
			return err
		}
		entry, err := q.UpdateUsageEntry(ctx, generated.UpdateUsageEntryParams{
			PatientID:      input.PatientID,
			AdministeredOn: input.AdministeredOn,
			Month:          targetMonth.String(),
			Quantity:       input.Quantity,
			Note:           note,
			UpdatedAt:      s.timestamp(),
			ID:             input.ID,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر تحديث قيد الاستخدام.")
		}
		newUsage := usage + input.Quantity
		balance, err := domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, newUsage)
		if err != nil {
			return err
		}
		result = UsageResult{Entry: entry, Inventory: inv, UsageTotal: newUsage, Balance: balance}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventUsageUpdated,
			EntityType: "usage_entry",
			EntityID:   idString(entry.ID),
			Month:      targetMonth.String(),
			Reason:     reason,
			Metadata: map[string]any{
				"patient_id": input.PatientID,
				"vaccine_id": original.VaccineID,
				"before": map[string]any{
					"patient_id":      original.PatientID,
					"quantity":        original.Quantity,
					"administered_on": original.AdministeredOn,
				},
				"after": map[string]any{
					"patient_id":      entry.PatientID,
					"quantity":        entry.Quantity,
					"administered_on": entry.AdministeredOn,
				},
			},
		})
	})
	if err != nil {
		return UsageResult{}, err
	}
	return result, nil
}

// Void cancels an entry with a required reason, never deleting history.
func (s *UsageService) Void(ctx context.Context, input VoidUsageInput) (UsageResult, error) {
	if input.ID <= 0 {
		return UsageResult{}, notFound("Usage entry")
	}
	reason, err := domain.RequireReason(input.Reason, "reason", "سبب الإلغاء")
	if err != nil {
		return UsageResult{}, err
	}
	var result UsageResult
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		original, err := q.GetUsageEntry(ctx, input.ID)
		if err != nil {
			return requireFound(err, "Usage entry")
		}
		if original.VoidedAt.Valid {
			return domain.NewError(domain.CodeInvalidInput, "", "قيد الاستخدام هذا ملغى بالفعل.")
		}
		inv, err := requireInitializedOpen(ctx, q, original.VaccineID, original.Month)
		if err != nil {
			return err
		}
		now := s.timestamp()
		entry, err := q.VoidUsageEntry(ctx, generated.VoidUsageEntryParams{
			VoidedAt:   sql.NullString{String: now, Valid: true},
			VoidReason: reason,
			UpdatedAt:  now,
			ID:         input.ID,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر إلغاء قيد الاستخدام.")
		}
		usage, err := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{
			VaccineID: original.VaccineID,
			Month:     original.Month,
		})
		if err != nil {
			return err
		}
		balance, err := domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, usage)
		if err != nil {
			return err
		}
		result = UsageResult{Entry: entry, Inventory: inv, UsageTotal: usage, Balance: balance}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventUsageVoided,
			EntityType: "usage_entry",
			EntityID:   idString(entry.ID),
			Month:      original.Month,
			Reason:     reason,
			Metadata: map[string]any{
				"patient_id":      original.PatientID,
				"vaccine_id":      original.VaccineID,
				"quantity":        original.Quantity,
				"administered_on": original.AdministeredOn,
			},
		})
	})
	if err != nil {
		return UsageResult{}, err
	}
	return result, nil
}

// ForVaccineDay returns private local detail for one day cell, including
// voided rows for traceability.
func (s *UsageService) ForVaccineDay(ctx context.Context, vaccineID int64, month, date string) ([]generated.ListUsageEntriesForVaccineDayRow, error) {
	if _, err := domain.ParseMonth(month); err != nil {
		return nil, err
	}
	if err := domain.DateInMonth(mustMonth(month), date); err != nil {
		return nil, err
	}
	if vaccineID <= 0 {
		return nil, notFound("Vaccine")
	}
	return s.store.Queries().ListUsageEntriesForVaccineDay(ctx, generated.ListUsageEntriesForVaccineDayParams{
		VaccineID:      vaccineID,
		Month:          month,
		AdministeredOn: date,
	})
}

// ForVaccineMonth returns private local detail for a vaccine/month.
func (s *UsageService) ForVaccineMonth(ctx context.Context, vaccineID int64, month string) ([]generated.ListUsageEntriesForVaccineMonthRow, error) {
	if _, err := domain.ParseMonth(month); err != nil {
		return nil, err
	}
	if vaccineID <= 0 {
		return nil, notFound("Vaccine")
	}
	return s.store.Queries().ListUsageEntriesForVaccineMonth(ctx, generated.ListUsageEntriesForVaccineMonthParams{
		VaccineID: vaccineID,
		Month:     month,
	})
}

// ForPatient returns recent private usage history for one patient.
func (s *UsageService) ForPatient(ctx context.Context, patientID int64) ([]generated.ListUsageEntriesForPatientRow, error) {
	if patientID <= 0 {
		return nil, notFound("Patient")
	}
	return s.store.Queries().ListUsageEntriesForPatient(ctx, generated.ListUsageEntriesForPatientParams{
		PatientID: patientID,
		Limit:     200,
	})
}

// Get returns a usage entry.
func (s *UsageService) Get(ctx context.Context, id int64) (generated.UsageEntry, error) {
	entry, err := s.store.Queries().GetUsageEntry(ctx, id)
	return entry, requireFound(err, "Usage entry")
}

func mustMonth(value string) domain.Month {
	m, _ := domain.ParseMonth(value)
	return m
}
