-- name: CreateLicelFile :one
INSERT INTO lidar.licelfiles (
    experiment_id, measurement_start, measurement_stop,
    n_datasets, laser_freq, is_background, raw_storage_id, filename
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetLicelFileByID :one
SELECT * FROM lidar.licelfiles WHERE id = $1 AND deleted_at IS NULL;

-- name: ListLicelFilesByExperiment :many
SELECT * FROM lidar.licelfiles
WHERE experiment_id = $1 AND deleted_at IS NULL
ORDER BY measurement_start ASC;

-- name: SoftDeleteLicelFile :exec
UPDATE lidar.licelfiles SET deleted_at = now() WHERE id = $1;

-- name: RestoreLicelFile :one
UPDATE lidar.licelfiles SET deleted_at = NULL WHERE id = $1
RETURNING *;
