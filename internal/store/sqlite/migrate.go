package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vaccineshit/migrations"
)

const migrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    checksum   TEXT NOT NULL,
    applied_at TEXT NOT NULL
);`

type migrationFile struct {
	version  string
	path     string
	body     string
	checksum string
}

// migrate applies checked-in migrations in filename order. When an existing
// database needs an upgrade, a consistent SQLite backup is written first.
func migrate(ctx context.Context, db *sql.DB, backupDir string) error {
	files, err := loadMigrations()
	if err != nil {
		return err
	}

	fresh, err := databaseIsFresh(ctx, db)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, migrationsTable); err != nil {
		return fmt.Errorf("initialize migration tracking: %w", err)
	}

	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	known := make(map[string]string, len(files))
	var pending []migrationFile
	for _, f := range files {
		known[f.version] = f.checksum
		if checksum, ok := applied[f.version]; ok {
			if checksum != f.checksum {
				return fmt.Errorf("migration %s was changed after it was applied; refusing to continue (create a new migration instead)", f.path)
			}
			continue
		}
		pending = append(pending, f)
	}
	for version := range applied {
		if _, ok := known[version]; !ok {
			return fmt.Errorf("database contains migration %s that this application does not know; it may have been created by a newer version", version)
		}
	}

	if len(pending) > 0 && !fresh {
		backupPath := filepath.Join(backupDir, "pre-migration-"+time.Now().UTC().Format("20060102T150405Z")+".db")
		if err := backupWithVacuum(ctx, db, backupPath); err != nil {
			return fmt.Errorf("back up database before migrations: %w", err)
		}
	}

	for _, f := range pending {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", f.path, err)
		}
		if _, err := tx.ExecContext(ctx, f.body); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", f.path, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, checksum, applied_at) VALUES (?, ?, ?)`,
			f.version, f.checksum, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", f.path, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", f.path, err)
		}
	}
	return nil
}

func loadMigrations() ([]migrationFile, error) {
	if len(migrations.Files) == 0 {
		return nil, errors.New("no embedded migrations found")
	}
	names := make([]string, 0, len(migrations.Files))
	for name := range migrations.Files {
		if strings.HasSuffix(name, ".sql") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	files := make([]migrationFile, 0, len(names))
	for _, name := range names {
		body := []byte(migrations.Files[name])
		sum := sha256.Sum256(body)
		files = append(files, migrationFile{
			version:  strings.TrimSuffix(name, ".sql"),
			path:     name,
			body:     string(body),
			checksum: hex.EncodeToString(sum[:]),
		})
	}
	if len(files) == 0 {
		return nil, errors.New("no embedded migrations found")
	}
	return files, nil
}

func loadAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	defer rows.Close()
	applied := map[string]string{}
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("read applied migrations: %w", err)
		}
		applied[version] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	return applied, nil
}

func databaseIsFresh(ctx context.Context, db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migrations'`,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("inspect database schema: %w", err)
	}
	return count == 0, nil
}

func backupWithVacuum(ctx context.Context, db *sql.DB, dest string) error {
	// VACUUM INTO is a consistent, concurrency-safe snapshot of the live
	// database. Never copy a live SQLite file while it is open.
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, dest); err != nil {
		return fmt.Errorf("vacuum into %s: %w", dest, err)
	}
	return nil
}
