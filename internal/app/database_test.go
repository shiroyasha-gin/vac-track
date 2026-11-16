package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vaccineshit/internal/domain"
	"vaccineshit/internal/service"
)

// freshStartDirs creates an application data directory with the given stale
// files and opens the app against it.
func appAtDir(t *testing.T, dir string) *App {
	t.Helper()
	app := newAppAt(dir, "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	app.now = func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }
	return app
}

func TestStartupCreatesMissingDatabaseAndQuarantinesOrphans(t *testing.T) {
	dir := t.TempDir()
	// A previous crash left WAL and shared-memory files behind without the main
	// database file. The new database must not adopt them.
	if err := os.WriteFile(filepath.Join(dir, "vaccineshit.db-wal"), []byte("orphan wal"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vaccineshit.db-shm"), []byte("orphan shm"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := appAtDir(t, dir)
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	info := app.Bootstrap()
	if !info.Ready || info.FatalError != "" {
		t.Fatalf("bootstrap = %+v", info)
	}
	if _, err := os.Stat(filepath.Join(dir, "vaccineshit.db")); err != nil {
		t.Fatalf("new database file: %v", err)
	}
	if current, err := os.ReadFile(filepath.Join(dir, "vaccineshit.db-wal")); err == nil && string(current) == "orphan wal" {
		t.Fatal("orphan WAL was adopted by the new database")
	}
	preserved := filepath.Join(dir, "backups")
	entries, err := os.ReadDir(preserved)
	if err != nil {
		t.Fatalf("recovery folder: %v", err)
	}
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "pre-initialize-") {
		t.Fatalf("recovery folders = %+v", entries)
	}
	quarantined, err := os.ReadFile(filepath.Join(preserved, entries[0].Name(), "vaccineshit.db-wal"))
	if err != nil || string(quarantined) != "orphan wal" {
		t.Fatalf("quarantined WAL = %q %v", quarantined, err)
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login to new database: %v", err)
	}
}

func TestStartupReplacesEmptyDatabaseFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vaccineshit.db"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	app := appAtDir(t, dir)
	if app.initErr != nil {
		t.Fatalf("startup with empty database: %v", app.initErr)
	}
	if !app.Bootstrap().Ready {
		t.Fatal("app did not start with an empty database file")
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("recovery folders = %+v %v", entries, err)
	}
}

func TestInitializeDatabasePreservesBrokenDatabase(t *testing.T) {
	dir := t.TempDir()
	brokenPath := filepath.Join(dir, "vaccineshit.db")
	if err := os.WriteFile(brokenPath, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := newAppAt(dir, "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.Bootstrap().Ready {
		t.Fatal("broken database unexpectedly started")
	}

	result := app.InitializeDatabase()
	if !result.OK {
		t.Fatalf("initialize database = %+v", result)
	}
	if result.PreservedPath == "" {
		t.Fatal("broken database was not preserved")
	}
	preserved, err := os.ReadFile(filepath.Join(result.PreservedPath, "vaccineshit.db"))
	if err != nil || string(preserved) != "not a sqlite database" {
		t.Fatalf("preserved database = %q %v", preserved, err)
	}
	if !app.Bootstrap().Ready {
		t.Fatal("app is not ready after initialization")
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login after initialization: %v", err)
	}
}

func TestInitializeDatabaseRefusesWhileDatabaseOpen(t *testing.T) {
	app := newTestApp(t)
	mustOK(t, app.CreateVaccine(VaccineInputDTO{Name: "Hepatitis B", Unit: "dose"}))

	result := app.InitializeDatabase()
	if result.OK || result.Code != string(domain.CodeConflict) {
		t.Fatalf("initialize while ready = %+v", result)
	}
	items, err := app.ListVaccines(false)
	if err != nil || len(items) != 1 || items[0].Name != "Hepatitis B" {
		t.Fatalf("vaccines after refused initialization = %+v %v", items, err)
	}
}

func TestInitializeDatabaseFallsBackNextToExecutable(t *testing.T) {
	dir := t.TempDir()
	brokenPath := filepath.Join(dir, "vaccineshit.db")
	if err := os.WriteFile(brokenPath, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newAppAt(dir, "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.Bootstrap().Ready {
		t.Fatal("broken database unexpectedly started")
	}

	// Make the recovery folder unusable so preserving the broken database and
	// opening a new one in the preferred directory both fail.
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	app.backupDir = filepath.Join(locked, "backups")
	fallbackDir := t.TempDir()
	app.fallbackDataDir = func() (string, error) { return fallbackDir, nil }

	result := app.InitializeDatabase()
	if !result.OK || result.Code != "initialized_fallback" {
		t.Fatalf("initialize database = %+v", result)
	}
	if !app.Bootstrap().Ready {
		t.Fatal("app is not ready after fallback initialization")
	}
	if app.dataDir != fallbackDir {
		t.Fatalf("data dir = %q, want %q", app.dataDir, fallbackDir)
	}
	if _, err := os.Stat(filepath.Join(fallbackDir, "vaccineshit.db")); err != nil {
		t.Fatalf("fallback database file: %v", err)
	}
	if kept, err := os.ReadFile(brokenPath); err != nil || string(kept) != "not a sqlite database" {
		t.Fatalf("broken database was modified: %q %v", kept, err)
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login after fallback initialization: %v", err)
	}
}

func TestInitializeDatabaseCreatesNewFileWithoutMovingOld(t *testing.T) {
	dir := t.TempDir()
	brokenPath := filepath.Join(dir, "vaccineshit.db")
	if err := os.WriteFile(brokenPath, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newAppAt(dir, "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.Bootstrap().Ready {
		t.Fatal("broken database unexpectedly started")
	}

	result, ok := app.initializeDatabaseInNewFileLocked(errors.New("sharing violation"), "")
	if !ok || !result.OK || result.Code != "initialized_new_file" {
		t.Fatalf("new-file initialization = %+v ok=%v", result, ok)
	}
	if app.dataDir != dir {
		t.Fatalf("data dir = %q, want %q", app.dataDir, dir)
	}
	if app.dbPath == brokenPath {
		t.Fatal("app still points at the broken database")
	}
	if _, err := os.Stat(app.dbPath); err != nil {
		t.Fatalf("recovery database file: %v", err)
	}
	if kept, err := os.ReadFile(brokenPath); err != nil || string(kept) != "not a sqlite database" {
		t.Fatalf("broken database was modified: %q %v", kept, err)
	}
	marker, err := os.ReadFile(filepath.Join(dir, activeDatabaseFileName))
	if err != nil || strings.TrimSpace(string(marker)) != filepath.Base(app.dbPath) {
		t.Fatalf("active database marker = %q %v", marker, err)
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login after new-file initialization: %v", err)
	}

	// The next start must reuse the recovery database.
	app.Shutdown(context.Background())
	restarted := newAppAt(dir, "test")
	t.Cleanup(func() { restarted.Shutdown(context.Background()) })
	if !restarted.Bootstrap().Ready {
		t.Fatalf("restart failed: %v", restarted.initErr)
	}
	if restarted.dbPath != app.dbPath {
		t.Fatalf("restart database = %q, want %q", restarted.dbPath, app.dbPath)
	}
}

func TestRememberedDatabasePathRejectsUnsafeNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"../outside.db", "/tmp/absolute.db", "notes.txt", "missing.db"} {
		if err := os.WriteFile(filepath.Join(dir, activeDatabaseFileName), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := rememberedDatabasePath(dir); got != filepath.Join(dir, "vaccineshit.db") {
			t.Fatalf("rememberedDatabasePath(%q) = %q", name, got)
		}
	}
}
