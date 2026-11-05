package publicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vaccineshit/internal/publishing"
)

const testToken = "test-publisher-token-0123456789"

func validSnapshot(t *testing.T, id string) publishing.Snapshot {
	t.Helper()
	snapshot, err := publishing.Build("2026-10", time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC), []publishing.Vaccine{
		{Name: "Influenza", Code: "FLU", Unit: "dose", OpeningBalance: 50, StockReceived: 20, TotalUsage: 30, Remaining: 40},
		{Name: "Tetanus", Code: "TET", Unit: "dose", OpeningBalance: 10, StockReceived: 0, TotalUsage: 4, Remaining: 6},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		snapshot.SnapshotID = id
	}
	return snapshot
}

func newTestServer(t *testing.T, dir, siteDir string, mut ...func(*Config)) (*httptest.Server, *FileStore) {
	t.Helper()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		AdminToken:              testToken,
		Store:                   store,
		PublicDir:               siteDir,
		Logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
		PublicRequestsPerMinute: 1000,
		AdminRequestsPerMinute:  1000,
	}
	for _, fn := range mut {
		fn(&cfg)
	}
	handler, err := NewHandler(cfg)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, store
}

func do(t *testing.T, method, url string, body []byte, token string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func publish(t *testing.T, server *httptest.Server, snapshot publishing.Snapshot, token string) (*http.Response, publishing.PublishResponse) {
	t.Helper()
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	resp := do(t, http.MethodPut, server.URL+"/api/admin/snapshot", body, token)
	defer resp.Body.Close()
	var out publishing.PublishResponse
	if resp.StatusCode/100 == 2 {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
	}
	return resp, out
}

func TestPublishRequiresBearerToken(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	for _, token := range []string{"", "wrong-token-value"} {
		resp, _ := publish(t, server, validSnapshot(t, ""), token)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: status = %d, want 401", token, resp.StatusCode)
		}
		if resp.Header.Get("WWW-Authenticate") == "" {
			t.Fatal("missing WWW-Authenticate header")
		}
	}
}

func TestPublishAndReadPublicSnapshot(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	snapshot := validSnapshot(t, "")
	resp, body := publish(t, server, snapshot, testToken)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("publish status = %d, want 201", resp.StatusCode)
	}
	if body.Duplicate || body.Version != 1 || body.ETag == "" {
		t.Fatalf("publish response = %+v", body)
	}

	public := do(t, http.MethodGet, server.URL+"/api/public/snapshot", nil, "")
	defer public.Body.Close()
	if public.StatusCode != http.StatusOK {
		t.Fatalf("public status = %d, want 200", public.StatusCode)
	}
	if public.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("public snapshot is missing CORS header")
	}
	quoted := `"` + body.ETag + `"`
	if public.Header.Get("ETag") != quoted {
		t.Fatalf("ETag = %q, want %q", public.Header.Get("ETag"), quoted)
	}
	if got := public.Header.Get("X-Publication-Version"); got != "1" {
		t.Fatalf("publication version header = %q, want 1", got)
	}
	if got := public.Header.Get("X-Published-At"); got == "" {
		t.Fatal("publication timestamp header is empty")
	}
	var published publishing.Snapshot
	if err := json.NewDecoder(public.Body).Decode(&published); err != nil {
		t.Fatal(err)
	}
	if published.SnapshotID != snapshot.SnapshotID {
		t.Fatalf("public snapshot id = %q, want %q", published.SnapshotID, snapshot.SnapshotID)
	}
	if got := published.Totals.TotalUsage; got != 34 {
		t.Fatalf("total usage = %d, want 34", got)
	}

	conditional := do(t, http.MethodGet, server.URL+"/api/public/snapshot", nil, "")
	conditional.Body.Close()
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/public/snapshot", nil)
	req.Header.Set("If-None-Match", quoted)
	notModified, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	notModified.Body.Close()
	if notModified.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want 304", notModified.StatusCode)
	}
}

func TestPublishIsIdempotentAndRejectsConflict(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	snapshot := validSnapshot(t, "")
	first, firstBody := publish(t, server, snapshot, testToken)
	if first.StatusCode != http.StatusCreated || firstBody.Version != 1 {
		t.Fatalf("first publish = %d %+v", first.StatusCode, firstBody)
	}
	retry, retryBody := publish(t, server, snapshot, testToken)
	if retry.StatusCode != http.StatusOK || !retryBody.Duplicate {
		t.Fatalf("retry = %d %+v, want 200 duplicate", retry.StatusCode, retryBody)
	}
	if retryBody.Version != firstBody.Version || retryBody.PublishedAt != firstBody.PublishedAt {
		t.Fatalf("retry changed publication metadata: %+v vs %+v", retryBody, firstBody)
	}

	conflict := snapshot
	conflict.Vaccines = append([]publishing.Vaccine(nil), snapshot.Vaccines...)
	conflict.Vaccines[0].Name = "Different vaccine"
	conflict.Totals = snapshot.Totals
	conflictResponse, _ := publish(t, server, conflict, testToken)
	if conflictResponse.StatusCode != http.StatusConflict {
		t.Fatalf("conflict status = %d, want 409", conflictResponse.StatusCode)
	}
}

func TestPublishRejectsUnknownFieldsAndPatientData(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	snapshot := validSnapshot(t, "")
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	payload["patients"] = []map[string]string{{"name": "Ada Lovelace"}}
	tainted, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp := do(t, http.MethodPut, server.URL+"/api/admin/snapshot", tainted, testToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("patient payload status = %d, want 400", resp.StatusCode)
	}
}

func TestPublishRejectsNonMonthlyAggregation(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	snapshot := validSnapshot(t, "")
	snapshot.Aggregation = "daily"
	resp, _ := publish(t, server, snapshot, testToken)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestPublishRejectsOversizedBody(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	snapshot := validSnapshot(t, "")
	snapshot.Vaccines[0].Name = strings.Repeat("a", maxSnapshotBytes+1024)
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	resp := do(t, http.MethodPut, server.URL+"/api/admin/snapshot", body, testToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
}

func TestUnpublishRemovesPublicSnapshot(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	if resp, _ := publish(t, server, validSnapshot(t, ""), testToken); resp.StatusCode != http.StatusCreated {
		t.Fatalf("publish status = %d", resp.StatusCode)
	}
	resp := do(t, http.MethodDelete, server.URL+"/api/admin/snapshot", nil, testToken)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unpublish status = %d, want 204", resp.StatusCode)
	}
	public := do(t, http.MethodGet, server.URL+"/api/public/snapshot", nil, "")
	defer public.Body.Close()
	if public.StatusCode != http.StatusNotFound {
		t.Fatalf("public status = %d, want 404", public.StatusCode)
	}
	// Unpublishing is idempotent.
	again := do(t, http.MethodDelete, server.URL+"/api/admin/snapshot", nil, testToken)
	again.Body.Close()
	if again.StatusCode != http.StatusNoContent {
		t.Fatalf("second unpublish status = %d, want 204", again.StatusCode)
	}
}

func TestPublicationSurvivesRestart(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	first, _ := newTestServer(t, dir, "")
	snapshot := validSnapshot(t, "")
	if resp, _ := publish(t, first, snapshot, testToken); resp.StatusCode != http.StatusCreated {
		t.Fatalf("publish status = %d", resp.StatusCode)
	}
	first.Close()

	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(Config{AdminToken: testToken, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	second := httptest.NewServer(handler)
	defer second.Close()
	resp := do(t, http.MethodGet, second.URL+"/api/public/snapshot", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status after restart = %d, want 200", resp.StatusCode)
	}
	var published publishing.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&published); err != nil {
		t.Fatal(err)
	}
	if published.SnapshotID != snapshot.SnapshotID {
		t.Fatalf("snapshot id after restart = %q", published.SnapshotID)
	}
}

func TestPublicRateLimit(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "", func(c *Config) {
		c.PublicRequestsPerMinute = 2
		c.AdminRequestsPerMinute = 1000
	})
	for i := 0; i < 2; i++ {
		resp := do(t, http.MethodGet, server.URL+"/api/public/snapshot", nil, "")
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("request %d was rate limited too early", i+1)
		}
	}
	resp := do(t, http.MethodGet, server.URL+"/api/public/snapshot", nil, "")
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third request status = %d, want 429", resp.StatusCode)
	}
}

func TestStaticPublicSite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><title>Public</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	server, _ := newTestServer(t, t.TempDir(), dir)
	resp := do(t, http.MethodGet, server.URL+"/", nil, "")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Public") {
		t.Fatalf("static status = %d body = %q", resp.StatusCode, body)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("content type = %q", resp.Header.Get("Content-Type"))
	}
	missing := do(t, http.MethodGet, server.URL+"/missing.txt", nil, "")
	missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", missing.StatusCode)
	}
}

func TestUnsupportedMethodAndHealth(t *testing.T) {
	t.Parallel()
	server, _ := newTestServer(t, t.TempDir(), "")
	resp := do(t, http.MethodPost, server.URL+"/api/admin/snapshot", []byte("{}"), testToken)
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", resp.StatusCode)
	}
	health := do(t, http.MethodGet, server.URL+"/healthz", nil, "")
	health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.StatusCode)
	}
}

func TestFileStoreRejectsIdCollision(t *testing.T) {
	t.Parallel()
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := validSnapshot(t, "")
	if _, _, err := store.Publish(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	changed := snapshot
	changed.Vaccines[0].TotalUsage++
	changed.Vaccines[0].Remaining--
	changed.Totals.TotalUsage++
	changed.Totals.Remaining--
	if _, _, err := store.Publish(context.Background(), changed); err != ErrSnapshotConflict {
		t.Fatalf("collision error = %v, want ErrSnapshotConflict", err)
	}
}
