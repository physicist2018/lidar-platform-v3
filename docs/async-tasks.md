# Async Tasks — документация

## Общая архитектура

Асинхронные задачи проходят через NATS JetStream. Каждая задача имеет запись в таблице `lidar.task_statuses`, где отслеживается её жизненный цикл.

```
Клиент → HTTP handler → CreateTaskUseCase → INSERT task_status (pending)
                                          → Publish NATS (dedup = task ID)

                                          ↓
                                    Worker (NATS consumer)
                                          → UPDATE task_status → processing
                                          → выполнение работы
                                          → UPDATE task_status → completed / failed
```

## Статусы задач

| Статус      | Описание                                                      |
|-------------|---------------------------------------------------------------|
| `pending`   | Задача создана и опубликована в NATS, но ещё не взята воркером |
| `processing`| Воркер начал обработку задачи                                 |
| `completed` | Задача выполнена успешно                                       |
| `failed`    | Задача завершилась с ошибкой (через NATS уйдёт в retry)       |

## Таблица `lidar.task_statuses`

| Колонка          | Тип          | Описание                                                     |
|------------------|--------------|--------------------------------------------------------------|
| `id`             | UUID PK      | Идентификатор задачи (совпадает с NATS dedupID)              |
| `subject`        | TEXT         | NATS subject (напр. `lidar.task.parse_experiment`)           |
| `status`         | TEXT         | `pending`, `processing`, `completed`, `failed`               |
| `task_params`    | JSONB        | Все параметры расчётов: `profile_ids`, `task_type`, ...      |
| `error_message`  | TEXT?        | Текст ошибки для `failed`                                    |
| `created_at`     | TIMESTAMPTZ  | Когда создана запись                                         |
| `updated_at`     | TIMESTAMPTZ  | Последнее обновление                                         |
| `started_at`     | TIMESTAMPTZ? | Когда перешла в `processing`                                 |
| `finished_at`    | TIMESTAMPTZ? | Когда перешла в `completed` или `failed`                     |

## Как добавить новую асинхронную задачу

### Шаг 1. Определить Subject

В `internal/lidar/ports/message_queue.go` добавить константу:

```go
const (
    SubjectParseExperiment   Subject = "lidar.task.parse_experiment"
    SubjectPrepareExperiment Subject = "lidar.task.prepare_experiment"
    SubjectProcessExperiment Subject = "lidar.task.process_experiment"
    SubjectNewTask           Subject = "lidar.task.new_task"   // <-- новая задача
)
```

### Шаг 2. Создать хендлер воркера

В `internal/worker/` создать файл хендлера, реализующий `TaskHandler`:

```go
package worker

import (
    "context"
    "fmt"
    "log"

    "github.com/google/uuid"
    "github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
    "github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

type NewTaskHandler struct {
    // ... репозитории, нужные для работы
    taskStatusRepo ports.TaskStatusRepository
}

func NewNewTaskHandler(
    // ... репозитории
    taskStatusRepo ports.TaskStatusRepository,
) *NewTaskHandler {
    return &NewTaskHandler{
        // ...
        taskStatusRepo: taskStatusRepo,
    }
}

func (h *NewTaskHandler) Subject() ports.Subject {
    return ports.SubjectNewTask
}

func (h *NewTaskHandler) Handle(ctx context.Context, data []byte) error {
    // 1. Извлечь ID задачи из data
    taskID, err := extractTaskID(data)
    if err != nil {
        return fmt.Errorf("parse task data: %w", err)
    }

    // 2. Отметить как processing
    h.updateTaskStatus(ctx, taskID, domain.TaskProcessing, "")

    // 3. Выполнить работу
    if err := h.doWork(ctx, data); err != nil {
        h.failTask(ctx, taskID, err)
        return err
    }

    // 4. Отметить как completed
    h.updateTaskStatus(ctx, taskID, domain.TaskCompleted, "")
    return nil
}

// Вспомогательные методы (best-effort, логируют ошибки, но не прерывают работу):

func (h *NewTaskHandler) updateTaskStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errMsg string) {
    if h.taskStatusRepo == nil {
        return
    }
    if err := h.taskStatusRepo.UpdateStatus(ctx, id, status, errMsg); err != nil {
        log.Printf("new_task: update task status: %v", err)
    }
}

func (h *NewTaskHandler) failTask(ctx context.Context, id uuid.UUID, err error) {
    if h.taskStatusRepo == nil {
        return
    }
    if err := h.taskStatusRepo.UpdateStatus(ctx, id, domain.TaskFailed, err.Error()); err != nil {
        log.Printf("new_task: update task status: %v", err)
    }
}
```

### Шаг 3. Зарегистрировать хендлер в `cmd/worker/main.go`

```go
w.Register(
    worker.NewParseExperimentHandler(..., taskStatusRepo),
    worker.NewNewTaskHandler(..., taskStatusRepo),  // <-- новый хендлер
)
```

### Шаг 4. Опубликовать задачу через `CreateTaskUseCase`

В `internal/lidar/application/` создать use case, который через `CreateTaskUseCase`
создаёт задачу и публикует её в NATS:

```go
// Создать задачу через универсальный use case
_, err := uc.createTask.Execute(ctx, &TaskRequest{
    Subject:  string(ports.SubjectNewTask),   // NATS subject
    TaskType: "new_task",                     // тип задачи (для task_params)
    TaskID:   taskUUID.String(),              // опционально; если не указать — сгенерируется
    Payload:  taskParams,                     // json.RawMessage с параметрами
})
if err != nil {
    return nil, fmt.Errorf("create task: %w", err)
}
```

`taskParams` — `json.RawMessage`, в который складываются все параметры расчёта:

```go
taskParams, _ := json.Marshal(map[string]any{
    "profile_ids":     req.ProfileIDs,
    "task_type":       req.TaskType,
    "background_type": "mean",
    "background_from": 80000.0,
    "trim_from":       20000.0,
    // любые другие параметры
})
```

`CreateTaskUseCase` сам:
1. Валидирует `subject` и `task_type`
2. Генерирует `taskID`, если не указан
3. Создаёт `TaskRecord` в БД со статусом `pending`
4. Публикует сообщение в NATS с `dedupID = taskID`

### Шаг 5. Прокинуть `CreateTaskUseCase` через DI

В `cmd/lidar/main.go`:

```go
createTaskUC := application.NewCreateTaskUseCase(msgQueue, taskStatusRepo)
createExpUC := application.NewCreateExperimentUseCase(
    fileStorage, storageObjRepo, experimentRepo, createTaskUC,
)
// Новый use case, которому нужно создавать задачи:
newUseCase := application.NewNewUseCase(createTaskUC, ...)
```

В `cmd/worker/main.go`:

```go
taskStatusRepo := repository.NewPostgresTaskStatusRepository(dbConn)

w.Register(
    worker.NewParseExperimentHandler(..., taskStatusRepo),
)
```

## Важные детали

### NATS dedupID = task ID

`dedupID` при публикации в NATS **обязательно** должен совпадать с `id` задачи в `task_statuses`. Это гарантирует идемпотентность —
если задача уже была опубликована, NATS отбросит дубликат. `CreateTaskUseCase` делает это автоматически.

### `task_params` — все параметры расчётов

Все параметры, которые раньше жили в отдельных колонках (`background_type`, `background_from`, `trim_from` и т.п.), теперь
хранятся в `task_params`. Это позволяет:

- Добавлять новые параметры без миграций БД
- Связывать задачу с любым набором профилей (`profile_ids`)
- Хранить произвольную конфигурацию обработки (`payload`)

### Best-effort обновление статуса

Обновления статуса в БД — best-effort. Если `UpdateStatus` вернул ошибку, она логируется, но работа не прерывается.
Это изолирует слой мониторинга от бизнес-логики.

### Универсальное создание задач

`CreateTaskUseCase` — единый способ создания любых асинхронных задач. Все use case'ы должны создавать задачи
только через него, а не напрямую через `TaskStatusRepository` и `MessageQueue`.

`TaskRequest` принимает:
- `subject` (обязательно) — NATS subject, на который публикуется задача
- `task_type` (обязательно) — тип задачи, сохраняется в `task_params`
- `task_id` (опционально) — если нужно явно задать ID задачи
- `payload` (опционально) — произвольные параметры в JSON

### Связь задачи с экспериментом

Поле `experiment_id` было удалено из `task_statuses`. Если задача привязана к эксперименту,
ID эксперимента передаётся через `task_params` или хранится в `payload`.

## Domain-типы

Пакет `internal/lidar/domain/task_status.go`:

```go
type TaskStatus string

const (
    TaskPending    TaskStatus = "pending"
    TaskProcessing TaskStatus = "processing"
    TaskCompleted  TaskStatus = "completed"
    TaskFailed     TaskStatus = "failed"
)

type TaskRecord struct {
    ID           uuid.UUID
    Subject      string
    Status       TaskStatus
    TaskParams   json.RawMessage  // параметры расчётов
    ErrorMessage string
    CreatedAt    time.Time
    UpdatedAt    time.Time
    StartedAt    *time.Time
    FinishedAt   *time.Time
}

func NewTaskRecord(
    id uuid.UUID,
    subject string,
    taskParams json.RawMessage,
) TaskRecord
```

## Port-интерфейс

Пакет `internal/lidar/ports/task_status_repository.go`:

```go
type TaskStatusRepository interface {
    Create(ctx context.Context, record *domain.TaskRecord) error
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errorMessage string) error
    FindByID(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error)
    FindAll(ctx context.Context) ([]domain.TaskRecord, error)
}
```

## Пример: полный lifecycle задачи parse_experiment

```
POST /api/v1/experiments
  ↓
CreateExperimentUseCase.Execute
  ├── upload files to MinIO
  ├── create experiment in DB
  └── createTaskUC.Execute (subject=parse_experiment, taskID=expID)
      ├── INSERT task_statuses (id=expID, status=pending,
      │       subject=parse_experiment)
      └── Publish NATS (subject=parse_experiment, data={...}, dedup=expID)
  ↓
Worker получает сообщение
ParseExperimentHandler.Handle
  ├── UPDATE task_statuses → status=processing, started_at=now()
  ├── download & parse archive
  ├── create LicelFile + LicelProfile records
  ├── update experiment TimeRange
  ├── process background (optional)
  ├── process meteo (optional)
  ├── UPDATE task_statuses → status=completed, finished_at=now()
  └── log "done"
```
