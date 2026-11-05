-- name: CreateMonthlyInventory :one
INSERT INTO monthly_inventory (
    vaccine_id, month, opening_balance, stock_received, status,
    carryover_overridden, carryover_reason, created_at, updated_at
)
VALUES (?, ?, ?, ?, 'open', ?, ?, ?, ?)
RETURNING *;

-- name: GetMonthlyInventory :one
SELECT * FROM monthly_inventory
WHERE vaccine_id = ? AND month = ?;

-- name: GetMonthlyInventoryByID :one
SELECT * FROM monthly_inventory WHERE id = ?;

-- name: UpdateMonthlyInventory :one
UPDATE monthly_inventory
SET opening_balance = ?, stock_received = ?, carryover_overridden = ?,
    carryover_reason = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: CloseMonthlyInventory :one
UPDATE monthly_inventory
SET status = 'closed', closing_balance = ?, closed_at = ?, reopened_at = NULL, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: ReopenMonthlyInventory :one
UPDATE monthly_inventory
SET status = 'open', closing_balance = NULL, closed_at = NULL, reopened_at = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: ListMonthlyInventoryForMonth :many
SELECT
    mi.id,
    mi.vaccine_id,
    mi.month,
    mi.opening_balance,
    mi.stock_received,
    mi.status,
    mi.closing_balance,
    mi.carryover_overridden,
    mi.carryover_reason,
    mi.closed_at,
    mi.created_at,
    mi.updated_at,
    v.name AS vaccine_name,
    v.code AS vaccine_code,
    v.unit AS vaccine_unit,
    v.is_archived AS vaccine_archived
FROM monthly_inventory mi
JOIN vaccines v ON v.id = mi.vaccine_id
WHERE mi.month = ?
ORDER BY v.name COLLATE NOCASE ASC;

-- name: ListMonthlyInventoryByVaccine :many
SELECT * FROM monthly_inventory
WHERE vaccine_id = ?
ORDER BY month DESC
LIMIT ?;

-- name: ListMonthlyInventoryRowsForMonth :many
SELECT * FROM monthly_inventory
WHERE month = ?
ORDER BY id ASC;

-- name: ListOpenMonthlyInventoryForMonth :many
SELECT * FROM monthly_inventory
WHERE month = ? AND status = 'open'
ORDER BY id ASC;

-- name: ListClosedMonthlyInventoryForMonth :many
SELECT * FROM monthly_inventory
WHERE month = ? AND status = 'closed'
ORDER BY id ASC;

-- name: CountClosedMonthlyInventoryForMonth :one
SELECT COUNT(*) FROM monthly_inventory
WHERE month = ? AND status = 'closed';

-- name: GetPreviousMonthlyInventory :one
SELECT * FROM monthly_inventory
WHERE vaccine_id = ? AND month = ?
LIMIT 1;
