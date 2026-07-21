-- +goose Up
CREATE TABLE lidar.task_statuses (
    id              UUID PRIMARY KEY,
    subject         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    experiment_id   UUID REFERENCES lidar.experiments(id) ON DELETE CASCADE,
    task_params     JSONB NOT NULL DEFAULT '{}',
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ
);

CREATE INDEX idx_task_statuses_experiment_id ON lidar.task_statuses(experiment_id);
CREATE INDEX idx_task_statuses_status ON lidar.task_statuses(status);

-- +goose Down
DROP TABLE IF EXISTS lidar.task_statuses;
