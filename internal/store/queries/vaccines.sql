-- name: CreateVaccine :one
INSERT INTO vaccines (name, code, description, category, unit, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetVaccine :one
SELECT * FROM vaccines WHERE id = ?;

-- name: GetVaccineByName :one
SELECT * FROM vaccines WHERE name = ? COLLATE NOCASE;

-- name: GetVaccineByCode :one
SELECT * FROM vaccines WHERE code = ? COLLATE NOCASE;

-- name: ListVaccines :many
SELECT * FROM vaccines
WHERE (CAST(sqlc.arg(include_archived) AS INTEGER) = 1 OR is_archived = 0)
ORDER BY is_archived ASC, name COLLATE NOCASE ASC;

-- name: UpdateVaccine :one
UPDATE vaccines
SET name = ?, code = ?, description = ?, category = ?, unit = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: SetVaccineArchived :one
UPDATE vaccines
SET is_archived = ?, archived_at = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: CountVaccineHistory :one
SELECT
    (SELECT COUNT(*) FROM usage_entries WHERE usage_entries.vaccine_id = ?) +
    (SELECT COUNT(*) FROM monthly_inventory WHERE monthly_inventory.vaccine_id = ?) +
    (SELECT COUNT(*) FROM appointment_items WHERE appointment_items.vaccine_id = ?) AS history_count;

-- name: DeleteVaccine :exec
DELETE FROM vaccines WHERE id = ?;
