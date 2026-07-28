# Changelog

## [Unreleased]

### Added
- **`subject` in `POST /api/v1/experiments/task`** — subject is now an explicit
  required field `"subject"` in the JSON request. Previously it was hardcoded to
  `lidar.task.process_experiment`, now any subject can be specified.
- **`task_id` in `POST /api/v1/experiments/task`** — optional field; auto-generated
  if empty. Allows explicitly setting the task ID (e.g. `expID` for a parse task).
- **Universal task creation** — `CreateExperimentUseCase` now creates tasks through
  `CreateTaskUseCase` instead of directly. Removed duplicate `queue` and
  `taskStatusRepo` dependencies from `CreateExperimentUseCase`.

### Removed
- **`experiment_id` from `lidar.task_statuses`** — column removed (migration
  `005_remove_experiment_id_from_task_statuses.sql`). Experiment association can
  be passed via `task_params` if needed.
- **`FindByExperimentID` from `TaskStatusRepository`** — method removed since it
  was unused in production code.
- **`ExperimentID` field from `GetTaskStatusResponse`** — no longer returned in API.

### Changed
- **`CreateTaskUseCase`** — now a universal use case: `subject` and optional
  `task_id` are passed explicitly in `TaskRequest`.
- **`NewCreateExperimentUseCase`** — signature changed: accepts `*CreateTaskUseCase`
  instead of `queue, taskStatusRepo`.
- **`NewTaskRecord`** — signature changed: `experimentID` parameter removed,
  three parameters remain: `id, subject, taskParams`.

## [Unreleased] (previous)

### Added
- **GET /api/v1/experiments/list** — новый эндпоинт для получения списка экспериментов
  с фильтрацией по временному диапазону.
  - `GET /api/v1/experiments/list` — все эксперименты
  - `GET /api/v1/experiments/list?start_time=2026-01-01T00:00:00Z&end_time=2026-12-31T23:59:59Z` — по диапазону
  - `start_time` и `end_time` опциональны (RFC3339), пагинация через `limit`/`offset`
  - `ListExperimentsUseCase`, хендлер `HandleListExperiments`, sqlc-запрос `ListExperimentsByTimeRange`
- **Adminer** — лёгкий веб-клиент для управления PostgreSQL, доступен через nginx
  по адресу `https://localhost/adminer/`. Заменил pgAdmin (проще, быстрее, не хранит
  конфигурации на диске).
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
- **Tests** (65 тестов, все проходят):
  - JWT middleware (11 тестов): валидный токен, отсутствующий заголовок, пустой Bearer,
    неверный формат, невалидный токен, неверный секрет, истекший токен, пустой secret fallback,
    извлечение user_id из контекста, extractBearerToken (6 кейсов).
  - Task handler (7 тестов): HandleCreateTask (invalid JSON, пустой ProfileID, пустой TaskType),
    HandleGetTaskStatus (200, invalid UUID, 404 not found, missing param).
  - Router (4 теста): /health без auth, /api/v1/* без auth (3 endpoints),
    с валидным токеном (501 = auth passed), с истекшим токеном (401).
  - Domain (17 тестов): TaskRecord (3), Experiment (2), TimeRange (3), GeoLocation (3),
    SoftDelete/Restore, LicelFile (3), LicelProfile (2), AtmosphereProfile (2),
    ObjectPath (2), StorageObject.
  - Use cases (6 тестов): CreateTask — пустой ProfileID, пустой TaskType, успех (с проверкой
    создания TaskRecord и публикации в NATS). GetTaskStatus — found, not found, with params.
  - Repository (8 тестов, с реальным PostgreSQL через testcontainers): Create + FindByID,
    Create с ExperimentID, дубликат ID, UpdateStatus (processing → completed),
    UpdateStatus (failed с error_message), FindByID not found, FindByExperimentID, FindAll.
- **Documentation**: `docs/async-tasks.md` — guide for adding new async tasks with status
  tracking.
- **Test dependencies**: `github.com/testcontainers/testcontainers-go` (v0.43.0) для
  интеграционных тестов с реальной PostgreSQL.

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
- **`docker-compose.yml`**: pgAdmin заменён на Adminer (`adminer:latest`) —
  более лёгкий и простой клиент для PostgreSQL, доступен через `/adminer/`.
- **`nginx/nginx.conf`**: location `/pgadmin/` заменён на `/adminer/` (proxy → adminer:8080).

### Fixed
- **`scripts/auth.sh`**: default `IDENTITY_URL` port corrected from `:8080` to `:8090`
  to match `run_identity.sh` configuration.
- **`docker-compose.yml`**: added missing `JWT_SECRET` environment variable to the `lidar`
  service — both identity and lidar now share the same secret.

### Removed
- Debug logging from `internal/identity/infrastructure/auth/jwt.go` and
  `internal/lidar/infrastructure/server/auth.go` after root cause analysis.
- `check_hmac/` — temporary test directory.
- `pgadmin/` — заменён на Adminer.
