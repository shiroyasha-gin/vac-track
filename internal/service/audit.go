package service

import (
	"context"
	"strings"

	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

// AuditFilter narrows the local audit review.
type AuditFilter struct {
	EventType string
	Month     string
	Limit     int64
	Offset    int64
}

// AuditPage is a page of audit events.
type AuditPage struct {
	Events []generated.AuditEvent
	Total  int64
}

// AuditService provides local, read-only audit review.
type AuditService struct {
	base
}

// List returns newest events first. Metadata is already minimized by writers.
func (s *AuditService) List(ctx context.Context, filter AuditFilter) (AuditPage, error) {
	filter.EventType = strings.TrimSpace(filter.EventType)
	if len(filter.EventType) > 80 {
		return AuditPage{}, domain.NewError(domain.CodeInvalidInput, "event_type", "مرشح حدث التدقيق طويل جداً.")
	}
	if filter.Month != "" {
		if _, err := domain.ParseMonth(filter.Month); err != nil {
			return AuditPage{}, err
		}
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	params := generated.ListAuditEventsParams{
		EventType:  filter.EventType,
		Month:      filter.Month,
		LimitRows:  filter.Limit,
		OffsetRows: filter.Offset,
	}
	events, err := s.store.Queries().ListAuditEvents(ctx, params)
	if err != nil {
		return AuditPage{}, err
	}
	total, err := s.store.Queries().CountAuditEvents(ctx, generated.CountAuditEventsParams{
		EventType: filter.EventType,
		Month:     filter.Month,
	})
	if err != nil {
		return AuditPage{}, err
	}
	return AuditPage{Events: events, Total: total}, nil
}
