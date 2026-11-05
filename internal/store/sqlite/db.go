// Package sqlite provides the local desktop database: connection setup,
// migrations, and write transactions.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"vaccineshit/internal/store"
	"vaccineshit/internal/store/generated"
)

// Config configures the local SQLite database.
type Config struct {
	Path        string
	BackupDir   string
	BusyTimeout time.Duration
}

// DB is the concrete store.Store implementation.
type DB struct {
	db        *sql.DB
	queries   *generated.Queries
	path      string
	backupDir string
	writeMu   sync.Mutex
	closeOnce sync.Once
	closeErr  error
}

var _ store.Store = (*DB)(nil)

// Open creates or opens the database, enables required pragmas, and applies
// pending migrations. The caller must call Close.
func Open(ctx context.Context, cfg Config) (*DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	abs, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	if cfg.BackupDir == "" {
		cfg.BackupDir = filepath.Join(filepath.Dir(abs), "backups")
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o700); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}
	if cfg.BusyTimeout <= 0 {
		cfg.BusyTimeout = 5 * time.Second
	}

	dsn := dataSourceName(abs, cfg.BusyTimeout)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// The driver applies pragmas per connection via the DSN. A small pool is
	// enough for a single-operator desktop app; writes are serialized below.
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(0)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := os.Chmod(abs, 0o600); err != nil && os.PathSeparator != '\\' {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("protect database file: %w", err)
	}

	db := &DB{
		db:        sqlDB,
		queries:   generated.New(sqlDB),
		path:      abs,
		backupDir: cfg.BackupDir,
	}
	if err := migrate(ctx, sqlDB, cfg.BackupDir); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func dataSourceName(path string, busyTimeout time.Duration) string {
	u := &url.URL{Scheme: "file", Path: path}
	q := u.Query()
	// modernc.org/sqlite supports repeatable _pragma values. Foreign keys are
	// enabled on every connection by the driver, not by a one-off statement.
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeout.Milliseconds()))
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "temp_store(MEMORY)")
	q.Set("_txlock", "immediate")
	u.RawQuery = q.Encode()
	return u.String()
}

// Queries implements store.Store.
func (d *DB) Queries() generated.Querier {
	return d.queries
}

// DB implements store.Store.
func (d *DB) DB() *sql.DB {
	return d.db
}

// Path implements store.Store.
func (d *DB) Path() string {
	return d.path
}

// WithTx implements store.Store. Writes are serialized inside this process;
// the transaction is BEGIN IMMEDIATE (driver _txlock) and balance checks are
// repeated inside it.
func (d *DB) WithTx(ctx context.Context, fn func(q generated.Querier) error) (err error) {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin database transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(generated.New(tx)); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit database transaction: %w", err)
	}
	return nil
}

// Checkpoint flushes the write-ahead log so external tooling sees a complete
// database file.
func (d *DB) Checkpoint(ctx context.Context) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_, err := d.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("checkpoint database: %w", err)
	}
	return nil
}

// Close implements store.Store.
func (d *DB) Close() error {
	d.closeOnce.Do(func() {
		d.writeMu.Lock()
		defer d.writeMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = d.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
		d.closeErr = d.db.Close()
	})
	return d.closeErr
}
