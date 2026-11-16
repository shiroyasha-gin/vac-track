// Package publicapi implements the separate, internet-facing HTTPS service that
// receives reviewed monthly aggregate snapshots from the desktop app and serves
// them to the public site. It never receives or stores patient-level data.
package publicapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"vaccineshit/internal/publishing"
)

const (
	maxSnapshotBytes = 256 << 10
	authRealm        = "vaccineshit-publisher"
)

// Config configures the public HTTP handler.
type Config struct {
	// AdminToken authenticates snapshot uploads. It is compared in constant
	// time and must never be logged.
	AdminToken string
	// Store persists public snapshots.
	Store Store
	// PublicDir holds the static public site. Empty disables static serving.
	PublicDir string
	// Logger receives sanitized operational logs.
	Logger *slog.Logger
	// PublicRequestsPerMinute and AdminRequestsPerMinute configure per-client-IP
	// token buckets. Zero uses safe defaults.
	PublicRequestsPerMinute int
	AdminRequestsPerMinute  int
	// TrustProxy makes the rate limiter use the first X-Forwarded-For address.
	// Enable only when a trusted reverse proxy overwrites that header.
	TrustProxy bool
}

type server struct {
	adminToken []byte
	store      Store
	publicDir  string
	logger     *slog.Logger
	public     *rateLimiter
	admin      *rateLimiter
	trustProxy bool
}

// NewHandler validates config and returns the complete public service handler.
func NewHandler(cfg Config) (http.Handler, error) {
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return nil, errors.New("admin token is required")
	}
	if cfg.Store == nil {
		return nil, errors.New("publication store is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	publicLimit := cfg.PublicRequestsPerMinute
	if publicLimit <= 0 {
		publicLimit = 120
	}
	adminLimit := cfg.AdminRequestsPerMinute
	if adminLimit <= 0 {
		adminLimit = 30
	}
	tokenHash := sha256.Sum256([]byte(cfg.AdminToken))
	s := &server{
		adminToken: tokenHash[:],
		store:      cfg.Store,
		publicDir:  strings.TrimSpace(cfg.PublicDir),
		logger:     logger,
		public:     newRateLimiter(publicLimit, time.Minute),
		admin:      newRateLimiter(adminLimit, time.Minute),
		trustProxy: cfg.TrustProxy,
	}

	mux := http.NewServeMux()
	mux.Handle("PUT /api/admin/snapshot", s.adminOnly(http.HandlerFunc(s.publish)))
	mux.Handle("DELETE /api/admin/snapshot", s.adminOnly(http.HandlerFunc(s.unpublish)))
	mux.Handle("GET /api/public/snapshot", s.publicOnly(http.HandlerFunc(s.current)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /", s.publicOnly(s.static()))
	return securityHeaders(recoverPanics(logger, mux)), nil
}

// securityHeaders applies conservative defaults to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		// Ignored over plain HTTP; honored when TLS terminates here or at a
		// reverse proxy that forwards the response headers.
		h.Set("Strict-Transport-Security", "max-age=31536000")
		h.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// recoverPanics keeps one bad request from terminating the public service and
// logs the panic locally instead of exposing it to the caller.
func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			value := recover()
			if value == nil {
				return
			}
			if value == http.ErrAbortHandler {
				panic(value)
			}
			logger.Error("panic serving request",
				"panic", fmt.Sprint(value),
				"method", r.Method,
				"path", r.URL.Path,
				"stack", string(debug.Stack()))
			writeError(w, http.StatusInternalServerError, "The request could not be completed.")
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *server) publicOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.public.allow(s.clientKey(r)) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusTooManyRequests, "Too many requests.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !s.admin.allow(s.clientKey(r)) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusTooManyRequests, "Too many requests.")
			return
		}
		if !s.authorized(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+authRealm+`"`)
			writeError(w, http.StatusUnauthorized, "Authentication required.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) authorized(r *http.Request) bool {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	presented := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if presented == "" {
		return false
	}
	hash := sha256.Sum256([]byte(presented))
	return subtle.ConstantTimeCompare(hash[:], s.adminToken) == 1
}

func (s *server) publish(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json.")
		return
	}
	var snapshot publishing.Snapshot
	if err := decodeStrictJSON(w, r, &snapshot); err != nil {
		status := http.StatusBadRequest
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error())
		return
	}
	if err := publishing.ValidateSnapshot(snapshot); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Snapshot rejected: "+err.Error())
		return
	}
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" && key != snapshot.SnapshotID {
		writeError(w, http.StatusBadRequest, "Idempotency-Key must match snapshotId.")
		return
	}

	pub, duplicate, err := s.store.Publish(r.Context(), snapshot)
	switch {
	case errors.Is(err, ErrSnapshotConflict):
		writeError(w, http.StatusConflict, "This snapshot id was already published with different content.")
		return
	case err != nil:
		s.logger.Error("publish snapshot", "error", err)
		writeError(w, http.StatusInternalServerError, "The snapshot could not be published.")
		return
	}
	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	w.Header().Set("ETag", quoteETag(pub.ETag))
	writeJSON(w, status, publishing.PublishResponse{
		Version:     pub.Version,
		PublishedAt: pub.PublishedAt,
		Duplicate:   duplicate,
		ETag:        pub.ETag,
	})
}

func (s *server) unpublish(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Delete(r.Context()); err != nil {
		s.logger.Error("unpublish snapshot", "error", err)
		writeError(w, http.StatusInternalServerError, "The snapshot could not be unpublished.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) current(w http.ResponseWriter, r *http.Request) {
	pub, err := s.store.Current(r.Context())
	if errors.Is(err, ErrNoPublication) {
		writeError(w, http.StatusNotFound, "No monthly statistics are published yet.")
		return
	}
	if err != nil {
		s.logger.Error("read current snapshot", "error", err)
		writeError(w, http.StatusInternalServerError, "The monthly statistics could not be read.")
		return
	}
	etag := quoteETag(pub.ETag)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Publication-Version", strconv.FormatInt(pub.Version, 10))
	w.Header().Set("X-Published-At", pub.PublishedAt)
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	// The public endpoint returns the allow-listed aggregate snapshot itself.
	// Publication metadata stays in headers so the response shape is exactly
	// the reviewed payload shown in the desktop preview.
	writeJSON(w, http.StatusOK, pub.Snapshot)
}

func (s *server) static() http.Handler {
	if s.publicDir == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "The public site is not configured.")
		})
	}
	files := http.FileServer(http.Dir(s.publicDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		full := filepath.Join(s.publicDir, filepath.FromSlash(name))
		rel, err := filepath.Rel(s.publicDir, full)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}
		if info, err := os.Stat(full); err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(name, ".html") {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		files.ServeHTTP(w, r)
	})
}

func (s *server) clientKey(r *http.Request) string {
	if s.trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if first, _, ok := strings.Cut(forwarded, ","); ok {
				forwarded = first
			}
			if ip := net.ParseIP(strings.TrimSpace(forwarded)); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxSnapshotBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return maxErr
		}
		if errors.Is(err, io.EOF) {
			//lint:ignore ST1005 this message is returned verbatim by the API
			return errors.New("Request body is empty.")
		}
		//lint:ignore ST1005 this message is returned verbatim by the API
		return errors.New("Request body is not a valid aggregate snapshot.")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		//lint:ignore ST1005 this message is returned verbatim by the API
		return errors.New("Request body must contain exactly one JSON object.")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func quoteETag(etag string) string {
	if etag == "" {
		return ""
	}
	return `"` + etag + `"`
}

func matchesETag(header, etag string) bool {
	if etag == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == etag || candidate == "*" {
			return true
		}
	}
	return false
}

// rateLimiter is a tiny per-key token bucket. It is deliberately in-memory:
// the public service is a single process and limits are a first line of
// defense, not an accounting system.
type rateLimiter struct {
	mu       sync.Mutex
	rate     float64
	burst    float64
	lastSeen map[string]*bucket
	lastGC   time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

func newRateLimiter(perWindow int, window time.Duration) *rateLimiter {
	perSecond := float64(perWindow) / window.Seconds()
	return &rateLimiter{
		rate:     perSecond,
		burst:    float64(perWindow),
		lastSeen: make(map[string]*bucket),
		lastGC:   time.Now(),
	}
}

func (l *rateLimiter) allow(key string) bool {
	if l == nil || l.rate <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastGC) > 10*time.Minute {
		for k, b := range l.lastSeen {
			if now.Sub(b.at) > 10*time.Minute {
				delete(l.lastSeen, k)
			}
		}
		l.lastGC = now
	}
	b, ok := l.lastSeen[key]
	if !ok {
		b = &bucket{tokens: l.burst, at: now}
		l.lastSeen[key] = b
	}
	b.tokens += now.Sub(b.at).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.at = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
