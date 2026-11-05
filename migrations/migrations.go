// Package migrations embeds the ordered SQLite schema migrations so the
// desktop application can apply them without relying on files next to the
// executable.
package migrations

//go:generate go run ./internal/genmigrations
