# Lidar Platform v3

Платформа для обработки LiDAR-данных.

---

## Identity — микросервис аутентификации

Регистрация, верификация email и JWT-авторизация.

### Архитектура

```
lidar-platform-v3/
├── cmd/identity/                    # Точка входа
│   ├── main.go                      # DI, запуск HTTP-сервера
│   └── Dockerfile                   # Многоступенчатая сборка
├── internal/identity/               # Бизнес-логика
│   ├── domain/                      # Entity, Value Objects, errors
│   ├── application/                 # Use cases (Register, Verify, Login)
│   ├── ports/                       # Интерфейсы (repository, mail, token)
│   └── infrastructure/              # Адаптеры
│       ├── repository/              # PostgreSQL (через sqlc)
│       ├── mail/                    # SMTP-отправка
│       ├── auth/                    # JWT (HS256)
│       └── server/                  # HTTP-роутер (chi), хендлеры
├── migrations/identity/             # Goose-миграции
├── queries/identity/                # sqlc-запросы
├── pkg/db/identity/                 # sqlc-генерация
├── docker-compose.yml               # postgres + identity
├── init-db.sh                       # Инициализация БД (схемы, роли)
└── sqlc.yml                         # Конфиг sqlc
```

### Принципы (DDD + SOLID)

| Слой | Отвечает за | Зависит от |
|---|---|---|
| **domain** | Бизнес-правила, entity, value objects | ничего (чистый Go) |
| **application** | Use cases (Register, Verify, Login) | `ports` (интерфейсы) |
| **ports** | Контракты (UserRepository, MailSender, TokenService) | `domain` |
| **infrastructure** | Реализации (PostgreSQL, SMTP, JWT, HTTP) | `ports` |

---

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

---

### Быстрый старт

```bash
# 1. Запустить PostgreSQL
docker compose up -d postgres

# 2. Собрать и запустить identity
docker compose up -d --build identity

# 3. Проверить логи
docker compose logs -f identity

# 4. Протестировать
curl -X POST http://localhost:8090/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'
```

#### Локально (без Docker)

```bash
# 1. Только БД в Docker
docker compose up -d postgres

# 2. Сервис локально
JWT_SECRET="my-secret" go run ./cmd/identity/
```

---

### Конфигурация

| Переменная | По умолчанию | Обязательная | Описание |
|---|---|---|---|
| `DATABASE_URL` | `postgresql://identity_user:pass@localhost:5432/main_db?search_path=identity&sslmode=disable` | — | Подключение к PostgreSQL |
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

### Docker

```bash
# Собрать образ
docker compose build identity

# Запустить
docker compose up -d identity

# Пересобрать и запустить
docker compose up -d --build identity

# Логи
docker compose logs -f identity
```

Образ основан на `alpine:3.19`, содержит только скомпилированный бинарник и миграции.

---

### SMTP-настройки

#### mail.ru / inbox.ru (STARTTLS, порт 587)

| Параметр | Значение |
|---|---|
| `SMTP_SERVER` | `smtp.mail.ru:587` |
| `SMTP_USERNAME` | полный email (user@inbox.ru) |
| `SMTP_PASSWORD` | пароль приложения |
| `SMTP_FROM` | полный email |

#### Яндекс (SMTPS, порт 465)

| Параметр | Значение |
|---|---|
| `SMTP_SERVER` | `smtp.yandex.ru:465` |
| `SMTP_USERNAME` | полный email (user@yandex.ru) |
| `SMTP_PASSWORD` | пароль приложения |
| `SMTP_FROM` | полный email |

> **Важно**: для Mail.ru и Яндекса используйте **пароль приложения**,
> а не обычный пароль от почты.

---

### Миграции (goose)

Миграции запускаются автоматически при старте сервиса.
Управление вручную (если нужно):

```bash
# Установка goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Статус миграций
goose -dir migrations/identity postgres "$DATABASE_URL" status

# Откатить последнюю
goose -dir migrations/identity postgres "$DATABASE_URL" down

# Накатить все
goose -dir migrations/identity postgres "$DATABASE_URL" up
```

---

### Разработка

#### Генерация sqlc

```bash
sqlc generate
```

После изменений в `migrations/identity/*.sql` или `queries/identity/*.sql`.

#### Добавление новой миграции

```sql
-- migrations/identity/002_something.sql

-- +goose Up
ALTER TABLE identity.users ADD COLUMN ...

-- +goose Down
ALTER TABLE identity.users DROP COLUMN ...
```

#### Структура layers (DDD)

| Пакет | Назначение | Пример |
|---|---|---|
| `domain/` | Бизнес-правила, независимые от инфраструктуры | `User`, `Email`, `Password`, `Verify()` |
| `ports/` | Интерфейсы, которые реализует инфраструктура | `UserRepository`, `MailSender`, `TokenService` |
| `application/` | Use cases, оркестрируют домен + порты | `RegisterUseCase`, `LoginUseCase` |
| `infrastructure/` | Адаптеры к внешним системам | PostgreSQL, SMTP, JWT, chi |

### Принципы

- **DDD** — бизнес-логика изолирована от инфраструктуры
- **sqlc** — типобезопасные SQL-запросы без ORM
- **goose** — версионирование схемы БД, автозапуск при старте
- **Graceful degradation** — без SMTP сервис работает, но логирует пропуск писем
- **Reply-To** — письма приходят от реального ящика, ответы уходят на noreply
