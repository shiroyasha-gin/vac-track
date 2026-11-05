package service

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

const (
	SettingLowStockThreshold = "low_stock_threshold"
	SettingPublisherEndpoint = "publisher_endpoint"
)

// SettingsService reads and writes local operator preferences.
type SettingsService struct {
	base
}

// Value returns a setting or "" when it has not been stored.
func (s *SettingsService) Value(ctx context.Context, key string) (string, error) {
	setting, err := s.store.Queries().GetSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// LowStockThreshold returns the configured threshold. Zero disables alerts.
func (s *SettingsService) LowStockThreshold(ctx context.Context) (int64, error) {
	value, err := s.Value(ctx, SettingLowStockThreshold)
	if err != nil {
		return 0, err
	}
	if value == "" {
		return 10, nil
	}
	threshold, err := strconv.ParseInt(value, 10, 64)
	if err != nil || threshold < 0 {
		return 10, nil
	}
	return threshold, nil
}

// SetLowStockThreshold validates and audits the local alert threshold.
func (s *SettingsService) SetLowStockThreshold(ctx context.Context, threshold int64) error {
	if threshold < 0 || threshold > 1_000_000 {
		return domain.NewError(domain.CodeInvalidInput, "threshold", "يجب أن يكون حد المخزون المنخفض بين 0 و1,000,000 جرعة.")
	}
	return s.store.WithTx(ctx, func(q generated.Querier) error {
		if _, err := q.UpsertSetting(ctx, generated.UpsertSettingParams{
			Key:       SettingLowStockThreshold,
			Value:     strconv.FormatInt(threshold, 10),
			UpdatedAt: s.timestamp(),
		}); err != nil {
			return mapPersistenceError(err, "تعذر حفظ حد المخزون المنخفض.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventSettingsUpdated,
			EntityType: "setting",
			EntityID:   SettingLowStockThreshold,
			Metadata:   map[string]any{"value": threshold},
		})
	})
}

// PublisherEndpoint returns the configured HTTPS publishing base URL.
func (s *SettingsService) PublisherEndpoint(ctx context.Context) (string, error) {
	return s.Value(ctx, SettingPublisherEndpoint)
}

// SetPublisherEndpoint stores a validated base URL. Credentials are stored
// separately in OS-secure storage (or a private local fallback file).
func (s *SettingsService) SetPublisherEndpoint(ctx context.Context, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" && len(endpoint) > 500 {
		return domain.NewError(domain.CodeInvalidInput, "endpoint", "عنوان النشر طويل جداً.")
	}
	return s.store.WithTx(ctx, func(q generated.Querier) error {
		if _, err := q.UpsertSetting(ctx, generated.UpsertSettingParams{
			Key:       SettingPublisherEndpoint,
			Value:     endpoint,
			UpdatedAt: s.timestamp(),
		}); err != nil {
			return mapPersistenceError(err, "تعذر حفظ عنوان النشر.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventSettingsUpdated,
			EntityType: "setting",
			EntityID:   SettingPublisherEndpoint,
			Metadata:   map[string]any{"configured": endpoint != ""},
		})
	})
}
