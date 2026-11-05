-- name: CreateInventoryAdjustment :one
INSERT INTO inventory_adjustments (
    vaccine_id, month, amount, balance_before, balance_after, reason, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListInventoryAdjustmentsForMonth :many
SELECT
    a.id,
    a.vaccine_id,
    a.month,
    a.amount,
    a.balance_before,
    a.balance_after,
    a.reason,
    a.created_at,
    v.name AS vaccine_name,
    v.unit AS vaccine_unit
FROM inventory_adjustments a
JOIN vaccines v ON v.id = a.vaccine_id
WHERE a.month = ?
ORDER BY a.created_at DESC, a.id DESC;

-- name: CountInventoryAdjustmentsForMonth :one
SELECT COUNT(*) FROM inventory_adjustments WHERE month = ?;
