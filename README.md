# Lidar Platform v3

Платформа для обработки LiDAR-данных.

## Архитектура

```
lidar-platform-v3/
├── cmd/                        # Точки входа микросервисов
│   └── identity/               # Identity-сервис (регистрация, верификация)
├── internal/
│   └── identity/               # Внутренняя реализация identity
│       ├── domain/             # Бизнес-сущности, value objects, ошибки
│       ├── application/        # Use cases (Register, Verify)
│       ├── ports/              # Интерфейсы (UserRepository, MailSender)
│       └── infrastructure/     # Адаптеры (PostgreSQL, SMTP, HTTP)
├── migrations/                 # SQL-миграции
│   └── identity/
├── queries/                    # sqlc-запросы
│   └── identity/
├── pkg/db/                     # Сгенерированный sqlc-код
│   └── identity/
├── docker-compose.yml          # Локальный запуск
├── sqlc.yml                    # Конфигурация sqlc
└── init-db.sh                  # Инициализация БД (схемы, пользователи)
```

## Identity-сервис

### API

#### `POST /register`

Регистрация нового пользователя.

```json
// Request
{ "email": "user@example.com", "password": "secret123" }

// 201 Created
{ "message": "user registered, verification email sent" }

// 409 Conflict
{ "error": "email already registered" }
```

#### `GET /verify?token=...&email=...`

Верификация по ссылке из письма. Редиректит на `{FRONTEND_URL}/verified?status=ok`
при успехе, или `?status=error&reason=...` при ошибке.

#### `POST /verify`

API-верификация для фронтенда.

```json
// Request
{ "token": "...", "email": "user@example.com" }

// 200 OK
{ "message": "email verified successfully" }
```

### Быстрый старт

```bash
# 1. Запустить PostgreSQL
docker compose up -d postgres

# 2. Применить миграцию
docker compose exec -T postgres psql -U user -d main_db -f migrations/identity/001_create_users.sql

# 3. Выдать права identity_user (первый раз)
docker compose exec -T postgres psql -U user -d main_db \
  -c "GRANT ALL ON ALL TABLES IN SCHEMA identity TO identity_user;"

# 4. Запустить сервис
SMTP_FROM=noreply@example.com go run ./cmd/identity/

# 5. Проверить
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'
```

### Конфигурация

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATABASE_URL` | `postgresql://identity_user:pass@localhost:5432/main_db?...` | Подключение к БД |
| `HTTP_ADDR` | `:8080` | Порт HTTP-сервера |
| `SMTP_SERVER` | — | SMTP-сервер (напр. `smtp.yandex.ru:465`) |
| `SMTP_USERNAME` | — | Логин для SMTP |
| `SMTP_PASSWORD` | — | Пароль для SMTP |
| `SMTP_FROM` | — | Адрес в поле Reply-To (noreply) |
| `VERIFY_BASE_URL` | `http://localhost:8080` | Базовый URL для ссылки верификации |
| `FRONTEND_URL` | — | URL для редиректа после верификации |

### Принципы

- **DDD** — бизнес-логика изолирована от инфраструктуры
- **sqlc** — типобезопасные SQL-запросы без ORM
- **Graceful degradation** — без SMTP сервис работает, но логирует пропуск писем
- **Reply-To** — письма приходят от реального ящика, ответы уходят на noreply

### Генерация кода sqlc

```bash
sqlc generate
```
