package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresTaskStatusRepository implements ports.TaskStatusRepository backed by sqlc.
type PostgresTaskStatusRepository struct {
	q *db.Queries
}

// NewPostgresTaskStatusRepository creates a new PostgresTaskStatusRepository.
func NewPostgresTaskStatusRepository(dbtx db.DBTX) *PostgresTaskStatusRepository {
	return &PostgresTaskStatusRepository{q: db.New(dbtx)}
}

// Create persists a new task record.
func (r *PostgresTaskStatusRepository) Create(ctx context.Context, record *domain.TaskRecord) error {
	taskParams := record.TaskParams
	if taskParams == nil {
		taskParams = json.RawMessage("{}")
	}
	_, err := r.q.CreateTaskStatus(ctx, db.CreateTaskStatusParams{
		ID:         record.ID,
		Subject:    record.Subject,
		Status:     string(record.Status),
		TaskParams: taskParams,
	})
	return err
}

// UpdateStatus transitions a task to a new status.
func (r *PostgresTaskStatusRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.TaskStatus,
	errorMessage string,
) error {
	return r.q.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		ID:           id,
		Status:       string(status),
		ErrorMessage: toNullString(errorMessage),
	})
}

// FindByID looks up a task record by ID.
func (r *PostgresTaskStatusRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
	u, err := r.q.GetTaskStatusByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapTaskStatus(u), nil
}

// FindAll returns all task records, newest first.
func (r *PostgresTaskStatusRepository) FindAll(ctx context.Context) ([]domain.TaskRecord, error) {
	rows, err := r.q.ListTaskStatuses(ctx)
	if err != nil {
		return nil, err
	}
	return mapTaskStatuses(rows), nil
}

// Delete removes a task record permanently.
func (r *PostgresTaskStatusRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTaskStatus(ctx, id)
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapTaskStatus(u db.LidarTaskStatus) *domain.TaskRecord {
	rec := &domain.TaskRecord{
		ID:         u.ID,
		Subject:    u.Subject,
		Status:     domain.TaskStatus(u.Status),
		TaskParams: u.TaskParams,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
	if u.ErrorMessage.Valid {
		rec.ErrorMessage = u.ErrorMessage.String
	}
	if u.StartedAt.Valid {
		rec.StartedAt = &u.StartedAt.Time
	}
	if u.FinishedAt.Valid {
		rec.FinishedAt = &u.FinishedAt.Time
	}
	return rec
}

func mapTaskStatuses(rows []db.LidarTaskStatus) []domain.TaskRecord {
	records := make([]domain.TaskRecord, len(rows))
	for i, row := range rows {
		records[i] = *mapTaskStatus(row)
	}
	return records
}
