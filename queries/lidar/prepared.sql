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

-- name: ListPreparedExperiments :many
SELECT DISTINCT pm.experiment_id, e.title, e.experiment_start, e.experiment_end
FROM lidar.prepared_profiles pp
JOIN lidar.prepared_meta pm ON pm.id = pp.prepared_meta_id
JOIN lidar.experiments e ON e.id = pm.experiment_id
WHERE pp.deleted_at IS NULL
ORDER BY e.experiment_start;

-- name: ListPreparedProfileWavelengths :many
SELECT DISTINCT lp.wavelength
FROM lidar.prepared_profiles pp
JOIN lidar.prepared_meta pm ON pm.id = pp.prepared_meta_id
JOIN lidar.licel_profiles lp ON lp.id = pp.licel_profile_id
WHERE pm.experiment_id = $1 AND pp.deleted_at IS NULL
ORDER BY lp.wavelength;

-- name: ListPreparedProfilePolarizations :many
SELECT DISTINCT lp.polarization
FROM lidar.prepared_profiles pp
JOIN lidar.prepared_meta pm ON pm.id = pp.prepared_meta_id
JOIN lidar.licel_profiles lp ON lp.id = pp.licel_profile_id
WHERE pm.experiment_id = sqlc.arg('experiment_id') AND pp.deleted_at IS NULL
  AND (sqlc.narg('wavelength')::real IS NULL OR lp.wavelength = sqlc.narg('wavelength'))
ORDER BY lp.polarization;

-- name: ListPreparedProfileDeviceIDs :many
SELECT DISTINCT lp.device_id
FROM lidar.prepared_profiles pp
JOIN lidar.prepared_meta pm ON pm.id = pp.prepared_meta_id
JOIN lidar.licel_profiles lp ON lp.id = pp.licel_profile_id
WHERE pm.experiment_id = sqlc.arg('experiment_id') AND pp.deleted_at IS NULL
  AND (sqlc.narg('wavelength')::real IS NULL OR lp.wavelength = sqlc.narg('wavelength'))
  AND (sqlc.narg('polarization')::text IS NULL OR lp.polarization = sqlc.narg('polarization'))
ORDER BY lp.device_id;

-- name: ListPreparedProfilesByExperiment :many
SELECT
    pp.id,
    pp.data,
    pp.created_at,
    lp.wavelength,
    lp.polarization,
    lp.device_id,
    lp.bin_width,
    lf.measurement_start,
    pm.background_type,
    pm.background_from,
    pm.trim_from
FROM lidar.prepared_profiles pp
JOIN lidar.prepared_meta pm ON pm.id = pp.prepared_meta_id
JOIN lidar.licel_profiles lp ON lp.id = pp.licel_profile_id
JOIN lidar.licelfiles lf ON lf.id = lp.licelfile_id
WHERE pm.experiment_id = sqlc.arg('experiment_id')
  AND pp.deleted_at IS NULL
  AND (sqlc.narg('wavelength')::real IS NULL OR lp.wavelength = sqlc.narg('wavelength'))
  AND (sqlc.narg('polarization')::text IS NULL OR lp.polarization = sqlc.narg('polarization'))
  AND (sqlc.narg('device_id')::text IS NULL OR lp.device_id = sqlc.narg('device_id'))
ORDER BY lp.wavelength, lp.polarization, lp.device_id;

-- name: DeletePreparedMetaByExperimentID :exec
DELETE FROM lidar.prepared_meta WHERE experiment_id = $1;
