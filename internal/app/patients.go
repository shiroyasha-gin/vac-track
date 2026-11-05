package app

import (
	"vaccineshit/internal/service"
)

// SearchPatients searches local records by name or patient code. The search is
// never sent over the network.
func (a *App) SearchPatients(search string, includeArchived bool) ([]PatientDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Patients.List(ctx, search, includeArchived)
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]PatientDTO, 0, len(items))
	for _, item := range items {
		out = append(out, patientDTO(item))
	}
	return out, nil
}

// ListActivePatients returns non-archived patients for the usage form.
func (a *App) ListActivePatients() ([]PatientDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Patients.ListActive(ctx)
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]PatientDTO, 0, len(items))
	for _, item := range items {
		out = append(out, patientDTO(item))
	}
	return out, nil
}

// CreatePatient adds a local patient. Only a name is required.
func (a *App) CreatePatient(input PatientInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	created, err := a.services.Patients.Create(ctx, service.PatientInput{
		FullName: input.FullName, PatientCode: input.PatientCode,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("Patient added.")
	result.EntityID = created.ID
	return result
}

// UpdatePatient edits a local patient record.
func (a *App) UpdatePatient(id int64, input PatientInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	updated, err := a.services.Patients.Update(ctx, id, service.PatientInput{
		FullName: input.FullName, PatientCode: input.PatientCode,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("Patient saved.")
	result.EntityID = updated.ID
	return result
}

// SetPatientArchived archives or reactivates a patient. Usage history remains.
func (a *App) SetPatientArchived(id int64, archived bool) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if _, err := a.services.Patients.SetArchived(ctx, id, archived); err != nil {
		return a.actionError(err)
	}
	if archived {
		return Result("Patient archived. Historical usage is retained locally.")
	}
	return Result("Patient reactivated.")
}

// PatientUsage returns private local usage history for one patient.
func (a *App) PatientUsage(patientID int64) ([]UsageRowDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Usage.ForPatient(ctx, patientID)
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]UsageRowDTO, 0, len(items))
	for _, item := range items {
		out = append(out, usageRowPatientDTO(item))
	}
	return out, nil
}
