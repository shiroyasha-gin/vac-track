-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveAdmins :one
SELECT COUNT(*) FROM users WHERE role = 'admin' AND is_active = 1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? COLLATE NOCASE;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, display_name, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY is_active DESC, username COLLATE NOCASE;

-- name: UpdateUser :one
UPDATE users
SET display_name = ?, role = ?, is_active = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = ?, updated_at = ?
WHERE id = ?
RETURNING *;
