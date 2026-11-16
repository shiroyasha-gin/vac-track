package service

import "testing"

func TestPatientListPagination(t *testing.T) {
	f := newFixture(t)
	names := []string{"أحمد علي", "بشرى محمد", "تامر سمير", "ثريا حسن", "جمال فؤاد"}
	created := make(map[string]int64, len(names))
	for _, name := range names {
		patient := f.patient(name)
		created[name] = patient.ID
	}

	first, err := f.svc.Patients.List(f.ctx, "", false, 2, 0)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if first.Total != int64(len(names)) {
		t.Fatalf("total = %d, want %d", first.Total, len(names))
	}
	if len(first.Items) != 2 {
		t.Fatalf("first page items = %d, want 2", len(first.Items))
	}

	last, err := f.svc.Patients.List(f.ctx, "", false, 2, 4)
	if err != nil {
		t.Fatalf("list last page: %v", err)
	}
	if last.Total != int64(len(names)) {
		t.Fatalf("last page total = %d, want %d", last.Total, len(names))
	}
	if len(last.Items) != 1 {
		t.Fatalf("last page items = %d, want 1", len(last.Items))
	}

	if _, err := f.svc.Patients.List(f.ctx, "", false, 2, 99); err != nil {
		t.Fatalf("list beyond end: %v", err)
	}

	// A limit outside the accepted range is clamped, never unbounded.
	huge, err := f.svc.Patients.List(f.ctx, "", false, 5000, 0)
	if err != nil {
		t.Fatalf("list clamped: %v", err)
	}
	if huge.Limit != 500 {
		t.Fatalf("clamped limit = %d, want 500", huge.Limit)
	}
}

func TestPatientListSearchAndArchive(t *testing.T) {
	f := newFixture(t)
	f.patient("سارة يوسف")
	archived := f.patient("سامي يوسف")
	if _, err := f.svc.Patients.SetArchived(f.ctx, archived.ID, true); err != nil {
		t.Fatalf("archive patient: %v", err)
	}
	f.patient("ليلى يوسف")

	active, err := f.svc.Patients.List(f.ctx, "يوسف", false, 50, 0)
	if err != nil {
		t.Fatalf("search active: %v", err)
	}
	if active.Total != 2 {
		t.Fatalf("active search total = %d, want 2", active.Total)
	}
	for _, item := range active.Items {
		if item.ID == archived.ID {
			t.Fatalf("archived patient %d appeared in active search", archived.ID)
		}
	}

	all, err := f.svc.Patients.List(f.ctx, "يوسف", true, 50, 0)
	if err != nil {
		t.Fatalf("search archived: %v", err)
	}
	if all.Total != 3 {
		t.Fatalf("all search total = %d, want 3", all.Total)
	}

	unknown, err := f.svc.Patients.List(f.ctx, "لا-يوجد-مريض", false, 50, 0)
	if err != nil {
		t.Fatalf("search miss: %v", err)
	}
	if unknown.Total != 0 || len(unknown.Items) != 0 {
		t.Fatalf("search miss = %d/%d, want 0/0", unknown.Total, len(unknown.Items))
	}

	// The shared picker must never return archived patients.
	picked, err := f.svc.Patients.SearchActive(f.ctx, "يوسف", 10)
	if err != nil {
		t.Fatalf("picker search: %v", err)
	}
	if len(picked) != 2 {
		t.Fatalf("picker results = %d, want 2", len(picked))
	}
	for _, item := range picked {
		if item.ID == archived.ID {
			t.Fatalf("archived patient %d appeared in picker", archived.ID)
		}
	}
}
