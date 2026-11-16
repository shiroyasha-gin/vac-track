// Package appdata resolves the per-user application data directory.
package appdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const AppDirName = "vaccineshit"

// Dir returns the OS-appropriate per-user data directory for the application.
// Tests and development previews may override it with VACCINESHIT_DATA_DIR.
func Dir() (string, error) {
	if override := os.Getenv("VACCINESHIT_DATA_DIR"); override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve VACCINESHIT_DATA_DIR: %w", err)
		}
		return abs, nil
	}

	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				return "", fmt.Errorf("resolve local application data directory: %w", err)
			}
		}
		return filepath.Join(base, "VaccineInventory"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "VaccineInventory"), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, AppDirName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", AppDirName), nil
	}
}

// EnsureDir creates the directory with private permissions.
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create application data directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("protect application data directory: %w", err)
	}
	return nil
}

// DirNextToExecutable returns the directory that holds the running executable.
// It is the last-resort data location when the per-user directory is not
// writable, such as a locked-down or roaming Windows profile.
func DirNextToExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return filepath.Dir(executable), nil
}

// EnsureExecutableDir checks that the executable's directory can hold the
// database. It creates the directory when missing but never changes its
// permissions: that directory also holds the application binary.
func EnsureExecutableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create fallback data directory: %w", err)
	}
	probe, err := os.CreateTemp(dir, ".vactrack-write-*")
	if err != nil {
		return fmt.Errorf("write to fallback data directory: %w", err)
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return nil
}

// ResolveDir returns the data directory. Windows builds are portable: the
// database, backups and log live next to the executable. If that directory is
// not writable, the app falls back to the per-user data directory.
func ResolveDir() (string, bool, error) {
	if override := os.Getenv("VACCINESHIT_DATA_DIR"); override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", false, err
		}
		if err := EnsureDir(abs); err != nil {
			return "", false, err
		}
		return abs, false, nil
	}
	if runtime.GOOS == "windows" {
		return resolvePortableDir()
	}
	preferred, err := Dir()
	if err != nil {
		return "", false, err
	}
	if err := EnsureDir(preferred); err != nil {
		return resolveFallback(preferred, err)
	}
	return preferred, false, nil
}

// resolvePortableDir prefers the executable's directory on Windows and falls
// back to the per-user directory when the executable lives somewhere read-only,
// such as Program Files.
func resolvePortableDir() (string, bool, error) {
	return resolvePortableDirAt(DirNextToExecutable, Dir)
}

// resolvePortableDirAt is the testable part of the Windows portable lookup.
func resolvePortableDirAt(portableDir, userDir func() (string, error)) (string, bool, error) {
	portable, err := portableDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve executable directory: %w", err)
	}
	if err := EnsureExecutableDir(portable); err == nil {
		return portable, false, nil
	}
	perUser, perUserErr := userDir()
	if perUserErr != nil {
		return "", false, fmt.Errorf("write next to executable %s: %v; resolve user data directory: %w", portable, err, perUserErr)
	}
	if perUserErr := EnsureDir(perUser); perUserErr != nil {
		return "", false, fmt.Errorf("write next to executable %s: %v; prepare user data directory %s: %w", portable, err, perUser, perUserErr)
	}
	return perUser, true, nil
}

// MigrateLegacyData copies an existing per-user database into a portable
// directory that has no database yet. It runs once, before the database opens.
// Existing files in the target directory are never overwritten.
func MigrateLegacyData(target string) (bool, error) {
	legacy, err := Dir()
	if err != nil {
		return false, err
	}
	if sameDir(legacy, target) {
		return false, nil
	}
	entries, err := os.ReadDir(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read legacy data directory %s: %w", legacy, err)
	}
	if hasDatabase(target) || !hasDatabase(legacy) {
		return false, nil
	}

	copied := false
	for _, entry := range entries {
		name := entry.Name()
		source := filepath.Join(legacy, name)
		destination := filepath.Join(target, name)
		if entry.IsDir() {
			if name != "backups" && name != "secrets" {
				continue
			}
			if _, err := os.Stat(destination); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return copied, fmt.Errorf("inspect migration destination %s: %w", destination, err)
			}
			if err := os.CopyFS(destination, os.DirFS(source)); err != nil {
				return copied, fmt.Errorf("copy %s to %s: %w", source, destination, err)
			}
			copied = true
			continue
		}
		if !legacyDataFile(name) {
			continue
		}
		if _, err := os.Stat(destination); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return copied, fmt.Errorf("inspect migration destination %s: %w", destination, err)
		}
		if err := copyPrivateFile(source, destination); err != nil {
			return copied, err
		}
		copied = true
	}
	return copied, nil
}

func legacyDataFile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "active-database.txt" ||
		lower == "application.log" ||
		strings.HasSuffix(lower, ".db") ||
		strings.HasSuffix(lower, ".db-wal") ||
		strings.HasSuffix(lower, ".db-shm")
}

func hasDatabase(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasSuffix(lower, ".db") || lower == "active-database.txt" {
			return true
		}
	}
	return false
}

func copyPrivateFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open legacy file %s: %w", source, err)
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create migrated file %s: %w", destination, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy legacy file %s: %w", source, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close migrated file %s: %w", destination, err)
	}
	return nil
}

func sameDir(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return absA == absB
}

// resolveFallback is the testable part of ResolveDir.
func resolveFallback(preferred string, preferredErr error) (string, bool, error) {
	fallback, err := DirNextToExecutable()
	if err != nil {
		return "", false, fmt.Errorf("%v; fallback: %w", preferredErr, err)
	}
	return resolveFallbackAt(preferred, fallback, preferredErr)
}

// resolveFallbackAt checks a concrete fallback directory.
func resolveFallbackAt(preferred, fallback string, preferredErr error) (string, bool, error) {
	if err := EnsureExecutableDir(fallback); err != nil {
		return "", false, fmt.Errorf("%v; fallback %s: %w", preferredErr, fallback, err)
	}
	return fallback, true, nil
}
