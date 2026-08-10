# Changelog

## [Unreleased]

### Added
- **Пакет `pkg/lidar/molecular`** — расчёт чисто молекулярного (релеевского)
  сигнала обратного рассеяния:
  - **`Compute`** — по сетке дальности `R`, зенитному углу `α`, длине волны `λ`
    и модели атмосферы (высота/температура/давление) выдаёт:
    - `βm(r)` — коэффициент молекулярного обратного рассеяния (м⁻¹·ср⁻¹),
    - `αm(r)` — коэффициент молекулярного ослабления (м⁻¹),
    - `T_m²(r)` — двустороннее молекулярное пропускание,
    - `M(r) = βm(r)·T_m²(r)` — range-corrected молекулярный сигнал
      (подаётся как `Model` в `pkg/smooth/tikhlidar`; калибровочная константа
      не включена — её оценивает потребитель).
  - Физика: плотность по закону идеального газа, показатель преломления по
    формуле Эдлена, сечения Релея (Bucholtz/Bodhaine) с фактором Кинга,
    молекулярное лидарное отношение `S_m = 8π/3`.
  - Высота из дальности `z = r·cos(α)`; интерполяция `T` линейная, `P` — в
    лог-пространстве (барометрический профиль): `exp(lin(ln P))`; выход за
    пределы модели — кламп к краям с предупреждением в лог.
  - Единицы входа как в домене: км/°C/гПа + градусы; выход в СИ.
  - **14 тестов**: опорные значения βm/αm на 532 нм (~1% к литературным),
    масштаб `λ⁻⁴`, интерполяция и кламп, однородное пропускание
    `T² = exp(−2αr)`, геометрия зенитного угла, валидация.

- **Пакет `pkg/lidar/ycf` + `cmd/ycf-molecular`** — HTTP-обработчик Yandex
  Cloud Function для вызова `molecular.Compute`:
  - `POST /` с телом `molecular.Input` (JSON, snake_case) → `molecular.Result`;
    ошибки валидации → 400, не-POST → 405, прочее → 500.
  - CORS-заголовки + обработка OPTIONS для вызовов из браузера.
  - Точка входа `main.Handler`; локальный `main()` — dev-сервер (`HTTP_ADDR`,
    по умолчанию `:8080`).
  - 6 тестов (httptest): успех, snake_case JSON, невалидный JSON, ошибка
    валидации, 405, CORS.
- **json-теги** в `molecular.Input`/`AtmosphereModel`/`Result`
  (`range`, `zenith_angle`, `wavelength`, `atmosphere`, `backscatter`...).

### Added
- **Пакет `pkg/smooth/tikhlidar`** — тихоновское сглаживание лидарных сигналов
  со штрафом на вторую производную и сигмоидной подтяжкой к молекулярному
  профилю:
  - **`SmoothProfile`** — сглаживание единичного профиля: на выходе сглаженный
    range-corrected сигнал `Ŝ(r)`; в дальней зоне (выше `Href`), где аэрозоля
    нет, сигнал заменяется молекулярным `C·M(r)`.
  - **`SmoothBatch`** — пачка профилей на общей сетке с дополнительным
    сглаживанием по времени (`ω`).
  - Параметры: `Href` (высота подтяжки, центр логистической сигмоиды),
    `L` (ширина переходной зоны), `q` (сила привязки), `λ` (сглаживание по
    дистанции), `ω` (сглаживание по времени), `[r0, r1]` (диапазон привязки
    для расчёта константы калибровки `C` — медиана `S/M`).
  - **Профиль весов `Weights`** — попрофильный множитель первого (fit) члена
    функционала: `Σ (1−w)·u·(S−Ŝ)²`; `uᵢ = 0` отключает fit в точке
    (например, для отбраковки шумных бинов).
  - **asinh-хелперы `SmoothProfileAsinh` / `SmoothBatchAsinh`** — сглаживание
    в лог-подобном масштабе: сигнал нормируется константой калибровки
    `C = median(S/M)` из `[r0,r1]` (в исходном домене), затем `asinh(x/eps)`
    применяется к сигналу и модели, веса = `|S_t|`, сглаживание, обратное
    преобразование `Ŝ = C·eps·sinh(Ŝ_t)`. Работает с большим динамическим
    диапазоном и отрицательными значениями сигнала.
  - **Отрицательные сигналы разрешены** — снято ограничение `Signal ≥ 0`
    (например, после вычета фона); `Model > 0` остаётся обязательным.
  - Сигналы **не скорректированы на `T²`**: дальняя зона заменяется модельным
    молекулярным сигналом, `C` поглощает неизвестное пропускание.
  - Решатели: gonum `BandCholesky` (1D, пятидиагональная СПД-система);
    матрично-свободный CG на gonum `blas64` (2D, блочная структура) — за
    подменяемым интерфейсом (замена на свои решатели в будущем).
  - Зависимость: `gonum.org/v1/gonum v0.17.0`.
  - **30 тестов**: сигмоида, калибровка, оператор второй производной
    (равномерная/неравномерная сетка), banded Cholesky и CG против плотного
    эталона, синтетическое восстановление профиля, эффекты `λ`/`q`/весов,
    пачка (`ω=0` ≡ попрофильно, `ω>0` снижает временную дисперсию),
    asinh-хелперы (round-trip, дальняя зона, эквивалентность ручному вызову,
    отрицательные сигналы, валидация), валидация.

### Added
- **Refresh token логика в identity** — двухтокенная модель авторизации:
  - **Access token** — JWT с TTL по умолчанию **1 час** (env `ACCESS_TOKEN_TTL`).
  - **Refresh token** — opaque-токен с TTL по умолчанию **30 дней** (env `REFRESH_TOKEN_TTL`),
    хранится **хэшированным (SHA-256)** в новой таблице `identity.refresh_tokens`
    (миграция `002_create_refresh_tokens.sql`).
  - **`POST /refresh`** — обмен refresh-токена на новую пару; каждый refresh **ротирует**
    токен (старый отзывается, выдаётся новый).
  - **Детекция повторного использования** — использование уже отозванного refresh-токена
    расценивается как кража: отзываются **все** refresh-токены пользователя.
  - **`POST /logout`** — отзыв refresh-токена (идемпотентный).
  - **`POST /login`** теперь возвращает `{ token, refresh_token, expires_in }`.
  - При выдаче refresh-токена сохраняются `user_agent` и `ip` для аудита.
  - Фронтенд: silent refresh — при `401` выполняется single-flight `POST /refresh`
    с повторной попыткой запроса; только при неудаче — очистка сессии и редирект на логин.
  - `scripts/auth.sh`: новые команды `refresh` и `refresh-token`; при логине
    сохраняется и refresh-токен.
- **Тесты identity** (21 тест): домен refresh-токенов (генерация, хэш, expiry, revoke),
  `RefreshUseCase` (успех с ротацией, неизвестный/отозванный/истёкший токен,
  disabled-пользователь, orphan-токен), `LogoutUseCase` (успех, идемпотентность),
  `RefreshHandler` (POST /refresh: успех, ошибки декодирования, отсутствие токена,
  неизвестный/отозванный/истёкший токен, `clientIP`).

### Changed
- **`BackgroundMean` в prepare-воркере** — среднее фона теперь вычитается из
  медианно-сглаженного сигнала (`smoothed`), а не из исходного; на выходе —
  сглаженный сигнал с вычтенным фоном. Тесты обновлены.
- **`JWTTokenService`** — конструктор принимает TTL access-токена
  (`NewJWTTokenService(secret, accessTTL)`); дефолт 1 час вместо 24 часов.
- **`LoginUseCase`** — выдаёт refresh-токен; `Execute` принимает `userAgent` и `ip`.
- **`UserRepository`** — добавлен метод `FindByID`.

### Removed
- **`LoginResult`** — заменён на общий `TokenPair` (`token`, `refresh_token`, `expires_in`).

### Fixed
- _(нет)_

### Added
- **Фронтенд SPA** — одностраничное приложение на Vite + vanilla JS:
  - `/login` — форма входа (JWT)
  - `/register` — регистрация с валидацией
  - `/verified` — результат верификации email
  - `/experiments` — список экспериментов с фильтром по датам
  - `/experiments/:id` — детали эксперимента, форма создания задачи,
    отслеживание статусов задач (автообновление каждые 5с)
  - `/upload` — multipart загрузка эксперимента (ZIP + LICEL + CSV опционально)
- **Nginx раздаёт фронтенд** — `location /` отдаёт SPA из `/usr/share/nginx/html`,
  API-запросы проксируются через nginx к соответствующим сервисам.
- **CreateExperimentResponse.parse_task_id** — ответ создания эксперимента теперь
  содержит ID автоматически созданной задачи `lidar.task.parse_experiment`.
- **Блокировка создания задач до завершения парсинга** — на странице деталей
  эксперимента форма создания задачи недоступна, пока `lidar.task.parse_experiment`
  не перейдёт в статус `completed` или `failed`.
- **Prepare experiment worker** — новый хендлер `PrepareExperimentHandler` для
  `lidar.task.prepare_experiment`. Делает поканальный вычет фона и обрезку профилей
  по высоте:
  - **Вычет фона**: `background_type: "file"` — поканальное вычитание фонового профиля;
    `background_type: "mean"` — вычет среднего арифметического хвоста профиля
    начиная с расстояния `background_from` (в метрах).
  - **Обрезка**: профили обрезаются до расстояния `trim_from` (в метрах).
  - Результат сохраняется в таблицы `lidar.prepared_meta` + `lidar.prepared_profiles`.
  - Payload задачи: `{"experiment_id", "background_type", "background_from", "trim_from"}`.
- **Domain models** — `PreparedMeta` и `PreparedProfile` для обработки профилей.
- **sqlc queries** — `CreatePreparedMeta`, `GetPreparedMetaByExperimentID`,
  `CreatePreparedProfile`, `ListPreparedProfilesByMetaID`.
- **Port interfaces** — `PreparedMetaRepository`, `PreparedProfileRepository`.
- **Postgres repositories** — реализация prepared-репозиториев.
- **Tests** (6 тестов) — `processProfile` core logic.
- **`subject` in `POST /api/v1/experiments/task`** — subject is now an explicit
  required field `"subject"` in the JSON request.
- **`task_id` in `POST /api/v1/experiments/task`** — optional field; auto-generated
  if empty.
- **Universal task creation** — `CreateExperimentUseCase` now creates tasks through
  `CreateTaskUseCase` instead of directly.
- **Prepared profiles API** — три новых эндпоинта для доступа к обработанным профилям:
  - `GET /api/v1/prepared-profiles/experiments` — список экспериментов с prepared данными
  - `GET /api/v1/prepared-profiles/filters?experiment_id=X` — доступные wavelength,
    polarization, device_id для каскадных селектов
  - `GET /api/v1/prepared-profiles?experiment_id=X&wavelength=...` — данные профилей с `data`
- **Страница `/prepared` во фронтенде** — просмотр prepared профилей с интерактивными
  графиками Plotly (heatmap, profile overlay, profile average). Каскадные селекты:
  эксперимент → длина волны → поляризация → device ID. Plotly установлен через npm.
- **Heatmap — ось X с реальным временем** — вместо номера профиля на оси X
  отображается реальное время измерения (`measurement_start` из LICEL-файла).
  SQL-запрос теперь JOINит таблицу `licelfiles`.
- **Преобразования сигнала на графиках** — новый селект "Преобразование":
  Raw, P×r², log₁₀(P×r²), log₁₀(P). Применяется ко всем типам графиков.
- **Документация по расширению фронтенда** — `docs/frontend.md`: как добавить
  страницу, API-функцию, маршрут, стили.
- **Название эксперимента в выпадающем списке** — в селекте экспериментов на
  странице `/prepared` отображается название и временной диапазон эксперимента
  вместо ID. `ListPreparedExperiments` JOINит таблицу `experiments`.
- **Медианный фильтр в BackgroundMean** — перед расчётом среднего хвоста профиль
  обрабатывается медианным фильтром (окно 3) для удаления шумовых выбросов.
- **Удаление задач** — новый эндпоинт `DELETE /api/v1/tasks/{taskID}`. Удаляет
  задачу и связанные результаты (`prepared_meta` + `prepared_profiles` для
  prepare-задач). Кнопка "Удалить" рядом с завершёнными задачами во фронтенде.

### Changed
- **`CreateTaskUseCase`** — теперь универсальный use case: `subject` и опциональный
  `task_id` передаются явно в `TaskRequest`. Идемпотентное поведение: если задача с
  таким `TaskID` уже существует, возвращает её статус без повторной публикации.
- **`NewCreateExperimentUseCase`** — сигнатура изменена: принимает `*CreateTaskUseCase`
  вместо `queue, taskStatusRepo`.
- **`NewTaskRecord`** — сигнатура изменена: убран параметр `experimentID`, осталось
  три параметра: `id, subject, taskParams`.
- **`ParseExperimentHandler`** — обновлён для работы с новым форматом сообщений
  от `CreateTaskUseCase` (JSON вместо raw UUID).
- **NATS dedup window**: 2 мин → **1 час** — предотвращает повторную обработку
  задачи, если один и тот же `dedupID` опубликован снова в течение часа.
- **NATS AckWait**: 30 с → **30 мин** — чтобы крупные эксперименты (672 профиля)
  успевали обработаться до редиливери NATS.
- **PrepareExperimentHandler** — добавлены логи: количество найденных пар сигнал+фон
  и прогресс обработки каждые 100 профилей.

### Removed
- **`experiment_id` from `lidar.task_statuses`** — column removed (migration
  `005_remove_experiment_id_from_task_statuses.sql`). Experiment association can
  be passed via `task_params` if needed.
- **`FindByExperimentID` from `TaskStatusRepository`** — method removed since it
  was unused in production code.
- **`ExperimentID` field from `GetTaskStatusResponse`** — no longer returned in API.

## [Unreleased] (previous)

### Added
- **GET /api/v1/experiments/list** — новый эндпоинт для получения списка экспериментов
  с фильтрацией по временному диапазону.
  - `GET /api/v1/experiments/list` — все эксперименты
  - `GET /api/v1/experiments/list?start_time=2026-01-01T00:00:00Z&end_time=2026-12-31T23:59:59Z` — по диапазону
  - `start_time` и `end_time` опциональны (RFC3339), пагинация через `limit`/`offset`
  - `ListExperimentsUseCase`, хендлер `HandleListExperiments`, sqlc-запрос `ListExperimentsByTimeRange`
- **Adminer** — лёгкий веб-клиент для управления PostgreSQL, доступен через nginx
  по адресу `https://localhost/adminer/`. Заменил pgAdmin (проще, быстрее, не хранит
  конфигурации на диске).
- **JWT authentication**: all `/api/v1/*` endpoints in the lidar service now require a
  valid Bearer JWT issued by the identity service.
  - `JWTAuthMiddleware` — chi middleware that validates HS256 JWTs against `JWT_SECRET`.
  - If `JWT_SECRET` is not set, an ephemeral random key is generated (dev mode).
  - User ID from token is injected into request context (`server.UserIDFromContext`).
  - `/health` remains public (no authentication required).
- **Task status tracking**: new `lidar.task_statuses` table tracks lifecycle of all async
  tasks (`pending` → `processing` → `completed` / `failed`).
  - Domain entity `TaskRecord`, port `TaskStatusRepository`, Postgres implementation.
  - Migration `004_create_task_statuses.sql` with indexes on `experiment_id` and `status`.
  - Integration in `CreateExperimentUseCase` and `CreateTaskUseCase` — creates a `pending`
    record before publishing to NATS.
  - Integration in `ParseExperimentHandler` — updates status to `processing` on start,
    `completed` on success, `failed` on error.
  - All calculation parameters stored in `task_params` JSONB column.
- **GET /api/v1/tasks/{taskID}**: new API endpoint to query task status.
  - `GetTaskStatusUseCase`, handler `HandleGetTaskStatus`, router registration.
  - Returns `200` with full status, `404` for unknown task, `400` for invalid UUID.
- **Tests** (65 тестов, все проходят):
  - JWT middleware (11 тестов): валидный токен, отсутствующий заголовок, пустой Bearer,
    неверный формат, невалидный токен, неверный секрет, истекший токен, пустой secret fallback,
    извлечение user_id из контекста, extractBearerToken (6 кейсов).
  - Task handler (7 тестов): HandleCreateTask (invalid JSON, пустой ProfileID, пустой TaskType),
    HandleGetTaskStatus (200, invalid UUID, 404 not found, missing param).
  - Router (4 теста): /health без auth, /api/v1/* без auth (3 endpoints),
    с валидным токеном (501 = auth passed), с истекшим токеном (401).
  - Domain (17 тестов): TaskRecord (3), Experiment (2), TimeRange (3), GeoLocation (3),
    SoftDelete/Restore, LicelFile (3), LicelProfile (2), AtmosphereProfile (2),
    ObjectPath (2), StorageObject.
  - Use cases (6 тестов): CreateTask — пустой ProfileID, пустой TaskType, успех (с проверкой
    создания TaskRecord и публикации в NATS). GetTaskStatus — found, not found, with params.
  - Repository (8 тестов, с реальным PostgreSQL через testcontainers): Create + FindByID,
    Create с ExperimentID, дубликат ID, UpdateStatus (processing → completed),
    UpdateStatus (failed с error_message), FindByID not found, FindByExperimentID, FindAll.
- **Documentation**: `docs/async-tasks.md` — guide for adding new async tasks with status
  tracking.
- **Test dependencies**: `github.com/testcontainers/testcontainers-go` (v0.43.0) для
  интеграционных тестов с реальной PostgreSQL.

### Changed
- **`ParseExperimentHandler.processArchive`**: now returns `(domain.TimeRange, error)` to
  propagate the global time range from all LICEL files in the archive.
- **`ParseExperimentHandler.Handle`**: updates experiment `TimeRange` after archive processing
  using the earliest start and latest stop across all files.
- **Constructor signatures**:
  - `NewCreateExperimentUseCase` — added `taskStatusRepo` parameter.
  - `NewCreateTaskUseCase` — added `taskStatusRepo` parameter.
  - `NewParseExperimentHandler` — added `taskStatusRepo` parameter.
  - `NewTaskHandler` — added `getTaskStatusUC` parameter.
- **`.gitignore`**: added `/worker` binary.
- **`docker-compose.yml`**: pgAdmin заменён на Adminer (`adminer:latest`) —
  более лёгкий и простой клиент для PostgreSQL, доступен через `/adminer/`.
- **`nginx/nginx.conf`**: location `/pgadmin/` заменён на `/adminer/` (proxy → adminer:8080).

### Fixed
- **`scripts/auth.sh`**: default `IDENTITY_URL` port corrected from `:8080` to `:8090`
  to match `run_identity.sh` configuration.
- **`docker-compose.yml`**: added missing `JWT_SECRET` environment variable to the `lidar`
  service — both identity and lidar now share the same secret.

### Removed
- Debug logging from `internal/identity/infrastructure/auth/jwt.go` and
  `internal/lidar/infrastructure/server/auth.go` after root cause analysis.
- `check_hmac/` — temporary test directory.
- `pgadmin/` — заменён на Adminer.
