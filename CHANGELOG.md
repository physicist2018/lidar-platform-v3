# Changelog

## [Unreleased]

### Added
- **JWT authentication**: all `/api/v1/*` endpoints in the lidar service now require a
  valid Bearer JWT issued by the identity service.
  - `JWTAuthMiddleware` — chi middleware that validates HS256 JWTs against `JWT_SECRET`.
  - If `JWT_SECRET` is not set, an ephemeral random key is generated (dev mode).
  - User ID from token is injected into request context (`server.UserIDFromContext`).
  - `/health` remains public (no authentication required).
- **Task status tracking**: new `lidar.task_statuses` table tracks lifecycle of all async
  tasks (`pending` → `processing` → `completed` / `failed`).
  - Domain entity `TaskRecord`, port `TaskStatusRepository`, Postgres implementation.
  - Migration `004_create_task_statuses.sql` with indexes on `experiment_id` and `status`.
  - Integration in `CreateExperimentUseCase` and `CreateTaskUseCase` — creates a `pending`
    record before publishing to NATS.
  - Integration in `ParseExperimentHandler` — updates status to `processing` on start,
    `completed` on success, `failed` on error.
  - All calculation parameters stored in `task_params` JSONB column.
- **GET /api/v1/tasks/{taskID}**: new API endpoint to query task status.
  - `GetTaskStatusUseCase`, handler `HandleGetTaskStatus`, router registration.
  - Returns `200` with full status, `404` for unknown task, `400` for invalid UUID.
- **Documentation**: `docs/async-tasks.md` — guide for adding new async tasks with status
  tracking.

### Changed
- **`ParseExperimentHandler.processArchive`**: now returns `(domain.TimeRange, error)` to
  propagate the global time range from all LICEL files in the archive.
- **`ParseExperimentHandler.Handle`**: updates experiment `TimeRange` after archive processing
  using the earliest start and latest stop across all files.
- **Constructor signatures**:
  - `NewCreateExperimentUseCase` — added `taskStatusRepo` parameter.
  - `NewCreateTaskUseCase` — added `taskStatusRepo` parameter.
  - `NewParseExperimentHandler` — added `taskStatusRepo` parameter.
  - `NewTaskHandler` — added `getTaskStatusUC` parameter.
- **`.gitignore`**: added `/worker` binary.

### Fixed
- **`scripts/auth.sh`**: default `IDENTITY_URL` port corrected from `:8080` to `:8090`
  to match `run_identity.sh` configuration.
- **`docker-compose.yml`**: added missing `JWT_SECRET` environment variable to the `lidar`
  service — both identity and lidar now share the same secret.

### Removed
- Debug logging from `internal/identity/infrastructure/auth/jwt.go` and
  `internal/lidar/infrastructure/server/auth.go` after root cause analysis.
- `check_hmac/` — temporary test directory.
