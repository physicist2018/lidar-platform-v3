# lidar-platform-v3

Lidar data processing platform. Ingests raw LICEL archives, parses profiles,
applies background correction, and prepares data for atmospheric analysis.

## Architecture

```
HTTP API (cmd/lidar)
  ├── POST   /api/v1/experiments/create  — upload experiment files
  ├── POST   /api/v1/experiments/task    — create processing task
  ├── GET    /api/v1/tasks/{taskID}      — query async task status
  └── GET    /health

NATS JetStream — async task queue
  ├── lidar.task.parse_experiment   — parse uploaded archive
  ├── lidar.task.prepare_experiment  — background correction (planned)
  └── lidar.task.process_experiment  — signal processing (planned)

Worker (cmd/worker)
  └── ParseExperimentHandler — downloads archive, creates LicelFile + LicelProfile records

PostgreSQL — domain storage
  └── lidar schema: experiments, licelfiles, licel_profiles, task_statuses, ...
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
│   ├── config/         Configuration
│   ├── domain/         Domain entities
│   ├── infrastructure/ 
│   │   ├── messaging/  NATS implementation
│   │   ├── repository/ Postgres implementations (sqlc)
│   │   ├── server/     HTTP handlers & router
│   │   └── storage/    MinIO implementation
│   └── ports/          Port interfaces (repositories, message queue, file storage)
└── worker/            Task handler interfaces + worker lifecycle

migrations/lidar/      Goose SQL migrations
queries/lidar/         sqlc query definitions
pkg/db/lidar/          Generated sqlc Go code

docs/
└── async-tasks.md     Guide for adding async tasks with status tracking
```

## Dependencies

- **Go** 1.22+
- **PostgreSQL** — domain storage
- **NATS JetStream** — async task queue
- **MinIO** (S3-compatible) — raw file storage
- **sqlc** — type-safe database queries
- **Goose** — database migrations
