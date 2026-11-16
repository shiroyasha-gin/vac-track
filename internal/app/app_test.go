package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vaccineshit/internal/publicapi"
	"vaccineshit/internal/publishing"
	"vaccineshit/internal/service"
)

const testPublisherToken = "test-publisher-token-0123456789"

type failingKeyring struct{}

func (failingKeyring) Set(service, user, password string) error { return errors.New("no keyring") }
func (failingKeyring) Get(service, user string) (string, error) { return "", errors.New("no keyring") }
func (failingKeyring) Delete(service, user string) error        { return errors.New("no keyring") }

func newTestApp(t *testing.T) *App {
	t.Helper()
	app := newAppAt(t.TempDir(), "test")
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	app.now = func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }
	app.credentials = publishing.NewCredentialsForTest(failingKeyring{}, publishing.DefaultFallbackPath(app.dataDir))
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login seeded operator: %v", err)
	}
	return app
}

func mustOK(t *testing.T, result ActionResult) {
	t.Helper()
	if !result.OK {
		t.Fatalf("action failed: %+v", result)
	}
}

func seedUsage(t *testing.T, app *App) (vaccineID, patientID int64) {
	t.Helper()
	vaccine := app.CreateVaccine(VaccineInputDTO{Name: "Influenza", Code: "FLU", Unit: "dose"})
	mustOK(t, vaccine)
	patient := app.CreatePatient(PatientInputDTO{FullName: "Ada Lovelace", PatientCode: "P-001"})
	mustOK(t, patient)
	mustOK(t, app.InitializeInventory(InitializeInventoryInputDTO{
		VaccineID: vaccine.EntityID, Month: "2026-11", OpeningBalance: 100, StockReceived: 20,
	}))
	mustOK(t, app.CreateUsage(CreateUsageInputDTO{
		PatientID: patient.EntityID, VaccineID: vaccine.EntityID,
		AdministeredOn: "2026-11-03", Quantity: 5, Note: "private clinical note",
	}))
	return vaccine.EntityID, patient.EntityID
}

func TestBootstrapReportsReady(t *testing.T) {
	app := newTestApp(t)
	info := app.Bootstrap()
	if !info.Ready || info.FatalError != "" {
		t.Fatalf("bootstrap = %+v", info)
	}
	if info.LowStockThreshold != 10 {
		t.Fatalf("threshold = %d, want 10", info.LowStockThreshold)
	}
	if info.Version != "test" || info.DataDir == "" {
		t.Fatalf("bootstrap metadata = %+v", info)
	}
}

func TestUsageRequiresInitializedOpenMonthAndStock(t *testing.T) {
	app := newTestApp(t)
	vaccine := app.CreateVaccine(VaccineInputDTO{Name: "Tetanus", Unit: "dose"})
	mustOK(t, vaccine)
	patient := app.CreatePatient(PatientInputDTO{FullName: "Grace Hopper"})
	mustOK(t, patient)

	notInitialized := app.CreateUsage(CreateUsageInputDTO{
		PatientID: patient.EntityID, VaccineID: vaccine.EntityID,
		AdministeredOn: "2026-11-01", Quantity: 1,
	})
	if notInitialized.OK || notInitialized.Code != "inventory_not_initialized" {
		t.Fatalf("uninitialized usage = %+v", notInitialized)
	}
	mustOK(t, app.InitializeInventory(InitializeInventoryInputDTO{
		VaccineID: vaccine.EntityID, Month: "2026-11", OpeningBalance: 3, StockReceived: 0,
	}))
	mustOK(t, app.CreateUsage(CreateUsageInputDTO{
		PatientID: patient.EntityID, VaccineID: vaccine.EntityID,
		AdministeredOn: "2026-11-01", Quantity: 2,
	}))
	excess := app.CreateUsage(CreateUsageInputDTO{
		PatientID: patient.EntityID, VaccineID: vaccine.EntityID,
		AdministeredOn: "2026-11-02", Quantity: 2,
	})
	if excess.OK || excess.Code != "insufficient_stock" {
		t.Fatalf("excess usage = %+v", excess)
	}
	overview, err := app.MonthOverview("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.Remaining != 1 || overview.Totals.UsageTotal != 2 {
		t.Fatalf("overview totals = %+v", overview.Totals)
	}

	mustOK(t, app.CloseMonth("2026-11"))
	closed := app.CreateUsage(CreateUsageInputDTO{
		PatientID: patient.EntityID, VaccineID: vaccine.EntityID,
		AdministeredOn: "2026-11-04", Quantity: 1,
	})
	if closed.OK || closed.Code != "month_closed" {
		t.Fatalf("closed usage = %+v", closed)
	}
	mustOK(t, app.ReopenMonth("2026-11", "Correcting a stock count"))
	mustOK(t, app.CreateUsage(CreateUsageInputDTO{
		PatientID: patient.EntityID, VaccineID: vaccine.EntityID,
		AdministeredOn: "2026-11-04", Quantity: 1,
	}))
}

func TestAppointmentCompletionFlowsIntoStockAndPatientFile(t *testing.T) {
	app := newTestApp(t)
	vaccine := app.CreateVaccine(VaccineInputDTO{Name: "Hepatitis B", Code: "HEPB", Unit: "dose"})
	mustOK(t, vaccine)
	patient := app.CreatePatient(PatientInputDTO{FullName: "Katherine Johnson", PatientCode: "P-042"})
	mustOK(t, patient)
	mustOK(t, app.InitializeInventory(InitializeInventoryInputDTO{
		VaccineID: vaccine.EntityID, Month: "2026-11", OpeningBalance: 10,
	}))

	created := app.CreateAppointment(AppointmentInputDTO{
		PatientID:   patient.EntityID,
		ScheduledOn: "2026-11-20", ScheduledTime: "09:30", Note: "second dose",
		Items: []AppointmentItemInputDTO{{VaccineID: vaccine.EntityID, Quantity: 2}},
	})
	mustOK(t, created)

	before, err := app.PatientDetail(patient.EntityID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Summary.TotalDoses != 2 || before.Summary.ScheduledAppointments != 1 || len(before.Usage) != 1 {
		t.Fatalf("detail before completion = %+v", before)
	}
	if len(before.Appointments) != 1 || before.Appointments[0].Status != "scheduled" {
		t.Fatalf("appointments before completion = %+v", before.Appointments)
	}

	completed := app.CompleteAppointment(CompleteAppointmentInputDTO{ID: created.EntityID, AdministeredOn: "2026-11-20"})
	mustOK(t, completed)
	retry := app.CompleteAppointment(CompleteAppointmentInputDTO{ID: created.EntityID, AdministeredOn: "2026-11-20"})
	if retry.OK || retry.Code != "conflict" {
		t.Fatalf("duplicate completion = %+v", retry)
	}

	after, err := app.PatientDetail(patient.EntityID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Summary.TotalDoses != 2 || after.Summary.VaccineCount != 1 || after.Summary.ScheduledAppointments != 0 || after.Summary.CompletedAppointments != 1 {
		t.Fatalf("detail after completion = %+v", after.Summary)
	}
	if len(after.Usage) != 1 || after.Usage[0].Quantity != 2 || after.Usage[0].AdministeredOn != "2026-11-20" {
		t.Fatalf("usage after completion = %+v", after.Usage)
	}
	if len(after.Appointments) != 1 || len(after.Appointments[0].Items) != 1 || after.Appointments[0].Items[0].UsageEntryID != after.Usage[0].ID || after.Appointments[0].Items[0].UsageQuantity != 2 {
		t.Fatalf("appointment usage link = %+v", after.Appointments)
	}

	overview, err := app.MonthOverview("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.UsageTotal != 2 || overview.Totals.Remaining != 8 {
		t.Fatalf("month totals after completion = %+v", overview.Totals)
	}
}

func TestAggregatePrivacyAndPublicationFlow(t *testing.T) {
	app := newTestApp(t)
	seedUsage(t, app)
	mustOK(t, app.CloseMonth("2026-11"))

	preview, err := app.PublicationPreview("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if preview.MonthState != "closed" || preview.IncludedRows != 1 || !preview.CanPublish {
		t.Fatalf("preview = %+v", preview)
	}
	body, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"Ada Lovelace", "P-001", "private clinical note"} {
		if strings.Contains(string(body), secret) {
			t.Fatalf("publication preview leaked %q", secret)
		}
	}
	if preview.Snapshot.Totals.TotalUsage != 5 || preview.Snapshot.Totals.Remaining != 115 {
		t.Fatalf("snapshot totals = %+v", preview.Snapshot.Totals)
	}

	store, err := publicapi.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := publicapi.NewHandler(publicapi.Config{
		AdminToken: testPublisherToken, Store: store,
		Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	mustOK(t, app.SetPublisherEndpoint(server.URL))
	mustOK(t, app.SetPublisherToken(testPublisherToken))

	result := app.PublishMonth("2026-11", false)
	if !result.OK || result.Version != 1 || result.SnapshotID == "" {
		t.Fatalf("publish = %+v", result)
	}
	status, err := app.PublicationStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !status.EndpointConfigured || !status.PublisherTokenConfigured {
		t.Fatalf("status = %+v", status)
	}
	if status.Latest == nil || status.Latest.Status != "published" || status.Latest.SnapshotID != result.SnapshotID {
		t.Fatalf("latest = %+v", status.Latest)
	}

	public, err := http.Get(server.URL + "/api/public/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer public.Body.Close()
	if public.StatusCode != http.StatusOK {
		t.Fatalf("public status = %d", public.StatusCode)
	}
	var published publishing.Snapshot
	if err := json.NewDecoder(public.Body).Decode(&published); err != nil {
		t.Fatal(err)
	}
	if published.SnapshotID != result.SnapshotID {
		t.Fatalf("public snapshot id = %q", published.SnapshotID)
	}

	unpublished := app.Unpublish()
	if !unpublished.OK {
		t.Fatalf("unpublish = %+v", unpublished)
	}
	after, err := http.Get(server.URL + "/api/public/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	after.Body.Close()
	if after.StatusCode != http.StatusNotFound {
		t.Fatalf("public status after unpublish = %d", after.StatusCode)
	}
	status, err = app.PublicationStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.Latest == nil || status.Latest.Status != "unpublished" {
		t.Fatalf("latest after unpublish = %+v", status.Latest)
	}
}

func TestPublishRejectsStalePreview(t *testing.T) {
	app := newTestApp(t)
	vaccineID, _ := seedUsage(t, app)
	mustOK(t, app.CloseMonth("2026-11"))
	if _, err := app.PublicationPreview("2026-11"); err != nil {
		t.Fatal(err)
	}
	mustOK(t, app.ReopenMonth("2026-11", "Counting correction"))
	mustOK(t, app.UpdateStock(UpdateInventoryInputDTO{
		VaccineID: vaccineID, Month: "2026-11", OpeningBalance: 101, StockReceived: 20,
	}))
	result := app.PublishMonth("2026-11", true)
	if result.OK || result.Code != "preview_stale" {
		t.Fatalf("stale publish = %+v", result)
	}
}

func TestPublishRequiresConfirmationForOpenMonth(t *testing.T) {
	app := newTestApp(t)
	seedUsage(t, app)
	preview, err := app.PublicationPreview("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Warnings) == 0 {
		t.Fatal("open month preview has no warning")
	}
	result := app.PublishMonth("2026-11", false)
	if result.OK || result.Code != "month_closed" {
		t.Fatalf("open publish = %+v", result)
	}
}

func TestBackupAndRestoreRoundTrip(t *testing.T) {
	app := newTestApp(t)
	mustOK(t, app.CreateVaccine(VaccineInputDTO{Name: "Hepatitis B", Unit: "dose"}))
	backupPath := filepath.Join(t.TempDir(), "roundtrip.db")
	created := app.backupTo(backupPath)
	if !created.OK || created.SHA256 == "" || created.SizeBytes == 0 {
		t.Fatalf("backup = %+v", created)
	}
	mustOK(t, app.CreateVaccine(VaccineInputDTO{Name: "Measles", Unit: "dose"}))
	before, err := app.ListVaccines(false)
	if err != nil || len(before) != 2 {
		t.Fatalf("vaccines before restore = %d, %v", len(before), err)
	}
	restored := app.restoreFrom(backupPath)
	if !restored.OK {
		t.Fatalf("restore = %+v", restored)
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login after restore: %v", err)
	}
	after, err := app.ListVaccines(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].Name != "Hepatitis B" {
		t.Fatalf("vaccines after restore = %+v", after)
	}
	page, err := app.AuditEvents(AuditFilterDTO{EventType: "backup.restored"})
	if err != nil || page.Total != 1 {
		t.Fatalf("restore audit = %+v %v", page, err)
	}
}

func TestRestoreRecoversFromUnopenableDatabase(t *testing.T) {
	good := newTestApp(t)
	mustOK(t, good.CreateVaccine(VaccineInputDTO{Name: "Polio", Unit: "dose"}))
	backupPath := filepath.Join(t.TempDir(), "good.db")
	if created := good.backupTo(backupPath); !created.OK {
		t.Fatalf("backup = %+v", created)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vaccineshit.db"), []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := newAppAt(dir, "test")
	t.Cleanup(func() { broken.Shutdown(context.Background()) })
	if broken.Bootstrap().Ready {
		t.Fatal("broken database unexpectedly started")
	}
	broken.now = func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }
	restored := broken.restoreFrom(backupPath)
	if !restored.OK {
		t.Fatalf("recovery restore = %+v", restored)
	}
	if !broken.Bootstrap().Ready {
		t.Fatal("app is not ready after recovery")
	}
	if _, err := broken.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login after recovery: %v", err)
	}
	items, err := broken.ListVaccines(false)
	if err != nil || len(items) != 1 || items[0].Name != "Polio" {
		t.Fatalf("recovered vaccines = %+v %v", items, err)
	}
}

func TestExportsAreScopedAndAudited(t *testing.T) {
	app := newTestApp(t)
	seedUsage(t, app)
	dir := t.TempDir()
	aggregatePath := filepath.Join(dir, "aggregate.csv")
	aggregate := app.exportAggregateCSVTo("2026-11", aggregatePath)
	if !aggregate.OK {
		t.Fatalf("aggregate export = %+v", aggregate)
	}
	aggregateBody, err := os.ReadFile(aggregatePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(aggregateBody), "Influenza") {
		t.Fatalf("aggregate missing vaccine: %s", aggregateBody)
	}
	if strings.Contains(string(aggregateBody), "Ada Lovelace") || strings.Contains(string(aggregateBody), "P-001") {
		t.Fatal("aggregate export leaked patient data")
	}

	patientPath := filepath.Join(dir, "patients.csv")
	patient := app.exportPatientUsageCSVTo("2026-11", patientPath)
	if !patient.OK {
		t.Fatalf("patient export = %+v", patient)
	}
	patientBody, err := os.ReadFile(patientPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patientBody), "Ada Lovelace") {
		t.Fatal("patient export is missing expected local patient detail")
	}
	if strings.Contains(string(patientBody), "private clinical note") {
		t.Fatal("patient export must omit usage notes")
	}
	info, err := os.Stat(patientPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("patient export mode = %o, want 600", info.Mode().Perm())
	}

	page, err := app.AuditEvents(AuditFilterDTO{EventType: "export.completed"})
	if err != nil || page.Total != 2 {
		t.Fatalf("export audit = %+v %v", page, err)
	}
	for _, event := range page.Events {
		if strings.Contains(event.Metadata, "Ada Lovelace") || strings.Contains(event.Metadata, "private clinical note") {
			t.Fatal("export audit leaked patient data")
		}
	}
}

func TestAuditEventsExcludePatientNamesAndNotes(t *testing.T) {
	app := newTestApp(t)
	seedUsage(t, app)
	page, err := app.AuditEvents(AuditFilterDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total == 0 {
		t.Fatal("no audit events recorded")
	}
	for _, event := range page.Events {
		serialized, _ := json.Marshal(event)
		for _, secret := range []string{"Ada Lovelace", "P-001", "private clinical note"} {
			if strings.Contains(string(serialized), secret) {
				t.Fatalf("audit event leaked %q: %s", secret, serialized)
			}
		}
	}
}
