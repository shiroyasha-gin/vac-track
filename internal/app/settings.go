package app

import (
	"strings"

	"vaccineshit/internal/domain"
	"vaccineshit/internal/publishing"
)

// Settings returns local configuration without any secret value.
func (a *App) Settings() (SettingsDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return SettingsDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	threshold, err := a.services.Settings.LowStockThreshold(ctx)
	if err != nil {
		return SettingsDTO{}, a.userError(err)
	}
	endpoint, err := a.services.Settings.PublisherEndpoint(ctx)
	if err != nil {
		return SettingsDTO{}, a.userError(err)
	}
	configured, storage := a.tokenState()
	return SettingsDTO{
		LowStockThreshold: threshold, PublisherEndpoint: endpoint,
		PublisherTokenConfigured: configured, PublisherTokenStorage: storage,
		DataDir: a.dataDir, DatabasePath: a.dbPath, BackupDir: a.backupDir,
	}, nil
}

// SetLowStockThreshold stores the local low-stock alert threshold.
func (a *App) SetLowStockThreshold(threshold int64) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if err := a.services.Settings.SetLowStockThreshold(ctx, threshold); err != nil {
		return a.actionError(err)
	}
	return Result("Low-stock threshold saved.")
}

// SetPublisherEndpoint stores a validated HTTPS publishing endpoint.
func (a *App) SetPublisherEndpoint(endpoint string) ActionResult {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		normalized, err := publishing.ValidateBaseURL(endpoint)
		if err != nil {
			return a.actionError(domain.NewError(domain.CodeInvalidInput, "endpoint", err.Error()))
		}
		endpoint = normalized
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if err := a.services.Settings.SetPublisherEndpoint(ctx, endpoint); err != nil {
		return a.actionError(err)
	}
	return Result("Publishing endpoint saved.")
}

// SetPublisherToken stores the publisher bearer token in OS-secure storage. The
// token is never returned, logged or stored in the SQLite database.
func (a *App) SetPublisherToken(token string) ActionResult {
	token = strings.TrimSpace(token)
	if len(token) < 24 {
		return failedResult(domain.CodeInvalidInput, "token", "Publisher token must be at least 24 characters.")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	storage, err := a.credentials.Save(publishing.PublisherAccount, token)
	if err != nil {
		a.logf("save publisher token", "error", err)
		return failedResult("credential_error", "token", "The publisher token could not be stored securely. Details were written to the local application log.")
	}
	if storage == publishing.StorageOSKeyring {
		return Result("Publisher token stored in the operating-system keychain.")
	}
	return Result("Publisher token stored in a private local file because no OS keychain is available.")
}

// ClearPublisherToken removes the stored publisher token from all secure
// storage locations.
func (a *App) ClearPublisherToken() ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.credentials == nil {
		return failedResult("internal_error", "", "Secure credential storage is not available.")
	}
	if err := a.credentials.Delete(publishing.PublisherAccount); err != nil {
		a.logf("delete publisher token", "error", err)
		return failedResult("credential_error", "", "The publisher token could not be removed. Details were written to the local application log.")
	}
	return Result("Publisher token removed.")
}
