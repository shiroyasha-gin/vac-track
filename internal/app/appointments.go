package app

import (
	"vaccineshit/internal/service"
)

// ListAppointments returns the private appointment schedule, including its
// linked usage status. No appointment data is published.
func (a *App) ListAppointments(filter AppointmentFilterDTO) ([]AppointmentDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Appointments.List(ctx, service.AppointmentFilter{
		Status: filter.Status, FromDate: filter.FromDate, ToDate: filter.ToDate,
	})
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]AppointmentDTO, 0, len(items))
	for _, item := range items {
		out = append(out, appointmentDTO(item))
	}
	return out, nil
}

// CreateAppointment schedules a visit and deducts the selected vaccines from stock immediately.
func (a *App) CreateAppointment(input AppointmentInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items := make([]service.AppointmentItemInput, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, service.AppointmentItemInput{VaccineID: item.VaccineID, Quantity: item.Quantity})
	}
	created, err := a.services.Appointments.Create(ctx, service.AppointmentInput{
		PatientID: input.PatientID, ScheduledOn: input.ScheduledOn,
		ScheduledTime: input.ScheduledTime, Note: input.Note, Items: items,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تم حجز الموعد وخصم الجرعات من مخزون الشهر المحدد.")
	result.EntityID = created.ID
	return result
}

// CreateTakenAppointment records an appointment that was already administered.
// It is created as completed and deducts the selected vaccines from stock.
func (a *App) CreateTakenAppointment(input AppointmentInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items := make([]service.AppointmentItemInput, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, service.AppointmentItemInput{VaccineID: item.VaccineID, Quantity: item.Quantity})
	}
	created, err := a.services.Appointments.CreateTaken(ctx, service.AppointmentInput{
		PatientID: input.PatientID, ScheduledOn: input.ScheduledOn,
		ScheduledTime: input.ScheduledTime, Note: input.Note, Items: items,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تم تسجيل الموعد واستهلاك الجرعات من المخزون.")
	result.EntityID = created.ID
	return result
}

// UpdateAppointment edits a scheduled visit and settles its linked usage rows
// so quantities, patient and date stay in sync with the stock ledger.
func (a *App) UpdateAppointment(id int64, input UpdateAppointmentInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items := make([]service.AppointmentItemInput, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, service.AppointmentItemInput{VaccineID: item.VaccineID, Quantity: item.Quantity})
	}
	updated, err := a.services.Appointments.Update(ctx, service.UpdateAppointmentInput{
		ID: id, PatientID: input.PatientID, ScheduledOn: input.ScheduledOn,
		ScheduledTime: input.ScheduledTime, Note: input.Note, Items: items, Reason: input.Reason,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تم تحديث الموعد وتسوية الجرعات المرتبطة به.")
	result.EntityID = updated.ID
	return result
}

// CompleteAppointment marks a scheduled visit as completed. Stock was already
// deducted when the appointment was created; a provided administered date moves
// the linked usage rows to the actual date.
func (a *App) CompleteAppointment(input CompleteAppointmentInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	completed, err := a.services.Appointments.Complete(ctx, service.CompleteAppointmentInput{
		ID: input.ID, AdministeredOn: input.AdministeredOn,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تم إكمال الموعد.")
	result.EntityID = completed.Appointment.ID
	return result
}

// CancelAppointment cancels a scheduled visit and voids its reserved doses.
func (a *App) CancelAppointment(input CancelAppointmentInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	cancelled, err := a.services.Appointments.Cancel(ctx, service.CancelAppointmentInput{
		ID: input.ID, Reason: input.Reason,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تم إلغاء الموعد وإرجاع الجرعات المحجوزة إلى المخزون.")
	result.EntityID = cancelled.ID
	return result
}
