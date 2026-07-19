# Lidar Platform v3

Платформа для обработки LiDAR-данных.

---

## Архитектура

```
lidar-platform-v3/
├── cmd/
│   ├── identity/                    # Точка входа identity
│   ├── lidar/                       # Точка входа lidar (HTTP API)
│   └── worker/                      # Точка входа worker (NATS consumer)
├── internal/
│   ├── identity/                    # Бизнес-логика identity
│   │   ├── domain/                  # Entity, Value Objects, errors
│   │   ├── application/             # Use cases (Register, Verify, Login)
│   │   ├── ports/                   # Интерфейсы (repository, mail, token)
│   │   └── infrastructure/          # Адаптеры (PostgreSQL, SMTP, JWT, HTTP)
│   ├── lidar/                       # Бизнес-логика lidar
│   │   ├── domain/                  # Entity, Value Objects, errors
│   │   ├── application/             # Use cases (CreateExperiment, CreateTask)
│   │   ├── ports/                   # Интерфейсы (repository, file storage, message queue)
│   │   ├── infrastructure/          # Адаптеры (PostgreSQL, MinIO, NATS, HTTP)
│   │   └── config/                  # Чтение env-переменных
│   └── worker/                      # Worker (NATS consumer)
│       ├── config/                  # Чтение env-переменных
│       ├── worker.go                # Worker Run/Stop
│       ├── handler.go               # TaskHandler interface
│       └── parse_experiment.go      # ParseExperiment handler
├── nginx/
│   ├── nginx.conf                   # HTTPS + proxy на identity, lidar, minio console
│   └── Dockerfile                   # nginx:alpine + self-signed cert
├── scripts/
│   └── create_experiment.sh         # Тестовый скрипт для API
├── testdata/                        # Тестовые файлы (LICEL, meteo, zip)
├── migrations/
│   ├── identity/                    # Goose-миграции identity
│   └── lidar/                       # Goose-миграции lidar
├── queries/                         # sqlc-запросы
├── pkg/db/                          # sqlc-генерация
├── docker-compose.yml               # nginx + postgres + minio + nats + identity + lidar + worker
├── sqlc.yml                         # Конфиг sqlc
└── init-db.sh                       # Инициализация БД (схемы, роли)
```

### Принципы (DDD + SOLID)

| Слой | Отвечает за | Зависит от |
|---|---|---|
| **domain** | Бизнес-правила, entity, value objects | ничего (чистый Go) |
| **application** | Use cases | `ports` (интерфейсы) |
| **ports** | Контракты (репозитории, сервисы) | `domain` |
| **infrastructure** | Реализации (PostgreSQL, SMTP, JWT, MinIO, NATS, HTTP) | `ports` |

---

## Identity — микросервис аутентификации

Регистрация, верификация email и JWT-авторизация.

### API

#### `POST /register`

```bash
curl -k https://localhost/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

| Код | Ответ | Когда |
|---|---|---|
| **201** | `{"message":"user registered, verification email sent"}` | Успех |
| **400** | `{"error":"invalid email format"}` | Невалидный email |
| **409** | `{"error":"email already registered"}` | Email занят |

#### `GET /verify?token=...&email=...`

Верификация по ссылке из письма. Редиректит на `{FRONTEND_URL}/verified?status=ok`.

#### `POST /login`

```bash
curl -k https://localhost/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

| Код | Ответ |
|---|---|
| **200** | `{"token":"eyJhbGciOiJIUzI1NiIs..."}` |
| **401** | `{"error":"invalid email or password"}` |
| **403** | `{"error":"account not verified"}` |

---

## Lidar — микросервис обработки LiDAR-данных

### API

#### `POST /api/v1/experiments/create`

Создаёт эксперимент: загружает файлы в MinIO + сохраняет метаданные в БД.

```bash
curl -k -X POST https://localhost/api/v1/experiments/create \
  -F "title=Test Experiment" \
  -F "zenith_angle=45.5" \
  -F "latitude=43.1" \
  -F "longitude=131.9" \
  -F "comments=optional" \
  -F "experiment_files=@testdata/archive.zip" \
  -F "background=@testdata/b2651321.051986" \
  -F "meteo=@testdata/meteo.csv"
```

| Код | Ответ |
|---|---|
| **201** | `{"id":"uuid","title":"...","zenith_angle":45.5,...}` |
| **400** | `{"error":"title is required"}` |

#### `POST /api/v1/experiments/task`

Создаёт задачу на обработку и публикует её в NATS.

```bash
curl -k -X POST https://localhost/api/v1/experiments/task \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": ["exp-uuid-123"],
    "task_type": "KLETT_FERNALD",
    "payload": {
      "boundary_altitude_km": 10.0,
      "extinction_km_1": 0.1,
      "method_version": "1.2.0"
    }
  }'
```

| Код | Ответ |
|---|---|
| **201** | `{"task_id":"uuid","status":"queued"}` |
| **400** | `{"error":"profile_id must not be empty"}` |

### Domain model

#### StorageObject

```go
type ObjectPath struct { Bucket, Path string }
type StorageObject struct {
    ID          uuid.UUID
    Path        ObjectPath
    Size        int64
    ETag        string
    ContentType string
    Metadata    map[string]any
    CreatedAt   time.Time
}
```

#### Experiment

```go
type TimeRange struct { Start, End time.Time }
type GeoLocation struct { Latitude, Longitude float32 }
type Experiment struct {
    ID          uuid.UUID
    Title       string
    ZenithAngle float32
    TimeRange   TimeRange
    GeoLocation GeoLocation
    StorageRefs ExperimentStorageRefs
    // + soft-delete
}
```

#### AtmosphereProfile, LicelFile, LicelProfile, PairedProfile

Полные определения в `internal/lidar/domain/`.

### File storage (MinIO/S3)

```go
type FileStorage interface {
    Upload(ctx, bucket, path, reader, size, contentType) (*ObjectInfo, error)
    UploadBytes(ctx, bucket, path, data, contentType) (*ObjectInfo, error)
    Download, Delete, Exists, GetInfo, PresignedGetURL, CreateBucket
}
```

### Message Queue (NATS JetStream)

```go
type MessageQueue interface {
    Publish(ctx, subject Subject, data []byte, dedupID string) error
    Subscribe(ctx, subject Subject, consumer string, handler MessageHandler) (Subscription, error)
    Close() error
}
```

Subjects:
- `lidar.task.parse_experiment`
- `lidar.task.prepare_experiment`
- `lidar.task.process_experiment`

### Репозитории

| Repository | Методы |
|---|---|
| `ExperimentRepository` | Create, FindByID, FindAll, Update, UpdateStorageRefs, SoftDelete, Restore |
| `StorageObjectRepository` | Create, FindByID, FindByBucketPath |
| `AtmosphereProfileRepository` | Create, FindByID, FindAll, Delete |
| `LicelFileRepository` | Create, FindByID, FindAllByExperimentID, SoftDelete, Restore |
| `LicelProfileRepository` | Create, FindByID, FindAllByLicelFileID, FindProfilesWithBackground, SoftDelete, Restore |

---

## Worker — микросервис асинхронной обработки

Подписывается на NATS subjects, диспатчит задачи по типу.

```
NATS (lidar.task.*)
  │
  ├─ lidar.task.parse_experiment   → ParseExperimentHandler
  ├─ lidar.task.prepare_experiment → (заглушка)
  └─ lidar.task.process_experiment → (заглушка)
```

Worker имеет доступ к БД, MinIO и NATS (только сеть backend).

### Запуск

```bash
# Docker
docker compose up -d --build worker

# Локально
NATS_URL="nats://localhost:4222" go run ./cmd/worker/
```

---

## Nginx — HTTPS-прокси

Все внешние запросы проходят через nginx на порту 443 (HTTPS). HTTP (80) редиректит на HTTPS.

### Routes

| Path | Target |
|---|---|
| `/register`, `/login`, `/verify` | `http://identity:8090` |
| `/api/*`, `/health` | `http://lidar:8091` |
| `/minio-console/` | `http://minio:9001` (WebSocket) |

Сертификат — self-signed (localhost). Для прода нужно заменить на Let's Encrypt.

---

## Docker

```bash
# Все сервисы
docker compose up -d --build

# Отдельные сервисы
docker compose up -d --build nginx
docker compose up -d --build lidar
docker compose up -d --build worker
```

### Сервисы

| Сервис | Host Port | Сеть | Назначение |
|---|---|---|---|
| **nginx** | **443 (HTTPS), 80** | frontend | Входная точка |
| postgres | ❌ | backend | PostgreSQL |
| minio | ❌ | frontend + backend | S3-хранилище |
| nats | ❌ | backend | Message Queue |
| identity | ❌ | frontend + backend | Аутентификация |
| lidar | ❌ | frontend + backend | HTTP API для LiDAR |
| worker | ❌ | backend | Фоновая обработка |

### Сети

```
frontend — nginx, identity, lidar, minio (доступно извне через nginx)
backend  — postgres, minio, nats, identity, lidar, worker (изолирована)
```

---

## Конфигурация

### Identity

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATABASE_URL` | `postgresql://identity_user:pass@...` | PostgreSQL |
| `HTTP_ADDR` | `:8080` | HTTP порт |
| `JWT_SECRET` | случайный | Секрет JWT |
| `SMTP_*` | — | SMTP-отправка |
| `VERIFY_BASE_URL` | `https://localhost` | Ссылка верификации в письме |
| `FRONTEND_URL` | `https://localhost` | URL для редиректа |

### Lidar / Worker

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATABASE_URL` | `postgresql://lidar_user:pass@...` | PostgreSQL (search_path=lidar) |
| `MIGRATIONS_DIR` | `migrations/lidar` | Путь к goose-миграциям (только lidar) |
| `HTTP_ADDR` | `:8091` | HTTP порт (только lidar) |
| `MINIO_ENDPOINT` | `minio:9000` | MinIO endpoint |
| `MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | `minioadmin` | MinIO secret key |
| `MINIO_USE_SSL` | `false` | MinIO TLS |
| `NATS_URL` | `nats://nats:4222` | NATS server |

---

## Тестирование

```bash
# Запустить тесты
go test ./...

# Отправить тестовый запрос через скрипт
./scripts/create_experiment.sh

# Отправить задачу на обработку
curl -k -X POST https://localhost/api/v1/experiments/task \
  -H "Content-Type: application/json" \
  -d '{"profile_id":["test"],"task_type":"KLETT_FERNALD","payload":{}}'

# Health check
curl -k https://localhost/health

# MinIO Console
open https://localhost/minio-console/
```

---

## Разработка

### Генерация sqlc

```bash
sqlc generate
```

### Добавление миграции

```sql
-- migrations/lidar/003_name.sql
-- +goose Up
ALTER TABLE lidar.experiments ADD COLUMN ...;

-- +goose Down
ALTER TABLE lidar.experiments DROP COLUMN ...;
```

### Добавление обработчика в Worker

#### 1. Создать файл хендлера

Хендлер реализует интерфейс `internal/worker/handler.go`:

```go
package worker

import "context"
import "github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"

type TaskHandler interface {
    Subject() ports.Subject
    Handle(ctx context.Context, data []byte) error
}
```

Пример хендлера — `prepare_experiment.go`:

```go
// internal/worker/prepare_experiment.go
package worker

import (
    "context"
    "log"
    "github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

type PrepareExperimentHandler struct {
    repo ports.ExperimentRepository
}

func NewPrepareExperimentHandler(repo ports.ExperimentRepository) *PrepareExperimentHandler {
    return &PrepareExperimentHandler{repo: repo}
}

func (h *PrepareExperimentHandler) Subject() ports.Subject {
    return ports.SubjectPrepareExperiment
}

func (h *PrepareExperimentHandler) Handle(ctx context.Context, data []byte) error {
    log.Printf("prepare: received %s", string(data))
    // return error → Nak (повторная доставка)
    return nil
}
```

#### 2. Зарегистрировать в `cmd/worker/main.go`

```go
w.Register(
    worker.NewParseExperimentHandler(experimentRepo, storageObjRepo, fileStorage),
    worker.NewPrepareExperimentHandler(experimentRepo),   // ← новый
)
```

#### 3. Собрать и проверить

```bash
go build ./cmd/worker/
```

Worker автоматически подпишется на `Subject()` нового хендлера.

> Если нужен новый subject — добавь константу в `internal/lidar/ports/message_queue.go`
> и используй её в `Subject()` и при публикации.

### Принципы

- **DDD** — бизнес-логика изолирована от инфраструктуры
- **sqlc** — типобезопасные SQL-запросы без ORM
- **goose** — версионирование схемы БД, автозапуск при старте
- **Graceful degradation** — без SMTP/NATS сервис работает, логирует пропуск
