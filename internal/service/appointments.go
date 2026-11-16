package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

const (
	appointmentScheduled = "scheduled"
	appointmentCompleted = "completed"
	appointmentCancelled = "cancelled"
)

// AppointmentItemInput is one vaccine line in a scheduled visit.
type AppointmentItemInput struct {
	VaccineID int64
	Quantity  int64
}

// AppointmentInput schedules a patient visit and consumes the specified
// vaccine quantities from stock immediately.
type AppointmentInput struct {
	PatientID     int64
	ScheduledOn   string
	ScheduledTime string
	Note          string
	Items         []AppointmentItemInput
}

// UpdateAppointmentInput edits a scheduled appointment that has not yet been
// completed. Linked usage rows are adjusted or voided so stock stays in sync.
type UpdateAppointmentInput struct {
	ID            int64
	PatientID     int64
	ScheduledOn   string
	ScheduledTime string
	Note          string
	Items         []AppointmentItemInput
	Reason        string
}

// CancelAppointmentInput cancels a scheduled appointment and voids its
// already-deducted usage, returning the reserved doses to stock.
type CancelAppointmentInput struct {
	ID     int64
	Reason string
}

// AppointmentFilter narrows the appointment list. Empty status and dates mean
// no filter.
type AppointmentFilter struct {
	Status   string
	FromDate string
	ToDate   string
}

// AppointmentResult includes the recalculated stock state after completion.
type AppointmentResult struct {
	Appointment generated.Appointment
	Usage       []UsageResult
}

// AppointmentItemView is one vaccine line and its linked stock impact.
type AppointmentItemView struct {
	ID                  int64
	AppointmentID       int64
	VaccineID           int64
	VaccineName         string
	VaccineCode         string
	VaccineUnit         string
	VaccineArchived     bool
	Quantity            int64
	UsageEntryID        int64
	UsageVoided         bool
	UsageVoidReason     string
	UsageAdministeredOn string
	UsageQuantity       int64
	CreatedAt           string
	UpdatedAt           string
}

// AppointmentView is a private scheduled visit with its vaccine lines.
type AppointmentView struct {
	ID              int64
	PatientID       int64
	PatientName     string
	PatientCode     string
	PatientArchived bool
	ScheduledOn     string
	ScheduledTime   string
	Note            string
	Status          string
	CancelReason    string
	CompletedAt     string
	CancelledAt     string
	CreatedAt       string
	UpdatedAt       string
	Items           []AppointmentItemView
}

// AppointmentService schedules visits and completes them through the same
// transactional usage path used by the monthly inventory table.
type AppointmentService struct {
	base
	usage *UsageService
}

func normalizeAppointmentItems(items []AppointmentItemInput) ([]AppointmentItemInput, error) {
	if len(items) == 0 {
		return nil, domain.NewError(domain.CodeInvalidInput, "items", "أضف لقاحاً واحداً على الأقل إلى الموعد.")
	}
	seen := make(map[int64]struct{}, len(items))
	out := make([]AppointmentItemInput, 0, len(items))
	for _, item := range items {
		if item.VaccineID <= 0 {
			return nil, notFound("Vaccine")
		}
		if _, err := domain.NewDoses(item.Quantity); err != nil {
			return nil, err
		}
		if _, ok := seen[item.VaccineID]; ok {
			return nil, domain.NewError(domain.CodeInvalidInput, "vaccineId", "لا يمكن تكرار اللقاح نفسه في موعد واحد.")
		}
		seen[item.VaccineID] = struct{}{}
		out = append(out, AppointmentItemInput{VaccineID: item.VaccineID, Quantity: item.Quantity})
	}
	return out, nil
}

func normalizeAppointmentInput(input AppointmentInput) (AppointmentInput, error) {
	if input.PatientID <= 0 {
		return input, notFound("Patient")
	}
	date, err := domain.ParseDate(input.ScheduledOn)
	if err != nil {
		return input, err
	}
	input.ScheduledOn = domain.FormatDate(date)
	input.ScheduledTime, err = normalizeAppointmentTime(input.ScheduledTime)
	if err != nil {
		return input, err
	}
	input.Note, err = domain.OptionalText(input.Note, "note", "ملاحظة الموعد", 300)
	if err != nil {
		return input, err
	}
	items, err := normalizeAppointmentItems(input.Items)
	if err != nil {
		return input, err
	}
	input.Items = items
	return input, nil
}

func normalizeAppointmentTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", domain.NewError(domain.CodeInvalidInput, "scheduled_time", "استخدم وقتاً صالحاً بصيغة HH:MM.")
	}
	return parsed.Format("15:04"), nil
}

func normalizeAppointmentFilter(filter AppointmentFilter) (AppointmentFilter, error) {
	filter.Status = strings.TrimSpace(filter.Status)
	switch filter.Status {
	case "", "all":
		filter.Status = ""
	case appointmentScheduled, appointmentCompleted, appointmentCancelled:
	default:
		return filter, domain.NewError(domain.CodeInvalidInput, "status", "حالة الموعد غير معروفة.")
	}
	if filter.FromDate != "" {
		date, err := domain.ParseDate(filter.FromDate)
		if err != nil {
			return filter, err
		}
		filter.FromDate = domain.FormatDate(date)
	}
	if filter.ToDate != "" {
		date, err := domain.ParseDate(filter.ToDate)
		if err != nil {
			return filter, err
		}
		filter.ToDate = domain.FormatDate(date)
	}
	if filter.FromDate != "" && filter.ToDate != "" && filter.FromDate > filter.ToDate {
		return filter, domain.NewError(domain.CodeInvalidDate, "from_date", "تاريخ البداية يجب أن يسبق تاريخ النهاية.")
	}
	return filter, nil
}

func ensureActiveAppointmentRefs(ctx context.Context, q generated.Querier, patientID int64, items []AppointmentItemInput) error {
	patient, err := q.GetPatient(ctx, patientID)
	if err != nil {
		return requireFound(err, "Patient")
	}
	if patient.IsArchived == 1 {
		return domain.NewError(domain.CodeArchived, "patient", "لا يمكن حجز موعد جديد لمريض مؤرشف.")
	}
	for _, item := range items {
		vaccine, err := q.GetVaccine(ctx, item.VaccineID)
		if err != nil {
			return requireFound(err, "Vaccine")
		}
		if vaccine.IsArchived == 1 {
			return domain.NewError(domain.CodeArchived, "vaccine", "لا يمكن حجز موعد للقاح مؤرشف.")
		}
	}
	return nil
}

// Create schedules an appointment and consumes the specified vaccine quantities
// from stock immediately. The appointment record and the usage/deduction are
// created in a single transaction so stock cannot diverge from the plan.
func (s *AppointmentService) Create(ctx context.Context, input AppointmentInput) (generated.Appointment, error) {
	return s.create(ctx, input, false)
}

// CreateTaken records an appointment that was already administered during the
// same visit. It is created as completed and its usage entries are the stock
// deduction, matching Create's immediate-deduction semantics.
func (s *AppointmentService) CreateTaken(ctx context.Context, input AppointmentInput) (generated.Appointment, error) {
	return s.create(ctx, input, true)
}

func (s *AppointmentService) create(ctx context.Context, input AppointmentInput, taken bool) (generated.Appointment, error) {
	normalized, err := normalizeAppointmentInput(input)
	if err != nil {
		return generated.Appointment{}, err
	}
	var created generated.Appointment
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		if err := ensureActiveAppointmentRefs(ctx, q, normalized.PatientID, normalized.Items); err != nil {
			return err
		}
		now := s.timestamp()
		if taken {
			created, err = q.CreateCompletedAppointment(ctx, generated.CreateCompletedAppointmentParams{
				PatientID:     normalized.PatientID,
				ScheduledOn:   normalized.ScheduledOn,
				ScheduledTime: normalized.ScheduledTime,
				Note:          normalized.Note,
				CompletedAt:   sql.NullString{String: now, Valid: true},
				CreatedAt:     now,
				UpdatedAt:     now,
			})
		} else {
			created, err = q.CreateAppointment(ctx, generated.CreateAppointmentParams{
				PatientID:     normalized.PatientID,
				ScheduledOn:   normalized.ScheduledOn,
				ScheduledTime: normalized.ScheduledTime,
				Note:          normalized.Note,
				CreatedAt:     now,
				UpdatedAt:     now,
			})
		}
		if err != nil {
			return mapPersistenceError(err, "تعذر حفظ الموعد.")
		}
		itemResults := make([]map[string]any, 0, len(normalized.Items))
		for _, item := range normalized.Items {
			usage, err := s.usage.createInTx(ctx, q, CreateUsageInput{
				PatientID:      normalized.PatientID,
				VaccineID:      item.VaccineID,
				AdministeredOn: normalized.ScheduledOn,
				Quantity:       item.Quantity,
				Note:           normalized.Note,
			}, created.ID)
			if err != nil {
				return err
			}
			apptItem, err := q.CreateAppointmentItem(ctx, generated.CreateAppointmentItemParams{
				AppointmentID: created.ID,
				VaccineID:     item.VaccineID,
				Quantity:      item.Quantity,
				CreatedAt:     now,
				UpdatedAt:     now,
			})
			if err != nil {
				return mapPersistenceError(err, "تعذر حفظ بنود الموعد.")
			}
			if _, err := q.SetAppointmentItemUsage(ctx, generated.SetAppointmentItemUsageParams{
				UsageEntryID: sql.NullInt64{Int64: usage.Entry.ID, Valid: true},
				UpdatedAt:    now,
				ID:           apptItem.ID,
			}); err != nil {
				return mapPersistenceError(err, "تعذر ربط جرعة الموعد.")
			}
			itemResults = append(itemResults, map[string]any{
				"item_id":        apptItem.ID,
				"vaccine_id":     item.VaccineID,
				"usage_entry_id": usage.Entry.ID,
				"quantity":       usage.Entry.Quantity,
			})
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventAppointmentCreated,
			EntityType: "appointment",
			EntityID:   idString(created.ID),
			Month:      administeredMonth(normalized.ScheduledOn),
			Metadata: map[string]any{
				"patient_id":     created.PatientID,
				"scheduled_on":   created.ScheduledOn,
				"scheduled_time": created.ScheduledTime,
				"items":          itemResults,
				"taken":          taken,
			},
		})
	})
	if err != nil {
		return generated.Appointment{}, err
	}
	return created, nil
}

// Update edits only a scheduled appointment that has not yet been completed.
func (s *AppointmentService) Update(ctx context.Context, input UpdateAppointmentInput) (generated.Appointment, error) {
	if input.ID <= 0 {
		return generated.Appointment{}, notFound("Appointment")
	}
	normalized, err := normalizeAppointmentInput(AppointmentInput{
		PatientID: input.PatientID, ScheduledOn: input.ScheduledOn,
		ScheduledTime: input.ScheduledTime, Note: input.Note, Items: input.Items,
	})
	if err != nil {
		return generated.Appointment{}, err
	}
	reason, err := domain.OptionalText(input.Reason, "reason", "سبب التعديل", 500)
	if err != nil {
		return generated.Appointment{}, err
	}
	var updated generated.Appointment
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		existing, err := q.GetAppointment(ctx, input.ID)
		if err != nil {
			return requireFound(err, "Appointment")
		}
		if existing.Status != appointmentScheduled {
			return domain.NewError(domain.CodeInvalidInput, "", "لا يمكن تعديل موعد مكتمل أو ملغى.")
		}
		if err := ensureActiveAppointmentRefs(ctx, q, normalized.PatientID, normalized.Items); err != nil {
			return err
		}
		existingItems, err := q.ListAppointmentItemsByAppointment(ctx, input.ID)
		if err != nil {
			return err
		}
		updated, err = q.UpdateAppointment(ctx, generated.UpdateAppointmentParams{
			PatientID:     normalized.PatientID,
			ScheduledOn:   normalized.ScheduledOn,
			ScheduledTime: normalized.ScheduledTime,
			Note:          normalized.Note,
			UpdatedAt:     s.timestamp(),
			ID:            input.ID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewError(domain.CodeConflict, "", "تغيّرت حالة الموعد. حدّث الصفحة وحاول مجدداً.")
		}
		if err != nil {
			return mapPersistenceError(err, "تعذر تحديث الموعد.")
		}

		// Reuse the existing usage entry for each vaccine that stays on the
		// appointment so its quantity, patient and date follow the edit.
		// Vaccines that are removed have their usage voided instead of deleted,
		// which restores their stock and keeps the audit trail intact.
		existingByVaccine := make(map[int64][]generated.ListAppointmentItemsByAppointmentRow, len(existingItems))
		for _, it := range existingItems {
			if it.UsageEntryID.Valid {
				existingByVaccine[it.VaccineID] = append(existingByVaccine[it.VaccineID], it)
			}
		}
		usageByVaccine := make(map[int64]int64, len(normalized.Items))
		for _, item := range normalized.Items {
			candidates := existingByVaccine[item.VaccineID]
			if len(candidates) > 0 {
				old := candidates[0]
				existingByVaccine[item.VaccineID] = candidates[1:]
				usage, err := s.usage.updateInTx(ctx, q, UpdateUsageInput{
					ID:             old.UsageEntryID.Int64,
					PatientID:      normalized.PatientID,
					AdministeredOn: normalized.ScheduledOn,
					Quantity:       item.Quantity,
					Note:           normalized.Note,
					Reason:         reason,
				}, true)
				if err != nil {
					return err
				}
				usageByVaccine[item.VaccineID] = usage.Entry.ID
				continue
			}
			usage, err := s.usage.createInTx(ctx, q, CreateUsageInput{
				PatientID:      normalized.PatientID,
				VaccineID:      item.VaccineID,
				AdministeredOn: normalized.ScheduledOn,
				Quantity:       item.Quantity,
				Note:           normalized.Note,
			}, updated.ID)
			if err != nil {
				return err
			}
			usageByVaccine[item.VaccineID] = usage.Entry.ID
		}
		voidReason := reason
		if voidReason == "" {
			voidReason = "تعديل الموعد: أُزيل اللقاح من الموعد."
		}
		for _, candidates := range existingByVaccine {
			for _, old := range candidates {
				if _, err := s.usage.voidInTx(ctx, q, old.UsageEntryID.Int64, voidReason); err != nil {
					return err
				}
			}
		}

		if err := q.DeleteAppointmentItems(ctx, input.ID); err != nil {
			return err
		}
		now := s.timestamp()
		for _, item := range normalized.Items {
			apptItem, err := q.CreateAppointmentItem(ctx, generated.CreateAppointmentItemParams{
				AppointmentID: updated.ID,
				VaccineID:     item.VaccineID,
				Quantity:      item.Quantity,
				CreatedAt:     now,
				UpdatedAt:     now,
			})
			if err != nil {
				return mapPersistenceError(err, "تعذر حفظ بنود الموعد.")
			}
			if _, err := q.SetAppointmentItemUsage(ctx, generated.SetAppointmentItemUsageParams{
				UsageEntryID: sql.NullInt64{Int64: usageByVaccine[item.VaccineID], Valid: true},
				UpdatedAt:    now,
				ID:           apptItem.ID,
			}); err != nil {
				return mapPersistenceError(err, "تعذر ربط جرعة الموعد.")
			}
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventAppointmentUpdated,
			EntityType: "appointment",
			EntityID:   idString(updated.ID),
			Reason:     reason,
			Metadata: map[string]any{
				"patient_id": updated.PatientID,
				"before": map[string]any{
					"patient_id":     existing.PatientID,
					"scheduled_on":   existing.ScheduledOn,
					"scheduled_time": existing.ScheduledTime,
				},
				"after": map[string]any{
					"patient_id":     updated.PatientID,
					"scheduled_on":   updated.ScheduledOn,
					"scheduled_time": updated.ScheduledTime,
					"items":          itemMetadata(normalized.Items),
				},
			},
		})
	})
	if err != nil {
		return generated.Appointment{}, err
	}
	return updated, nil
}

// CompleteAppointmentInput marks a scheduled visit as completed. Stock was
// already deducted when the appointment was created, so no new usage entry is
// made. When AdministeredOn is set, the linked usage rows move to that actual
// date, which requires both the original and target months to be open.
type CompleteAppointmentInput struct {
	ID             int64
	AdministeredOn string
}

func (s *AppointmentService) Complete(ctx context.Context, input CompleteAppointmentInput) (AppointmentResult, error) {
	if input.ID <= 0 {
		return AppointmentResult{}, notFound("Appointment")
	}
	var result AppointmentResult
	err := s.store.WithTx(ctx, func(q generated.Querier) error {
		existing, err := q.GetAppointment(ctx, input.ID)
		if err != nil {
			return requireFound(err, "Appointment")
		}
		if existing.Status != appointmentScheduled {
			return domain.NewError(domain.CodeConflict, "", "لا يمكن إكمال موعد مكتمل أو ملغى.")
		}
		patient, err := q.GetPatient(ctx, existing.PatientID)
		if err != nil {
			return requireFound(err, "Patient")
		}
		if patient.IsArchived == 1 {
			return domain.NewError(domain.CodeArchived, "patient", "لا يمكن إكمال موعد لمريض مؤرشف.")
		}
		items0, _ := q.ListAppointmentItemsByAppointment(ctx, input.ID)
		for _, it := range items0 {
			v, err := q.GetVaccine(ctx, it.VaccineID)
			if err != nil {
				return requireFound(err, "Vaccine")
			}
			if v.IsArchived == 1 {
				return domain.NewError(domain.CodeArchived, "vaccine", "لا يمكن إكمال موعد للقاح مؤرشف.")
			}
		}
		now := s.timestamp()
		updated, err := q.CompleteAppointment(ctx, generated.CompleteAppointmentParams{
			CompletedAt: sql.NullString{String: now, Valid: true},
			UpdatedAt:   now,
			ID:          input.ID,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر إكمال الموعد.")
		}
		if strings.TrimSpace(input.AdministeredOn) != "" {
			date, err := domain.ParseDate(input.AdministeredOn)
			if err != nil {
				return err
			}
			administeredOn := domain.FormatDate(date)
			for _, it := range items0 {
				if !it.UsageEntryID.Valid {
					continue
				}
				entry, err := q.GetUsageEntry(ctx, it.UsageEntryID.Int64)
				if err != nil {
					return requireFound(err, "Usage entry")
				}
				if entry.AdministeredOn == administeredOn {
					continue
				}
				if _, err := s.usage.updateInTx(ctx, q, UpdateUsageInput{
					ID:             entry.ID,
					PatientID:      entry.PatientID,
					AdministeredOn: administeredOn,
					Quantity:       entry.Quantity,
					Note:           entry.Note,
					Reason:         "إكمال الموعد بتاريخ الإعطاء الفعلي.",
				}, true); err != nil {
					return err
				}
			}
		}
		result.Appointment = updated
		items, err := q.ListAppointmentItemsByAppointment(ctx, input.ID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.UsageEntryID.Valid {
				entry, err := q.GetUsageEntry(ctx, item.UsageEntryID.Int64)
				if err == nil {
					inv, _ := requireInitializedOpen(ctx, q, entry.VaccineID, entry.Month)
					usage, _ := q.SumActiveUsageForVaccineMonth(ctx, generated.SumActiveUsageForVaccineMonthParams{VaccineID: entry.VaccineID, Month: entry.Month})
					balance, _ := domain.ComputeBalance(inv.OpeningBalance, inv.StockReceived, usage)
					result.Usage = append(result.Usage, UsageResult{Entry: entry, Inventory: inv, UsageTotal: usage, Balance: balance})
				}
			}
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventAppointmentCompleted,
			EntityType: "appointment",
			EntityID:   idString(updated.ID),
			Metadata:   map[string]any{"patient_id": updated.PatientID},
		})
	})
	if err != nil {
		return AppointmentResult{}, err
	}
	return result, nil
}

// Cancel closes a scheduled appointment and voids its already-deducted usage.
func (s *AppointmentService) Cancel(ctx context.Context, input CancelAppointmentInput) (generated.Appointment, error) {
	if input.ID <= 0 {
		return generated.Appointment{}, notFound("Appointment")
	}
	reason, err := domain.RequireReason(input.Reason, "reason", "سبب الإلغاء")
	if err != nil {
		return generated.Appointment{}, err
	}
	var cancelled generated.Appointment
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		existing, err := q.GetAppointment(ctx, input.ID)
		if err != nil {
			return requireFound(err, "Appointment")
		}
		if existing.Status != appointmentScheduled {
			return domain.NewError(domain.CodeConflict, "", "لا يمكن إلغاء موعد مكتمل أو ملغى.")
		}
		now := s.timestamp()
		updated, err := q.CancelAppointment(ctx, generated.CancelAppointmentParams{
			CancelReason: reason,
			CancelledAt:  sql.NullString{String: now, Valid: true},
			UpdatedAt:    now,
			ID:           input.ID,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر إلغاء الموعد.")
		}
		cancelled = updated
		items, err := q.ListAppointmentItemsByAppointment(ctx, input.ID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.UsageEntryID.Valid {
				if _, err := q.VoidUsageEntry(ctx, generated.VoidUsageEntryParams{
					VoidedAt:   sql.NullString{String: now, Valid: true},
					VoidReason: reason,
					UpdatedAt:  now,
					ID:         item.UsageEntryID.Int64,
				}); err != nil {
					return mapPersistenceError(err, "تعذر إلغاء قيد الاستخدام.")
				}
			}
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventAppointmentCancelled,
			EntityType: "appointment",
			EntityID:   idString(updated.ID),
			Reason:     reason,
			Metadata:   map[string]any{"patient_id": updated.PatientID},
		})
	})
	if err != nil {
		return generated.Appointment{}, err
	}
	return cancelled, nil
}

// List returns joined appointment rows for the private UI.
func (s *AppointmentService) List(ctx context.Context, filter AppointmentFilter) ([]AppointmentView, error) {
	normalized, err := normalizeAppointmentFilter(filter)
	if err != nil {
		return nil, err
	}
	return s.listViews(ctx, normalized, 0, 0, 500)
}

// ForPatient returns the patient's complete appointment history.
func (s *AppointmentService) ForPatient(ctx context.Context, patientID int64) ([]AppointmentView, error) {
	if patientID <= 0 {
		return nil, notFound("Patient")
	}
	return s.listViews(ctx, AppointmentFilter{}, patientID, 0, 200)
}

// Get returns one joined appointment row.
func (s *AppointmentService) Get(ctx context.Context, id int64) (AppointmentView, error) {
	if id <= 0 {
		return AppointmentView{}, notFound("Appointment")
	}
	views, err := s.listViews(ctx, AppointmentFilter{}, 0, id, 1)
	if err != nil {
		return AppointmentView{}, err
	}
	if len(views) == 0 {
		return AppointmentView{}, notFound("Appointment")
	}
	return views[0], nil
}

func (s *AppointmentService) listViews(ctx context.Context, filter AppointmentFilter, patientID, appointmentID, limit int64) ([]AppointmentView, error) {
	rows, err := s.store.Queries().ListAppointments(ctx, generated.ListAppointmentsParams{
		AppointmentID: appointmentID,
		PatientID:     patientID,
		Status:        filter.Status,
		FromDate:      filter.FromDate,
		ToDate:        filter.ToDate,
		LimitCount:    limit,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	itemRows := []generated.ListAppointmentItemsByAppointmentsRow{}
	if len(ids) > 0 {
		itemRows, err = s.store.Queries().ListAppointmentItemsByAppointments(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	itemsByAppointment := make(map[int64][]AppointmentItemView)
	for _, item := range itemRows {
		itemsByAppointment[item.AppointmentID] = append(itemsByAppointment[item.AppointmentID], AppointmentItemView{
			ID:                  item.ID,
			AppointmentID:       item.AppointmentID,
			VaccineID:           item.VaccineID,
			VaccineName:         item.VaccineName,
			VaccineCode:         item.VaccineCode,
			VaccineUnit:         item.VaccineUnit,
			VaccineArchived:     item.VaccineArchived == 1,
			Quantity:            item.Quantity,
			UsageEntryID:        item.UsageEntryID.Int64,
			UsageVoided:         item.UsageVoidedAt.Valid,
			UsageVoidReason:     item.UsageVoidReason.String,
			UsageAdministeredOn: item.UsageAdministeredOn.String,
			UsageQuantity:       item.UsageQuantity.Int64,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		})
	}
	views := make([]AppointmentView, 0, len(rows))
	for _, row := range rows {
		completedAt := ""
		if row.CompletedAt.Valid {
			completedAt = row.CompletedAt.String
		}
		cancelledAt := ""
		if row.CancelledAt.Valid {
			cancelledAt = row.CancelledAt.String
		}
		views = append(views, AppointmentView{
			ID:              row.ID,
			PatientID:       row.PatientID,
			PatientName:     row.PatientName,
			PatientCode:     row.PatientCode,
			PatientArchived: row.PatientArchived == 1,
			ScheduledOn:     row.ScheduledOn,
			ScheduledTime:   row.ScheduledTime,
			Note:            row.Note,
			Status:          row.Status,
			CancelReason:    row.CancelReason,
			CompletedAt:     completedAt,
			CancelledAt:     cancelledAt,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
			Items:           itemsByAppointment[row.ID],
		})
	}
	return views, nil
}

func itemMetadata(items []AppointmentItemInput) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"vaccine_id": item.VaccineID,
			"quantity":   item.Quantity,
		})
	}
	return out
}

func administeredMonth(scheduledOn string) string {
	if len(scheduledOn) >= 7 {
		return scheduledOn[:7]
	}
	return ""
}
