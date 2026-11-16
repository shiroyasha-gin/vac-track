package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vaccineshit/internal/atomicfile"
	"vaccineshit/internal/publishing"
)

// ErrNoPublication means nothing is currently published.
var ErrNoPublication = errors.New("no snapshot is currently published")

// ErrSnapshotConflict means a snapshot id was reused with different content.
var ErrSnapshotConflict = errors.New("snapshot id was already published with different content")

// Publication is a stored, reviewed aggregate snapshot plus publication
// metadata. It never includes patient-level data.
type Publication struct {
	Version     int64               `json:"version"`
	PublishedAt string              `json:"publishedAt"`
	ETag        string              `json:"etag"`
	Snapshot    publishing.Snapshot `json:"snapshot"`
}

// Store persists the currently published aggregate snapshot and remembers
// earlier snapshots so client retries are idempotent.
type Store interface {
	// Publish makes snapshot current. duplicate reports that this exact
	// snapshot id and content were already published.
	Publish(ctx context.Context, snapshot publishing.Snapshot) (pub Publication, duplicate bool, err error)
	// Current returns the currently published snapshot.
	Current(ctx context.Context) (Publication, error)
	// Delete removes the current publication. Historical snapshot ids remain
	// known so retries cannot silently change previous content.
	Delete(ctx context.Context) error
}

// FileStore stores publications as JSON files under one private directory. The
// single current pointer is replaced atomically, so a crash cannot leave a
// half-written public snapshot.
type FileStore struct {
	dir string
	mu  sync.Mutex
	now func() time.Time
}

// NewFileStore prepares dir with private permissions.
func NewFileStore(dir string) (*FileStore, error) {
	if dir == "" {
		return nil, errors.New("publication store directory is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve publication store directory: %w", err)
	}
	for _, sub := range []string{abs, filepath.Join(abs, "published")} {
		if err := os.MkdirAll(sub, 0o700); err != nil {
			return nil, fmt.Errorf("create publication store: %w", err)
		}
	}
	return &FileStore{dir: abs, now: func() time.Time { return time.Now().UTC() }}, nil
}

// SetClock overrides the publication clock. It exists for deterministic tests.
func (s *FileStore) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

func (s *FileStore) currentPath() string {
	return filepath.Join(s.dir, "current.json")
}

func (s *FileStore) snapshotPath(id string) string {
	return filepath.Join(s.dir, "published", id+".json")
}

// Publish validates and stores one aggregate snapshot. Snapshot ids must be
// unique; reusing an id with identical bytes is an idempotent retry.
func (s *FileStore) Publish(ctx context.Context, snapshot publishing.Snapshot) (Publication, bool, error) {
	if err := ctx.Err(); err != nil {
		return Publication{}, false, err
	}
	if err := publishing.ValidateSnapshot(snapshot); err != nil {
		return Publication{}, false, fmt.Errorf("invalid snapshot: %w", err)
	}
	etag, err := publishing.Hash(snapshot)
	if err != nil {
		return Publication{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.readPublication(s.snapshotPath(snapshot.SnapshotID))
	if err == nil {
		if existing.ETag != etag {
			return Publication{}, false, ErrSnapshotConflict
		}
		// Re-point current at the known snapshot so a retry after an unpublish
		// or crash restores exactly the reviewed content.
		if err := s.writePublication(s.currentPath(), existing); err != nil {
			return Publication{}, false, err
		}
		return existing, true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Publication{}, false, err
	}

	current, err := s.readPublication(s.currentPath())
	version := int64(0)
	if err == nil {
		version = current.Version
	} else if !errors.Is(err, os.ErrNotExist) {
		return Publication{}, false, err
	}

	pub := Publication{
		Version:     version + 1,
		PublishedAt: s.now().UTC().Format(time.RFC3339),
		ETag:        etag,
		Snapshot:    snapshot,
	}
	if err := s.writePublication(s.snapshotPath(snapshot.SnapshotID), pub); err != nil {
		return Publication{}, false, err
	}
	if err := s.writePublication(s.currentPath(), pub); err != nil {
		return Publication{}, false, err
	}
	return pub, false, nil
}

// Current reads the current pointer. A missing pointer means 404, not an empty
// snapshot.
func (s *FileStore) Current(ctx context.Context) (Publication, error) {
	if err := ctx.Err(); err != nil {
		return Publication{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pub, err := s.readPublication(s.currentPath())
	if errors.Is(err, os.ErrNotExist) {
		return Publication{}, ErrNoPublication
	}
	return pub, err
}

// Delete removes the current pointer. Published history is retained for
// idempotency and diagnostics.
func (s *FileStore) Delete(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.currentPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current publication: %w", err)
	}
	return nil
}

func (s *FileStore) readPublication(path string) (Publication, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Publication{}, err
	}
	var pub Publication
	if err := json.Unmarshal(body, &pub); err != nil {
		return Publication{}, fmt.Errorf("read publication record: %w", err)
	}
	if err := publishing.ValidateSnapshot(pub.Snapshot); err != nil {
		return Publication{}, fmt.Errorf("stored publication is invalid: %w", err)
	}
	return pub, nil
}

func (s *FileStore) writePublication(path string, pub Publication) error {
	body, err := json.Marshal(pub)
	if err != nil {
		return fmt.Errorf("encode publication: %w", err)
	}
	if err := atomicfile.Write(path, body, 0o600); err != nil {
		return fmt.Errorf("replace publication file: %w", err)
	}
	return nil
}
