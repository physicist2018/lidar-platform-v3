-- +goose Up
ALTER TABLE lidar.atmosphere_profiles
    ADD COLUMN experiment_id UUID NOT NULL UNIQUE
    REFERENCES lidar.experiments(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE lidar.atmosphere_profiles DROP COLUMN experiment_id;
