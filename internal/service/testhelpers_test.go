package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"vaccineshit/internal/store/generated"
	"vaccineshit/internal/store/sqlite"
)

type fixture struct {
	t   *testing.T
	db  *sqlite.DB
	svc *Services
	ctx context.Context
}

var fixedNow = time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(context.Background(), sqlite.Config{
		Path:      filepath.Join(dir, "app.db"),
		BackupDir: filepath.Join(dir, "backups"),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &fixture{
		t:   t,
		db:  db,
		ctx: context.Background(),
		svc: New(db, Options{Now: func() time.Time { return fixedNow }}),
	}
}

func (f *fixture) vaccine(name string) generated.Vaccine {
	f.t.Helper()
	v, err := f.svc.Vaccines.Create(f.ctx, VaccineInput{Name: name, Unit: "dose"})
	if err != nil {
		f.t.Fatalf("create vaccine %q: %v", name, err)
	}
	return v
}

func (f *fixture) patient(name string) generated.Patient {
	f.t.Helper()
	p, err := f.svc.Patients.Create(f.ctx, PatientInput{FullName: name})
	if err != nil {
		f.t.Fatalf("create patient %q: %v", name, err)
	}
	return p
}

func (f *fixture) initInventory(vaccineID int64, month string, opening, received int64) generated.MonthlyInventory {
	f.t.Helper()
	inv, err := f.svc.Inventory.Initialize(f.ctx, InitializeInventoryInput{
		VaccineID:      vaccineID,
		Month:          month,
		OpeningBalance: opening,
		StockReceived:  received,
		OverrideReason: "Synthetic test carryover override",
	})
	if err != nil {
		f.t.Fatalf("initialize %d/%s: %v", vaccineID, month, err)
	}
	return inv
}

func (f *fixture) createUsage(patientID, vaccineID int64, date string, qty int64) UsageResult {
	f.t.Helper()
	result, err := f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patientID,
		VaccineID:      vaccineID,
		AdministeredOn: date,
		Quantity:       qty,
	})
	if err != nil {
		f.t.Fatalf("create usage: %v", err)
	}
	return result
}

func (f *fixture) countUsage() int {
	f.t.Helper()
	var count int
	if err := f.db.DB().QueryRowContext(f.ctx, "SELECT COUNT(*) FROM usage_entries").Scan(&count); err != nil {
		f.t.Fatalf("count usage: %v", err)
	}
	return count
}

func (f *fixture) countAudit(eventType string) int {
	f.t.Helper()
	var count int
	if err := f.db.DB().QueryRowContext(f.ctx, "SELECT COUNT(*) FROM audit_events WHERE event_type = ?", eventType).Scan(&count); err != nil {
		f.t.Fatalf("count audit: %v", err)
	}
	return count
}
