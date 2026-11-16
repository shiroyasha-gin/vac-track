// Package audit records concise, local audit events. Events must never contain
// patient names, notes, publishing tokens, or other unnecessary sensitive data.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"vaccineshit/internal/store/generated"
)

// Event types. Keep values stable so audit review filters remain useful.
const (
	EventVaccineCreated     = "vaccine.created"
	EventVaccineUpdated     = "vaccine.updated"
	EventVaccineArchived    = "vaccine.archived"
	EventVaccineReactivated = "vaccine.reactivated"
	EventVaccineDeleted     = "vaccine.deleted"

	EventPatientCreated     = "patient.created"
	EventPatientUpdated     = "patient.updated"
	EventPatientArchived    = "patient.archived"
	EventPatientReactivated = "patient.reactivated"

	EventInventoryInitialized  = "inventory.initialized"
	EventInventoryStockUpdated = "inventory.stock_updated"
	EventInventoryAdjusted     = "inventory.adjusted"
	EventCarryoverOverridden   = "inventory.carryover_overridden"

	EventMonthClosed   = "month.closed"
	EventMonthReopened = "month.reopened"

	EventUsageCreated = "usage.created"
	EventUsageUpdated = "usage.updated"
	EventUsageVoided  = "usage.voided"

	EventAppointmentCreated   = "appointment.created"
	EventAppointmentUpdated   = "appointment.updated"
	EventAppointmentCompleted = "appointment.completed"
	EventAppointmentCancelled = "appointment.cancelled"

	EventBackupCreated  = "backup.created"
	EventBackupRestored = "backup.restored"

	EventPublicationPublished   = "publication.published"
	EventPublicationUnpublished = "publication.unpublished"
	EventPublicationFailed      = "publication.failed"

	EventSettingsUpdated = "settings.updated"
	EventExportCompleted = "export.completed"

	EventLoginSucceeded  = "auth.login_succeeded"
	EventPasswordChanged = "auth.password_changed"

	EventUserCreated       = "user.created"
	EventUserUpdated       = "user.updated"
	EventUserDeactivated   = "user.deactivated"
	EventUserReactivated   = "user.reactivated"
	EventUserPasswordReset = "user.password_reset"
)

type actorContextKey struct{}

// Actor identifies the signed-in operator that caused a mutation. The app layer
// attaches it to every operation context; audit.Record copies it into metadata
// without forcing every service call site to repeat the identity.
type Actor struct {
	ID       int64
	Username string
}

// WithActor returns a context carrying the acting operator.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// ActorFromContext returns the operator attached by the app layer, if any.
func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	return actor, ok && actor.ID > 0
}

// Event is the allow-listed audit payload. Metadata must contain numeric IDs,
// counts and non-sensitive labels only.
type Event struct {
	Type       string
	EntityType string
	EntityID   string
	Month      string
	Reason     string
	Metadata   map[string]any
}

// Record writes one audit event through the supplied transaction-bound
// querier. Callers must pass the same querier used for the mutation so both
// commit or roll back together.
func Record(ctx context.Context, q generated.Querier, now time.Time, event Event) error {
	if strings.TrimSpace(event.Type) == "" || strings.TrimSpace(event.EntityType) == "" {
		return fmt.Errorf("audit event type and entity type are required")
	}
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if actor, ok := ActorFromContext(ctx); ok {
		// Copy so callers keep ownership of their map and cannot overwrite the
		// signed-in identity.
		withActor := make(map[string]any, len(metadata)+2)
		for key, value := range metadata {
			withActor[key] = value
		}
		withActor["actor_id"] = actor.ID
		withActor["actor_username"] = actor.Username
		metadata = withActor
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	reason := strings.TrimSpace(event.Reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	_, err = q.CreateAuditEvent(ctx, generated.CreateAuditEventParams{
		EventType:  event.Type,
		EntityType: event.EntityType,
		EntityID:   event.EntityID,
		Month:      event.Month,
		Reason:     reason,
		Metadata:   string(body),
		CreatedAt:  now.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}
