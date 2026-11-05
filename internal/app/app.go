// Package app is the Wails binding layer. It exposes explicit, sanitized DTOs
// to the private desktop UI, serializes operations, and owns startup, backup
// restore and publication orchestration. No public DTO contains patient names,
// patient codes, notes or individual usage events.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vaccineshit/internal/appdata"
	"vaccineshit/internal/backup"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/publishing"
	"vaccineshit/internal/service"
	"vaccineshit/internal/store/sqlite"
)

const appVersionFallback = "0.1.0"

// App is the object bound to the Wails frontend.
type App struct {
	mu      sync.Mutex
	ctx     context.Context
	version string

	dataDir   string
	dbPath    string
	backupDir string
	logPath   string
	logger    *slog.Logger
	logFile   *os.File

	store       *sqlite.DB
	services    *service.Services
	backups     *backup.Manager
	credentials *publishing.Credentials
	client      *publishing.Client
	initErr     error
	now         func() time.Time

	pendingPreview *previewState
}

type previewState struct {
	month    string
	hash     string
	snapshot publishing.Snapshot
	state    string
}

// NewApp resolves the OS user-data directory and opens the local database. A
// startup failure is retained so the UI can show a recovery screen instead of
// failing to bind. The Wails main function always receives a usable App.
func NewApp(version string) *App {
	if strings.TrimSpace(version) == "" {
		version = appVersionFallback
	}
	app := &App{version: version, ctx: context.Background()}
	dataDir, err := appdata.Dir()
	if err != nil {
		app.initErr = fmt.Errorf("resolve application data directory: %w", err)
		return app
	}
	app.configurePaths(dataDir)
	if err := appdata.EnsureDir(dataDir); err != nil {
		app.initErr = err
		return app
	}
	app.initLogging()
	app.mu.Lock()
	defer app.mu.Unlock()
	if err := app.openLocked(); err != nil {
		app.initErr = err
	}
	return app
}

// newAppAt is the deterministic constructor used by tests.
func newAppAt(dataDir, version string) *App {
	app := &App{version: version, ctx: context.Background()}
	app.configurePaths(dataDir)
	app.initLogging()
	if err := appdata.EnsureDir(dataDir); err != nil {
		app.initErr = err
		return app
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	if err := app.openLocked(); err != nil {
		app.initErr = err
	}
	return app
}

func (a *App) configurePaths(dataDir string) {
	a.dataDir = dataDir
	a.dbPath = filepath.Join(dataDir, "vaccineshit.db")
	a.backupDir = filepath.Join(dataDir, "backups")
	a.logPath = filepath.Join(dataDir, "application.log")
	a.now = func() time.Time { return time.Now().UTC() }
	a.credentials = publishing.NewCredentials(publishing.DefaultFallbackPath(dataDir))
	a.client = publishing.NewClient()
	a.backups = backup.New(nil, a.dbPath, a.backupDir)
}

func (a *App) initLogging() {
	file, err := os.OpenFile(a.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		a.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
		return
	}
	_ = file.Chmod(0o600)
	a.logFile = file
	a.logger = slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// Startup records the Wails runtime context used by native file dialogs.
func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx != nil {
		a.ctx = ctx
	}
}

// Shutdown flushes and closes the local database.
func (a *App) Shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		if err := a.store.Close(); err != nil {
			a.logf("close database", "error", err)
		}
		a.store = nil
		a.services = nil
	}
	if a.logFile != nil {
		if err := a.logFile.Sync(); err != nil {
			_ = err
		}
		_ = a.logFile.Close()
		a.logFile = nil
	}
}

// openLocked opens the database and wires the service graph. a.mu must be held.
func (a *App) openLocked() error {
	if a.store != nil {
		_ = a.store.Close()
		a.store = nil
		a.services = nil
	}
	ctx, cancel := context.WithTimeout(a.context(), 30*time.Second)
	defer cancel()
	store, err := sqlite.Open(ctx, sqlite.Config{
		Path:      a.dbPath,
		BackupDir: a.backupDir,
	})
	if err != nil {
		return err
	}
	a.store = store
	a.services = service.New(store, service.Options{})
	a.backups = backup.New(store.DB(), store.Path(), a.backupDir)
	a.initErr = nil
	a.pendingPreview = nil
	return nil
}

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// readyLocked converts a startup failure into a safe error.
func (a *App) readyLocked() error {
	if a.services == nil {
		if a.initErr != nil {
			return fmt.Errorf("the local database is not available: %w", a.initErr)
		}
		return errors.New("the local database is not available")
	}
	return nil
}

// Bootstrap describes startup state. It never fails: a fatal startup error is a
// value so the UI can render a recovery screen with restore controls.
func (a *App) Bootstrap() BootstrapInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	info := BootstrapInfo{
		DataDir:      a.dataDir,
		DatabasePath: a.dbPath,
		Version:      a.version,
		Today:        time.Now().UTC().Format("2006-01-02"),
		CurrentMonth: time.Now().UTC().Format("2006-01"),
	}
	if a.services == nil {
		message := "The local database could not be opened."
		if a.initErr != nil {
			message = a.initErr.Error()
		}
		info.FatalError = message
		return info
	}
	info.Ready = true
	ctx, cancel := context.WithTimeout(a.context(), 10*time.Second)
	defer cancel()
	threshold, err := a.services.Settings.LowStockThreshold(ctx)
	if err != nil {
		a.logf("read low-stock threshold", "error", err)
	} else {
		info.LowStockThreshold = threshold
	}
	endpoint, err := a.services.Settings.PublisherEndpoint(ctx)
	if err != nil {
		a.logf("read publisher endpoint", "error", err)
	}
	info.PublisherEndpoint = endpoint
	info.PublisherTokenConfigured, info.PublisherTokenStorage = a.tokenState()
	return info
}

// tokenState reports whether a publisher token exists without returning it.
func (a *App) tokenState() (bool, string) {
	if a.credentials == nil {
		return false, ""
	}
	_, storage, err := a.credentials.Load(publishing.PublisherAccount)
	if err == nil {
		return true, string(storage)
	}
	if !errors.Is(err, publishing.ErrCredentialsNotFound) {
		a.logf("read publisher credential", "error", err)
	}
	return false, ""
}

func (a *App) logf(message string, args ...any) {
	if a.logger != nil {
		a.logger.Error(message, args...)
	}
}

// userError preserves structured domain errors and sanitizes everything else so
// raw SQL, paths or credential details never reach the UI.
func (a *App) userError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := domain.AsError(err); ok {
		return err
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("The operation was cancelled.")
	}
	a.logf("operation failed", "error", err)
	return errors.New("The operation could not be completed. Details were written to the local application log.")
}

func (a *App) actionError(err error) ActionResult {
	if err == nil {
		return failedResult(domain.CodeInvalidInput, "", "The operation did not complete.")
	}
	if de, ok := domain.AsError(err); ok {
		return failedResult(de.Code, de.Field, de.Message)
	}
	a.logf("operation failed", "error", err)
	return failedResult("internal_error", "", "The operation could not be completed. Details were written to the local application log.")
}

func (a *App) readyError() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.readyLocked()
}

func (a *App) opContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(a.context(), 30*time.Second)
}
