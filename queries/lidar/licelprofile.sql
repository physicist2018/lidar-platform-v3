-- name: CreateLicelProfile :one
INSERT INTO lidar.licel_profiles (
    licelfile_id, n_data_points, high_voltage, bin_width,
    wavelength, polarization, device_id,
    n_shots, discr_level, data
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetLicelProfileByID :one
SELECT * FROM lidar.licel_profiles WHERE id = $1 AND deleted_at IS NULL;

-- name: ListLicelProfilesByLicelFile :many
SELECT * FROM lidar.licel_profiles
WHERE licelfile_id = $1 AND deleted_at IS NULL
ORDER BY wavelength, polarization;

-- name: SoftDeleteLicelProfile :exec
UPDATE lidar.licel_profiles SET deleted_at = now(), updated_at = now() WHERE id = $1;

-- name: RestoreLicelProfile :one
UPDATE lidar.licel_profiles SET deleted_at = NULL, updated_at = now() WHERE id = $1
RETURNING *;

-- name: ListBackgroundProfilesByExperiment :many
SELECT lp.id, lp.licelfile_id, lp.n_data_points, lp.high_voltage, lp.bin_width,
       lp.wavelength, lp.polarization, lp.device_id, lp.n_shots, lp.discr_level,
       lp.data, lp.created_at, lp.updated_at, lp.deleted_at
FROM lidar.licel_profiles lp
JOIN lidar.licelfiles lf ON lf.id = lp.licelfile_id
WHERE lf.experiment_id = $1
  AND lf.is_background = TRUE
  AND lp.deleted_at IS NULL
  AND lf.deleted_at IS NULL
ORDER BY lp.wavelength, lp.polarization;

-- name: FindProfilesWithBackgroundByExperiment :many
WITH background_profiles AS (
    SELECT
        lf.experiment_id,
        lp.device_id,
        lp.wavelength,
        lp.polarization,
        lp.id AS bg_profile_id,
        lp.data AS bg_data,
        lp.n_data_points AS bg_points,
        lp.licelfile_id AS bg_licelfile_id,
        lp.bin_width AS bg_bin_width
    FROM lidar.licelfiles lf
    INNER JOIN lidar.licel_profiles lp ON lp.licelfile_id = lf.id
    WHERE lf.experiment_id = $1
        AND lf.is_background = true
        AND lf.deleted_at IS NULL
        AND lp.deleted_at IS NULL
        AND array_length(lp.data, 1) > 0
),
signal_profiles AS (
    SELECT
        lf.experiment_id,
        lp.device_id,
        lp.wavelength,
        lp.polarization,
        lp.id AS signal_profile_id,
        lp.data AS signal_data,
        lp.n_data_points AS signal_points,
        lp.licelfile_id AS signal_licelfile_id,
        lp.bin_width AS signal_bin_width
    FROM lidar.licelfiles lf
    INNER JOIN lidar.licel_profiles lp ON lp.licelfile_id = lf.id
    WHERE lf.experiment_id = $1
        AND lf.is_background = false
        AND lf.deleted_at IS NULL
        AND lp.deleted_at IS NULL
        AND array_length(lp.data, 1) > 0
)
SELECT
    s.experiment_id,
    s.signal_profile_id,
    s.device_id,
    s.wavelength,
    s.polarization,
    s.signal_data,
    s.signal_points,
    s.signal_licelfile_id,
    s.signal_bin_width,
    b.bg_profile_id,
    b.bg_data,
    b.bg_points,
    b.bg_licelfile_id,
    b.bg_bin_width,
    CASE
        WHEN b.bg_profile_id IS NULL THEN 'NO_BACKGROUND'
        WHEN s.signal_points = b.bg_points THEN 'OK'
        ELSE 'MISMATCH'
    END AS data_length_match
FROM signal_profiles s
LEFT JOIN background_profiles b
    ON s.experiment_id = b.experiment_id
    AND s.device_id = b.device_id
    AND s.wavelength = b.wavelength
    AND s.polarization = b.polarization
ORDER BY s.device_id, s.wavelength, s.polarization;
