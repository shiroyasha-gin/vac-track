package publishing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PublishResponse is the hosted service's result.
type PublishResponse struct {
	Version     int64  `json:"version"`
	PublishedAt string `json:"publishedAt"`
	Duplicate   bool   `json:"duplicate"`
	ETag        string `json:"etag"`
}

// Client uploads reviewed aggregate snapshots to a separate hosted service.
type Client struct {
	HTTP *http.Client
}

// NewClient returns a client with conservative production timeouts.
func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 20 * time.Second}}
}

// ValidateBaseURL normalizes a publishing base URL. HTTPS is required; plain
// HTTP is accepted only for loopback development.
func ValidateBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("publishing endpoint is not configured")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("publishing endpoint is not a valid URL")
	}
	if u.User != nil {
		return "", errors.New("publishing endpoint must not contain credentials")
	}
	if u.Fragment != "" || u.RawQuery != "" {
		return "", errors.New("publishing endpoint must not contain a query or fragment")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("publishing endpoint must use HTTPS (HTTP is allowed only for localhost testing)")
		}
	} else if u.Scheme != "https" {
		return "", errors.New("publishing endpoint must use HTTPS")
	}
	if u.Host == "" {
		return "", errors.New("publishing endpoint must include a host")
	}
	return u.String(), nil
}

// Publish sends one snapshot over HTTPS. Retries are safe because the server
// de-duplicates by snapshotId.
func (c *Client) Publish(ctx context.Context, baseURL, token string, snapshot Snapshot) (PublishResponse, error) {
	if strings.TrimSpace(token) == "" {
		return PublishResponse{}, errors.New("publisher token is not configured")
	}
	base, err := ValidateBaseURL(baseURL)
	if err != nil {
		return PublishResponse{}, err
	}
	if err := ValidateSnapshot(snapshot); err != nil {
		return PublishResponse{}, fmt.Errorf("snapshot is not safe to publish: %w", err)
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return PublishResponse{}, fmt.Errorf("encode snapshot: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, base+"/api/admin/snapshot", bytes.NewReader(body))
	if err != nil {
		return PublishResponse{}, fmt.Errorf("build publish request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Idempotency-Key", snapshot.SnapshotID)
	req.Header.Set("User-Agent", "vaccineshit-desktop/1")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return PublishResponse{}, fmt.Errorf("publish snapshot: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return PublishResponse{}, fmt.Errorf("read publish response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PublishResponse{}, responseError(resp.StatusCode, payload)
	}
	var out PublishResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return PublishResponse{}, errors.New("publishing service returned an unexpected response")
	}
	return out, nil
}

// Unpublish removes the current public snapshot.
func (c *Client) Unpublish(ctx context.Context, baseURL, token string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("publisher token is not configured")
	}
	base, err := ValidateBaseURL(baseURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, base+"/api/admin/snapshot", nil)
	if err != nil {
		return fmt.Errorf("build unpublish request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "vaccineshit-desktop/1")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("unpublish snapshot: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp.StatusCode, payload)
	}
	return nil
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func responseError(status int, payload []byte) error {
	var body struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(payload, &body)
	message := strings.TrimSpace(body.Error)
	if message == "" {
		message = http.StatusText(status)
	}
	if len(message) > 200 {
		message = message[:200]
	}
	return fmt.Errorf("publishing service returned HTTP %d: %s", status, message)
}
