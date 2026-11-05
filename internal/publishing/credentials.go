package publishing

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

// ErrCredentialsNotFound means no publisher token is stored.
var ErrCredentialsNotFound = errors.New("no publisher credential is stored")

// StorageKind says where a credential was stored.
type StorageKind string

const (
	StorageOSKeyring StorageKind = "os_keyring"
	StorageLocalFile StorageKind = "local_file"
)

const (
	credentialService = "vaccineshit"
	PublisherAccount  = "publisher-token"
)

// KeyringBackend is the small part of the OS keychain API used here.
type KeyringBackend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type systemKeyring struct{}

func (systemKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}
func (systemKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}
func (systemKeyring) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

// FileCredentialStore is the fallback when no OS keychain is available. The
// directory is 0700 and the file is 0600; the token is never written to logs or
// the SQLite database.
type FileCredentialStore struct {
	Path string
}

// Save writes the token atomically with private permissions.
func (s *FileCredentialStore) Save(account, secret string) (StorageKind, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("publisher token is empty")
	}
	if s.Path == "" {
		return "", errors.New("credential fallback path is not configured")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return "", fmt.Errorf("create credential directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.Path), ".publisher-token-*.tmp")
	if err != nil {
		return "", fmt.Errorf("reserve credential file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return "", fmt.Errorf("protect credential file: %w", err)
	}
	if _, err := temp.WriteString(strings.TrimSpace(secret)); err != nil {
		temp.Close()
		return "", fmt.Errorf("write credential file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", fmt.Errorf("sync credential file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, s.Path); err != nil {
		return "", fmt.Errorf("finalize credential file: %w", err)
	}
	if err := os.Chmod(s.Path, 0o600); err != nil {
		return "", fmt.Errorf("protect credential file: %w", err)
	}
	return StorageLocalFile, nil
}

// Load reads a fallback credential.
func (s *FileCredentialStore) Load(account string) (string, error) {
	if s.Path == "" {
		return "", ErrCredentialsNotFound
	}
	body, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrCredentialsNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read credential file: %w", err)
	}
	secret := strings.TrimSpace(string(body))
	if secret == "" {
		return "", ErrCredentialsNotFound
	}
	return secret, nil
}

// Delete removes a fallback credential.
func (s *FileCredentialStore) Delete(account string) error {
	if s.Path == "" {
		return nil
	}
	if err := os.Remove(s.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete credential file: %w", err)
	}
	return nil
}

// Credentials prefers the OS keychain and falls back to a private local file.
type Credentials struct {
	backend  KeyringBackend
	fallback *FileCredentialStore
}

// NewCredentials wires OS keychain storage with a local fallback.
func NewCredentials(fallback *FileCredentialStore) *Credentials {
	return &Credentials{backend: systemKeyring{}, fallback: fallback}
}

// NewCredentialsForTest allows an injected keychain backend.
func NewCredentialsForTest(backend KeyringBackend, fallback *FileCredentialStore) *Credentials {
	return &Credentials{backend: backend, fallback: fallback}
}

// Save stores the publisher token and returns where it landed. If the OS
// keychain is unavailable, the token is written to the protected fallback file
// and no secret is ever logged.
func (c *Credentials) Save(account, secret string) (StorageKind, error) {
	if c.backend != nil {
		if err := c.backend.Set(credentialService, account, secret); err == nil {
			if c.fallback != nil {
				_ = c.fallback.Delete(account)
			}
			return StorageOSKeyring, nil
		}
	}
	if c.fallback == nil {
		return "", errors.New("no secure credential storage is available on this system")
	}
	return c.fallback.Save(account, secret)
}

// Load retrieves the publisher token, preferring the OS keychain.
func (c *Credentials) Load(account string) (string, StorageKind, error) {
	if c.backend != nil {
		secret, err := c.backend.Get(credentialService, account)
		if err == nil && strings.TrimSpace(secret) != "" {
			return strings.TrimSpace(secret), StorageOSKeyring, nil
		}
	}
	if c.fallback != nil {
		secret, err := c.fallback.Load(account)
		if err == nil {
			return secret, StorageLocalFile, nil
		}
		if !errors.Is(err, ErrCredentialsNotFound) {
			return "", "", err
		}
	}
	return "", "", ErrCredentialsNotFound
}

// Delete removes the token from both storage locations.
func (c *Credentials) Delete(account string) error {
	var firstErr error
	if c.backend != nil {
		if err := c.backend.Delete(credentialService, account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			// Keychain unavailability is not fatal to local deletion; the
			// fallback file may still exist.
			firstErr = err
		}
	}
	if c.fallback != nil {
		if err := c.fallback.Delete(account); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// DefaultFallbackPath places the fallback token in the app data directory.
func DefaultFallbackPath(dataDir string) *FileCredentialStore {
	return &FileCredentialStore{Path: filepath.Join(dataDir, "secrets", "publisher-token")}
}
