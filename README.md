# Lidar Platform v3

Платформа для обработки LiDAR-данных.

---

## Архитектура

```
lidar-platform-v3/
├── cmd/
│   ├── identity/                    # Точка входа identity
│   └── lidar/                       # Точка входа lidar (WIP)
├── internal/
│   ├── identity/                    # Бизнес-логика identity
│   │   ├── domain/                  # Entity, Value Objects, errors
│   │   ├── application/             # Use cases (Register, Verify, Login)
│   │   ├── ports/                   # Интерфейсы (repository, mail, token)
│   │   └── infrastructure/          # Адаптеры (PostgreSQL, SMTP, JWT, HTTP)
│   └── lidar/                       # Бизнес-логика lidar
│       ├── domain/                  # Entity, Value Objects, errors
│       ├── application/             # Use cases (WIP)
│       ├── ports/                   # Интерфейсы (repository, file storage)
│       └── infrastructure/          # Адаптеры (PostgreSQL, MinIO/S3)
├── migrations/
│   ├── identity/                    # Goose-миграции identity
│   └── lidar/                       # Goose-миграции lidar
├── queries/
│   ├── identity/                    # sqlc-запросы identity
│   └── lidar/                       # sqlc-запросы lidar
├── pkg/db/
│   ├── identity/                    # sqlc-генерация identity
│   └── lidar/                       # sqlc-генерация lidar
├── docker-compose.yml               # postgres + identity + minio
├── sqlc.yml                         # Конфиг sqlc
└── init-db.sh                       # Инициализация БД (схемы, роли)
```

### Принципы (DDD + SOLID)

| Слой | Отвечает за | Зависит от |
|---|---|---|
| **domain** | Бизнес-правила, entity, value objects | ничего (чистый Go) |
| **application** | Use cases | `ports` (интерфейсы) |
| **ports** | Контракты (репозитории, сервисы) | `domain` |
| **infrastructure** | Реализации (PostgreSQL, SMTP, JWT, MinIO, HTTP) | `ports` |

---

## Identity — микросервис аутентификации

Регистрация, верификация email и JWT-авторизация.

### API

#### `POST /register`

Создаёт пользователя со статусом `pending` и отправляет письмо со ссылкой для верификации.

```bash
xh POST http://localhost:8090/register email==user@example.com password==secret123
```

```http
POST /register
Content-Type: application/json

{ "email": "user@example.com", "password": "secret123" }
```

| Код | Ответ | Когда |
|---|---|---|
| **201** | `{"message":"user registered, verification email sent"}` | Успех |
| **400** | `{"error":"invalid email format"}` | Невалидный email |
| **400** | `{"error":"password must be at least 8 characters"}` | Слабый пароль |
| **409** | `{"error":"email already registered"}` | Email уже занят |

---

#### `GET /verify?token=...&email=...`

Верификация по ссылке из письма. Редиректит браузер на фронтенд.

```http
GET /verify?token=abc123...&email=user@example.com
```

| Статус | Редирект |
|---|---|
| Успех | `{FRONTEND_URL}/verified?status=ok` |
| Невалидный/истёкший токен | `{FRONTEND_URL}/verified?status=error&reason=invalid_token` |
| Уже верифицирован | `{FRONTEND_URL}/verified?status=error&reason=already_verified` |
| Нет параметров | `{FRONTEND_URL}/verified?status=error&reason=missing_params` |

Если `FRONTEND_URL` не задан — возвращает JSON.

---

#### `POST /login`

Проверяет email + пароль, возвращает JWT.

```bash
xh POST http://localhost:8090/login email==user@example.com password==secret123
```

```http
POST /login
Content-Type: application/json

{ "email": "user@example.com", "password": "secret123" }
```

| Код | Ответ | Когда |
|---|---|---|
| **200** | `{"token":"eyJhbGciOiJIUzI1NiIs..."}` | Успех |
| **400** | `{"error":"invalid email format"}` | Невалидный email |
| **401** | `{"error":"invalid email or password"}` | Неверные креды |
| **403** | `{"error":"account not verified"}` | Email не подтверждён |

JWT содержит:

```json
{
  "user_id": "ba723dca-8528-427a-b2ec-8780846d7d3d",
  "exp": 1784349552,
  "iat": 1784263152
}
```

- **Алгоритм**: HS256
- **Срок жизни**: 24 часа
- **Секрет**: задаётся через `JWT_SECRET`

---

### Полный цикл (curl)

```bash
# 1. Регистрация
curl -X POST http://localhost:8090/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'

# 2. Получить токен из БД (только для dev)
TOKEN=$(docker compose exec -T postgres psql -U user -d main_db -t -A \
  -c "SELECT verification_token FROM identity.users WHERE email='user@example.com';")

# 3. Верифицировать
curl -X POST http://localhost:8090/verify \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"user@example.com\",\"token\":\"$TOKEN\"}"

# 4. Авторизация
curl -X POST http://localhost:8090/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'

# 5. Использовать JWT (пример для другого сервиса)
curl http://localhost:8090/protected-resource \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Конфигурация identity

| Переменная | По умолчанию | Обязательная | Описание |
|---|---|---|---|
| `DATABASE_URL` | `postgresql://identity_user:pass@localhost:5432/main_db?...` | — | Подключение к PostgreSQL |
| `HTTP_ADDR` | `:8080` | — | Адрес HTTP-сервера |
| `JWT_SECRET` | случайный (сгорает при рестарте) | **да** | Секрет для подписи JWT |
| `MIGRATIONS_DIR` | `migrations/identity` | — | Путь к goose-миграциям |
| `SMTP_SERVER` | — | для писем | SMTP-сервер (напр. `smtp.mail.ru:587`) |
| `SMTP_USERNAME` | — | для писем | Логин для SMTP |
| `SMTP_PASSWORD` | — | для писем | Пароль для SMTP |
| `SMTP_FROM` | — | для писем | Адрес в поле Reply-To |
| `VERIFY_BASE_URL` | `http://localhost:8080` | — | Базовый URL для ссылки верификации |
| `FRONTEND_URL` | — | для редиректа | URL фронтенда для редиректа |

---

## Lidar — микросервис обработки LiDAR-данных

### Domain model

#### StorageObject

Реестр файлов в S3/MinIO. Хранит метаинформацию: bucket, path, размер, etag, content-type, metadata.

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

LiDAR-эксперимент: временной диапазон, координаты, угол, профиль атмосферы, ссылки на файлы.

```go
type TimeRange struct { Start, End time.Time }
type GeoLocation struct { Latitude, Longitude float32 }
type Experiment struct {
    ID                  uuid.UUID
    Title               string
    ZenithAngle         float32
    TimeRange           TimeRange
    GeoLocation         GeoLocation
    AtmosphereProfileID uuid.UUID
    StorageRefs         ExperimentStorageRefs
    // + soft-delete: DeletedAt *time.Time
}
```

#### AtmosphereProfile

Вертикальный профиль атмосферы: высота, температура, давление.

```go
type AtmosphereProfile struct {
    ID          uuid.UUID
    Altitude    []float64   // км
    Temperature []float64   // °C
    Pressure    []float64   // гПа
    CreatedAt   time.Time
}
```

#### LicelFile

Сырой LICEL-файл: принадлежит эксперименту, содержит метаданные измерения, ссылку на raw storage.

```go
type LicelFile struct {
    ID               uuid.UUID
    ExperimentID     uuid.UUID
    MeasurementRange TimeRange
    IsBackground     bool
    RawStorageID     uuid.UUID
    // + soft-delete
}
```

#### LicelProfile

Профиль из LICEL-файла: данные измерения, параметры устройства, длина волны, поляризация.

```go
type LicelProfile struct {
    ID           uuid.UUID
    LicelFileID  uuid.UUID
    NDataPoints  int32
    Data         []float64
    Wavelength   float32
    Polarization string
    DeviceID     string
    // + soft-delete
}
```

#### PairedProfile

Read model: сигнальный профиль + соответствующий фоновый (по device_id, wavelength, polarization).

```go
type PairedProfile struct {
    Signal      ProfileData
    Background  *ProfileData   // nil если фона нет
    MatchStatus MatchStatus    // OK / NO_BACKGROUND / MISMATCH
}
```

### File storage (MinIO/S3)

```go
type FileStorage interface {
    CreateBucket(ctx, bucket) error
    Upload(ctx, bucket, path, reader, size, contentType) error
    UploadBytes(ctx, bucket, path, data, contentType) error
    Download(ctx, bucket, path, writer) error
    Delete(ctx, bucket, path) error
    Exists(ctx, bucket, path) (bool, error)
    GetInfo(ctx, bucket, path) (*ObjectInfo, error)
    PresignedGetURL(ctx, bucket, path, expiry) (string, error)
}
```

**MinIOFileStorage** — адаптер через `minio-go/v7`.

**Config:**
| Параметр | Описание |
|---|---|
| `Endpoint` | Адрес MinIO/S3 (e.g. `play.min.io`) |
| `AccessKey` | Access key |
| `SecretKey` | Secret key |
| `UseSSL` | Использовать HTTPS |

### Репозитории (PostgreSQL через sqlc)

| Repository | Методы |
|---|---|
| `ExperimentRepository` | Create, FindByID, FindAll, Update, UpdateStorageRefs, SoftDelete, Restore |
| `AtmosphereProfileRepository` | Create, FindByID, FindAll, Delete |
| `LicelFileRepository` | Create, FindByID, FindAllByExperimentID, SoftDelete, Restore |
| `LicelProfileRepository` | Create, FindByID, FindAllByLicelFileID, FindProfilesWithBackground, SoftDelete, Restore |

---

## Docker

```bash
# Собрать и запустить всё
docker compose up -d --build

# Только identity
docker compose up -d --build identity

# Только БД
docker compose up -d postgres
```

---

## Миграции (goose)

Миграции запускаются автоматически при старте сервиса.

```bash
# Статус миграций
goose -dir migrations/identity postgres "$DATABASE_URL" status
goose -dir migrations/lidar postgres "$DATABASE_URL" status

# Накатить все
goose -dir migrations/identity postgres "$DATABASE_URL" up
goose -dir migrations/lidar postgres "$DATABASE_URL" up
```

---

## Разработка

### Генерация sqlc

```bash
sqlc generate
```

После изменений в `migrations/*.sql` или `queries/*.sql`.

### Добавление миграции

```sql
-- migrations/lidar/002_name.sql

-- +goose Up
ALTER TABLE lidar.experiments ADD COLUMN ...

-- +goose Down
ALTER TABLE lidar.experiments DROP COLUMN ...
```
