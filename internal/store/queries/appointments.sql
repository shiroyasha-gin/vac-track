-- name: CreateAppointment :one
INSERT INTO appointments (
    patient_id, scheduled_on, scheduled_time, note, status, created_at, updated_at
)
VALUES (?, ?, ?, ?, 'scheduled', ?, ?)
RETURNING *;

-- name: CreateCompletedAppointment :one
INSERT INTO appointments (
    patient_id, scheduled_on, scheduled_time, note, status, completed_at, created_at, updated_at
)
VALUES (?, ?, ?, ?, 'completed', ?, ?, ?)
RETURNING *;

-- name: GetAppointment :one
SELECT * FROM appointments WHERE id = ?;

-- name: GetAppointmentByUsageEntryID :one
SELECT a.* FROM appointments a
JOIN appointment_items ai ON ai.appointment_id = a.id
WHERE ai.usage_entry_id = ?;

-- name: UpdateAppointment :one
UPDATE appointments
SET patient_id = ?, scheduled_on = ?, scheduled_time = ?, note = ?, updated_at = ?
WHERE id = ? AND status = 'scheduled'
RETURNING *;

-- name: CompleteAppointment :one
UPDATE appointments
SET status = 'completed', completed_at = ?, updated_at = ?
WHERE id = ? AND status = 'scheduled'
RETURNING *;

-- name: CancelAppointment :one
UPDATE appointments
SET status = 'cancelled', cancel_reason = ?, cancelled_at = ?, updated_at = ?
WHERE id = ? AND status = 'scheduled'
RETURNING *;

-- name: DeleteAppointmentItems :exec
DELETE FROM appointment_items WHERE appointment_id = ?;

-- name: CreateAppointmentItem :one
INSERT INTO appointment_items (
    appointment_id, vaccine_id, quantity, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: SetAppointmentItemUsage :one
UPDATE appointment_items
SET usage_entry_id = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: ListAppointmentItemsByAppointment :many
SELECT
    ai.id,
    ai.appointment_id,
    ai.vaccine_id,
    ai.quantity,
    ai.usage_entry_id,
    ai.created_at,
    ai.updated_at,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit,
    v.is_archived AS vaccine_archived,
    ue.voided_at AS usage_voided_at,
    ue.void_reason AS usage_void_reason,
    ue.administered_on AS usage_administered_on,
    ue.quantity AS usage_quantity
FROM appointment_items ai
JOIN vaccines v ON v.id = ai.vaccine_id
LEFT JOIN usage_entries ue ON ue.id = ai.usage_entry_id
WHERE ai.appointment_id = ?
ORDER BY ai.id;

-- name: ListAppointmentItemsByAppointments :many
SELECT
    ai.id,
    ai.appointment_id,
    ai.vaccine_id,
    ai.quantity,
    ai.usage_entry_id,
    ai.created_at,
    ai.updated_at,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit,
    v.is_archived AS vaccine_archived,
    ue.voided_at AS usage_voided_at,
    ue.void_reason AS usage_void_reason,
    ue.administered_on AS usage_administered_on,
    ue.quantity AS usage_quantity
FROM appointment_items ai
JOIN vaccines v ON v.id = ai.vaccine_id
LEFT JOIN usage_entries ue ON ue.id = ai.usage_entry_id
WHERE ai.appointment_id IN (sqlc.slice('appointment_ids'))
ORDER BY ai.appointment_id, ai.id;

-- name: ListAppointments :many
SELECT
    a.id,
    a.patient_id,
    a.scheduled_on,
    a.scheduled_time,
    a.note,
    a.status,
    a.cancel_reason,
    a.completed_at,
    a.cancelled_at,
    a.created_at,
    a.updated_at,
    p.full_name AS patient_name,
    p.patient_code AS patient_code,
    p.is_archived AS patient_archived
FROM appointments a
JOIN patients p ON p.id = a.patient_id
WHERE (CAST(sqlc.arg(appointment_id) AS INTEGER) = 0 OR a.id = CAST(sqlc.arg(appointment_id) AS INTEGER))
  AND (CAST(sqlc.arg(patient_id) AS INTEGER) = 0 OR a.patient_id = CAST(sqlc.arg(patient_id) AS INTEGER))
  AND (CAST(sqlc.arg(status) AS TEXT) = '' OR a.status = CAST(sqlc.arg(status) AS TEXT))
  AND (CAST(sqlc.arg(from_date) AS TEXT) = '' OR a.scheduled_on >= CAST(sqlc.arg(from_date) AS TEXT))
  AND (CAST(sqlc.arg(to_date) AS TEXT) = '' OR a.scheduled_on <= CAST(sqlc.arg(to_date) AS TEXT))
ORDER BY
    CASE WHEN a.status = 'scheduled' THEN 0 ELSE 1 END ASC,
    CASE WHEN a.status = 'scheduled' THEN a.scheduled_on END ASC,
    CASE WHEN a.status = 'scheduled' THEN a.scheduled_time END ASC,
    a.scheduled_on DESC,
    a.scheduled_time DESC,
    a.id DESC
LIMIT CAST(sqlc.arg(limit_count) AS INTEGER);
