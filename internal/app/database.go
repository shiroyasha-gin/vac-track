package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vaccineshit/internal/atomicfile"
	"vaccineshit/internal/backup"
	"vaccineshit/internal/domain"
)

// databaseFileSuffixes lists the files SQLite uses for the local database.
// WAL and shared-memory files must move with the main file so a fresh start
// never mixes old pages with a new database.
var databaseFileSuffixes = []string{"", "-wal", "-shm"}

// InitializeDatabaseResultDTO reports a manual fresh start from the recovery
// screen. PreservedPath points at the folder that keeps the previous files.
type InitializeDatabaseResultDTO struct {
	OK            bool   `json:"ok"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	PreservedPath string `json:"preservedPath,omitempty"`
}

// InitializeDatabase creates a new empty local database after moving any
// existing database files into the backup folder. It is only available while
// the app has no open database, so it cannot wipe a running station.
func (a *App) InitializeDatabase() InitializeDatabaseResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		return InitializeDatabaseResultDTO{
			Code:    string(domain.CodeConflict),
			Message: "قاعدة البيانات تعمل بالفعل. لم يتم إنشاء قاعدة بيانات جديدة.",
		}
	}
	return a.initializeDatabaseLocked(true)
}

// initializeDatabaseLocked performs the fresh start and retries once in the
// executable's directory when the preferred data directory cannot be used.
func (a *App) initializeDatabaseLocked(allowFallback bool) InitializeDatabaseResultDTO {
	preserved, err := a.preserveDatabaseFilesLocked("pre-initialize")
	if err != nil {
		a.logf("preserve database files before initialization", "error", err)
		if allowFallback {
			if result, ok := a.initializeDatabaseInNewFileLocked(err, ""); ok {
				return result
			}
			return a.initializeDatabaseAtFallbackLocked(err, "")
		}
		return InitializeDatabaseResultDTO{
			Code:    "initialize_failed",
			Message: "تعذر حفظ ملفات قاعدة البيانات السابقة. لم يتم إنشاء قاعدة بيانات جديدة.",
		}
	}
	if err := a.openLocked(); err != nil {
		a.logf("initialize new database", "error", err)
		a.initErr = err
		if preserved != "" {
			if rollbackErr := a.restorePreservedFilesLocked(preserved); rollbackErr != nil {
				a.logf("restore preserved database files", "error", rollbackErr)
				if allowFallback {
					if result, ok := a.initializeDatabaseInNewFileLocked(err, preserved); ok {
						return result
					}
					return a.initializeDatabaseAtFallbackLocked(err, preserved)
				}
				return InitializeDatabaseResultDTO{
					Code: "initialize_failed",
					Message: fmt.Sprintf(
						"تعذر إنشاء قاعدة بيانات جديدة، وتعذر إرجاع الملفات السابقة. الملفات محفوظة في %s.",
						preserved,
					),
					PreservedPath: preserved,
				}
			}
			// The old files are back in place; their backup copy is gone.
			preserved = ""
		}
		if allowFallback {
			if result, ok := a.initializeDatabaseInNewFileLocked(err, preserved); ok {
				return result
			}
			return a.initializeDatabaseAtFallbackLocked(err, preserved)
		}
		return InitializeDatabaseResultDTO{
			Code:          "initialize_failed",
			Message:       "تعذر إنشاء قاعدة بيانات جديدة. " + a.userError(err).Error(),
			PreservedPath: preserved,
		}
	}
	return initializedResult(preserved)
}

// initializeDatabaseAtFallbackLocked retries a failed fresh start in the
// executable's directory. The preferred directory and any preserved files stay
// untouched, so the old database remains available for a later repair.
func (a *App) initializeDatabaseAtFallbackLocked(cause error, preserved string) InitializeDatabaseResultDTO {
	previousDir := a.dataDir
	if err := a.rebaseToFallbackLocked(); err != nil {
		a.logf("initialize database in fallback directory", "error", err)
		return InitializeDatabaseResultDTO{
			Code: "initialize_failed",
			Message: fmt.Sprintf(
				"تعذر إنشاء قاعدة بيانات جديدة في %s، وتعذر استخدام مجلد التطبيق: %s",
				previousDir, err.Error(),
			),
			PreservedPath: preserved,
		}
	}
	if err := a.openLocked(); err != nil {
		a.logf("open fallback database", "error", err)
		a.rebaseLocked(previousDir)
		a.initErr = cause
		return InitializeDatabaseResultDTO{
			Code:          "initialize_failed",
			Message:       fmt.Sprintf("تعذر إنشاء قاعدة بيانات جديدة في %s أو بجوار ملف التطبيق: %s", previousDir, err.Error()),
			PreservedPath: preserved,
		}
	}
	result := initializedResult(preserved)
	result.Code = "initialized_fallback"
	result.Message = fmt.Sprintf(
		"تعذر استخدام مجلد البيانات الأصلي (%s). أُنشئت قاعدة بيانات جديدة بجوار ملف التطبيق في %s. سجّل الدخول باسم admin وكلمة المرور admin123 ثم غيّر كلمة المرور.",
		previousDir, a.dataDir,
	)
	if preserved != "" {
		result.Message += " حُفظت ملفات قاعدة البيانات السابقة في " + preserved + "."
	}
	return result
}

// activeDatabaseFileName stores the database file chosen after a recovery.
// It keeps a recovered database in use on the next start without moving the
// original broken file.
const activeDatabaseFileName = "active-database.txt"

// rememberedDatabasePath returns the recovery database when its marker and
// file both exist; otherwise it returns the default database path.
func rememberedDatabasePath(dataDir string) string {
	defaultPath := filepath.Join(dataDir, "vaccineshit.db")
	body, err := os.ReadFile(filepath.Join(dataDir, activeDatabaseFileName))
	if err != nil {
		return defaultPath
	}
	name := strings.TrimSpace(string(body))
	if name == "" || filepath.Base(name) != name || !strings.HasSuffix(strings.ToLower(name), ".db") {
		return defaultPath
	}
	path := filepath.Join(dataDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return defaultPath
	}
	return path
}

// initializeDatabaseInNewFileLocked creates a new database beside the current
// one instead of moving a file that Windows may still hold open. The chosen
// name is recorded so the next start uses it.
func (a *App) initializeDatabaseInNewFileLocked(cause error, preserved string) (InitializeDatabaseResultDTO, bool) {
	previousPath := a.dbPath
	previousBackups := a.backups
	path, err := a.newDatabasePathLocked()
	if err != nil {
		a.logf("choose new database path", "error", err)
		return InitializeDatabaseResultDTO{}, false
	}
	a.dbPath = path
	a.backups = backup.New(nil, path, a.backupDir)
	if err := a.openLocked(); err != nil {
		a.logf("initialize database in new file", "error", err)
		removeDatabaseFiles(path)
		a.dbPath = previousPath
		a.backups = previousBackups
		a.initErr = cause
		return InitializeDatabaseResultDTO{}, false
	}
	if err := a.rememberDatabaseNameLocked(filepath.Base(path)); err != nil {
		a.logf("record recovery database path", "error", err)
		if a.store != nil {
			_ = a.store.Close()
			a.store = nil
			a.services = nil
		}
		removeDatabaseFiles(path)
		a.dbPath = previousPath
		a.backups = previousBackups
		a.initErr = cause
		return InitializeDatabaseResultDTO{}, false
	}
	message := fmt.Sprintf(
		"تعذر فتح قاعدة البيانات القديمة. أُنشئت قاعدة بيانات جديدة فارغة في %s. بقيت الملفات القديمة في %s. سجّل الدخول باسم admin وكلمة المرور admin123 ثم غيّر كلمة المرور.",
		path, previousPath,
	)
	if preserved != "" {
		message += " حُفظت نسخة من الملفات السابقة في " + preserved + "."
	}
	return InitializeDatabaseResultDTO{
		OK:            true,
		Code:          "initialized_new_file",
		Message:       message,
		PreservedPath: preserved,
	}, true
}

func (a *App) newDatabasePathLocked() (string, error) {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for index := 0; index < 100; index++ {
		name := fmt.Sprintf("vaccineshit-recovered-%s.db", stamp)
		if index > 0 {
			name = fmt.Sprintf("vaccineshit-recovered-%s-%d.db", stamp, index)
		}
		path := filepath.Join(a.dataDir, name)
		if databaseFilesExist(path) {
			continue
		}
		return path, nil
	}
	return "", errors.New("no free recovery database name")
}

func databaseFilesExist(path string) bool {
	for _, suffix := range databaseFileSuffixes {
		if _, err := os.Stat(path + suffix); err == nil {
			return true
		} else if !errors.Is(err, os.ErrNotExist) {
			return true
		}
	}
	return false
}

func removeDatabaseFiles(path string) {
	for _, suffix := range databaseFileSuffixes {
		_ = os.Remove(path + suffix)
	}
}

func (a *App) rememberDatabaseNameLocked(name string) error {
	path := filepath.Join(a.dataDir, activeDatabaseFileName)
	if err := atomicfile.Write(path, []byte(name+"\n"), 0o600); err != nil {
		return fmt.Errorf("write active database marker: %w", err)
	}
	return nil
}

func initializedResult(preserved string) InitializeDatabaseResultDTO {
	message := "تم إنشاء قاعدة بيانات جديدة فارغة. سجّل الدخول باسم admin وكلمة المرور admin123 ثم غيّر كلمة المرور."
	if preserved != "" {
		message += " حُفظت ملفات قاعدة البيانات السابقة في " + preserved + "."
	}
	return InitializeDatabaseResultDTO{OK: true, Message: message, PreservedPath: preserved}
}

// prepareDatabaseLocked makes a missing or empty database explicit. SQLite
// creates the file on open; this step logs the fresh start and moves orphaned
// WAL or shared-memory files aside so they cannot contaminate the new database.
// a.mu must be held.
func (a *App) prepareDatabaseLocked() error {
	info, err := os.Stat(a.dbPath)
	missing := errors.Is(err, os.ErrNotExist)
	if err != nil && !missing {
		return fmt.Errorf("inspect local database: %w", err)
	}
	empty := err == nil && info.Size() == 0
	if err == nil && !empty {
		return nil
	}
	preserved, err := a.preserveDatabaseFilesLocked("pre-initialize")
	if err != nil {
		return err
	}
	if preserved != "" {
		a.logf("preserved orphaned database files", "dir", preserved)
	}
	if missing {
		a.logf("initializing new local database", "path", a.dbPath)
	} else {
		a.logf("initializing empty local database", "path", a.dbPath)
	}
	return nil
}

// preserveDatabaseFilesLocked moves the database and its WAL files into a
// timestamped folder in the backup directory. It returns an empty path when no
// database file exists.
func (a *App) preserveDatabaseFilesLocked(prefix string) (string, error) {
	files := make([]string, 0, len(databaseFileSuffixes))
	for _, suffix := range databaseFileSuffixes {
		path := a.dbPath + suffix
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect database file %s: %w", path, err)
		}
	}
	if len(files) == 0 {
		return "", nil
	}
	dir, err := a.newRecoveryDirLocked(prefix)
	if err != nil {
		return "", err
	}
	moved := make([]string, 0, len(files))
	for _, source := range files {
		destination := filepath.Join(dir, filepath.Base(source))
		if err := os.Rename(source, destination); err != nil {
			for _, path := range moved {
				_ = os.Rename(filepath.Join(dir, filepath.Base(path)), path)
			}
			_ = os.Remove(dir)
			return "", fmt.Errorf("move database file %s: %w", source, err)
		}
		moved = append(moved, source)
	}
	return dir, nil
}

func (a *App) newRecoveryDirLocked(prefix string) (string, error) {
	base := filepath.Join(a.backupDir, prefix+"-"+time.Now().UTC().Format("20060102T150405Z"))
	dir := base
	for index := 1; ; index++ {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			break
		} else if err != nil {
			return "", fmt.Errorf("inspect recovery folder: %w", err)
		}
		dir = fmt.Sprintf("%s-%d", base, index)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create recovery folder: %w", err)
	}
	return dir, nil
}

// restorePreservedFilesLocked removes any files a failed fresh start created
// and puts the preserved database files back in place. a.mu must be held.
func (a *App) restorePreservedFilesLocked(dir string) error {
	if dir == "" {
		return nil
	}
	for _, suffix := range databaseFileSuffixes {
		if err := os.Remove(a.dbPath + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove new database file: %w", err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read recovery folder: %w", err)
	}
	for _, entry := range entries {
		source := filepath.Join(dir, entry.Name())
		if err := os.Rename(source, filepath.Join(a.dbPath, entry.Name())); err != nil {
			return fmt.Errorf("restore database file %s: %w", entry.Name(), err)
		}
	}
	_ = os.Remove(dir)
	return nil
}

// databaseTimeout keeps fresh initialization bounded like normal startup.
const databaseTimeout = 30 * time.Second

func (a *App) databaseContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(a.context(), databaseTimeout)
}
