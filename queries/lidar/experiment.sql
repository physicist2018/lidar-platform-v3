-- name: CreateExperiment :one
INSERT INTO lidar.experiments (
    title, comments, zenith_angle,
    experiment_start, experiment_end, longitude, latitude,
    atmosphere_profile_id,
    experiments_storage_id, background_storage_id, meteo_storage_id
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7,
    $8,
    $9, $10, $11
)
RETURNING *;

-- name: GetExperimentByID :one
SELECT * FROM lidar.experiments WHERE id = $1 AND deleted_at IS NULL;

-- name: ListExperiments :many
SELECT * FROM lidar.experiments
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateExperiment :one
UPDATE lidar.experiments SET
    title = COALESCE($2, title),
    comments = COALESCE($3, comments),
    zenith_angle = COALESCE($4, zenith_angle),
    longitude = COALESCE($5, longitude),
    latitude = COALESCE($6, latitude),
    experiment_start = COALESCE($7, experiment_start),
    experiment_end = COALESCE($8, experiment_end),
    atmosphere_profile_id = COALESCE($9, atmosphere_profile_id),
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateExperimentStorageRefs :one
UPDATE lidar.experiments SET
    experiments_storage_id = $2,
    background_storage_id = $3,
    meteo_storage_id = $4,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteExperiment :exec
UPDATE lidar.experiments SET deleted_at = now(), updated_at = now() WHERE id = $1;

-- name: RestoreExperiment :one
UPDATE lidar.experiments SET deleted_at = NULL, updated_at = now() WHERE id = $1
RETURNING *;
