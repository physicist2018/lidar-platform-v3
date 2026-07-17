# Changelog

## 1.0.0 (2026-07-17)

### Initial release — Identity microservice

Микросервис аутентификации и верификации пользователей.

#### Added

- **Регистрация** — `POST /register` с email и паролем, валидация, bcrypt-хеширование,
  сохранение в БД со статусом `pending`.
- **Верификация по email** — `GET /verify?token=...&email=...` по ссылке из письма.
  Проверка токена, срока действия, совпадения email. Редирект на фронтенд.
- **API-верификация** — `POST /verify` с JSON-телом для обратной совместимости.
- **SMTP-отправка писем** — поддержка SMTPS (port 465, implicit TLS) и STARTTLS
  (ports 587/25). Graceful degradation при отсутствии конфигурации.
- **Безопасность** — bcrypt для паролей, случайные токены (32 байта → hex),
  срок действия токена 24ч, `Reply-To: noreply@...` для запрета ответов.
- **sqlc-генерация** — типобезопасные PostgreSQL-запросы.
- **DDD-архитектура** — чёткое разделение domain / application / ports / infrastructure.
- **Docker** — многоступенчатая сборка (`cmd/identity/Dockerfile`),
  `docker-compose.yml` с postgres и identity-сервисом.

#### Configuration

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `HTTP_ADDR` | Адрес HTTP-сервера (по умолч. `:8080`) |
| `SMTP_SERVER` | SMTP-сервер (напр. `smtp.yandex.ru:465`) |
| `SMTP_USERNAME` | Логин для SMTP-аутентификации |
| `SMTP_PASSWORD` | Пароль для SMTP-аутентификации |
| `SMTP_FROM` | Адрес для Reply-To (напр. `noreply@example.com`) |
| `VERIFY_BASE_URL` | Публичный URL эндпоинта verify для ссылки в письме |
| `FRONTEND_URL` | URL для редиректа после верификации |
