package service

import "testing"

func TestArchivedVaccinesExcludedFromLowStockTotals(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	active := f.vaccine("Active vaccine")
	archived := f.vaccine("Archived vaccine")
	f.initInventory(active.ID, "2026-11", 100, 0)
	f.initInventory(archived.ID, "2026-11", 1, 0)
	if _, err := f.svc.Vaccines.SetArchived(f.ctx, archived.ID, true); err != nil {
		t.Fatal(err)
	}

	overview, err := f.svc.Reporting.MonthOverview(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Totals.ActiveVaccines != 1 || overview.Totals.InitializedVaccines != 1 {
		t.Fatalf("active/initialized totals = %+v", overview.Totals)
	}
	if overview.Totals.LowStockCount != 0 {
		t.Fatalf("low stock count = %d, want 0 for archived-only low stock", overview.Totals.LowStockCount)
	}
	for _, row := range overview.Rows {
		if row.VaccineID == archived.ID && row.LowStock {
			t.Fatalf("archived vaccine row marked low stock: %+v", row)
		}
	}

	report, err := f.svc.Reporting.Report(f.ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals.LowStockCount != 0 {
		t.Fatalf("report low stock count = %d, want 0", report.Totals.LowStockCount)
	}
}
