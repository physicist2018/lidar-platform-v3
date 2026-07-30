# lidar-platform-v3

Lidar data processing platform. Ingests raw LICEL archives, parses profiles,
applies background correction, and prepares data for atmospheric analysis.

## Architecture

```
Frontend SPA (Vite + vanilla JS)  — https://localhost / localhost:5173
  ├── /login, /register          — auth via identity service
  ├── /experiments               — list / detail / upload
  └── /experiments/:id           — task management

Nginx (SSL termination + routing)
  ├── / → frontend SPA
  ├── /login, /register, /verify  → identity:8090
  ├── /api/*                     → lidar:8091
  └── /adminer/*                 → adminer:8080

HTTP API (cmd/lidar)
  ├── POST   /api/v1/experiments/create           — upload experiment files
  ├── POST   /api/v1/experiments/task             — create processing task
  ├── GET    /api/v1/experiments/list              — list experiments (with time filter)
  ├── GET    /api/v1/tasks/{taskID}               — query async task status
  ├── DELETE /api/v1/tasks/{taskID}               — delete task and results
  ├── GET    /api/v1/prepared-profiles             — query processed profiles
  ├── GET    /api/v1/prepared-profiles/experiments — list experiments with prepared data
  ├── GET    /api/v1/prepared-profiles/filters     — available wavelength/polarization/device_id
  └── GET    /health                              — health check (no auth)

NATS JetStream — async task queue
  ├── lidar.task.parse_experiment    — parse uploaded archive
  ├── lidar.task.prepare_experiment   — background correction
  └── lidar.task.process_experiment  — signal processing (planned)

Worker (cmd/worker)
  ├── ParseExperimentHandler   — downloads archive, creates LicelFile + LicelProfile records
  └── PrepareExperimentHandler — background removal and profile trimming

PostgreSQL — domain storage
  ├── experiments, licelfiles, licel_profiles, task_statuses
  └── prepared_meta, prepared_profiles — processed data
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

### Docker Compose (full stack)

```bash
# Build frontend
cd frontend && npm run build && cd ..

# Start all services
docker compose up -d --build

# Open https://localhost in browser
```

### Development mode (hot reload)

```bash
# Terminal 1 — backend services
docker compose up -d identity lidar worker nats minio postgres

# Terminal 2 — frontend dev server
cd frontend && npm run dev
# → http://localhost:5173 (API proxies through nginx at https://localhost)
```

### Local dev (without Docker)

```bash
# Terminal 1 — identity service
bash run_identity.sh

# Terminal 2 — lidar API + worker
bash run_lidar.sh
bash run_worker.sh

# Terminal 3 — frontend
cd frontend && npm run dev

# Terminal 4 — register and login
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

frontend/
├── index.html          SPA entry point
├── vite.config.js      Proxy + build config
└── src/
    ├── api.js          HTTP client (JWT, 401 redirect)
    ├── router.js       Hash-based SPA router
    ├── store.js        Application state
    ├── styles.css      Full UI kit (cards, tables, forms, badges)
    └── pages/          Login, register, verified, experiments, upload

scripts/
└── auth.sh            Auth helper script (register, login, token)

migrations/lidar/      Goose SQL migrations
queries/lidar/         sqlc query definitions
pkg/db/lidar/          Generated sqlc Go code

docs/
├── async-tasks.md     Guide for adding async tasks with status tracking
├── frontend.md        Guide for extending the frontend (pages, API, routing)
```

## Tests

Проект содержит **68 тестов** (Go backend), покрывающих ключевые компоненты.

### Как запустить

```bash
# Backend unit tests
go test ./internal/...

# Frontend build verification
cd frontend && npm run build
```

### Тестовые файлы

| Файл | Тестов | Что тестирует |
|------|--------|---------------|
| `server/auth_test.go` | 11 | JWT middleware |
| `server/task_handler_test.go` | 8 | HandleCreateTask, HandleGetTaskStatus |
| `server/router_test.go` | 4 | Health + auth checks |
| `server/experiment_handler_test.go` | 9 | Experiment creation handler |
| `domain/domain_test.go` | 16 | All domain entities |
| `application/create_task_test.go` | 4 | CreateTaskUseCase (empty subject, empty task_type, success, idempotent) |
| `application/get_task_status_test.go` | 3 | GetTaskStatusUseCase |
| `repository/task_status_repo_test.go` | 7 | TaskStatusRepository (real PG via testcontainers) |
| `worker/prepare_experiment_test.go` | 6 | processProfile core logic |

## Dependencies

- **Go** 1.22+
- **PostgreSQL** — domain storage
- **NATS JetStream** — async task queue
- **MinIO** (S3-compatible) — raw file storage
- **sqlc** — type-safe database queries
- **Goose** — database migrations
- **Node.js 20+** — frontend build
