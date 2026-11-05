package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/store/generated"
)

// CreateBackup asks for a destination and writes a verified consistent backup.
func (a *App) CreateBackup() BackupResultDTO {
	a.mu.Lock()
	ctx := a.ctx
	dir := a.backupDir
	ready := a.readyLocked() == nil
	a.mu.Unlock()
	if !ready {
		return backupFailed("The local database is not available, so a backup cannot be created.")
	}
	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		DefaultDirectory:     dir,
		DefaultFilename:      "vaccineshit-backup-" + time.Now().UTC().Format("20060102-150405") + ".db",
		Title:                "Save verified backup",
		CanCreateDirectories: true,
		Filters:              []runtime.FileFilter{{DisplayName: "SQLite database (*.db)", Pattern: "*.db"}},
	})
	if err != nil {
		return backupFailed("The save dialog could not be opened.")
	}
	if strings.TrimSpace(path) == "" {
		return BackupResultDTO{Code: "cancelled", Message: "Backup cancelled."}
	}
	return a.backupTo(path)
}

// BackupTo creates a verified backup at an explicit path.
func (a *App) backupTo(path string) BackupResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return backupFailed(a.userError(err).Error())
	}
	ctx, cancel := a.opContext()
	defer cancel()
	info, err := a.backups.Create(ctx, path)
	if err != nil {
		a.logf("create backup", "error", err)
		return backupFailed("The backup could not be created. Details were written to the local application log.")
	}
	if a.store != nil {
		if err := a.store.WithTx(ctx, func(q generated.Querier) error {
			return audit.Record(ctx, q, a.now(), audit.Event{
				Type: audit.EventBackupCreated, EntityType: "backup",
				EntityID: filepath.Base(info.Path),
				Metadata: map[string]any{"size_bytes": info.SizeBytes, "sha256": info.SHA256},
			})
		}); err != nil {
			a.logf("record backup audit event", "error", err)
		}
	}
	return backupResult(info)
}

// RestoreBackup asks for a backup file and restores it after validation. The UI
// must obtain explicit confirmation before calling this method.
func (a *App) RestoreBackup() BackupResultDTO {
	a.mu.Lock()
	ctx := a.ctx
	dir := a.backupDir
	a.mu.Unlock()
	path, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		DefaultDirectory: dir,
		Title:            "Choose a verified backup to restore",
		Filters:          []runtime.FileFilter{{DisplayName: "SQLite database (*.db)", Pattern: "*.db"}},
	})
	if err != nil {
		return backupFailed("The open dialog could not be opened.")
	}
	if strings.TrimSpace(path) == "" {
		return BackupResultDTO{Code: "cancelled", Message: "Restore cancelled."}
	}
	return a.restoreFrom(path)
}

// RestoreFrom validates a backup, preserves the current database, installs the
// backup and reopens the service graph. If reopening fails, the pre-restore
// database is put back automatically.
func (a *App) restoreFrom(path string) BackupResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, cancel := a.opContext()
	defer cancel()

	if a.store != nil {
		if err := a.store.Close(); err != nil {
			a.logf("close database before restore", "error", err)
			return backupFailed("The current database could not be closed safely. Restore was not attempted.")
		}
		a.store = nil
		a.services = nil
	}
	prep, err := a.backups.PrepareRestore(ctx, path)
	if err != nil {
		a.logf("prepare restore", "error", err)
		a.reopenQuietlyLocked()
		return backupFailed("The chosen backup failed validation. The current database was not changed.")
	}
	if err := prep.CommitAfterClose(); err != nil {
		a.logf("commit restore", "error", err)
		prep.Abort()
		a.reopenQuietlyLocked()
		return backupFailed("The backup could not be installed. The current database was not changed.")
	}
	if err := a.openLocked(); err != nil {
		a.logf("open restored database", "error", err)
		if rollbackErr := prep.Rollback(); rollbackErr != nil {
			a.logf("roll back restored database", "error", rollbackErr)
			return backupFailed(fmt.Sprintf(
				"The restored database could not be opened and automatic rollback failed. Recovery files are in %s.", a.backupDir))
		}
		if reopenErr := a.openLocked(); reopenErr != nil {
			a.logf("reopen original database", "error", reopenErr)
			return backupFailed("The restored database could not be opened. The original database was restored but could not be reopened.")
		}
		return backupFailed("The restored database could not be opened. The original database was restored.")
	}
	if a.store != nil {
		if err := a.store.WithTx(ctx, func(q generated.Querier) error {
			return audit.Record(ctx, q, a.now(), audit.Event{
				Type: audit.EventBackupRestored, EntityType: "backup",
				EntityID: filepath.Base(path), Metadata: map[string]any{"validated": true},
			})
		}); err != nil {
			a.logf("record restore audit event", "error", err)
		}
	}
	return BackupResultDTO{
		OK: true, Message: "Backup restored and verified. The previous database was preserved in the backup folder.",
		Path: path,
	}
}

func (a *App) reopenQuietlyLocked() {
	if err := a.openLocked(); err != nil {
		a.logf("reopen database after failed restore", "error", err)
		a.initErr = err
	}
}

func backupFailed(message string) BackupResultDTO {
	return BackupResultDTO{Code: "backup_error", Message: message}
}

// ExportAggregateCSV asks for a destination and exports aggregate-only data.
func (a *App) ExportAggregateCSV(month string) ExportResultDTO {
	path, result, ok := a.chooseExportPath("aggregate-"+month+".csv", "Export monthly aggregate CSV")
	if !ok {
		return result
	}
	return a.exportAggregateCSVTo(month, path)
}

// ExportAggregateCSVTo writes the aggregate monthly table. It cannot contain
// patient-level information.
func (a *App) exportAggregateCSVTo(month, path string) ExportResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return exportFailed(a.userError(err).Error())
	}
	ctx, cancel := a.opContext()
	defer cancel()
	body, err := a.services.CSV.AggregateCSV(ctx, month)
	if err != nil {
		return exportFailed(a.actionError(err).Message)
	}
	if err := writeFileAtomic(path, body); err != nil {
		a.logf("write aggregate export", "error", err)
		return exportFailed("The aggregate export could not be written.")
	}
	a.recordExport(ctx, "aggregate", month, len(body), path, false)
	return ExportResultDTO{OK: true, Message: "Aggregate CSV exported.", Path: path, Bytes: int64(len(body))}
}

// ExportPatientUsageCSV asks for a destination and exports local, confidential
// patient-level usage. Notes are omitted by the service.
func (a *App) ExportPatientUsageCSV(month string) ExportResultDTO {
	path, result, ok := a.chooseExportPath("patient-usage-"+month+".csv", "Export confidential patient-level usage")
	if !ok {
		return result
	}
	return a.exportPatientUsageCSVTo(month, path)
}

// ExportPatientUsageCSVTo writes the confidential local patient-level export.
// The caller must have displayed the privacy warning and obtained confirmation.
func (a *App) exportPatientUsageCSVTo(month, path string) ExportResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return exportFailed(a.userError(err).Error())
	}
	ctx, cancel := a.opContext()
	defer cancel()
	body, err := a.services.CSV.PatientUsageCSV(ctx, month)
	if err != nil {
		return exportFailed(a.actionError(err).Message)
	}
	if err := writeFileAtomic(path, body); err != nil {
		a.logf("write patient export", "error", err)
		return exportFailed("The patient-level export could not be written.")
	}
	a.recordExport(ctx, "patient_usage", month, len(body), path, true)
	return ExportResultDTO{
		OK: true, Message: "Confidential patient-level CSV exported. Keep this file local.", Path: path, Bytes: int64(len(body)),
	}
}

func (a *App) chooseExportPath(defaultName, title string) (string, ExportResultDTO, bool) {
	a.mu.Lock()
	ctx := a.ctx
	ready := a.readyLocked() == nil
	a.mu.Unlock()
	if !ready {
		return "", exportFailed("The local database is not available."), false
	}
	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		DefaultFilename:      defaultName,
		Title:                title,
		CanCreateDirectories: true,
		Filters:              []runtime.FileFilter{{DisplayName: "CSV file (*.csv)", Pattern: "*.csv"}},
	})
	if err != nil {
		return "", exportFailed("The save dialog could not be opened."), false
	}
	if strings.TrimSpace(path) == "" {
		return "", ExportResultDTO{Code: "cancelled", Message: "Export cancelled."}, false
	}
	return path, ExportResultDTO{}, true
}

func (a *App) recordExport(ctx context.Context, kind, month string, bytes int, path string, patientData bool) {
	if a.store == nil {
		return
	}
	err := a.store.WithTx(ctx, func(q generated.Querier) error {
		return audit.Record(ctx, q, a.now(), audit.Event{
			Type: audit.EventExportCompleted, EntityType: "export",
			EntityID: filepath.Base(path), Month: month,
			Metadata: map[string]any{
				"kind": kind, "bytes": bytes, "contains_patient_data": patientData,
			},
		})
	})
	if err != nil {
		a.logf("record export audit event", "error", err)
	}
}

func exportFailed(message string) ExportResultDTO {
	return ExportResultDTO{Code: "export_error", Message: message}
}

func writeFileAtomic(path string, body []byte) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("export path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".vaccineshit-export-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
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
	return os.Rename(tempPath, path)
}
