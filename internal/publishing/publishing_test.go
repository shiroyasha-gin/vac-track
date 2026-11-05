package publishing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validSnapshot(t *testing.T) Snapshot {
	t.Helper()
	s, err := Build("2026-11", time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), []Vaccine{
		{Name: "Measles", Unit: "dose", OpeningBalance: 8, StockReceived: 12, TotalUsage: 5, Remaining: 15},
		{Name: "BCG", Unit: "dose", OpeningBalance: 2, StockReceived: 0, TotalUsage: 1, Remaining: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBuildSnapshotRecomputesTotalsAndHasNoPatientShape(t *testing.T) {
	t.Parallel()
	s := validSnapshot(t)
	if s.Totals.VaccineCount != 2 || s.Totals.OpeningBalance != 10 || s.Totals.StockReceived != 12 || s.Totals.TotalUsage != 6 || s.Totals.Remaining != 16 {
		t.Fatalf("totals = %+v", s.Totals)
	}
	body, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{"patient", "Patient", "Synthetic Patient", "note", "administeredOn"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("public snapshot JSON contains forbidden patient-level field %q: %s", forbidden, text)
		}
	}
	if err := ValidateSnapshot(s); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSnapshotRejectsInconsistentPayloads(t *testing.T) {
	t.Parallel()
	s := validSnapshot(t)
	s.Totals.Remaining++
	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("inconsistent totals accepted")
	}

	s = validSnapshot(t)
	s.Aggregation = "daily"
	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("daily aggregation accepted")
	}

	s = validSnapshot(t)
	s.Vaccines[0].Remaining = -1
	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("negative aggregate accepted")
	}

	s = validSnapshot(t)
	s.Month = "2026-13"
	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("invalid month accepted")
	}
}

func TestValidateBaseURL(t *testing.T) {
	t.Parallel()
	if got, err := ValidateBaseURL("https://stats.example.org/"); err != nil || got != "https://stats.example.org" {
		t.Fatalf("https base = %q, %v", got, err)
	}
	if _, err := ValidateBaseURL("http://stats.example.org"); err == nil {
		t.Fatal("plain HTTP remote endpoint accepted")
	}
	if got, err := ValidateBaseURL("http://127.0.0.1:8080"); err != nil || got != "http://127.0.0.1:8080" {
		t.Fatalf("loopback endpoint = %q, %v", got, err)
	}
	if _, err := ValidateBaseURL("https://user:secret@stats.example.org"); err == nil {
		t.Fatal("userinfo endpoint accepted")
	}
}

func TestPublishClientSendsTokenAndIdempotencyKey(t *testing.T) {
	t.Parallel()
	s := validSnapshot(t)
	var gotAuth, gotKey string
	var gotBody Snapshot
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("Idempotency-Key")
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":3,"publishedAt":"2026-11-15T12:00:00Z","duplicate":false,"etag":"abc"}`))
	}))
	defer server.Close()

	client := NewClient()
	result, err := client.Publish(context.Background(), server.URL, "secret-token", s)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 3 || result.Duplicate {
		t.Fatalf("response = %+v", result)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatal("missing bearer token")
	}
	if gotKey != s.SnapshotID {
		t.Fatalf("idempotency key = %q, want %q", gotKey, s.SnapshotID)
	}
	if gotBody.Month != "2026-11" || len(gotBody.Vaccines) != 2 {
		t.Fatalf("posted snapshot = %+v", gotBody)
	}
}

func TestPublishClientErrorsDoNotLeakToken(t *testing.T) {
	t.Parallel()
	s := validSnapshot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid publisher token"}`))
	}))
	defer server.Close()
	_, err := NewClient().Publish(context.Background(), server.URL, "super-secret-token", s)
	if err == nil {
		t.Fatal("expected publish failure")
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatal("error leaked the token")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnpublishClient(t *testing.T) {
	t.Parallel()
	var method, auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, auth = r.Method, r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err := NewClient().Unpublish(context.Background(), server.URL, "token"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || auth != "Bearer token" {
		t.Fatalf("method=%s auth=%s", method, auth)
	}
}

type fakeKeyring struct {
	secret  string
	fail    bool
	deleted bool
}

func (f *fakeKeyring) Set(service, user, password string) error {
	if f.fail {
		return errors.New("keyring unavailable")
	}
	f.secret = password
	return nil
}
func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.fail {
		return "", errors.New("keyring unavailable")
	}
	return f.secret, nil
}
func (f *fakeKeyring) Delete(service, user string) error {
	if f.fail {
		return errors.New("keyring unavailable")
	}
	f.deleted = true
	f.secret = ""
	return nil
}

func TestCredentialsPreferKeyring(t *testing.T) {
	t.Parallel()
	fallbackPath := filepath.Join(t.TempDir(), "secrets", "token")
	if err := os.MkdirAll(filepath.Dir(fallbackPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallbackPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	backend := &fakeKeyring{}
	creds := NewCredentialsForTest(backend, &FileCredentialStore{Path: fallbackPath})
	stored, err := creds.Save(PublisherAccount, "from-keyring")
	if err != nil {
		t.Fatal(err)
	}
	if stored != StorageOSKeyring {
		t.Fatalf("storage = %s, want keyring", stored)
	}
	secret, source, err := creds.Load(PublisherAccount)
	if err != nil || secret != "from-keyring" || source != StorageOSKeyring {
		t.Fatalf("load = %q %s %v", secret, source, err)
	}
	if _, err := os.Stat(fallbackPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("fallback token file was not removed after keyring save")
	}
}

func TestCredentialsFallbackFileIsPrivate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "secrets", "token")
	creds := NewCredentialsForTest(&fakeKeyring{fail: true}, &FileCredentialStore{Path: path})
	stored, err := creds.Save(PublisherAccount, "  local-secret  ")
	if err != nil {
		t.Fatal(err)
	}
	if stored != StorageLocalFile {
		t.Fatalf("storage = %s, want local file", stored)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 600", stat.Mode().Perm())
	}
	secret, source, err := creds.Load(PublisherAccount)
	if err != nil || secret != "local-secret" || source != StorageLocalFile {
		t.Fatalf("load = %q %s %v", secret, source, err)
	}
	if _, _, err := creds.Load("other"); !errors.Is(err, ErrCredentialsNotFound) && err != nil {
		// The fallback is intentionally account-agnostic; a missing file is the
		// only not-found case.
		t.Fatalf("load missing = %v", err)
	}
}
