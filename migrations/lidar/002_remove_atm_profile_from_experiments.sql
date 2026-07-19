-- +goose Up
ALTER TABLE lidar.experiments DROP COLUMN atmosphere_profile_id;

-- +goose Down
ALTER TABLE lidar.experiments ADD COLUMN atmosphere_profile_id UUID
    REFERENCES lidar.atmosphere_profiles(id) ON DELETE RESTRICT;
