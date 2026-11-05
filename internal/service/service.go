// Package service contains use cases and business-rule enforcement. Every
// stock-affecting operation is transactional and writes its audit event in the
// same transaction.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/store"
	"vaccineshit/internal/store/generated"
)

// Options configures the service graph. Now is injectable for deterministic
// tests; production uses the system UTC clock.
type Options struct {
	Now func() time.Time
}

// Services is the application service graph.
type Services struct {
	Vaccines  *VaccineService
	Patients  *PatientService
	Inventory *InventoryService
	Usage     *UsageService
	Reporting *ReportingService
	Audit     *AuditService
	Settings  *SettingsService
	CSV       *CSVService
}

// New wires services around a store. It performs no queries.
func New(st store.Store, opts Options) *Services {
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	base := base{store: st, now: opts.Now}
	settings := &SettingsService{base: base}
	auditSvc := &AuditService{base: base}
	reporting := &ReportingService{base: base, settings: settings, inventory: nil, usage: nil}
	inventory := &InventoryService{base: base}
	usage := &UsageService{base: base}
	reporting.inventory = inventory
	reporting.usage = usage
	return &Services{
		Vaccines:  &VaccineService{base: base},
		Patients:  &PatientService{base: base},
		Inventory: inventory,
		Usage:     usage,
		Reporting: reporting,
		Audit:     auditSvc,
		Settings:  settings,
		CSV:       &CSVService{base: base, reporting: reporting},
	}
}

type base struct {
	store store.Store
	now   func() time.Time
}

func (b base) timestamp() string {
	return b.now().UTC().Format(time.RFC3339)
}

func (b base) nowUTC() time.Time {
	return b.now().UTC()
}

func (b base) record(ctx context.Context, q generated.Querier, event audit.Event) error {
	return audit.Record(ctx, q, b.nowUTC(), event)
}

// mapPersistenceError converts expected SQLite constraint failures into
// user-facing domain errors without leaking SQL. Unexpected errors are
// returned wrapped for the application layer to log and sanitize.
func mapPersistenceError(err error, conflictMessage string) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unique constraint failed"):
		return domainConflict(conflictMessage)
	case strings.Contains(msg, "foreign key constraint failed"):
		return domainConflict("لم يعد سجل مرتبط موجوداً. حدّث الصفحة وحاول مجدداً.")
	case strings.Contains(msg, "check constraint failed"):
		return domainInvalid("توجد قيمة مخزنة تخالف إحدى قواعد المخزون. حدّث الصفحة وحاول مجدداً.")
	case errors.Is(err, context.Canceled):
		return err
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("انتهت مهلة العملية: %w", err)
	default:
		return err
	}
}
