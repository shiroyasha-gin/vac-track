// Package atomicfile writes and replaces files without leaving a partially
// written destination. It also handles Windows, where os.Rename cannot replace
// an existing file.
package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Replace moves src over dst. On Windows the destination is removed first when
// a direct rename is refused.
func Replace(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(src, dst)
}

// Write creates parent directories, writes body to a temporary file in the
// destination directory, syncs it, and atomically replaces path.
func Write(path string, body []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".atomic-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(perm); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(body); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return Replace(tempPath, path)
}
