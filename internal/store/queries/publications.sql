-- name: CreatePublication :one
INSERT INTO publications (
    snapshot_id, month, status, endpoint, version, payload_sha256, published_at, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPublication :one
SELECT * FROM publications WHERE id = ?;

-- name: GetPublicationBySnapshotID :one
SELECT * FROM publications WHERE snapshot_id = ?;

-- name: GetLatestPublication :one
SELECT * FROM publications
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListPublications :many
SELECT * FROM publications
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: MarkPublicationUnpublished :one
UPDATE publications
SET status = 'unpublished', unpublished_at = ?
WHERE id = ?
RETURNING *;

-- name: RepublishPublication :one
UPDATE publications
SET status = 'published',
    endpoint = ?,
    version = ?,
    payload_sha256 = ?,
    published_at = ?,
    unpublished_at = NULL
WHERE id = ?
RETURNING *;
