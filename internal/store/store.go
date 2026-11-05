// Package store defines the narrow data-access surface used by services.
// sqlc-generated queries implement the typed query layer; SQLite provides the
// transaction entry point.
package store

import (
	"context"
	"database/sql"

	"vaccineshit/internal/store/generated"
)

// Store is the service-facing data access interface. Services never open SQL
// connections themselves.
type Store interface {
	// Queries returns the sqlc query set bound to the read pool.
	Queries() generated.Querier
	// WithTx runs fn atomically in a write transaction. Audit events that
	// describe a change must be written through the provided querier so they
	// commit or roll back with the change itself.
	WithTx(ctx context.Context, fn func(q generated.Querier) error) error
	// DB returns the underlying pool for maintenance operations such as
	// consistent SQLite backups. It must not be used by feature services.
	DB() *sql.DB
	// Path returns the absolute database file path.
	Path() string
	// Checkpoint flushes the WAL before backup, shutdown or restore.
	Checkpoint(ctx context.Context) error
	// Close flushes and closes the database.
	Close() error
}
