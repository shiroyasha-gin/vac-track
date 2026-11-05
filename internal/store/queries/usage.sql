-- name: CreateUsageEntry :one
INSERT INTO usage_entries (
    patient_id, vaccine_id, administered_on, month, quantity, note, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUsageEntry :one
SELECT * FROM usage_entries WHERE id = ?;

-- name: GetUsageEntryDetail :one
SELECT
    ue.id,
    ue.patient_id,
    ue.vaccine_id,
    ue.administered_on,
    ue.month,
    ue.quantity,
    ue.note,
    ue.voided_at,
    ue.void_reason,
    ue.created_at,
    ue.updated_at,
    p.full_name AS patient_name,
    p.patient_code AS patient_code,
    p.is_archived AS patient_archived,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit,
    v.is_archived AS vaccine_archived
FROM usage_entries ue
JOIN patients p ON p.id = ue.patient_id
JOIN vaccines v ON v.id = ue.vaccine_id
WHERE ue.id = ?;

-- name: UpdateUsageEntry :one
UPDATE usage_entries
SET patient_id = ?, administered_on = ?, month = ?, quantity = ?, note = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: VoidUsageEntry :one
UPDATE usage_entries
SET voided_at = ?, void_reason = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: ListUsageEntriesForVaccineDay :many
SELECT
    ue.id,
    ue.patient_id,
    ue.vaccine_id,
    ue.administered_on,
    ue.month,
    ue.quantity,
    ue.note,
    ue.voided_at,
    ue.void_reason,
    ue.created_at,
    ue.updated_at,
    p.full_name AS patient_name,
    p.patient_code AS patient_code,
    p.is_archived AS patient_archived,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit
FROM usage_entries ue
JOIN patients p ON p.id = ue.patient_id
JOIN vaccines v ON v.id = ue.vaccine_id
WHERE ue.vaccine_id = ? AND ue.month = ? AND ue.administered_on = ?
ORDER BY ue.created_at DESC, ue.id DESC;

-- name: ListUsageEntriesForVaccineMonth :many
SELECT
    ue.id,
    ue.patient_id,
    ue.vaccine_id,
    ue.administered_on,
    ue.month,
    ue.quantity,
    ue.note,
    ue.voided_at,
    ue.void_reason,
    ue.created_at,
    ue.updated_at,
    p.full_name AS patient_name,
    p.patient_code AS patient_code,
    p.is_archived AS patient_archived,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit
FROM usage_entries ue
JOIN patients p ON p.id = ue.patient_id
JOIN vaccines v ON v.id = ue.vaccine_id
WHERE ue.vaccine_id = ? AND ue.month = ?
ORDER BY ue.administered_on ASC, ue.created_at ASC;

-- name: ListUsageEntriesForPatient :many
SELECT
    ue.id,
    ue.patient_id,
    ue.vaccine_id,
    ue.administered_on,
    ue.month,
    ue.quantity,
    ue.note,
    ue.voided_at,
    ue.void_reason,
    ue.created_at,
    ue.updated_at,
    p.full_name AS patient_name,
    p.patient_code AS patient_code,
    p.is_archived AS patient_archived,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit
FROM usage_entries ue
JOIN patients p ON p.id = ue.patient_id
JOIN vaccines v ON v.id = ue.vaccine_id
WHERE ue.patient_id = ?
ORDER BY ue.administered_on DESC, ue.id DESC
LIMIT ?;

-- name: SumActiveUsageForVaccineMonth :one
SELECT CAST(COALESCE(SUM(quantity), 0) AS INTEGER) FROM usage_entries
WHERE vaccine_id = ? AND month = ? AND voided_at IS NULL;

-- name: SumActiveUsageForVaccineMonthExcluding :one
SELECT CAST(COALESCE(SUM(quantity), 0) AS INTEGER) FROM usage_entries
WHERE vaccine_id = ? AND month = ? AND voided_at IS NULL AND id <> ?;

-- name: SumActiveUsageForMonth :one
SELECT CAST(COALESCE(SUM(quantity), 0) AS INTEGER) FROM usage_entries
WHERE month = ? AND voided_at IS NULL;

-- name: ListDailyUsageTotalsForMonth :many
SELECT vaccine_id, administered_on, CAST(SUM(quantity) AS INTEGER) AS total
FROM usage_entries
WHERE month = ? AND voided_at IS NULL
GROUP BY vaccine_id, administered_on
ORDER BY administered_on ASC;

-- name: ListUsageTotalsByVaccineForMonth :many
SELECT vaccine_id, CAST(COALESCE(SUM(quantity), 0) AS INTEGER) AS total
FROM usage_entries
WHERE month = ? AND voided_at IS NULL
GROUP BY vaccine_id;

-- name: ListClosedMonthUsageHistory :many
SELECT
    mi.month AS month,
    CAST(COALESCE(SUM(CASE WHEN ue.voided_at IS NULL THEN ue.quantity ELSE 0 END), 0) AS INTEGER) AS usage_total
FROM monthly_inventory mi
LEFT JOIN usage_entries ue
    ON ue.vaccine_id = mi.vaccine_id AND ue.month = mi.month
WHERE mi.vaccine_id = ? AND mi.month < ? AND mi.status = 'closed'
GROUP BY mi.month
ORDER BY mi.month DESC
LIMIT ?;

-- name: ListUsageEntriesForMonthExport :many
SELECT
    ue.id,
    ue.administered_on,
    ue.month,
    ue.quantity,
    ue.voided_at,
    ue.void_reason,
    p.full_name AS patient_name,
    p.patient_code AS patient_code,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit
FROM usage_entries ue
JOIN patients p ON p.id = ue.patient_id
JOIN vaccines v ON v.id = ue.vaccine_id
WHERE ue.month = ?
ORDER BY ue.administered_on ASC, v.name COLLATE NOCASE ASC, p.full_name COLLATE NOCASE ASC;

-- name: ListActiveUsageEntriesForMonth :many
SELECT * FROM usage_entries
WHERE month = ? AND voided_at IS NULL
ORDER BY administered_on ASC, vaccine_id ASC, id ASC;
