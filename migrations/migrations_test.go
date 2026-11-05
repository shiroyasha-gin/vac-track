package migrations

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestGeneratedFilesMatchSQL prevents shipping a binary whose compiled
// migrations differ from the canonical .sql files. Run `go generate
// ./migrations` after adding or editing a migration.
func TestGeneratedFilesMatchSQL(t *testing.T) {
	names, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)

	if len(names) != len(Files) {
		t.Fatalf("generated migrations count %d does not match %d .sql files; run go generate ./migrations", len(Files), len(names))
	}
	for _, name := range names {
		want, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := Files[name]
		if !ok {
			t.Fatalf("generated migrations are missing %s; run go generate ./migrations", name)
		}
		if got != string(want) {
			t.Fatalf("generated migration %s is stale; run go generate ./migrations", name)
		}
	}
}
