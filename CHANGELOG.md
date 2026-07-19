# Changelog

## 1.1.0 (2026-07-19)

### Lidar domain models + infrastructure + services

Добавлены доменные модели, HTTP API, NATS, Worker, nginx для LiDAR-сервиса.

#### Added

- **StorageObject** — domain model для реестра файлов в S3/MinIO (ObjectPath value object, metadata, functional options)
- **MinIO/S3 FileStorage** — port + MinIO-адаптер (Upload, UploadBytes, Download, Delete, Exists, GetInfo, PresignedGetURL, CreateBucket)
- **Experiment** — domain model (TimeRange, GeoLocation value objects, soft-delete lifecycle)
- **ExperimentRepository** — port + Postgres-реализация через sqlc (CRUD, UpdateStorageRefs, SoftDelete, Restore)
- **AtmosphereProfile** — domain model (массивы altitude, temperature, pressure с валидацией длины)
- **AtmosphereProfileRepository** — port + Postgres-реализация
- **LicelFile** — domain model (LICEL-файл эксперимента, reuse TimeRange, soft-delete)
- **LicelFileRepository** — port + Postgres-реализация
- **LicelProfile** — domain model (профиль из LICEL-файла, data array, PointAt, soft-delete)
- **LicelProfileRepository** — port + Postgres-реализация, включая FindProfilesWithBackground
- **PairedProfile** — read model для спаренных сигнал+фон с MatchStatus
- **NATS JetStream MessageQueue** — асинхронная публикация/подписка, дедупликация, durable consumer
- **StorageObjectRepository** — port + Postgres-реализация для реестра файлов
- **HTTP сервер (chi)** — роутер, JSON-хелперы, graceful shutdown
- **POST /api/v1/experiments/create** — multipart endpoint: загрузка файлов в MinIO, создание StorageObject, создание Experiment, публикация в NATS
- **POST /api/v1/experiments/task** — JSON-RPC endpoint: приём задачи с алгоритмом (KLETT_FERNALD и др.), публикация в NATS, возврат task_id
- **Worker** — отдельный микросервис: подписка на NATS, диспатч задач по subject (parse, process)
- **ParseExperimentHandler** — обработчик задач (заглушка)
- **Bash-скрипт** — `scripts/create_experiment.sh` для тестирования API с файлами из `testdata/`
- **Nginx** — HTTPS (self-signed), прокси на identity, lidar, MinIO Console (/minio-console/). HTTP → HTTPS redirect.
- **Docker сети** — frontend (nginx, identity, lidar, minio) + backend (postgres, minio, nats, identity, lidar, worker)

#### Changed

- sqlc: добавлены RestoreLicelFile, RestoreLicelProfile, RestoreExperiment, UpdateExperimentStorageRefs
- sqlc: исправлено именование LiceL → Licel, добавлен префикс lidar. в запросах
- **Experiment** — удалён AtmosphereProfileID (профиль создаётся после загрузки данных)
- **Upload** — возвращает `ObjectInfo` (ETag, Size, ContentType)
- **Config** — выделен в отдельный пакет `internal/lidar/config`
- **init-db.sh** — добавлен `GRANT CREATE ON DATABASE` для identity_user и lidar_user
- **docker-compose** — добавлены nginx, minio, nats, worker; убраны host-порты postgres/minio/nats; 2 сети (frontend + backend)

#### Fixed

- NATS consumer name — точки заменены на дефисы (sanitize)
- goose миграции — работают через основное подключение (без DATABASE_URL_MIGRATIONS)
- nginx 413 Request Entity Too Large — добавлен `client_max_body_size 2g`

---

## 1.0.0 (2026-07-17)

### Initial release — Identity microservice

Микросервис аутентификации, верификации и авторизации пользователей.

#### Added

- **Регистрация** — `POST /register` с email и паролем, валидация, bcrypt-хеширование,
  сохранение в БД со статусом `pending`.
- **Верификация по email** — `GET /verify?token=...&email=...` по ссылке из письма.
  Проверка токена, срока действия, совпадения email. Редирект на фронтенд.
- **Авторизация по JWT** — `POST /login` с email и паролем, проверка статуса `active`.
  Выдача JWT (HS256) с `user_id` и `exp` (24ч).
- **SMTP-отправка писем** — поддержка SMTPS (port 465, implicit TLS) и STARTTLS
  (ports 587/25). Graceful degradation при отсутствии конфигурации.
- **Миграции через goose** — автоматический запуск при старте сервиса.
- **Безопасность** — bcrypt для паролей, случайные токены (32 байта → hex),
  срок действия токена 24ч, `Reply-To` для запрета ответов.
  Subject письма encoded по RFC 2047 (без raw UTF-8 в заголовках).
- **sqlc-генерация** — типобезопасные PostgreSQL-запросы.
- **DDD-архитектура** — чёткое разделение domain / application / ports / infrastructure.
- **Docker** — многоступенчатая сборка, docker-compose с postgres и identity.

---

