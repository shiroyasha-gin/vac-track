// Package appdata resolves the per-user application data directory.
package appdata

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
