// Package backup provides consistent SQLite backups and staged restores.
// Live database files are never copied blindly; snapshots are created with
// VACUUM INTO and restores are validated before replacing anything.
package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Info describes a created backup file.
type Info struct {
	Path      string
	SizeBytes int64
	SHA256    string
	CreatedAt string
}

// Manager creates and validates backups for one database path.
type Manager struct {
	db        *sql.DB
	dbPath    string
	backupDir string
}

// New returns a backup manager. db must be the live handle.
func New(db *sql.DB, dbPath, backupDir string) *Manager {
	return &Manager{db: db, dbPath: dbPath, backupDir: backupDir}
}

// Create writes a consistent backup to dest using VACUUM INTO, validates it,
// and atomically moves it into place.
func (m *Manager) Create(ctx context.Context, dest string) (Info, error) {
	if strings.TrimSpace(dest) == "" {
		return Info{}, errors.New("backup destination is required")
	}
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return Info{}, fmt.Errorf("resolve backup destination: %w", err)
	}
	dbAbs, err := filepath.Abs(m.dbPath)
	if err != nil {
		return Info{}, fmt.Errorf("resolve database path: %w", err)
	}
	if samePath(destAbs, dbAbs) {
		return Info{}, errors.New("backup destination must not be the live database file")
	}
	if err := os.MkdirAll(filepath.Dir(destAbs), 0o700); err != nil {
		return Info{}, fmt.Errorf("create backup directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(destAbs), ".vaccineshit-backup-*.tmp")
	if err != nil {
		return Info{}, fmt.Errorf("reserve backup destination: %w", err)
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return Info{}, err
	}
	// VACUUM INTO requires that the destination does not exist.
	if err := os.Remove(tempPath); err != nil {
		return Info{}, err
	}
	defer os.Remove(tempPath)

	if _, err := m.db.ExecContext(ctx, `VACUUM INTO ?`, tempPath); err != nil {
		return Info{}, fmt.Errorf("create consistent database backup: %w", err)
	}
	if err := ensureFile(tempPath); err != nil {
		return Info{}, err
	}
	if err := Validate(ctx, tempPath); err != nil {
		return Info{}, fmt.Errorf("validate new backup: %w", err)
	}
	hash, size, err := hashFile(tempPath)
	if err != nil {
		return Info{}, err
	}
	if err := replaceFile(tempPath, destAbs); err != nil {
		return Info{}, fmt.Errorf("finalize backup: %w", err)
	}
	return Info{
		Path:      destAbs,
		SizeBytes: size,
		SHA256:    hash,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// Validate opens a backup read-only and checks integrity, foreign keys and the
// required application tables. It never migrates or modifies the file.
func Validate(ctx context.Context, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve backup: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}
	if info.IsDir() || info.Size() == 0 {
		return errors.New("backup is not a SQLite database file")
	}
	u := &url.URL{Scheme: "file", Path: abs}
	q := u.Query()
	q.Set("mode", "ro")
	q.Add("_pragma", "query_only(1)")
	q.Add("_pragma", "busy_timeout(2000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return fmt.Errorf("open backup for validation: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return fmt.Errorf("run integrity check: %w", err)
	}
	ok := false
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			rows.Close()
			return fmt.Errorf("read integrity check: %w", err)
		}
		if result != "ok" {
			rows.Close()
			return fmt.Errorf("backup integrity check failed: %s", result)
		}
		ok = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read integrity check: %w", err)
	}
	rows.Close()
	if !ok {
		return errors.New("backup integrity check returned no result")
	}

	fkRows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("run foreign key check: %w", err)
	}
	if fkRows.Next() {
		fkRows.Close()
		return errors.New("backup contains foreign key violations")
	}
	if err := fkRows.Err(); err != nil {
		fkRows.Close()
		return fmt.Errorf("read foreign key check: %w", err)
	}
	fkRows.Close()

	required := []string{
		"schema_migrations", "vaccines", "patients", "monthly_inventory",
		"usage_entries", "inventory_adjustments", "audit_events", "app_settings",
	}
	for _, table := range required {
		var count int
		if err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			return fmt.Errorf("inspect backup schema: %w", err)
		}
		if count != 1 {
			return fmt.Errorf("backup is missing required table %q", table)
		}
	}
	return nil
}

// RestorePreparation is a validated backup copied next to the live database.
// Call CommitAfterClose only after the live database handle is closed.
type RestorePreparation struct {
	dbPath       string
	stagingPath  string
	recoveryDir  string
	recoveryPath string
	committed    bool
}

// PrepareRestore validates a backup and stages a verified copy. The live
// database is not modified until CommitAfterClose is called.
func (m *Manager) PrepareRestore(ctx context.Context, backupPath string) (*RestorePreparation, error) {
	absSource, err := filepath.Abs(backupPath)
	if err != nil {
		return nil, fmt.Errorf("resolve backup: %w", err)
	}
	dbAbs, err := filepath.Abs(m.dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if samePath(absSource, dbAbs) {
		return nil, errors.New("choose a backup file, not the live database")
	}
	if err := Validate(ctx, absSource); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(m.backupDir, 0o700); err != nil {
		return nil, fmt.Errorf("create recovery directory: %w", err)
	}
	staging, err := os.CreateTemp(filepath.Dir(dbAbs), ".vaccineshit-restore-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("reserve restore staging file: %w", err)
	}
	stagingPath := staging.Name()
	if err := staging.Close(); err != nil {
		_ = os.Remove(stagingPath)
		return nil, err
	}
	p := &RestorePreparation{
		dbPath:      dbAbs,
		stagingPath: stagingPath,
		recoveryDir: m.backupDir,
	}
	if err := copyFile(absSource, stagingPath); err != nil {
		p.Abort()
		return nil, fmt.Errorf("stage backup copy: %w", err)
	}
	if err := ensureFile(stagingPath); err != nil {
		p.Abort()
		return nil, err
	}
	if err := Validate(ctx, stagingPath); err != nil {
		p.Abort()
		return nil, fmt.Errorf("validate staged backup: %w", err)
	}
	return p, nil
}

// CommitAfterClose moves the current database into the recovery directory and
// atomically installs the staged backup. The database handle must be closed.
func (p *RestorePreparation) CommitAfterClose() error {
	if p.committed {
		return errors.New("restore was already committed")
	}
	_ = os.Remove(p.dbPath + "-wal")
	_ = os.Remove(p.dbPath + "-shm")
	recovery := filepath.Join(p.recoveryDir, "pre-restore-"+time.Now().UTC().Format("20060102T150405Z")+".db")
	if err := os.Rename(p.dbPath, recovery); err != nil {
		return fmt.Errorf("preserve current database: %w", err)
	}
	if err := os.Rename(p.stagingPath, p.dbPath); err != nil {
		// Put the original database back immediately.
		if restoreErr := os.Rename(recovery, p.dbPath); restoreErr != nil {
			return fmt.Errorf("install restored database: %w (original remains at %s; recovery failed: %v)", err, recovery, restoreErr)
		}
		return fmt.Errorf("install restored database: %w", err)
	}
	p.stagingPath = ""
	p.recoveryPath = recovery
	p.committed = true
	return nil
}

// Rollback puts the pre-restore database back after a failed open. The failed
// restored copy is retained beside the recovery file for diagnosis.
func (p *RestorePreparation) Rollback() error {
	if !p.committed {
		p.Abort()
		return nil
	}
	failed := filepath.Join(p.recoveryDir, "failed-restore-"+time.Now().UTC().Format("20060102T150405Z")+".db")
	if err := os.Rename(p.dbPath, failed); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("move failed restored database aside: %w", err)
	}
	_ = os.Remove(p.dbPath + "-wal")
	_ = os.Remove(p.dbPath + "-shm")
	if err := os.Rename(p.recoveryPath, p.dbPath); err != nil {
		return fmt.Errorf("restore original database (original at %s): %w", p.recoveryPath, err)
	}
	p.committed = false
	return nil
}

// RecoveryPath is the preserved pre-restore database after a successful
// commit. Empty before commit.
func (p *RestorePreparation) RecoveryPath() string {
	return p.recoveryPath
}

// Abort removes an uncommitted staging copy.
func (p *RestorePreparation) Abort() {
	if p.stagingPath != "" {
		_ = os.Remove(p.stagingPath)
		p.stagingPath = ""
	}
}

func ensureFile(path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect backup file: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open backup for sync: %w", err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync backup: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return out.Close()
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(src, dst)
}

func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(ai, bi)
	}
	return false
}
