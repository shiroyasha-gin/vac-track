// Command publicd runs the separate internet-facing publication service. It
// serves only reviewed monthly aggregate snapshots and the static public site.
// It is deployed independently from the desktop application and never has
// access to the desktop SQLite database.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vaccineshit/internal/publicapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("public service stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	token := strings.TrimSpace(os.Getenv("PUBLIC_ADMIN_TOKEN"))
	if len(token) < 24 {
		return errors.New("PUBLIC_ADMIN_TOKEN must be set to at least 24 characters")
	}
	dataDir := envOr("PUBLIC_DATA_DIR", filepath.Join("data", "public"))
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	store, err := publicapi.NewFileStore(dataDir)
	if err != nil {
		return err
	}
	siteDir := envOr("PUBLIC_SITE_DIR", "public")
	if info, err := os.Stat(siteDir); err != nil || !info.IsDir() {
		logger.Warn("public site directory is missing; static site serving is disabled", "dir", siteDir)
		siteDir = ""
	}

	handler, err := publicapi.NewHandler(publicapi.Config{
		AdminToken:              token,
		Store:                   store,
		PublicDir:               siteDir,
		Logger:                  logger,
		PublicRequestsPerMinute: envInt("PUBLIC_RATE_LIMIT", 120),
		AdminRequestsPerMinute:  envInt("PUBLIC_ADMIN_RATE_LIMIT", 30),
		TrustProxy:              envBool("PUBLIC_TRUST_PROXY"),
	})
	if err != nil {
		return err
	}

	addr := envOr("PUBLIC_LISTEN_ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	cert := os.Getenv("PUBLIC_TLS_CERT")
	key := os.Getenv("PUBLIC_TLS_KEY")
	if (cert == "") != (key == "") {
		return errors.New("PUBLIC_TLS_CERT and PUBLIC_TLS_KEY must be set together")
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("public service listening", "addr", addr, "tls", cert != "")
		if cert != "" {
			errCh <- server.ListenAndServeTLS(cert, key)
			return
		}
		logger.Warn("serving plain HTTP; terminate TLS at a trusted reverse proxy")
		errCh <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("shutting down public service")
	return server.Shutdown(ctx)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
