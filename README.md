# lidar-platform-v3

Платформа обработки лидарных данных. Принимает сырые архивы LICEL, разбирает
профили, выполняет коррекцию фона и готовит данные для атмосферного анализа.

## Архитектура

```
Frontend SPA (Vite + vanilla JS)  — https://localhost / localhost:5173
  ├── /login, /register          — авторизация через identity-сервис
  ├── /experiments               — список / детали / загрузка
  └── /experiments/:id           — управление задачами

Nginx (SSL-терминация + маршрутизация)
  ├── / → frontend SPA
  ├── /login, /register, /verify, /refresh, /logout → identity:8090
  ├── /api/*                     → lidar:8091
  └── /adminer/*                 → adminer:8080

HTTP API (cmd/lidar)
  ├── POST   /api/v1/experiments/create           — загрузка файлов эксперимента
  ├── POST   /api/v1/experiments/task             — создание задачи обработки
  ├── GET    /api/v1/experiments/list              — список экспериментов (с фильтром по времени)
  ├── GET    /api/v1/tasks/{taskID}               — статус асинхронной задачи
  ├── DELETE /api/v1/tasks/{taskID}               — удаление задачи и результатов
  ├── GET    /api/v1/prepared-profiles             — запрос обработанных профилей
  ├── GET    /api/v1/prepared-profiles/experiments — список экспериментов с prepared-данными
  ├── GET    /api/v1/prepared-profiles/filters     — доступные wavelength/polarization/device_id
  └── GET    /health                              — проверка здоровья (без авторизации)

NATS JetStream — очередь асинхронных задач
  ├── lidar.task.parse_experiment    — разбор загруженного архива
  ├── lidar.task.prepare_experiment   — коррекция фона
  └── lidar.task.process_experiment  — обработка сигнала (планируется)

Worker (cmd/worker)
  ├── ParseExperimentHandler   — скачивает архив, создаёт записи LicelFile + LicelProfile
  └── PrepareExperimentHandler — вычет фона и обрезка профилей

PostgreSQL — хранилище доменных данных
  ├── experiments, licelfiles, licel_profiles, task_statuses
  └── prepared_meta, prepared_profiles — обработанные данные
```

## Аутентификация

Все эндпоинты `/api/v1/*` требуют Bearer JWT, выпущенный identity-сервисом.

1. **Регистрация** через identity-сервис: `POST /register`
2. **Верификация** аккаунта (ссылка из письма или вручную через psql)
3. **Вход** для получения пары токенов: `POST /login`
4. **Использование access-токена** в запросах к lidar API:

```bash
curl -H 'Authorization: Bearer <TOKEN>' http://localhost:8091/api/v1/tasks/<taskID>
```

### Пара токенов (access + refresh)

`POST /login` и `POST /refresh` возвращают пару токенов:

```json
{ "token": "<jwt>", "refresh_token": "<opaque>", "expires_in": 3600 }
```

- **Access token** — короткоживущий JWT (по умолчанию **1 час**, env `ACCESS_TOKEN_TTL`);
  передаётся как `Authorization: Bearer` и валидируется lidar-сервисом.
- **Refresh token** — долгоживущий opaque-токен (по умолчанию **30 дней**, env
  `REFRESH_TOKEN_TTL`); хранится **в хэшированном виде (SHA-256)** в таблице
  `identity.refresh_tokens` и используется только для получения новой пары. Каждый
  refresh **ротирует** токен (старый отзывается, выдаётся новый). Повторное
  использование уже отозванного токена расценивается как кража и отзывает **все**
  refresh-токены пользователя.

Эндпоинты:

| Метод | Путь | Тело | Описание |
|-------|------|------|----------|
| `POST` | `/refresh` | `{ "refresh_token": "..." }` | Ротация refresh-токена, возврат новой пары |
| `POST` | `/logout` | `{ "refresh_token": "..." }` | Отзыв refresh-токена (идемпотентный) |

Фронтенд обновляет сессию незаметно: при `401` выполняется single-flight `POST /refresh`
и запрос повторяется один раз; только при неудачном refresh сессия очищается и
происходит редирект на страницу входа.

Скрипт `scripts/auth.sh` автоматизирует регистрацию, вход и refresh:

```bash
# Регистрация + вход
IDENTITY_URL=http://localhost:8090 ./scripts/auth.sh full user@example.com mypassword

# Показать сохранённый access-токен
./scripts/auth.sh token

# Обновить пару токенов
./scripts/auth.sh refresh
```

Identity и lidar-сервисы должны использовать один и тот же `JWT_SECRET`.
В docker-compose это настроено автоматически.

## Быстрый старт

### Docker Compose (полный стек)

```bash
# Сборка фронтенда
cd frontend && npm run build && cd ..

# Запуск всех сервисов
docker compose up -d --build

# Открыть https://localhost в браузере
```

### Режим разработки (hot reload)

```bash
# Терминал 1 — backend-сервисы
docker compose up -d identity lidar worker nats minio postgres

# Терминал 2 — dev-сервер фронтенда
cd frontend && npm run dev
# → http://localhost:5173 (API проксируется через nginx на https://localhost)
```

### Локальная разработка (без Docker)

```bash
# Терминал 1 — identity-сервис
bash run_identity.sh

# Терминал 2 — lidar API + worker
bash run_lidar.sh
bash run_worker.sh

# Терминал 3 — фронтенд
cd frontend && npm run dev

# Терминал 4 — регистрация и вход
./scripts/auth.sh full user@example.com mypassword
```

## Структура проекта

```
cmd/
├── lidar/          HTTP API сервер
├── worker/         NATS consumer / worker задач
└── identity/       Сервис аутентификации

internal/
├── lidar/
│   ├── application/    Use cases
│   ├── config/         Конфигурация (JWT_SECRET, MinIO, NATS)
│   ├── domain/         Доменные сущности
│   ├── infrastructure/ 
│   │   ├── messaging/  Реализация NATS
│   │   ├── repository/ Реализации Postgres (sqlc)
│   │   ├── server/     HTTP-хендлеры, роутер, JWT middleware
│   │   └── storage/    Реализация MinIO
│   └── ports/          Интерфейсы портов (репозитории, очередь сообщений, файловое хранилище)
└── worker/            Интерфейсы хендлеров задач + жизненный цикл worker'а

frontend/
├── index.html          Точка входа SPA
├── vite.config.js      Proxy + конфиг сборки
└── src/
    ├── api.js          HTTP-клиент (JWT, silent refresh, 401)
    ├── router.js       Hash-based SPA роутер
    ├── store.js        Состояние приложения
    ├── styles.css      Полный UI-кит (карточки, таблицы, формы, бейджи)
    └── pages/          Login, register, verified, experiments, upload, prepared

scripts/
└── auth.sh            Вспомогательный скрипт авторизации (register, login, refresh, token)

migrations/lidar/      Goose SQL-миграции
queries/lidar/         Определения sqlc-запросов
pkg/db/lidar/          Сгенерированный sqlc Go-код
pkg/smooth/tikhlidar/  Тихоновское сглаживание лидарных сигналов
                       (сигмоидная подтяжка к молекулярному профилю)

docs/
├── async-tasks.md     Гайд по добавлению асинхронных задач со статусами
├── frontend.md        Гайд по расширению фронтенда (страницы, API, роутинг)
```

## Тесты

Проект содержит **97 тестов** (Go backend), покрывающих ключевые компоненты.

### Как запустить

```bash
# Unit-тесты backend
go test ./internal/...

# Проверка сборки фронтенда
cd frontend && npm run build
```

### Тестовые файлы

| Файл | Тестов | Что тестирует |
|------|--------|---------------|
| `identity/domain/refresh_token_test.go` | 5 | Генерация refresh-токенов, SHA-256 хэш, срок действия, revoke |
| `identity/application/refresh_test.go` | 6 | RefreshUseCase (ротация, неизвестный/отозванный/истёкший токен, disabled-пользователь, orphan) |
| `identity/application/logout_test.go` | 3 | LogoutUseCase (успех, идемпотентность) |
| `identity/server/refresh_handler_test.go` | 7 | Хендлер POST /refresh, clientIP |
| `server/auth_test.go` | 11 | JWT middleware |
| `server/task_handler_test.go` | 7 | HandleCreateTask, HandleGetTaskStatus |
| `server/router_test.go` | 4 | Health + проверки авторизации |
| `server/experiment_handler_test.go` | 10 | Хендлер создания эксперимента |
| `domain/domain_test.go` | 22 | Все доменные сущности |
| `application/create_task_test.go` | 4 | CreateTaskUseCase (empty subject, empty task_type, success, idempotent) |
| `application/get_task_status_test.go` | 3 | GetTaskStatusUseCase |
| `repository/task_status_repo_test.go` | 8 | TaskStatusRepository (реальный PG через testcontainers) |
| `worker/prepare_experiment_test.go` | 7 | Ключевая логика processProfile |

## Зависимости

- **Go** 1.22+
- **PostgreSQL** — хранилище доменных данных
- **NATS JetStream** — очередь асинхронных задач
- **MinIO** (S3-совместимый) — хранилище сырых файлов
- **sqlc** — типобезопасные SQL-запросы
- **Goose** — миграции БД
- **Node.js 20+** — сборка фронтенда
- **gonum** — численные решатели (banded Cholesky, blas64) в `pkg/smooth/tikhlidar`
