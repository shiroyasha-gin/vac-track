package service

import (
	"context"
	"database/sql"
	"strings"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

// PatientInput deliberately contains the minimum fields required by the
// product. Patient names are local-only and are never used in public snapshots
// or audit metadata.
type PatientInput struct {
	FullName    string
	PatientCode string
}

// PatientService manages local patient records.
type PatientService struct {
	base
}

func normalizePatientInput(input PatientInput) (PatientInput, error) {
	var err error
	input.FullName, err = domain.RequiredText(input.FullName, "full_name", "اسم المريض", 120)
	if err != nil {
		return input, err
	}
	input.PatientCode, err = domain.OptionalText(input.PatientCode, "patient_code", "رمز المريض", 40)
	if err != nil {
		return input, err
	}
	return input, nil
}

// Create adds an active patient. Only a name is required.
func (s *PatientService) Create(ctx context.Context, input PatientInput) (generated.Patient, error) {
	normalized, err := normalizePatientInput(input)
	if err != nil {
		return generated.Patient{}, err
	}
	var created generated.Patient
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		now := s.timestamp()
		created, err = q.CreatePatient(ctx, generated.CreatePatientParams{
			FullName:    normalized.FullName,
			PatientCode: normalized.PatientCode,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		if err != nil {
			return mapPersistenceError(err, "يوجد مريض بهذا الرمز مسبقاً.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventPatientCreated,
			EntityType: "patient",
			EntityID:   idString(created.ID),
			Metadata:   map[string]any{"has_code": created.PatientCode != ""},
		})
	})
	if err != nil {
		return generated.Patient{}, err
	}
	return created, nil
}

// Update edits a patient's local record. Audit metadata intentionally omits
// the patient's name.
func (s *PatientService) Update(ctx context.Context, id int64, input PatientInput) (generated.Patient, error) {
	if id <= 0 {
		return generated.Patient{}, notFound("Patient")
	}
	normalized, err := normalizePatientInput(input)
	if err != nil {
		return generated.Patient{}, err
	}
	var updated generated.Patient
	err = s.store.WithTx(ctx, func(q generated.Querier) error {
		if _, err := q.GetPatient(ctx, id); err != nil {
			return requireFound(err, "Patient")
		}
		updated, err = q.UpdatePatient(ctx, generated.UpdatePatientParams{
			FullName:    normalized.FullName,
			PatientCode: normalized.PatientCode,
			UpdatedAt:   s.timestamp(),
			ID:          id,
		})
		if err != nil {
			return mapPersistenceError(err, "يوجد مريض بهذا الرمز مسبقاً.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventPatientUpdated,
			EntityType: "patient",
			EntityID:   idString(id),
			Metadata:   map[string]any{"has_code": updated.PatientCode != ""},
		})
	})
	if err != nil {
		return generated.Patient{}, err
	}
	return updated, nil
}

// SetArchived archives or reactivates a patient. Historical usage remains.
func (s *PatientService) SetArchived(ctx context.Context, id int64, archived bool) (generated.Patient, error) {
	if id <= 0 {
		return generated.Patient{}, notFound("Patient")
	}
	var result generated.Patient
	err := s.store.WithTx(ctx, func(q generated.Querier) error {
		if _, err := q.GetPatient(ctx, id); err != nil {
			return requireFound(err, "Patient")
		}
		flag := int64(0)
		archivedAt := sql.NullString{}
		eventType := audit.EventPatientReactivated
		if archived {
			flag = 1
			archivedAt = sql.NullString{String: s.timestamp(), Valid: true}
			eventType = audit.EventPatientArchived
		}
		var err error
		result, err = q.SetPatientArchived(ctx, generated.SetPatientArchivedParams{
			IsArchived: flag,
			ArchivedAt: archivedAt,
			UpdatedAt:  s.timestamp(),
			ID:         id,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر تحديث المريض.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       eventType,
			EntityType: "patient",
			EntityID:   idString(id),
			Metadata:   map[string]any{"usage_retained": true},
		})
	})
	if err != nil {
		return generated.Patient{}, err
	}
	return result, nil
}

// PatientPage is one bounded page of the local patient directory.
type PatientPage struct {
	Items  []generated.Patient
	Total  int64
	Limit  int64
	Offset int64
}

// List searches locally by name or patient code. No remote request is made and
// the result is always bounded so a large directory cannot freeze the UI.
func (s *PatientService) List(ctx context.Context, search string, includeArchived bool, limit, offset int64) (PatientPage, error) {
	search = strings.TrimSpace(search)
	if len([]rune(search)) > 80 {
		return PatientPage{}, domain.NewError(domain.CodeInvalidInput, "search", "نص البحث طويل جداً.")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	flag := int64(0)
	if includeArchived {
		flag = 1
	}
	q := s.store.Queries()
	total, err := q.CountPatients(ctx, generated.CountPatientsParams{
		IncludeArchived: flag,
		Search:          search,
	})
	if err != nil {
		return PatientPage{}, err
	}
	items, err := q.ListPatients(ctx, generated.ListPatientsParams{
		IncludeArchived: flag,
		Search:          search,
		LimitCount:      limit,
		OffsetRows:      offset,
	})
	if err != nil {
		return PatientPage{}, err
	}
	return PatientPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

// SearchActive returns a bounded set of active patients for search-first pickers.
func (s *PatientService) SearchActive(ctx context.Context, search string, limit int64) ([]generated.Patient, error) {
	search = strings.TrimSpace(search)
	if len([]rune(search)) > 80 {
		return nil, domain.NewError(domain.CodeInvalidInput, "search", "نص البحث طويل جداً.")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.store.Queries().SearchActivePatients(ctx, generated.SearchActivePatientsParams{
		Search:     search,
		LimitCount: limit,
	})
}

// Get returns one patient.
func (s *PatientService) Get(ctx context.Context, id int64) (generated.Patient, error) {
	p, err := s.store.Queries().GetPatient(ctx, id)
	return p, requireFound(err, "Patient")
}
