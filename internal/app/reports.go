package app

import (
	"vaccineshit/internal/service"
)

// MonthReport returns local statistics for a month.
func (a *App) MonthReport(month string) (MonthReportDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return MonthReportDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	report, err := a.services.Reporting.Report(ctx, month)
	if err != nil {
		return MonthReportDTO{}, a.userError(err)
	}
	return monthReportDTO(report), nil
}

// Forecast estimates next-month demand from closed months. It is a planning aid,
// not a medical recommendation.
func (a *App) Forecast(month string) (ForecastDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return ForecastDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	forecast, err := a.services.Reporting.Forecast(ctx, month)
	if err != nil {
		return ForecastDTO{}, a.userError(err)
	}
	return forecastDTO(forecast), nil
}

// AuditEvents returns minimized local audit records, newest first.
func (a *App) AuditEvents(filter AuditFilterDTO) (AuditPageDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return AuditPageDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	page, err := a.services.Audit.List(ctx, service.AuditFilter{
		EventType: filter.EventType, Month: filter.Month, Limit: filter.Limit, Offset: filter.Offset,
	})
	if err != nil {
		return AuditPageDTO{}, a.userError(err)
	}
	events := make([]AuditEventDTO, 0, len(page.Events))
	for _, event := range page.Events {
		events = append(events, AuditEventDTO{
			ID: event.ID, EventType: event.EventType, EntityType: event.EntityType,
			EntityID: event.EntityID, Month: event.Month, Reason: event.Reason,
			Metadata: parseMetadata(event.Metadata), CreatedAt: event.CreatedAt,
		})
	}
	return AuditPageDTO{Events: events, Total: page.Total}, nil
}
