package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"vaccineshit/internal/store/generated"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(context.Background(), Config{
		Path:        filepath.Join(dir, "app.db"),
		BackupDir:   filepath.Join(dir, "backups"),
		BusyTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createVaccineParams(now string) generated.CreateVaccineParams {
	return generated.CreateVaccineParams{
		Name:      "Measles",
		Unit:      "dose",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestOpenMigratesAndPersists(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := context.Background()

	var fk int
	if err := db.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	var journal string
	if err := db.DB().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journal)
	}

	tables := []string{"vaccines", "patients", "monthly_inventory", "usage_entries", "inventory_adjustments", "audit_events", "publications", "app_settings", "schema_migrations"}
	for _, table := range tables {
		var count int
		if err := db.DB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("table %s was not created", table)
		}
	}

	var threshold string
	if err := db.DB().QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key='low_stock_threshold'").Scan(&threshold); err != nil {
		t.Fatal(err)
	}
	if threshold != "10" {
		t.Fatalf("low stock threshold = %q, want 10", threshold)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Queries().CreateVaccine(ctx, createVaccineParams(now)); err != nil {
		t.Fatal(err)
	}
	path := db.Path()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(context.Background(), Config{Path: path, BackupDir: filepath.Join(filepath.Dir(path), "backups")})
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer reopened.Close()
	var name string
	if err := reopened.DB().QueryRowContext(ctx, "SELECT name FROM vaccines WHERE id=1").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Measles" {
		t.Fatalf("name after reopen = %q", name)
	}
}

func TestConstraintsAndForeignKeys(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	if _, err := db.Queries().CreateVaccine(ctx, createVaccineParams(now)); err != nil {
		t.Fatal(err)
	}
	// Negative opening balance must fail the CHECK.
	_, err := db.DB().ExecContext(ctx,
		`INSERT INTO monthly_inventory (vaccine_id, month, opening_balance, stock_received, created_at, updated_at)
		 VALUES (1, '2026-01', -1, 0, ?, ?)`, now, now)
	if err == nil {
		t.Fatal("negative opening balance was accepted")
	}
	// Missing vaccine must fail the foreign key.
	_, err = db.DB().ExecContext(ctx,
		`INSERT INTO monthly_inventory (vaccine_id, month, opening_balance, stock_received, created_at, updated_at)
		 VALUES (999, '2026-01', 0, 0, ?, ?)`, now, now)
	if err == nil {
		t.Fatal("foreign key violation was accepted")
	}
	// Zero-stock setup is explicitly allowed.
	if _, err := db.DB().ExecContext(ctx,
		`INSERT INTO monthly_inventory (vaccine_id, month, opening_balance, stock_received, created_at, updated_at)
		 VALUES (1, '2026-01', 0, 0, ?, ?)`, now, now); err != nil {
		t.Fatalf("explicit zero stock setup rejected: %v", err)
	}
	// Duplicate month record is rejected.
	_, err = db.DB().ExecContext(ctx,
		`INSERT INTO monthly_inventory (vaccine_id, month, opening_balance, stock_received, created_at, updated_at)
		 VALUES (1, '2026-01', 0, 0, ?, ?)`, now, now)
	if err == nil {
		t.Fatal("duplicate vaccine/month inventory was accepted")
	}
}

func TestWriteTransactionRollsBack(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	wantErr := errors.New("stop")
	err := db.WithTx(ctx, func(q generated.Querier) error {
		if _, err := q.CreateVaccine(ctx, createVaccineParams(now)); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithTx error = %v, want %v", err, wantErr)
	}
	var vaccines int
	if err := db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM vaccines").Scan(&vaccines); err != nil {
		t.Fatal(err)
	}
	if vaccines != 0 {
		t.Fatalf("rolled-back transaction left %d vaccines", vaccines)
	}

	if err := db.WithTx(ctx, func(q generated.Querier) error {
		_, err := q.CreateVaccine(ctx, createVaccineParams(now))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM vaccines").Scan(&vaccines); err != nil {
		t.Fatal(err)
	}
	if vaccines != 1 {
		t.Fatalf("committed transaction has %d vaccines, want 1", vaccines)
	}
}
