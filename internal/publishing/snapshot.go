// Package publishing defines the allow-listed aggregate snapshot boundary and
// the HTTPS publisher client. No type in this package can carry patient names,
// patient identifiers, usage notes or individual usage events.
package publishing

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"vaccineshit/internal/domain"
)

const SchemaVersion = 1

var snapshotIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

// Totals are monthly aggregate totals across all included vaccines.
type Totals struct {
	VaccineCount   int   `json:"vaccineCount"`
	OpeningBalance int64 `json:"openingBalance"`
	StockReceived  int64 `json:"stockReceived"`
	TotalUsage     int64 `json:"totalUsage"`
	Remaining      int64 `json:"remaining"`
}

// Vaccine is the only per-vaccine public shape. It contains aggregate doses
// only, never patient-level records.
type Vaccine struct {
	Name           string `json:"name"`
	Code           string `json:"code,omitempty"`
	Unit           string `json:"unit"`
	OpeningBalance int64  `json:"openingBalance"`
	StockReceived  int64  `json:"stockReceived"`
	TotalUsage     int64  `json:"totalUsage"`
	Remaining      int64  `json:"remaining"`
}

// Snapshot is the reviewed, monthly aggregate publication payload.
type Snapshot struct {
	SchemaVersion int       `json:"schemaVersion"`
	SnapshotID    string    `json:"snapshotId"`
	Month         string    `json:"month"`
	Aggregation   string    `json:"aggregation"`
	GeneratedAt   string    `json:"generatedAt"`
	Totals        Totals    `json:"totals"`
	Vaccines      []Vaccine `json:"vaccines"`
}

// NewSnapshotID returns an unpredictable identifier used for idempotent
// publishing and local publication metadata.
func NewSnapshotID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate snapshot id: %w", err)
	}
	return "snap_" + hex.EncodeToString(buf), nil
}

// Build aggregates only explicitly allowed monthly fields. It recomputes
// totals from the rows so a caller cannot accidentally publish inconsistent
// summaries.
func Build(month string, generatedAt time.Time, vaccines []Vaccine) (Snapshot, error) {
	id, err := NewSnapshotID()
	if err != nil {
		return Snapshot{}, err
	}
	rows := make([]Vaccine, len(vaccines))
	copy(rows, vaccines)
	totals := Totals{VaccineCount: len(rows)}
	for _, row := range rows {
		totals.OpeningBalance += row.OpeningBalance
		totals.StockReceived += row.StockReceived
		totals.TotalUsage += row.TotalUsage
		totals.Remaining += row.Remaining
	}
	snapshot := Snapshot{
		SchemaVersion: SchemaVersion,
		SnapshotID:    id,
		Month:         month,
		Aggregation:   "monthly",
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		Totals:        totals,
		Vaccines:      rows,
	}
	if err := ValidateSnapshot(snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// ValidateSnapshot enforces the public payload shape, non-negative whole-dose
// quantities, and internally consistent totals.
func ValidateSnapshot(s Snapshot) error {
	if s.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported snapshot schema version %d", s.SchemaVersion)
	}
	if !snapshotIDPattern.MatchString(s.SnapshotID) {
		return fmt.Errorf("invalid snapshot id")
	}
	if _, err := domain.ParseMonth(s.Month); err != nil {
		return fmt.Errorf("invalid snapshot month")
	}
	if s.Aggregation != "monthly" {
		return fmt.Errorf("public snapshots must use monthly aggregation")
	}
	if _, err := time.Parse(time.RFC3339, s.GeneratedAt); err != nil {
		return fmt.Errorf("invalid generatedAt timestamp")
	}
	if len(s.Vaccines) > 1000 {
		return fmt.Errorf("too many vaccine rows")
	}
	if s.Totals.VaccineCount != len(s.Vaccines) {
		return fmt.Errorf("vaccineCount does not match vaccines")
	}
	var totals Totals
	totals.VaccineCount = len(s.Vaccines)
	for i, row := range s.Vaccines {
		if strings.TrimSpace(row.Name) == "" {
			return fmt.Errorf("vaccine row %d has no name", i)
		}
		if strings.TrimSpace(row.Unit) == "" {
			return fmt.Errorf("vaccine row %d has no unit", i)
		}
		if row.OpeningBalance < 0 || row.StockReceived < 0 || row.TotalUsage < 0 || row.Remaining < 0 {
			return fmt.Errorf("vaccine row %d contains a negative aggregate", i)
		}
		expectedRemaining := row.OpeningBalance + row.StockReceived - row.TotalUsage
		if row.Remaining != expectedRemaining {
			return fmt.Errorf("vaccine row %d balance is inconsistent", i)
		}
		totals.OpeningBalance += row.OpeningBalance
		totals.StockReceived += row.StockReceived
		totals.TotalUsage += row.TotalUsage
		totals.Remaining += row.Remaining
	}
	if totals != s.Totals {
		return fmt.Errorf("snapshot totals do not match vaccine rows")
	}
	return nil
}

// Hash returns a stable SHA-256 of the canonical JSON payload.
func Hash(s Snapshot) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// ContentHash hashes the reviewed aggregate content without the per-preview
// snapshot id or generation timestamp. The desktop uses it to detect whether
// source data changed between preview and publish.
func ContentHash(s Snapshot) (string, error) {
	canonical := struct {
		SchemaVersion int       `json:"schemaVersion"`
		Month         string    `json:"month"`
		Aggregation   string    `json:"aggregation"`
		Totals        Totals    `json:"totals"`
		Vaccines      []Vaccine `json:"vaccines"`
	}{
		SchemaVersion: s.SchemaVersion, Month: s.Month, Aggregation: s.Aggregation,
		Totals: s.Totals, Vaccines: s.Vaccines,
	}
	body, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// Marshal returns compact JSON.
func Marshal(s Snapshot) ([]byte, error) {
	return json.Marshal(s)
}
