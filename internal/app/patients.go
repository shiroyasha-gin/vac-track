package app

import (
	"vaccineshit/internal/service"
)

// SearchPatients returns one bounded page of local records by name or patient
// code. The search is never sent over the network.
func (a *App) SearchPatients(search string, includeArchived bool, limit, offset int64) (PatientPageDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return PatientPageDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	page, err := a.services.Patients.List(ctx, search, includeArchived, limit, offset)
	if err != nil {
		return PatientPageDTO{}, a.userError(err)
	}
	items := make([]PatientDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, patientDTO(item))
	}
	return PatientPageDTO{
		Items: items, Total: page.Total, Limit: page.Limit, Offset: page.Offset,
	}, nil
}

// SearchPatientsLimited returns a small active-patient result set for pickers.
func (a *App) SearchPatientsLimited(search string, limit int64) ([]PatientDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Patients.SearchActive(ctx, search, limit)
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
	if err := a.authReadyLocked(); err != nil {
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
	result := Result("تمت إضافة المريض.")
	result.EntityID = created.ID
	return result
}

// UpdatePatient edits a local patient record.
func (a *App) UpdatePatient(id int64, input PatientInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
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
	result := Result("تم حفظ بيانات المريض.")
	result.EntityID = updated.ID
	return result
}

// SetPatientArchived archives or reactivates a patient. Usage history remains.
func (a *App) SetPatientArchived(id int64, archived bool) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if _, err := a.services.Patients.SetArchived(ctx, id, archived); err != nil {
		return a.actionError(err)
	}
	if archived {
		return Result("تمت أرشفة المريض مع الاحتفاظ بسجل الاستخدامات محلياً.")
	}
	return Result("تمت إعادة تنشيط المريض.")
}

// PatientDetail returns one complete local patient file: identity, dose
// history, vaccine coverage and appointments. It never leaves the workstation.
func (a *App) PatientDetail(patientID int64) (PatientDetailDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return PatientDetailDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	patient, err := a.services.Patients.Get(ctx, patientID)
	if err != nil {
		return PatientDetailDTO{}, a.userError(err)
	}
	usageItems, err := a.services.Usage.ForPatient(ctx, patientID)
	if err != nil {
		return PatientDetailDTO{}, a.userError(err)
	}
	appointmentItems, err := a.services.Appointments.ForPatient(ctx, patientID)
	if err != nil {
		return PatientDetailDTO{}, a.userError(err)
	}

	usage := make([]UsageRowDTO, 0, len(usageItems))
	summary := PatientSummaryDTO{}
	vaccines := make(map[int64]struct{})
	for _, item := range usageItems {
		usage = append(usage, usageRowPatientDTO(item))
		if item.VoidedAt.Valid {
			continue
		}
		summary.TotalDoses += item.Quantity
		vaccines[item.VaccineID] = struct{}{}
		if summary.LastDoseOn == "" {
			summary.LastDoseOn = item.AdministeredOn
		}
	}
	summary.VaccineCount = len(vaccines)

	appointments := make([]AppointmentDTO, 0, len(appointmentItems))
	for _, item := range appointmentItems {
		appointments = append(appointments, appointmentDTO(item))
		switch item.Status {
		case "scheduled":
			summary.ScheduledAppointments++
			if summary.NextAppointmentOn == "" {
				summary.NextAppointmentOn = item.ScheduledOn
			}
		case "completed":
			summary.CompletedAppointments++
		}
	}

	return PatientDetailDTO{
		Patient: patientDTO(patient), Summary: summary,
		Usage: usage, Appointments: appointments,
	}, nil
}

// PatientUsage returns private local usage history for one patient.
func (a *App) PatientUsage(patientID int64) ([]UsageRowDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
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
