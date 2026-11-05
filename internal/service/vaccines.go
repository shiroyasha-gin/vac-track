package service

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

// VaccineInput is validated by the vaccine service.
type VaccineInput struct {
	Name        string
	Code        string
	Description string
	Category    string
	Unit        string
}

// VaccineService manages the catalog.
type VaccineService struct {
	base
}

func normalizeVaccineInput(input VaccineInput) (VaccineInput, error) {
	var err error
	input.Name, err = domain.RequiredText(input.Name, "name", "اسم اللقاح", 120)
	if err != nil {
		return input, err
	}
	input.Code, err = domain.OptionalText(input.Code, "code", "رمز اللقاح", 40)
	if err != nil {
		return input, err
	}
	input.Category, err = domain.OptionalText(input.Category, "category", "الفئة", 60)
	if err != nil {
		return input, err
	}
	input.Description, err = domain.OptionalText(input.Description, "description", "الوصف", 500)
	if err != nil {
		return input, err
	}
	if input.Unit == "" {
		input.Unit = "جرعة"
	}
	input.Unit, err = domain.OptionalText(input.Unit, "unit", "الوحدة", 20)
	if err != nil {
		return input, err
	}
	if input.Unit == "" {
		input.Unit = "جرعة"
	}
	return input, nil
}

// Create adds an active vaccine and audits the creation.
func (s *VaccineService) Create(ctx context.Context, input VaccineInput) (generated.Vaccine, error) {
	normalized, err := normalizeVaccineInput(input)
	if err != nil {
		return generated.Vaccine{}, err
	}
	var created generated.Vaccine
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		if _, err := q.GetVaccineByName(ctx, normalized.Name); err == nil {
			return domainConflict("يوجد لقاح بهذا الاسم مسبقاً (بما في ذلك اللقاحات المؤرشفة).")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if normalized.Code != "" {
			if _, err := q.GetVaccineByCode(ctx, normalized.Code); err == nil {
				return domainConflict("يوجد لقاح بهذا الرمز مسبقاً.")
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		now := s.timestamp()
		created, err = q.CreateVaccine(ctx, generated.CreateVaccineParams{
			Name:        normalized.Name,
			Code:        normalized.Code,
			Description: normalized.Description,
			Category:    normalized.Category,
			Unit:        normalized.Unit,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		if err != nil {
			return mapPersistenceError(err, "يوجد لقاح بهذا الاسم أو الرمز مسبقاً.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventVaccineCreated,
			EntityType: "vaccine",
			EntityID:   idString(created.ID),
			Metadata: map[string]any{
				"name": created.Name,
				"code": created.Code,
			},
		})
	})
	if err != nil {
		return generated.Vaccine{}, err
	}
	return created, nil
}

// Update edits catalog metadata. Historical usage rows are never changed.
func (s *VaccineService) Update(ctx context.Context, id int64, input VaccineInput) (generated.Vaccine, error) {
	if id <= 0 {
		return generated.Vaccine{}, notFound("Vaccine")
	}
	normalized, err := normalizeVaccineInput(input)
	if err != nil {
		return generated.Vaccine{}, err
	}
	var updated generated.Vaccine
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		existing, err := q.GetVaccine(ctx, id)
		if err != nil {
			return requireFound(err, "Vaccine")
		}
		nameCheck, err := q.GetVaccineByName(ctx, normalized.Name)
		if err == nil && nameCheck.ID != id {
			return domainConflict("A vaccine with this name already exists.")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if normalized.Code != "" {
			codeCheck, err := q.GetVaccineByCode(ctx, normalized.Code)
			if err == nil && codeCheck.ID != id {
				return domainConflict("يوجد لقاح بهذا الرمز مسبقاً.")
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		updated, err = q.UpdateVaccine(ctx, generated.UpdateVaccineParams{
			Name:        normalized.Name,
			Code:        normalized.Code,
			Description: normalized.Description,
			Category:    normalized.Category,
			Unit:        normalized.Unit,
			UpdatedAt:   s.timestamp(),
			ID:          id,
		})
		if err != nil {
			return mapPersistenceError(err, "يوجد لقاح بهذا الاسم أو الرمز مسبقاً.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventVaccineUpdated,
			EntityType: "vaccine",
			EntityID:   idString(id),
			Metadata: map[string]any{
				"before": map[string]any{"name": existing.Name, "code": existing.Code, "unit": existing.Unit},
				"after":  map[string]any{"name": updated.Name, "code": updated.Code, "unit": updated.Unit},
			},
		})
	})
	if err != nil {
		return generated.Vaccine{}, err
	}
	return updated, nil
}

// SetArchived archives or reactivates a vaccine. Archived vaccines remain in
// historical reports but are rejected for new routine usage.
func (s *VaccineService) SetArchived(ctx context.Context, id int64, archived bool) (generated.Vaccine, error) {
	if id <= 0 {
		return generated.Vaccine{}, notFound("Vaccine")
	}
	var result generated.Vaccine
	err := s.store.WithTx(ctx, func(q generated.Querier) error {
		existing, err := q.GetVaccine(ctx, id)
		if err != nil {
			return requireFound(err, "Vaccine")
		}
		flag := int64(0)
		archivedAt := sql.NullString{}
		eventType := audit.EventVaccineReactivated
		if archived {
			flag = 1
			archivedAt = sql.NullString{String: s.timestamp(), Valid: true}
			eventType = audit.EventVaccineArchived
		}
		result, err = q.SetVaccineArchived(ctx, generated.SetVaccineArchivedParams{
			IsArchived: flag,
			ArchivedAt: archivedAt,
			UpdatedAt:  s.timestamp(),
			ID:         id,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر تحديث اللقاح.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       eventType,
			EntityType: "vaccine",
			EntityID:   idString(id),
			Metadata:   map[string]any{"name": existing.Name},
		})
	})
	if err != nil {
		return generated.Vaccine{}, err
	}
	return result, nil
}

// Delete permanently removes an archived vaccine only when it has no usage or
// monthly inventory history.
func (s *VaccineService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return notFound("Vaccine")
	}
	return s.store.WithTx(ctx, func(q generated.Querier) error {
		vaccine, err := q.GetVaccine(ctx, id)
		if err != nil {
			return requireFound(err, "Vaccine")
		}
		if vaccine.IsArchived == 0 {
			return domain.NewError(domain.CodeUnsafeDelete, "", "أرشف اللقاح قبل حذفه نهائياً.")
		}
		history, err := q.CountVaccineHistory(ctx, generated.CountVaccineHistoryParams{
			VaccineID:   id,
			VaccineID_2: id,
		})
		if err != nil {
			return err
		}
		if history > 0 {
			return domain.NewError(domain.CodeUnsafeDelete, "", "لهذا اللقاح مخزون أو سجل استخدام ولا يمكن حذفه. أبقِه مؤرشفاً بدلاً من ذلك.")
		}
		if err := q.DeleteVaccine(ctx, id); err != nil {
			return mapPersistenceError(err, "تعذر حذف اللقاح.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventVaccineDeleted,
			EntityType: "vaccine",
			EntityID:   idString(id),
			Metadata:   map[string]any{"name": vaccine.Name},
		})
	})
}

// List returns active vaccines, optionally including archived ones.
func (s *VaccineService) List(ctx context.Context, includeArchived bool) ([]generated.Vaccine, error) {
	flag := int64(0)
	if includeArchived {
		flag = 1
	}
	return s.store.Queries().ListVaccines(ctx, flag)
}

// Get returns one vaccine.
func (s *VaccineService) Get(ctx context.Context, id int64) (generated.Vaccine, error) {
	v, err := s.store.Queries().GetVaccine(ctx, id)
	return v, requireFound(err, "Vaccine")
}

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}
