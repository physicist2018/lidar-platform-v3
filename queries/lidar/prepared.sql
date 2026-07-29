-- name: CreatePreparedMeta :one
INSERT INTO lidar.prepared_meta (
    experiment_id, background_type, background_from, trim_from
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetPreparedMetaByExperimentID :one
SELECT * FROM lidar.prepared_meta WHERE experiment_id = $1;

-- name: CreatePreparedProfile :one
INSERT INTO lidar.prepared_profiles (
    prepared_meta_id, licel_profile_id, data
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: ListPreparedProfilesByMetaID :many
SELECT * FROM lidar.prepared_profiles
WHERE prepared_meta_id = $1 AND deleted_at IS NULL
ORDER BY created_at;
