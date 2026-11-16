package service

import (
	"strings"
	"testing"

	"vaccineshit/internal/domain"
)

func TestUsageRequiresMonthlyStockInitialization(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Measles")
	patient := f.patient("Synthetic Patient A")

	_, err := f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-03",
		Quantity:       1,
	})
	if !domain.IsCode(err, domain.CodeNotInitialized) {
		t.Fatalf("error = %v, want %v", err, domain.CodeNotInitialized)
	}
	if got := f.countUsage(); got != 0 {
		t.Fatalf("usage rows = %d, want 0 after rejected entry", got)
	}
	if got := f.countAudit("usage.created"); got != 0 {
		t.Fatalf("usage audit events = %d, want 0", got)
	}
}

func TestZeroStockSetupUnlocksUsageWhenOpeningSufficient(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("BCG")
	patient := f.patient("Synthetic Patient B")
	f.initInventory(vaccine.ID, "2026-11", 8, 0)

	result := f.createUsage(patient.ID, vaccine.ID, "2026-11-04", 3)
	if result.Balance != 5 {
		t.Fatalf("balance = %d, want 5", result.Balance)
	}
	if result.UsageTotal != 3 {
		t.Fatalf("usage total = %d, want 3", result.UsageTotal)
	}
}

func TestExplicitZeroStockIsInitializedButRejectsUsage(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Polio")
	patient := f.patient("Synthetic Patient C")
	f.initInventory(vaccine.ID, "2026-11", 0, 0)

	_, err := f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-05",
		Quantity:       1,
	})
	if !domain.IsCode(err, domain.CodeInsufficientStock) {
		t.Fatalf("error = %v, want %v", err, domain.CodeInsufficientStock)
	}
	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	row := overview.Rows[0]
	if !row.Initialized || row.Status != "open" {
		t.Fatalf("explicit zero-stock row not initialized: %+v", row)
	}
}

func TestBalanceFormulaAndNegativeStockGuard(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("DTP")
	patient := f.patient("Synthetic Patient D")
	f.initInventory(vaccine.ID, "2026-11", 8, 12)

	f.createUsage(patient.ID, vaccine.ID, "2026-11-06", 5)
	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	row := overview.Rows[0]
	if row.Remaining != 15 || row.OpeningBalance != 8 || row.StockReceived != 12 || row.UsageTotal != 5 {
		t.Fatalf("row = %+v, want 8 + 12 - 5 = 15", row)
	}

	before := f.countUsage()
	_, err = f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-07",
		Quantity:       16,
	})
	if !domain.IsCode(err, domain.CodeInsufficientStock) {
		t.Fatalf("error = %v, want %v", err, domain.CodeInsufficientStock)
	}
	if after := f.countUsage(); after != before {
		t.Fatalf("usage rows = %d, want unchanged %d", after, before)
	}
	overview, _ = f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if overview.Rows[0].Remaining != 15 {
		t.Fatalf("remaining changed after rejected entry: %d", overview.Rows[0].Remaining)
	}
}

func TestUsageDateMustHaveMatchingInitializedMonth(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Hepatitis B")
	patient := f.patient("Synthetic Patient E")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)

	_, err := f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-12-01",
		Quantity:       1,
	})
	if !domain.IsCode(err, domain.CodeNotInitialized) {
		t.Fatalf("error = %v, want uninitialized December", err)
	}
}

func TestClosedMonthRejectsUsageAndStockEdits(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Measles")
	patient := f.patient("Synthetic Patient F")
	f.initInventory(vaccine.ID, "2026-11", 20, 0)
	f.createUsage(patient.ID, vaccine.ID, "2026-11-08", 4)

	if _, err := f.svc.Inventory.CloseMonth(f.ctx, "2026-11"); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-09",
		Quantity:       1,
	})
	if !domain.IsCode(err, domain.CodeMonthClosed) {
		t.Fatalf("usage error = %v, want closed month", err)
	}
	_, err = f.svc.Inventory.UpdateStock(f.ctx, UpdateInventoryInput{
		VaccineID:      vaccine.ID,
		Month:          "2026-11",
		OpeningBalance: 20,
		StockReceived:  5,
	})
	if !domain.IsCode(err, domain.CodeMonthClosed) {
		t.Fatalf("stock error = %v, want closed month", err)
	}

	if _, err := f.svc.Inventory.ReopenMonth(f.ctx, "2026-11", "Correct a stock count"); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-09",
		Quantity:       1,
	})
	if err != nil {
		t.Fatalf("usage after reopen failed: %v", err)
	}
}

func TestVoidIncreasesRemainingAndRequiresReason(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Rotavirus")
	patient := f.patient("Synthetic Patient G")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	created := f.createUsage(patient.ID, vaccine.ID, "2026-11-10", 4)

	_, err := f.svc.Usage.Void(f.ctx, VoidUsageInput{ID: created.Entry.ID, Reason: " "})
	if !domain.IsCode(err, domain.CodeReasonRequired) {
		t.Fatalf("empty reason error = %v", err)
	}
	result, err := f.svc.Usage.Void(f.ctx, VoidUsageInput{ID: created.Entry.ID, Reason: "Data entry error"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Balance != 10 || result.UsageTotal != 0 {
		t.Fatalf("after void balance=%d usage=%d, want 10/0", result.Balance, result.UsageTotal)
	}
	if result.Entry.VoidedAt.Valid != true {
		t.Fatal("void timestamp not set")
	}
	var reason string
	if err := f.db.DB().QueryRowContext(f.ctx, "SELECT void_reason FROM usage_entries WHERE id=?", created.Entry.ID).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "Data entry error" {
		t.Fatalf("void reason = %q", reason)
	}
	if got := f.countAudit("usage.voided"); got != 1 {
		t.Fatalf("void audit events = %d, want 1", got)
	}
}

func TestCarryoverDefaultsToPriorClosingBalance(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Varicella")
	patient := f.patient("Synthetic Patient H")
	f.initInventory(vaccine.ID, "2026-11", 8, 12)
	f.createUsage(patient.ID, vaccine.ID, "2026-11-11", 5)
	if _, err := f.svc.Inventory.CloseMonth(f.ctx, "2026-11"); err != nil {
		t.Fatal(err)
	}

	suggestion, err := f.svc.Inventory.Suggestion(f.ctx, vaccine.ID, "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	if suggestion.Amount != 15 {
		t.Fatalf("suggested opening = %d, want 15", suggestion.Amount)
	}
	if !suggestion.PriorClosed || suggestion.Source != "prior_month_closing" {
		t.Fatalf("suggestion = %+v", suggestion)
	}
	inv := f.initInventory(vaccine.ID, "2026-12", suggestion.Amount, 0)
	if inv.OpeningBalance != 15 {
		t.Fatalf("December opening = %d, want 15", inv.OpeningBalance)
	}
}

func TestCarryoverOverrideRequiresReason(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Yellow Fever")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	if _, err := f.svc.Inventory.CloseMonth(f.ctx, "2026-11"); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.Inventory.Initialize(f.ctx, InitializeInventoryInput{
		VaccineID:      vaccine.ID,
		Month:          "2026-12",
		OpeningBalance: 7,
		StockReceived:  0,
	})
	if !domain.IsCode(err, domain.CodeReasonRequired) {
		t.Fatalf("error = %v, want reason required", err)
	}
	_, err = f.svc.Inventory.Initialize(f.ctx, InitializeInventoryInput{
		VaccineID:      vaccine.ID,
		Month:          "2026-12",
		OpeningBalance: 7,
		StockReceived:  0,
		OverrideReason: "Physical count correction",
	})
	if err != nil {
		t.Fatalf("reasoned override rejected: %v", err)
	}
	if got := f.countAudit("inventory.initialized"); got != 2 {
		t.Fatalf("initialization audits = %d, want 2", got)
	}
}

func TestArchivedVaccineHistoryRemainsButNewUsageIsRejected(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Mumps")
	patient := f.patient("Synthetic Patient I")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	f.createUsage(patient.ID, vaccine.ID, "2026-11-12", 2)
	if _, err := f.svc.Vaccines.SetArchived(f.ctx, vaccine.ID, true); err != nil {
		t.Fatal(err)
	}

	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range overview.Rows {
		if row.VaccineID == vaccine.ID {
			found = true
			if !row.Archived || row.UsageTotal != 2 {
				t.Fatalf("archived history row = %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("archived vaccine disappeared from historical report")
	}

	_, err = f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-13",
		Quantity:       1,
	})
	if !domain.IsCode(err, domain.CodeArchived) {
		t.Fatalf("new usage on archived vaccine error = %v", err)
	}
	forecast, err := f.svc.Reporting.Forecast(f.ctx, "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range forecast.Rows {
		if row.VaccineID == vaccine.ID {
			t.Fatal("archived vaccine offered in forecast")
		}
	}
}

func TestEditUsageRevalidatesStockAndRecordsAudit(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Japanese Encephalitis")
	patient := f.patient("Synthetic Patient J")
	f.initInventory(vaccine.ID, "2026-11", 10, 0)
	created := f.createUsage(patient.ID, vaccine.ID, "2026-11-14", 4)

	result, err := f.svc.Usage.Update(f.ctx, UpdateUsageInput{
		ID:             created.Entry.ID,
		PatientID:      patient.ID,
		AdministeredOn: "2026-11-14",
		Quantity:       8,
		Reason:         "Corrected dose count",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Balance != 2 {
		t.Fatalf("balance after edit = %d, want 2", result.Balance)
	}
	_, err = f.svc.Usage.Update(f.ctx, UpdateUsageInput{
		ID:             created.Entry.ID,
		PatientID:      patient.ID,
		AdministeredOn: "2026-11-14",
		Quantity:       11,
	})
	if !domain.IsCode(err, domain.CodeInsufficientStock) {
		t.Fatalf("over-stock edit error = %v", err)
	}
	if got := f.countAudit("usage.updated"); got != 1 {
		t.Fatalf("usage update audits = %d, want 1", got)
	}
}

func TestAdjustmentIsAuditedAndCannotGoNegative(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Cholera")
	f.initInventory(vaccine.ID, "2026-11", 5, 5)

	_, err := f.svc.Inventory.Adjust(f.ctx, AdjustmentInput{
		VaccineID: vaccine.ID,
		Month:     "2026-11",
		Amount:    -11,
		Reason:    "Recount",
	})
	if err == nil {
		t.Fatal("negative stock adjustment was accepted")
	}
	result, err := f.svc.Inventory.Adjust(f.ctx, AdjustmentInput{
		VaccineID: vaccine.ID,
		Month:     "2026-11",
		Amount:    -2,
		Reason:    "Broken vial",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Balance != 8 {
		t.Fatalf("balance = %d, want 8", result.Balance)
	}
	if got := f.countAudit("inventory.adjusted"); got != 1 {
		t.Fatalf("adjustment audits = %d, want 1", got)
	}
}

func TestMonthCloseStoresClosingBalanceAndBlocksNewInventory(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Typhoid")
	other := f.vaccine("Rabies")
	f.initInventory(vaccine.ID, "2026-11", 6, 4)
	if _, err := f.svc.Inventory.CloseMonth(f.ctx, "2026-11"); err != nil {
		t.Fatal(err)
	}
	inv, err := f.svc.Inventory.Suggestion(f.ctx, vaccine.ID, "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Amount != 10 {
		t.Fatalf("closing suggestion = %d, want 10", inv.Amount)
	}
	_, err = f.svc.Inventory.Initialize(f.ctx, InitializeInventoryInput{
		VaccineID: other.ID,
		Month:     "2026-11",
	})
	if !domain.IsCode(err, domain.CodeMonthClosed) {
		t.Fatalf("late initialization error = %v, want month closed", err)
	}
}

func TestLeapFebruaryDailyColumns(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Influenza")
	patient := f.patient("Synthetic Patient K")
	f.initInventory(vaccine.ID, "2024-02", 100, 0)
	f.createUsage(patient.ID, vaccine.ID, "2024-02-29", 2)

	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2024-02")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Days != 29 {
		t.Fatalf("days = %d, want 29", overview.Days)
	}
	row := overview.Rows[0]
	if len(row.DailyUsage) != 29 {
		t.Fatalf("daily columns = %d, want 29", len(row.DailyUsage))
	}
	if row.DailyUsage[28] != 2 {
		t.Fatalf("Feb 29 usage = %d, want 2", row.DailyUsage[28])
	}
	report, err := f.svc.Reporting.Report(f.ctx, "2024-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Daily) != 29 || report.Daily[28].Total != 2 {
		t.Fatalf("daily report = %+v", report.Daily)
	}
}

func TestAuditEventWrittenInSameTransaction(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Pertussis")
	patient := f.patient("Synthetic Patient L")
	f.initInventory(vaccine.ID, "2026-11", 1, 0)

	_, err := f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-15",
		Quantity:       2,
	})
	if err == nil {
		t.Fatal("expected insufficient stock")
	}
	if got := f.countAudit("usage.created"); got != 0 {
		t.Fatalf("audit rows from rejected usage = %d, want 0", got)
	}
}

func TestAggregateAndPatientCSVPrivacyBoundary(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Measles")
	patient := f.patient("Synthetic Unique Patient Name")
	f.initInventory(vaccine.ID, "2026-11", 10, 2)
	f.createUsage(patient.ID, vaccine.ID, "2026-11-16", 3)

	aggregate, err := f.svc.CSV.AggregateCSV(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(aggregate), patient.FullName) {
		t.Fatal("aggregate CSV leaked a patient name")
	}
	if !strings.Contains(string(aggregate), "2026-11") {
		t.Fatal("aggregate CSV is missing its report month")
	}
	patientCSV, err := f.svc.CSV.PatientUsageCSV(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	text := string(patientCSV)
	if !strings.Contains(text, "تحذير") || !strings.Contains(text, "معلومات صحية") || !strings.Contains(text, patient.FullName) {
		t.Fatal("patient CSV is missing its warning or patient data")
	}
	if strings.Contains(text, "Synthetic Patient A") {
		t.Fatal("patient CSV includes unrelated synthetic patients")
	}
}

func TestForecastUsesMostRecentCompleteMonthUntilThreeExist(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Shingles")
	patient := f.patient("Synthetic Patient M")
	var usageByMonth = []struct {
		month string
		qty   int64
	}{
		{"2026-08", 4},
		{"2026-09", 6},
		{"2026-10", 8},
	}
	for _, item := range usageByMonth {
		f.initInventory(vaccine.ID, item.month, 20, 0)
		f.createUsage(patient.ID, vaccine.ID, item.month+"-05", item.qty)
		if _, err := f.svc.Inventory.CloseMonth(f.ctx, item.month); err != nil {
			t.Fatal(err)
		}
	}
	forecast, err := f.svc.Reporting.Forecast(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if len(forecast.Rows) != 1 {
		t.Fatalf("forecast rows = %d, want 1", len(forecast.Rows))
	}
	row := forecast.Rows[0]
	if row.Method != "three_month_average" || row.SampleCount != 3 || row.EstimatedDemand != 6 {
		t.Fatalf("forecast = %+v, want 3-month average 6", row)
	}
}

func TestPatientArchiveRetainsHistory(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("HPV")
	patient := f.patient("Synthetic Patient N")
	f.initInventory(vaccine.ID, "2026-11", 5, 0)
	created := f.createUsage(patient.ID, vaccine.ID, "2026-11-17", 1)

	if _, err := f.svc.Patients.SetArchived(f.ctx, patient.ID, true); err != nil {
		t.Fatal(err)
	}
	history, err := f.svc.Usage.ForPatient(f.ctx, patient.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].ID != created.Entry.ID {
		t.Fatalf("patient history after archive = %+v", history)
	}
	_, err = f.svc.Usage.Create(f.ctx, CreateUsageInput{
		PatientID:      patient.ID,
		VaccineID:      vaccine.ID,
		AdministeredOn: "2026-11-18",
		Quantity:       1,
	})
	if !domain.IsCode(err, domain.CodeArchived) {
		t.Fatalf("archived patient usage error = %v", err)
	}
}

func TestUnsafeVaccineDeleteRejected(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("Japanese Encephalitis")
	f.initInventory(vaccine.ID, "2026-11", 1, 0)
	if _, err := f.svc.Vaccines.SetArchived(f.ctx, vaccine.ID, true); err != nil {
		t.Fatal(err)
	}
	err := f.svc.Vaccines.Delete(f.ctx, vaccine.ID)
	if !domain.IsCode(err, domain.CodeUnsafeDelete) {
		t.Fatalf("delete error = %v, want unsafe delete", err)
	}
}

func TestAuditFilterAndPagination(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.vaccine("First")
	f.vaccine("Second")
	page, err := f.svc.Audit.List(f.ctx, AuditFilter{EventType: "vaccine.created", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Events) != 1 {
		t.Fatalf("audit page = %+v", page)
	}
	if page.Events[0].EventType != "vaccine.created" {
		t.Fatalf("event type = %s", page.Events[0].EventType)
	}
}

func TestMonthOverviewReportsUninitializedActiveVaccine(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	vaccine := f.vaccine("BCG")
	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.State != domain.MonthNotInitialized {
		t.Fatalf("state = %s, want not_initialized", overview.State)
	}
	if overview.Totals.UninitializedActive != 1 || overview.Rows[0].VaccineID != vaccine.ID {
		t.Fatalf("overview = %+v", overview)
	}
}
