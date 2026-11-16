package appdata

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDirHonorsExplicitOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "data")
	t.Setenv("VACCINESHIT_DATA_DIR", dir)

	got, fallback, err := ResolveDir()
	if err != nil {
		t.Fatalf("ResolveDir() error = %v", err)
	}
	if fallback {
		t.Fatal("explicit override unexpectedly used the executable fallback")
	}
	if got != dir {
		t.Fatalf("ResolveDir() = %q, want %q", got, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("data directory = %v %v", info, err)
	}
}

func TestResolveFallbackAtUsesWritableDirectory(t *testing.T) {
	fallback := filepath.Join(t.TempDir(), "app")
	preferred := filepath.Join(t.TempDir(), "locked", "data")

	got, used, err := resolveFallbackAt(preferred, fallback, errors.New("permission denied"))
	if err != nil {
		t.Fatalf("resolveFallbackAt() error = %v", err)
	}
	if !used || got != fallback {
		t.Fatalf("fallback = %q, used = %v", got, used)
	}
	if info, err := os.Stat(fallback); err != nil || !info.IsDir() {
		t.Fatalf("fallback directory = %v %v", info, err)
	}
}

func TestResolveFallbackAtRejectsUnwritableDirectory(t *testing.T) {
	fallback := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(fallback, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveFallbackAt("/preferred", fallback, errors.New("permission denied"))
	if err == nil {
		t.Fatal("resolveFallbackAt() accepted a file as the fallback directory")
	}
}

func TestResolvePortableDirPrefersExecutableFolder(t *testing.T) {
	portable := t.TempDir()
	perUser := filepath.Join(t.TempDir(), "per-user")

	got, fallback, err := resolvePortableDirAt(
		func() (string, error) { return portable, nil },
		func() (string, error) { return perUser, nil },
	)
	if err != nil {
		t.Fatalf("resolvePortableDirAt() error = %v", err)
	}
	if fallback || got != portable {
		t.Fatalf("portable = %q, fallback = %v; want %q", got, fallback, portable)
	}
	if _, err := os.Stat(filepath.Join(perUser)); !os.IsNotExist(err) {
		t.Fatalf("per-user fallback directory was created: %v", err)
	}
}

func TestResolvePortableDirFallsBackToUserFolder(t *testing.T) {
	portable := filepath.Join(t.TempDir(), "VacTrack.exe")
	if err := os.WriteFile(portable, []byte("exe"), 0o600); err != nil {
		t.Fatal(err)
	}
	perUser := filepath.Join(t.TempDir(), "per-user")

	got, fallback, err := resolvePortableDirAt(
		func() (string, error) { return portable, nil },
		func() (string, error) { return perUser, nil },
	)
	if err != nil {
		t.Fatalf("resolvePortableDirAt() error = %v", err)
	}
	if !fallback || got != perUser {
		t.Fatalf("portable = %q, fallback = %v; want %q", got, fallback, perUser)
	}
	if info, err := os.Stat(perUser); err != nil || !info.IsDir() {
		t.Fatalf("per-user directory = %v %v", info, err)
	}
}

func TestMigrateLegacyDataCopiesDatabaseAndAuxiliaryFiles(t *testing.T) {
	legacy := t.TempDir()
	target := filepath.Join(t.TempDir(), "portable")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VACCINESHIT_DATA_DIR", legacy)

	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(legacy, "vaccineshit.db"), "database")
	write(filepath.Join(legacy, "vaccineshit.db-wal"), "wal")
	write(filepath.Join(legacy, "application.log"), "log")
	write(filepath.Join(legacy, "backups", "old.db"), "backup")
	write(filepath.Join(legacy, "secrets", "publisher-token"), "token")
	write(filepath.Join(legacy, "notes.txt"), "not app data")
	write(filepath.Join(target, "VacTrack.exe"), "binary")

	migrated, err := MigrateLegacyData(target)
	if err != nil {
		t.Fatalf("MigrateLegacyData() error = %v", err)
	}
	if !migrated {
		t.Fatal("MigrateLegacyData() did not migrate")
	}
	for _, name := range []string{"vaccineshit.db", "vaccineshit.db-wal", "application.log", "backups/old.db", "secrets/publisher-token"} {
		if _, err := os.Stat(filepath.Join(target, name)); err != nil {
			t.Errorf("migrated file %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "notes.txt")); !os.IsNotExist(err) {
		t.Errorf("unrelated file was copied: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(target, "VacTrack.exe")); err != nil || string(body) != "binary" {
		t.Errorf("existing executable changed: %q %v", body, err)
	}

	migrated, err = MigrateLegacyData(target)
	if err != nil || migrated {
		t.Fatalf("second MigrateLegacyData() = %v %v, want false nil", migrated, err)
	}
}

func TestEnsureExecutableDirKeepsDirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	before, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureExecutableDir(dir); err != nil {
		t.Fatalf("EnsureExecutableDir() error = %v", err)
	}
	after, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode().Perm() != after.Mode().Perm() {
		t.Fatalf("permissions changed from %v to %v", before.Mode().Perm(), after.Mode().Perm())
	}
}
