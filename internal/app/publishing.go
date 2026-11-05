package app

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/publishing"
	"vaccineshit/internal/store/generated"
)

// PublicationPreview builds exactly the monthly aggregate payload that would
// be uploaded. It is a pure preview: no request is made until PublishMonth.
func (a *App) PublicationPreview(month string) (PublicationPreviewDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return PublicationPreviewDTO{}, a.userError(err)
	}
	preview, err := a.buildPreviewLocked(month)
	if err != nil {
		return PublicationPreviewDTO{}, a.userError(err)
	}
	return preview, nil
}

func (a *App) buildPreviewLocked(month string) (PublicationPreviewDTO, error) {
	ctx, cancel := a.opContext()
	defer cancel()
	overview, err := a.services.Reporting.MonthOverview(ctx, month)
	if err != nil {
		return PublicationPreviewDTO{}, err
	}
	rows := make([]publishing.Vaccine, 0, len(overview.Rows))
	for _, row := range overview.Rows {
		if !row.Initialized && row.UsageTotal == 0 {
			continue
		}
		rows = append(rows, publishing.Vaccine{
			Name: row.VaccineName, Code: row.VaccineCode, Unit: row.Unit,
			OpeningBalance: row.OpeningBalance, StockReceived: row.StockReceived,
			TotalUsage: row.UsageTotal, Remaining: row.Remaining,
		})
	}
	if len(rows) == 0 {
		return PublicationPreviewDTO{}, domain.NewError(domain.CodeNotInitialized, "month",
			"Initialize monthly stock before building a public snapshot.")
	}
	snapshot, err := publishing.Build(overview.Month, a.now(), rows)
	if err != nil {
		return PublicationPreviewDTO{}, fmt.Errorf("build public snapshot: %w", err)
	}
	hash, err := publishing.ContentHash(snapshot)
	if err != nil {
		return PublicationPreviewDTO{}, err
	}
	warnings := make([]string, 0, 2)
	if overview.State != domain.MonthClosed {
		warnings = append(warnings,
			"This month is not closed. Published totals are provisional and may change until the month is closed.")
	}
	if overview.Totals.UninitializedActive > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d active vaccine(s) have no monthly stock row and are excluded from this aggregate.",
			overview.Totals.UninitializedActive))
	}
	preview := PublicationPreviewDTO{
		Month:        overview.Month,
		MonthState:   string(overview.State),
		Snapshot:     snapshot,
		Warnings:     warnings,
		CanPublish:   len(rows) > 0,
		IncludedRows: len(rows),
		GeneratedAt:  overview.GeneratedAt,
	}
	a.pendingPreview = &previewState{
		month: overview.Month, hash: hash, snapshot: snapshot, state: string(overview.State),
	}
	return preview, nil
}

// PublishMonth uploads the reviewed preview. If the month is still open the
// caller must pass confirmOpen after acknowledging the warning. If the source
// data changed since the preview, publishing is refused until the operator
// reviews a fresh preview.
func (a *App) PublishMonth(month string, confirmOpen bool) PublicationResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return publicationFailed(a.actionError(err))
	}
	if a.pendingPreview == nil || a.pendingPreview.month != month {
		return publicationFailed(failedResult(domain.CodeInvalidInput, "month",
			"Review the public snapshot before publishing it."))
	}
	if a.pendingPreview.state != string(domain.MonthClosed) && !confirmOpen {
		return publicationFailed(failedResult(domain.CodeMonthClosed, "month",
			"This month is not closed. Close the month or explicitly confirm publishing provisional figures."))
	}
	// Rebuild to ensure the reviewed figures still match the live database.
	reviewed := a.pendingPreview
	fresh, err := a.buildPreviewLocked(month)
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	freshHash, err := publishing.ContentHash(fresh.Snapshot)
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	if freshHash != reviewed.hash {
		a.pendingPreview = nil
		return publicationFailed(failedResult("preview_stale", "month",
			"Monthly data changed after the preview. Review the new public snapshot before publishing."))
	}
	a.pendingPreview = reviewed
	snapshot := reviewed.snapshot

	ctx, cancel := a.opContext()
	defer cancel()
	endpoint, err := a.services.Settings.PublisherEndpoint(ctx)
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	if strings.TrimSpace(endpoint) == "" {
		return publicationFailed(failedResult(domain.CodeInvalidInput, "endpoint",
			"Configure the HTTPS publishing endpoint in Settings first."))
	}
	token, _, err := a.credentials.Load(publishing.PublisherAccount)
	if errors.Is(err, publishing.ErrCredentialsNotFound) {
		return publicationFailed(failedResult(domain.CodeInvalidInput, "token",
			"Publisher token is not configured. Add it in Settings."))
	}
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	response, err := a.client.Publish(ctx, endpoint, token, snapshot)
	if err != nil {
		a.logf("publish snapshot", "error", err)
		return publicationFailed(failedResult("publish_failed", "",
			"The public service could not be reached or rejected the snapshot. Nothing was changed locally."))
	}

	hash, err := publishing.Hash(snapshot)
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	err = a.store.WithTx(ctx, func(q generated.Querier) error {
		existing, err := q.GetPublicationBySnapshotID(ctx, snapshot.SnapshotID)
		switch {
		case err == nil:
			if existing.PayloadSha256 != hash {
				return domain.NewError(domain.CodeConflict, "", "The snapshot id was already recorded with different content.")
			}
			if existing.Status != "published" {
				if _, err := q.RepublishPublication(ctx, generated.RepublishPublicationParams{
					Endpoint: endpoint, Version: response.Version, PayloadSha256: hash,
					PublishedAt: response.PublishedAt, ID: existing.ID,
				}); err != nil {
					return err
				}
			}
		case errors.Is(err, sql.ErrNoRows):
			if _, err := q.CreatePublication(ctx, generated.CreatePublicationParams{
				SnapshotID: snapshot.SnapshotID, Month: snapshot.Month, Status: "published",
				Endpoint: endpoint, Version: response.Version, PayloadSha256: hash,
				PublishedAt: response.PublishedAt, CreatedAt: a.now().Format(time.RFC3339),
			}); err != nil {
				return err
			}
		default:
			return err
		}
		return audit.Record(ctx, q, a.now(), audit.Event{
			Type: audit.EventPublicationPublished, EntityType: "publication",
			EntityID: snapshot.SnapshotID, Month: snapshot.Month,
			Metadata: map[string]any{
				"version": response.Version, "duplicate": response.Duplicate,
				"payload_sha256": hash, "vaccine_count": len(snapshot.Vaccines),
			},
		})
	})
	if err != nil {
		a.logf("record successful publication", "error", err)
		return publicationFailed(failedResult("publish_record_failed", "",
			"The snapshot was accepted by the public service but could not be recorded locally. Retry publishing; the service de-duplicates by snapshot id."))
	}
	a.pendingPreview = nil
	message := "Public snapshot published."
	if response.Duplicate {
		message = "Public snapshot was already published; the service confirmed the existing version."
	}
	return PublicationResultDTO{
		OK: true, Message: message, SnapshotID: snapshot.SnapshotID,
		Version: response.Version, PublishedAt: response.PublishedAt, Duplicate: response.Duplicate,
	}
}

// Unpublish removes the current public snapshot. No local data is deleted.
func (a *App) Unpublish() PublicationResultDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return publicationFailed(a.actionError(err))
	}
	ctx, cancel := a.opContext()
	defer cancel()
	endpoint, err := a.services.Settings.PublisherEndpoint(ctx)
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	if strings.TrimSpace(endpoint) == "" {
		return publicationFailed(failedResult(domain.CodeInvalidInput, "endpoint",
			"Configure the HTTPS publishing endpoint in Settings first."))
	}
	token, _, err := a.credentials.Load(publishing.PublisherAccount)
	if errors.Is(err, publishing.ErrCredentialsNotFound) {
		return publicationFailed(failedResult(domain.CodeInvalidInput, "token",
			"Publisher token is not configured. Add it in Settings."))
	}
	if err != nil {
		return publicationFailed(a.actionError(err))
	}
	if err := a.client.Unpublish(ctx, endpoint, token); err != nil {
		a.logf("unpublish snapshot", "error", err)
		return publicationFailed(failedResult("publish_failed", "",
			"The public service could not be reached. The public snapshot may still be visible."))
	}

	latest, err := a.store.Queries().GetLatestPublication(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return publicationFailed(a.actionError(err))
	}
	if err == nil && latest.Status == "published" {
		txErr := a.store.WithTx(ctx, func(q generated.Querier) error {
			if _, err := q.MarkPublicationUnpublished(ctx, generated.MarkPublicationUnpublishedParams{
				UnpublishedAt: sql.NullString{String: a.now().Format(time.RFC3339), Valid: true},
				ID:            latest.ID,
			}); err != nil {
				return err
			}
			return audit.Record(ctx, q, a.now(), audit.Event{
				Type: audit.EventPublicationUnpublished, EntityType: "publication",
				EntityID: latest.SnapshotID, Month: latest.Month,
				Metadata: map[string]any{"version": latest.Version},
			})
		})
		if txErr != nil {
			a.logf("record unpublish", "error", txErr)
			return publicationFailed(failedResult("publish_record_failed", "",
				"The public snapshot was removed but the local publication history could not be updated."))
		}
	}
	return PublicationResultDTO{OK: true, Message: "Public snapshot removed."}
}

// PublicationStatus reports configuration and recent publication history.
func (a *App) PublicationStatus() (PublicationStatusDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return PublicationStatusDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	endpoint, err := a.services.Settings.PublisherEndpoint(ctx)
	if err != nil {
		return PublicationStatusDTO{}, a.userError(err)
	}
	configured, storage := a.tokenState()
	history, err := a.store.Queries().ListPublications(ctx, 20)
	if err != nil {
		return PublicationStatusDTO{}, a.userError(err)
	}
	out := PublicationStatusDTO{
		EndpointConfigured:       strings.TrimSpace(endpoint) != "",
		PublisherTokenConfigured: configured,
		PublisherTokenStorage:    storage,
		History:                  make([]PublicationRecordDTO, 0, len(history)),
	}
	for _, item := range history {
		out.History = append(out.History, publicationRecordDTO(item))
	}
	if len(out.History) > 0 {
		out.Latest = &out.History[0]
	}
	return out, nil
}

func publicationFailed(result ActionResult) PublicationResultDTO {
	return PublicationResultDTO{
		OK: false, Code: result.Code, Field: result.Field, Message: result.Message,
	}
}
