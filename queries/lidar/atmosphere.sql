-- Atmosphere Profiles
-- name: CreateAtmosphereProfile :one
INSERT INTO lidar.atmosphere_profiles (experiment_id, altitude, temperature, pressure)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAtmosphereProfileByID :one
SELECT * FROM lidar.atmosphere_profiles WHERE id = $1;

-- name: ListAtmosphereProfiles :many
SELECT * FROM lidar.atmosphere_profiles
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteAtmosphereProfile :exec
DELETE FROM lidar.atmosphere_profiles WHERE id = $1;
