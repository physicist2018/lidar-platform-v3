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

## Dependencies

- **Go** 1.22+
- **PostgreSQL** — domain storage
- **NATS JetStream** — async task queue
- **MinIO** (S3-compatible) — raw file storage
- **sqlc** — type-safe database queries
- **Goose** — database migrations
