-- Storage Objects
-- name: CreateStorageObject :one
INSERT INTO lidar.storage_objects (bucket, path, size_bytes, etag, content_type, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStorageObjectByID :one
SELECT * FROM lidar.storage_objects WHERE id = $1;

-- name: GetStorageObjectByBucketPath :one
SELECT * FROM lidar.storage_objects WHERE bucket = $1 AND path = $2;

-- name: DeleteStorageObject :exec
DELETE FROM lidar.storage_objects WHERE id = $1;
