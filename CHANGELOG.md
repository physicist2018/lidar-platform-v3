# Changelog

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
- **Миграции через goose** — автоматический запуск при старте сервиса,
  embedded-миграции в Docker-образе.
- **Безопасность** — bcrypt для паролей, случайные токены (32 байта → hex),
  срок действия токена 24ч, `Reply-To` для запрета ответов.
  Subject письма encoded по RFC 2047 (без raw UTF-8 в заголовках).
- **sqlc-генерация** — типобезопасные PostgreSQL-запросы.
- **DDD-архитектура** — чёткое разделение domain / application / ports / infrastructure.
- **Docker** — многоступенчатая сборка (`cmd/identity/Dockerfile`),
  `docker-compose.yml` с postgres и identity-сервисом.

#### Removed

- **POST /verify** — удалён в пользу `GET /verify` (ссылка из письма).

#### Fixed

- **Docker build context** — изменён с `./cmd/identity` на корень проекта
  для доступа к `go.mod` и миграциям.
- **DB readiness** — retry ping с backoff (до 10 попыток) +
  healthcheck на postgres + `condition: service_healthy`.
- **SMTPUTF8** — заголовок Subject закодирован по RFC 2047 (B-encoding),
  чтобы не требовать SMTPUTF8 при доставке.
- **SMTP настройки** — переключение с порта 465 (SMTPS) на 587 (STARTTLS).

#### Configuration

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `HTTP_ADDR` | Адрес HTTP-сервера (по умолч. `:8080`) |
| `JWT_SECRET` | Секрет для подписи JWT (по умолч. случайный — нестабильный) |
| `MIGRATIONS_DIR` | Путь к директории с goose-миграциями |
| `SMTP_SERVER` | SMTP-сервер (напр. `smtp.mail.ru:587`) |
| `SMTP_USERNAME` | Логин для SMTP-аутентификации |
| `SMTP_PASSWORD` | Пароль для SMTP-аутентификации |
| `SMTP_FROM` | Адрес для Reply-To (напр. `noreply@example.com`) |
| `VERIFY_BASE_URL` | Публичный URL эндпоинта verify для ссылки в письме |
| `FRONTEND_URL` | URL для редиректа после верификации |
