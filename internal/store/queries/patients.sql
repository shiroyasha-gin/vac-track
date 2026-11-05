-- name: CreatePatient :one
INSERT INTO patients (full_name, patient_code, created_at, updated_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetPatient :one
SELECT * FROM patients WHERE id = ?;

-- name: ListPatients :many
SELECT * FROM patients
WHERE (CAST(sqlc.arg(include_archived) AS INTEGER) = 1 OR is_archived = 0)
  AND (
    CAST(sqlc.arg(search) AS TEXT) = ''
    OR full_name LIKE '%' || CAST(sqlc.arg(search) AS TEXT) || '%'
    OR patient_code LIKE '%' || CAST(sqlc.arg(search) AS TEXT) || '%'
  )
ORDER BY is_archived ASC, full_name COLLATE NOCASE ASC;

-- name: ListActivePatients :many
SELECT * FROM patients
WHERE is_archived = 0
ORDER BY full_name COLLATE NOCASE ASC;

-- name: UpdatePatient :one
UPDATE patients
SET full_name = ?, patient_code = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: SetPatientArchived :one
UPDATE patients
SET is_archived = ?, archived_at = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: CountPatientUsage :one
SELECT COUNT(*) FROM usage_entries WHERE patient_id = ?;
