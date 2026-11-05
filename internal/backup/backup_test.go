package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vaccineshit/internal/store/generated"
	"vaccineshit/internal/store/sqlite"
)

func setup(t *testing.T) (string, string, *sqlite.DB, *Manager) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	backupDir := filepath.Join(dir, "backups")
	db, err := sqlite.Open(context.Background(), sqlite.Config{Path: dbPath, BackupDir: backupDir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	manager := New(db.DB(), dbPath, backupDir)
	return dir, dbPath, db, manager
}

func addVaccine(t *testing.T, db *sqlite.DB, name string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Queries().CreateVaccine(context.Background(), generated.CreateVaccineParams{
		Name: name, Unit: "dose", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

func vaccineNames(t *testing.T, path string) []string {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT name FROM vaccines ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return names
}

func TestCreateBackupIsConsistentAndValidated(t *testing.T) {
	t.Parallel()
	_, _, db, manager := setup(t)
	addVaccine(t, db, "Measles")
	dest := filepath.Join(t.TempDir(), "snapshot.db")

	info, err := manager.Create(context.Background(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.SizeBytes == 0 || info.SHA256 == "" {
		t.Fatalf("backup info = %+v", info)
	}
	if err := Validate(context.Background(), dest); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions = %o, want 600", stat.Mode().Perm())
	}

	// Later live changes must not alter the existing snapshot.
	addVaccine(t, db, "Later")
	if names := vaccineNames(t, dest); len(names) != 1 || names[0] != "Measles" {
		t.Fatalf("snapshot names = %v, want [Measles]", names)
	}
}

func TestValidateRejectsInvalidFiles(t *testing.T) {
	t.Parallel()
	_, dbPath, _, _ := setup(t)
	bad := filepath.Join(t.TempDir(), "bad.db")
	if err := os.WriteFile(bad, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Validate(context.Background(), bad); err == nil {
		t.Fatal("invalid backup file was accepted")
	}
	if err := Validate(context.Background(), dbPath+".missing"); err == nil {
		t.Fatal("missing backup file was accepted")
	}
}

func TestStagedRestoreAndRollback(t *testing.T) {
	t.Parallel()
	dir, dbPath, db, manager := setup(t)
	addVaccine(t, db, "Original")
	backupPath := filepath.Join(dir, "backups", "original.db")
	if _, err := manager.Create(context.Background(), backupPath); err != nil {
		t.Fatal(err)
	}
	addVaccine(t, db, "Later")

	prepared, err := manager.PrepareRestore(context.Background(), backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Checkpoint(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := prepared.CommitAfterClose(); err != nil {
		t.Fatal(err)
	}
	if names := vaccineNames(t, dbPath); len(names) != 1 || names[0] != "Original" {
		t.Fatalf("restored names before rollback = %v", names)
	}
	if err := prepared.Rollback(); err != nil {
		t.Fatal(err)
	}
	if names := vaccineNames(t, dbPath); len(names) != 2 {
		t.Fatalf("names after rollback = %v, want original live database with 2 rows", names)
	}
}

func TestPrepareRestoreRejectsWithoutTouchingDatabase(t *testing.T) {
	t.Parallel()
	dir, dbPath, db, manager := setup(t)
	addVaccine(t, db, "Still Here")
	bad := filepath.Join(dir, "bad.db")
	if err := os.WriteFile(bad, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PrepareRestore(context.Background(), bad); err == nil {
		t.Fatal("invalid restore was prepared")
	}
	if names := vaccineNames(t, dbPath); len(names) != 1 || names[0] != "Still Here" {
		t.Fatalf("live database changed after rejected restore: %v", names)
	}
}
