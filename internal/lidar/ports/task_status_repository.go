package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// TaskStatusRepository provides read/write access to async task status records.
type TaskStatusRepository interface {
	// Create persists a new task record.
	Create(ctx context.Context, record *domain.TaskRecord) error

	// UpdateStatus transitions a task to a new status.
	// If errorMessage is non-empty, it overwrites the stored error message.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errorMessage string) error

	// FindByID looks up a task record by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error)

	// FindAll returns all task records, newest first.
	FindAll(ctx context.Context) ([]domain.TaskRecord, error)
}
