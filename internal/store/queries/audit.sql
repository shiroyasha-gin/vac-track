-- name: CreateAuditEvent :one
INSERT INTO audit_events (event_type, entity_type, entity_id, month, reason, metadata, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListAuditEvents :many
SELECT * FROM audit_events
WHERE (CAST(sqlc.arg(event_type) AS TEXT) = '' OR event_type = CAST(sqlc.arg(event_type) AS TEXT))
  AND (CAST(sqlc.arg(month) AS TEXT) = '' OR month = CAST(sqlc.arg(month) AS TEXT))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_rows) OFFSET sqlc.arg(offset_rows);

-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events
WHERE (CAST(sqlc.arg(event_type) AS TEXT) = '' OR event_type = CAST(sqlc.arg(event_type) AS TEXT))
  AND (CAST(sqlc.arg(month) AS TEXT) = '' OR month = CAST(sqlc.arg(month) AS TEXT));
