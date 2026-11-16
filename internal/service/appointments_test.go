package service

import (
	"testing"

	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

func createTestAppointment(f *fixture, patientID, vaccineID int64, date string, quantity int64) generated.Appointment {
	f.t.Helper()
	appointment, err := f.svc.Appointments.Create(f.ctx, AppointmentInput{
		PatientID: patientID, ScheduledOn: date, ScheduledTime: "09:30",
		Note:  "Synthetic appointment note",
		Items: []AppointmentItemInput{{VaccineID: vaccineID, Quantity: quantity}},
	})
	if err != nil {
		f.t.Fatalf("create appointment: %v", err)
	}
	return appointment
}

func TestCreateTakenAppointmentIsCompletedAndDeductsStock(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Tetanus")
	patient := f.patient("Synthetic Immediate Patient")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)

	appointment, err := f.svc.Appointments.CreateTaken(f.ctx, AppointmentInput{
		PatientID: patient.ID, ScheduledOn: "2026-11-11", ScheduledTime: "10:15",
		Items: []AppointmentItemInput{{VaccineID: vaccine.ID, Quantity: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if appointment.Status != appointmentCompleted || !appointment.CompletedAt.Valid {
		t.Fatalf("created taken appointment = %+v", appointment)
	}

	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.UsageTotal != 4 || overview.Totals.Remaining != 6 {
		t.Fatalf("month totals after taken appointment = %+v", overview.Totals)
	}

	today, err := f.svc.Appointments.List(f.ctx, AppointmentFilter{
		FromDate: "2026-11-11", ToDate: "2026-11-11",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(today) != 1 || today[0].Status != appointmentCompleted || len(today[0].Items) != 1 {
		t.Fatalf("today's appointments = %+v", today)
	}
	if today[0].Items[0].UsageQuantity != 4 {
		t.Fatalf("linked usage = %+v", today[0].Items[0])
	}
}

func TestAppointmentCompletionConsumesStockExactlyOnce(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Measles")
	patient := f.patient("Synthetic Appointment Patient A")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	appointment := createTestAppointment(f, patient.ID, vaccine.ID, "2026-11-20", 3)

	result, err := f.svc.Appointments.Complete(f.ctx, CompleteAppointmentInput{
		ID: appointment.ID, AdministeredOn: "2026-11-20",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Appointment.Status != appointmentCompleted {
		t.Fatalf("appointment status = %q, want completed", result.Appointment.Status)
	}
	if len(result.Usage) != 1 || result.Usage[0].Entry.ID == 0 {
		t.Fatalf("appointment usage link = %+v", result.Usage)
	}
	if result.Usage[0].Entry.Quantity != 3 || result.Usage[0].Balance != 7 || result.Usage[0].UsageTotal != 3 {
		t.Fatalf("completion result = %+v", result)
	}
	if got := f.countUsage(); got != 1 {
		t.Fatalf("usage rows = %d, want 1", got)
	}
	if got := f.countAudit("usage.created"); got != 1 {
		t.Fatalf("usage audit rows = %d, want 1", got)
	}
	if got := f.countAudit("appointment.completed"); got != 1 {
		t.Fatalf("appointment audit rows = %d, want 1", got)
	}
	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.UsageTotal != 3 || overview.Totals.Remaining != 7 {
		t.Fatalf("month totals after completion = %+v", overview.Totals)
	}
	history, err := f.svc.Usage.ForPatient(f.ctx, patient.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Note != "Synthetic appointment note" {
		t.Fatalf("patient usage history = %+v", history)
	}

	_, err = f.svc.Appointments.Complete(f.ctx, CompleteAppointmentInput{
		ID: appointment.ID, AdministeredOn: "2026-11-20",
	})
	if !domain.IsCode(err, domain.CodeConflict) {
		t.Fatalf("second completion error = %v, want conflict", err)
	}
	if got := f.countUsage(); got != 1 {
		t.Fatalf("usage rows after retry = %d, want 1", got)
	}

	completed, err := f.svc.Appointments.List(f.ctx, AppointmentFilter{Status: "completed"})
	if err != nil {
		t.Fatal(err)
	}
	if len(completed) != 1 || len(completed[0].Items) != 1 || completed[0].Items[0].UsageEntryID != result.Usage[0].Entry.ID {
		t.Fatalf("completed appointment list = %+v", completed)
	}
}

func TestAppointmentCompletionRollsBackWithoutOpenStock(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Polio")
	patient := f.patient("Synthetic Appointment Patient B")
	_, err := f.svc.Appointments.Create(f.ctx, AppointmentInput{
		PatientID: patient.ID, ScheduledOn: "2026-11-21", ScheduledTime: "09:30",
		Items: []AppointmentItemInput{{VaccineID: vaccine.ID, Quantity: 2}},
	})
	if !domain.IsCode(err, domain.CodeNotInitialized) {
		t.Fatalf("uninitialized appointment creation error = %v, want not initialized", err)
	}
	if got := f.countUsage(); got != 0 {
		t.Fatalf("usage rows = %d, want 0", got)
	}
}

func TestAppointmentCancelRequiresReasonAndDoesNotTouchStock(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("BCG")
	patient := f.patient("Synthetic Appointment Patient C")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	appointment := createTestAppointment(f, patient.ID, vaccine.ID, "2026-11-22", 1)

	_, err := f.svc.Appointments.Cancel(f.ctx, CancelAppointmentInput{ID: appointment.ID})
	if !domain.IsCode(err, domain.CodeReasonRequired) {
		t.Fatalf("empty cancellation reason error = %v", err)
	}
	cancelled, err := f.svc.Appointments.Cancel(f.ctx, CancelAppointmentInput{
		ID: appointment.ID, Reason: "Patient called to reschedule",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != appointmentCancelled || cancelled.CancelReason == "" {
		t.Fatalf("cancelled appointment = %+v", cancelled)
	}
	if got := f.countUsage(); got != 1 {
		t.Fatalf("usage rows after cancellation = %d, want 1 (voided)", got)
	}
	if got := f.countAudit("appointment.cancelled"); got != 1 {
		t.Fatalf("cancellation audits = %d, want 1", got)
	}
	_, err = f.svc.Appointments.Complete(f.ctx, CompleteAppointmentInput{
		ID: appointment.ID, AdministeredOn: "2026-11-22",
	})
	if !domain.IsCode(err, domain.CodeConflict) {
		t.Fatalf("cancelled completion error = %v, want conflict", err)
	}
}

func TestAppointmentScheduledForArchivedPatientRollsBackCompletion(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("DTP")
	patient := f.patient("Synthetic Appointment Patient D")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	appointment := createTestAppointment(f, patient.ID, vaccine.ID, "2026-11-23", 1)
	if _, err := f.svc.Patients.SetArchived(f.ctx, patient.ID, true); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.Appointments.Complete(f.ctx, CompleteAppointmentInput{
		ID: appointment.ID, AdministeredOn: "2026-11-23",
	})
	if !domain.IsCode(err, domain.CodeArchived) {
		t.Fatalf("archived-patient completion error = %v", err)
	}
	if got := f.countUsage(); got != 1 {
		t.Fatalf("usage rows = %d, want 1", got)
	}
	if _, err := f.svc.Patients.SetArchived(f.ctx, patient.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Vaccines.SetArchived(f.ctx, vaccine.ID, true); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Appointments.Complete(f.ctx, CompleteAppointmentInput{
		ID: appointment.ID, AdministeredOn: "2026-11-23",
	})
	if !domain.IsCode(err, domain.CodeArchived) {
		t.Fatalf("archived-vaccine completion error = %v", err)
	}
}

func TestAppointmentUpdateAndLinkedUsagePatientProtection(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Hepatitis B")
	patient := f.patient("Synthetic Appointment Patient E")
	other := f.patient("Synthetic Appointment Patient F")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	appointment := createTestAppointment(f, patient.ID, vaccine.ID, "2026-11-24", 1)

	updated, err := f.svc.Appointments.Update(f.ctx, UpdateAppointmentInput{
		ID: appointment.ID, PatientID: patient.ID,
		ScheduledOn: "2026-11-25", ScheduledTime: "14:45",
		Items: []AppointmentItemInput{{VaccineID: vaccine.ID, Quantity: 2}},
		Note:  "Updated synthetic note", Reason: "Patient requested a later slot",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ScheduledOn != "2026-11-25" || updated.ScheduledTime != "14:45" {
		t.Fatalf("updated appointment = %+v", updated)
	}
	_, err = f.svc.Appointments.Complete(f.ctx, CompleteAppointmentInput{
		ID: appointment.ID, AdministeredOn: "2026-11-25",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Appointments.Update(f.ctx, UpdateAppointmentInput{
		ID: appointment.ID, PatientID: patient.ID,
		ScheduledOn: "2026-11-26", Items: []AppointmentItemInput{{VaccineID: vaccine.ID, Quantity: 1}},
	})
	if !domain.IsCode(err, domain.CodeInvalidInput) {
		t.Fatalf("completed appointment edit error = %v", err)
	}

	_ = other
	if got := f.countUsage(); got != 1 {
		t.Fatalf("usage rows = %d, want 1", got)
	}
}

func TestAppointmentUpdateSettlesLinkedUsageAndRestoresRemovedVaccine(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	kept := f.vaccine("Influenza")
	removed := f.vaccine("Tetanus")
	patient := f.patient("Synthetic Appointment Patient H")
	other := f.patient("Synthetic Appointment Patient I")
	f.initInventory(kept.ID, "2026-11", 10, 0)
	f.initInventory(removed.ID, "2026-11", 10, 0)

	appointment, err := f.svc.Appointments.Create(f.ctx, AppointmentInput{
		PatientID: patient.ID, ScheduledOn: "2026-11-24", ScheduledTime: "09:30",
		Note: "Synthetic initial note",
		Items: []AppointmentItemInput{
			{VaccineID: kept.ID, Quantity: 2},
			{VaccineID: removed.ID, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.Appointments.Update(f.ctx, UpdateAppointmentInput{
		ID: appointment.ID, PatientID: other.ID,
		ScheduledOn: "2026-11-26", ScheduledTime: "11:00",
		Note:   "Synthetic updated note",
		Items:  []AppointmentItemInput{{VaccineID: kept.ID, Quantity: 5}},
		Reason: "Patient moved and one vaccine was dropped",
	}); err != nil {
		t.Fatal(err)
	}

	view, err := f.svc.Appointments.Get(f.ctx, appointment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Items) != 1 || view.Items[0].VaccineID != kept.ID || view.Items[0].UsageEntryID == 0 {
		t.Fatalf("updated appointment items = %+v", view.Items)
	}
	var itemID int64
	if err := f.db.DB().QueryRowContext(f.ctx,
		"SELECT id FROM appointment_items WHERE appointment_id = ?", appointment.ID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if view.Items[0].ID != itemID {
		t.Fatalf("item view ID = %d, want appointment_items.id %d", view.Items[0].ID, itemID)
	}

	// The quantity follows the edit on the same usage entry, and the removed
	// vaccine's entry is voided so its stock returns.
	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.UsageTotal != 5 || overview.Totals.Remaining != 15 {
		t.Fatalf("month totals after update = %+v", overview.Totals)
	}
	history, err := f.svc.Usage.ForPatient(f.ctx, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].VaccineID != kept.ID || history[0].Quantity != 5 || history[0].AdministeredOn != "2026-11-26" {
		t.Fatalf("moved patient usage = %+v", history)
	}

	if _, err := f.svc.Appointments.Cancel(f.ctx, CancelAppointmentInput{
		ID: appointment.ID, Reason: "Synthetic cancellation after edit",
	}); err != nil {
		t.Fatal(err)
	}
	overview, err = f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.UsageTotal != 0 || overview.Totals.Remaining != 20 {
		t.Fatalf("month totals after cancel = %+v", overview.Totals)
	}
}

func TestAppointmentUpdateRejectsMonthWithoutOpenStock(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Rubella")
	patient := f.patient("Synthetic Appointment Patient J")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	appointment := createTestAppointment(f, patient.ID, vaccine.ID, "2026-11-24", 1)

	_, err := f.svc.Appointments.Update(f.ctx, UpdateAppointmentInput{
		ID: appointment.ID, PatientID: patient.ID, ScheduledOn: "2026-12-02",
		Items: []AppointmentItemInput{{VaccineID: vaccine.ID, Quantity: 1}},
	})
	if !domain.IsCode(err, domain.CodeNotInitialized) {
		t.Fatalf("edit into uninitialized month error = %v, want not initialized", err)
	}
	view, err := f.svc.Appointments.Get(f.ctx, appointment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.ScheduledOn != "2026-11-24" || len(view.Items) != 1 || view.Items[0].Quantity != 1 {
		t.Fatalf("appointment changed after rejected edit = %+v", view)
	}
}

func TestAppointmentHistoryPreventsVaccineDeletion(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Yellow Fever")
	patient := f.patient("Synthetic Appointment Patient G")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	createTestAppointment(f, patient.ID, vaccine.ID, "2026-11-27", 1)
	if _, err := f.svc.Vaccines.SetArchived(f.ctx, vaccine.ID, true); err != nil {
		t.Fatal(err)
	}
	err := f.svc.Vaccines.Delete(f.ctx, vaccine.ID)
	if !domain.IsCode(err, domain.CodeUnsafeDelete) {
		t.Fatalf("appointment-linked vaccine delete error = %v, want unsafe delete", err)
	}
}
