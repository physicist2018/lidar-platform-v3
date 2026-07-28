-- +goose Up
ALTER TABLE lidar.task_statuses DROP COLUMN experiment_id;
DROP INDEX IF EXISTS lidar.idx_task_statuses_experiment_id;

-- +goose Down
ALTER TABLE lidar.task_statuses ADD COLUMN experiment_id UUID REFERENCES lidar.experiments(id) ON DELETE CASCADE;
CREATE INDEX idx_task_statuses_experiment_id ON lidar.task_statuses(experiment_id);
