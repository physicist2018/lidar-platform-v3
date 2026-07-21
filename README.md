# lidar-platform-v3

Lidar data processing platform. Ingests raw LICEL archives, parses profiles,
applies background correction, and prepares data for atmospheric analysis.

## Architecture

```
HTTP API (cmd/lidar)
  ├── POST   /api/v1/experiments/create  — upload experiment files
  ├── POST   /api/v1/experiments/task    — create processing task
  ├── GET    /api/v1/tasks/{taskID}      — query async task status
  └── GET    /health                     — health check (no auth)

NATS JetStream — async task queue
  ├── lidar.task.parse_experiment   — parse uploaded archive
  ├── lidar.task.prepare_experiment  — background correction (planned)
  └── lidar.task.process_experiment  — signal processing (planned)

Worker (cmd/worker)
  └── ParseExperimentHandler — downloads archive, creates LicelFile + LicelProfile records

PostgreSQL — domain storage
  └── lidar schema: experiments, licelfiles, licel_profiles, task_statuses, ...
```

## Authentication

All `/api/v1/*` endpoints require a Bearer JWT issued by the identity service.

1. **Register** via identity service: `POST /register`
2. **Verify** account (email link or manual via psql)
3. **Login** to get a token: `POST /login`
4. **Use token** in requests to lidar API:

```bash
curl -H 'Authorization: Bearer <TOKEN>' http://localhost:8091/api/v1/tasks/<taskID>
```

The helper script `scripts/auth.sh` automates registration and login:

```bash
# Register + login
IDENTITY_URL=http://localhost:8090 ./scripts/auth.sh full user@example.com mypassword

# Show saved token
./scripts/auth.sh token
```

Both identity and lidar services must share the same `JWT_SECRET` environment variable.
In docker-compose, this is configured automatically.

## Quick start

```bash
# Terminal 1 — identity service
bash run_identity.sh

# Terminal 2 — lidar API + worker
bash run_lidar.sh
bash run_worker.sh

# Terminal 3 — register and login
./scripts/auth.sh full user@example.com mypassword
```

## Project structure

```
cmd/
├── lidar/          HTTP API server
├── worker/         NATS consumer / task worker
└── identity/       Authentication service

internal/
├── lidar/
│   ├── application/    Use cases
│   ├── config/         Configuration (JWT_SECRET, MinIO, NATS)
│   ├── domain/         Domain entities
│   ├── infrastructure/ 
│   │   ├── messaging/  NATS implementation
│   │   ├── repository/ Postgres implementations (sqlc)
│   │   ├── server/     HTTP handlers, router, JWT middleware
│   │   └── storage/    MinIO implementation
│   └── ports/          Port interfaces (repositories, message queue, file storage)
└── worker/            Task handler interfaces + worker lifecycle

scripts/
└── auth.sh            Auth helper script (register, login, token)

migrations/lidar/      Goose SQL migrations
queries/lidar/         sqlc query definitions
pkg/db/lidar/          Generated sqlc Go code

docs/
├── async-tasks.md     Guide for adding async tasks with status tracking
```

## Tests

Проект содержит **65 тестов**, покрывающих ключевые компоненты.

### Как запустить

```bash
# Все тесты
go test ./internal/...

# Только unit-тесты (без Docker/БД)
go test ./internal/lidar/domain/... \
       ./internal/lidar/application/... \
       ./internal/lidar/infrastructure/server/...

# Интеграционные тесты с PostgreSQL (требует Docker)
DOCKER_TEST=1 go test ./internal/lidar/infrastructure/repository/...

# Интеграционные тесты с существующей БД
TEST_DATABASE_URL=postgresql://user:pass@localhost:5432/main_db?search_path=lidar&sslmode=disable \
  go test ./internal/lidar/infrastructure/repository/...
```

### Тестовые файлы

| Файл | Тестов | Что тестирует |
|------|--------|---------------|
| `server/auth_test.go` | 11 | JWT middleware: валидный токен → 200 + user_id; отсутствует Authorization → 401; пустой Bearer → 401; неверный формат → 401; невалидный токен → 401; неверный секрет → 401; истекший токен → 401; пустой secret fallback; UserIDFromContext; extractBearerToken (6 кейсов) |
| `server/task_handler_test.go` | 7 | HandleCreateTask: invalid JSON → 400; пустой ProfileID → 400; пустой TaskType → 400. HandleGetTaskStatus: успех → 200 + статус; невалидный UUID → 400; не найдено → 404; отсутствует taskID → 400 |
| `server/router_test.go` | 4 | /health без auth → 200; /api/v1/* без auth → 401 (3 endpoints); с валидным токеном → 501 (auth прошёл); с истекшим токеном → 401 |
| `server/experiment_handler_test.go` | 9 | HandleCreateExperiment: успех со всеми полями, успех без опциональных файлов, отсутствует title → 400, отсутствует zenith_angle → 400, неверный zenith_angle → 400, отсутствует latitude → 400, отсутствует longitude → 400, отсутствует experiment_files → 400, use case error → 500 |
| `domain/domain_test.go` | 17 | TaskRecord: создание с experiment_id, без experiment_id, nil params. Experiment: конструктор по умолчанию, с опциями. TimeRange: валидный, inverted, равные. GeoLocation: валидные координаты, неверная широта, неверная долгота. SoftDelete/Restore эксперимента. LicelFile: базовый, с filename, soft delete. LicelProfile: валидный, несовпадение длины данных, PointAt. AtmosphereProfile: валидный, несовпадение длин. ObjectPath: Key/String, пустой bucket/path. StorageObject: конструктор с опциями |
| `application/create_task_test.go` | 3 | CreateTaskUseCase: пустой ProfileID → error, пустой TaskType → error, успех (создание TaskRecord + публикация в NATS) |
| `application/get_task_status_test.go` | 3 | GetTaskStatusUseCase: найдено → response, не найдено → error, with params (failed + error message + JSON params) |
| `repository/task_status_repo_test.go` | 8 | TaskStatusRepository (реальный PostgreSQL через testcontainers): Create + FindByID; Create с experiment_id; дубликат ID → error; UpdateStatus (processing → completed + started_at/finished_at); UpdateStatus failed + error_message; FindByID not found → ErrObjectNotFound; FindByExperimentID (2 tasks); FindAll |

## Dependencies

- **Go** 1.22+
- **PostgreSQL** — domain storage
- **NATS JetStream** — async task queue
- **MinIO** (S3-compatible) — raw file storage
- **sqlc** — type-safe database queries
- **Goose** — database migrations
