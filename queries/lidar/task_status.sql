-- Task Statuses
-- name: CreateTaskStatus :one
INSERT INTO lidar.task_statuses (
    id, subject, status, task_params
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateTaskStatus :exec
UPDATE lidar.task_statuses SET
    status             = $2,
    error_message      = $3,
    updated_at         = now(),
    started_at         = CASE
                            WHEN $2 = 'processing' AND started_at IS NULL THEN now()
                            ELSE started_at
                         END,
    finished_at        = CASE
                            WHEN $2 IN ('completed', 'failed') AND finished_at IS NULL THEN now()
                            ELSE finished_at
                         END
WHERE id = $1;

-- name: GetTaskStatusByID :one
SELECT * FROM lidar.task_statuses WHERE id = $1;

-- name: ListTaskStatuses :many
SELECT * FROM lidar.task_statuses ORDER BY created_at DESC;
