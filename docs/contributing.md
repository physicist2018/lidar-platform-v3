# Как расширять проект

Документация по добавлению новых эндпоинтов, бизнес-логики и воркеров.

---

## 1. Как добавлять HTTP-ручки

### 1.1. Структура

```
HTTP Request → Router → Handler → Use Case → Repository → PostgreSQL
                        (server/)   (application/)    (repository/)
```

Каждый слой отвечает только за своё:
- **Handler** — парсинг HTTP-запроса, валидация формата, формирование ответа
- **Use Case** — бизнес-логика, оркестрация вызовов репозиториев
- **Repository** — работа с БД через sqlc

### 1.2. Пошаговая инструкция

#### Шаг 1. Создать/дополнить Use Case (application/)

Если ручка только читает данные — создайте новый use case.
Если ручка модифицирует данные — дополните существующий или создайте новый.

```go
// internal/lidar/application/list_experiments.go
package application

type ExperimentItem struct {
    ID        uuid.UUID `json:"id"`
    Title     string    `json:"title"`
    StartTime time.Time `json:"experiment_start"`
    EndTime   time.Time `json:"experiment_end"`
}

type ListExperimentsResponse struct {
    Experiments []ExperimentItem `json:"experiments"`
    Count       int              `json:"count"`
}

type ListExperimentsUseCase struct {
    repo ports.ExperimentRepository
}

func NewListExperimentsUseCase(repo ports.ExperimentRepository) *ListExperimentsUseCase {
    return &ListExperimentsUseCase{repo: repo}
}

func (uc *ListExperimentsUseCase) Execute(ctx context.Context, startTime, endTime time.Time) (*ListExperimentsResponse, error) {
    experiments, err := uc.repo.FindByTimeRange(ctx, startTime, endTime)
    if err != nil {
        return nil, err
    }
    items := make([]ExperimentItem, len(experiments))
    for i, exp := range experiments {
        items[i] = mapExperimentToItem(exp)
    }
    return &ListExperimentsResponse{Experiments: items, Count: len(items)}, nil
}
```

#### Шаг 2. Определить интерфейс use case в handler (server/)

```go
// internal/lidar/infrastructure/server/experiment_handler.go
type ListExperimentsUseCase interface {
    Execute(ctx context.Context, startTime, endTime time.Time) (*application.ListExperimentsResponse, error)
}

type ExperimentHandler struct {
    createUC CreateExperimentUseCase
    listUC   ListExperimentsUseCase    // <-- новый use case
}

func NewExperimentHandler(createUC CreateExperimentUseCase, listUC ListExperimentsUseCase) *ExperimentHandler {
    return &ExperimentHandler{createUC: createUC, listUC: listUC}
}
```

#### Шаг 3. Написать метод-обработчик (server/)

```go
func (h *ExperimentHandler) HandleListExperiments(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    
    // Парсинг query-параметров
    var startTime, endTime time.Time
    if st := q.Get("start_time"); st != "" {
        t, err := time.Parse(time.RFC3339, st)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "invalid start_time")
            return
        }
        startTime = t
    }
    
    result, err := h.listUC.Execute(r.Context(), startTime, endTime)
    if err != nil {
        log.Printf("list experiments error: %v", err)
        RespondWithError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    
    RespondWithJSON(w, http.StatusOK, result)
}
```

#### Шаг 4. Зарегистрировать роут (server/router.go)

```go
r.Get("/experiments/list", func(w http.ResponseWriter, r *http.Request) {
    if expHandler == nil {
        RespondWithError(w, http.StatusNotImplemented, "not available")
        return
    }
    expHandler.HandleListExperiments(w, r)
})
```

**Важно:** все `/api/v1/*` роуты защищены JWT-мидлварой. Если роут должен быть публичным — регистрируйте его вне `r.Route("/api/v1", ...)`.

#### Шаг 5. Зарегистрировать use case в DI (cmd/lidar/main.go)

```go
listExpUC := application.NewListExperimentsUseCase(experimentRepo)
expHandler := server.NewExperimentHandler(createExpUC, listExpUC)  // <-- передать новый UC
```

### 1.3. Обработка ошибок

| Ситуация | Код ответа |
|----------|-----------|
| Невалидный входной параметр | `400 Bad Request` |
| Объект не найден | `404 Not Found` (`domain.ErrObjectNotFound`) |
| Ошибка бизнес-логики | `400 Bad Request` с текстом ошибки |
| Внутренняя ошибка | `500 Internal Server Error` |

```go
if errors.Is(err, domain.ErrObjectNotFound) {
    RespondWithError(w, http.StatusNotFound, "experiment not found")
    return
}
```

---

## 2. Как добавлять бизнес-логику

### 2.1. Слои бизнес-логики

```
Use Case (application/) → Domain Entity (domain/) → Repository Port (ports/)
                                                          ↓
                                              Repository Impl (repository/)
```

### 2.2. Когда создавать use case

- **Новая бизнес-операция** → новый файл в `application/`
- **Дополнение существующей операции** → дополнить существующий use case
- **Простое чтение/запись** → можно сразу в handler, но лучше через use case

### 2.3. Работа с репозиториями

**Порт (интерфейс):**
```go
// internal/lidar/ports/experiment_repository.go
type ExperimentRepository interface {
    FindByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]domain.Experiment, error)
    // ...
}
```

**Реализация:**
```go
// internal/lidar/infrastructure/repository/experiment_repo.go
func (r *PostgresExperimentRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]domain.Experiment, error) {
    rows, err := r.q.ListExperimentsByTimeRange(ctx, db.ListExperimentsByTimeRangeParams{
        ExperimentStart: startTime,
        ExperimentEnd:   endTime,
        Limit:  int32(limit),
        Offset: int32(offset),
    })
    if err != nil {
        return nil, err
    }
    experiments := make([]domain.Experiment, len(rows))
    for i, row := range rows {
        experiments[i] = *mapExperiment(row)
    }
    return experiments, nil
}
```

**sqlc-запрос:**
```sql
-- queries/lidar/experiment.sql
-- name: ListExperimentsByTimeRange :many
SELECT * FROM lidar.experiments
WHERE deleted_at IS NULL
  AND experiment_start >= $1
  AND experiment_end <= $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
```

После добавления запроса:
```bash
sqlc generate
```

### 2.4. Когда добавлять sqlc-запрос

1. Новая таблица → создать `.sql` файл в `queries/lidar/`
2. Новый запрос к существующей таблице → дополнить существующий `.sql` файл
3. После изменений → `sqlc generate` (код появится в `pkg/db/lidar/`)

### 2.5. Migrations

```bash
# Создать новую миграцию
touch migrations/lidar/005_my_feature.sql
```

Формат:
```sql
-- +goose Up
ALTER TABLE lidar.experiments ADD COLUMN my_column TEXT;

-- +goose Down
ALTER TABLE lidar.experiments DROP COLUMN my_column;
```

---

## 3. Как добавлять воркеры (async tasks)

### 3.1. Архитектура

```
HTTP → CreateTaskUseCase → NATS Publish (subject=lidar.task.*)
                              ↓
Worker → NATS Subscribe → TaskHandler.Handle()
```

### 3.2. Полная инструкция

#### Шаг 1. Определить Subject (ports/message_queue.go)

```go
const (
    SubjectParseExperiment   Subject = "lidar.task.parse_experiment"
    SubjectPrepareExperiment Subject = "lidar.task.prepare_experiment"
    SubjectProcessExperiment Subject = "lidar.task.process_experiment"
)
```

#### Шаг 2. Создать хендлер воркера (worker/)

```go
// internal/worker/prepare_experiment.go
package worker

type PrepareExperimentHandler struct {
    experimentRepo        ports.ExperimentRepository
    licelFileRepo         ports.LicelFileRepository
    licelProfileRepo      ports.LicelProfileRepository
    preparedMetaRepo      ports.PreparedMetaRepository
    taskStatusRepo        ports.TaskStatusRepository
}

func NewPrepareExperimentHandler(
    experimentRepo ports.ExperimentRepository,
    licelFileRepo ports.LicelFileRepository,
    licelProfileRepo ports.LicelProfileRepository,
    preparedMetaRepo ports.PreparedMetaRepository,
    taskStatusRepo ports.TaskStatusRepository,
) *PrepareExperimentHandler {
    return &PrepareExperimentHandler{
        experimentRepo:   experimentRepo,
        licelFileRepo:    licelFileRepo,
        licelProfileRepo: licelProfileRepo,
        preparedMetaRepo: preparedMetaRepo,
        taskStatusRepo:   taskStatusRepo,
    }
}

func (h *PrepareExperimentHandler) Subject() ports.Subject {
    return ports.SubjectPrepareExperiment
}

func (h *PrepareExperimentHandler) Handle(ctx context.Context, data []byte) error {
    // data — это JSON из NATS (task_id, task_type, payload, created_at)
    var msg struct {
        TaskID  string          `json:"task_id"`
        Payload json.RawMessage `json:"payload"`
    }
    if err := json.Unmarshal(data, &msg); err != nil {
        return fmt.Errorf("parse message: %w", err)
    }

    taskUUID, _ := uuid.Parse(msg.TaskID)
    h.updateTaskStatus(ctx, taskUUID, domain.TaskProcessing, "")

    // Парсинг payload с параметрами задачи
    var payload PreparePayload
    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        h.failTask(ctx, taskUUID, err)
        return fmt.Errorf("parse payload: %w", err)
    }

    // ... бизнес-логика ...

    h.updateTaskStatus(ctx, taskUUID, domain.TaskCompleted, "")
    return nil
}
```

#### Шаг 3. Зарегистрировать хендлер в cmd/worker/main.go

```go
w.Register(
    worker.NewParseExperimentHandler(...),
    worker.NewPrepareExperimentHandler(...),  // <-- новый хендлер
)
```

#### Шаг 4. Опубликовать задачу (из use case)

```go
// Где-то в application/
taskID := uuid.New().String()
payload := map[string]any{
    "experimentID": expID,
    "bgr":          map[string]any{"type": "file"},
    "crop":         map[string]any{"range1": 20000},
}
payloadBytes, _ := json.Marshal(payload)

msg := map[string]any{
    "task_id":    taskID,
    "task_type":  "prepare_for_processing",
    "payload":    payloadBytes,
    "created_at": time.Now().UTC().Format(time.RFC3339),
}
data, _ := json.Marshal(msg)

// Создать запись в task_statuses
taskRecord := domain.NewTaskRecord(
    uuid.MustParse(taskID),
    string(ports.SubjectPrepareExperiment),
    &expID,
    payloadBytes,
)
uc.taskStatusRepo.Create(ctx, &taskRecord)

// Опубликовать в NATS
uc.queue.Publish(ctx, ports.SubjectPrepareExperiment, data, taskID)
```

#### Шаг 5. Получать задачу через существующую ручку

```
POST /api/v1/experiments/task
{
    "task_type": "prepare_for_processing",
    "payload": {
        "experimentID": "...",
        "bgr": {"type": "file"},
        "crop": {"range1": 20000}
    }
}
```

### 3.3. Отслеживание статуса задачи

Все воркеры автоматически получают отслеживание статуса через `taskStatusRepo`:

| Статус | Когда |
|--------|-------|
| `pending` | Задача создана, опубликована в NATS |
| `processing` | Воркер начал обработку |
| `completed` | Воркер успешно завершил |
| `failed` | Воркер вернул ошибку |

Проверить статус:
```
GET /api/v1/tasks/{taskID}
```

### 3.4. Жизненный цикл задачи

```
POST /api/v1/experiments/task
  ↓
CreateTaskUseCase
  ├── INSERT task_statuses (status=pending)
  └── Publish NATS (subject=lidar.task.*)
  ↓
Worker получает сообщение
Handler.Handle()
  ├── UPDATE task_statuses → processing
  ├── бизнес-логика
  ├── UPDATE task_statuses → completed / failed
  └── Ack / Nak
```

### 3.5. Структура NATS-сообщения

```json
{
    "task_id":    "uuid",
    "task_type":  "prepare_for_processing",
    "profile_id": ["..."],            // опционально
    "payload":    { ... },            // параметры задачи
    "created_at": "2026-07-21T10:00:00Z"
}
```

---

## 4. Тестирование

### 4.1. Unit-тесты хендлеров

Паттерн: `httptest.NewRequest` + mock use case:

```go
func TestHandleListExperiments(t *testing.T) {
    mockUC := &mockListExperimentsUC{
        executeFunc: func(ctx context.Context, start, end time.Time) (*application.ListExperimentsResponse, error) {
            return &application.ListExperimentsResponse{...}, nil
        },
    }
    handler := NewExperimentHandler(nil, mockUC)
    
    req := httptest.NewRequest(http.MethodGet, "/api/v1/experiments/list", nil)
    w := httptest.NewRecorder()
    
    handler.HandleListExperiments(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### 4.2. Интеграционные тесты репозиториев

Требуют PostgreSQL (через testcontainers или `TEST_DATABASE_URL`):

```bash
DOCKER_TEST=1 go test ./internal/lidar/infrastructure/repository/...
```

---

## 5. Структура проекта (шпаргалка)

```
cmd/
├── lidar/main.go      — HTTP API server (DI, запуск)
├── worker/main.go     — NATS worker (DI, запуск)
└── identity/main.go   — Authentication service

internal/
├── lidar/
│   ├── application/    — Use cases (бизнес-логика)
│   ├── config/         — Конфигурация (env vars)
│   ├── domain/         — Сущности и value objects
│   ├── infrastructure/
│   │   ├── messaging/  — NATS JetStream реализация
│   │   ├── repository/ — PostgreSQL реализации (sqlc)
│   │   ├── server/     — HTTP handlers, router, middleware
│   │   └── storage/    — MinIO реализация
│   └── ports/          — Интерфейсы репозиториев
└── worker/             — Task handlers

migrations/lidar/       — Goose SQL миграции
queries/lidar/          — sqlc query definitions
pkg/db/lidar/           — sqlc generated code
```
