package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the lifecycle state of an async task.
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskProcessing TaskStatus = "processing"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
)

// TaskRecord is an entity representing the status of a single async task.
type TaskRecord struct {
	ID           uuid.UUID
	Subject      string
	Status       TaskStatus
	ExperimentID *uuid.UUID // nil for profile-level tasks
	TaskParams   json.RawMessage
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// NewTaskRecord creates a new TaskRecord with pending status.
func NewTaskRecord(
	id uuid.UUID,
	subject string,
	experimentID *uuid.UUID,
	taskParams json.RawMessage,
) TaskRecord {
	now := time.Now()
	return TaskRecord{
		ID:           id,
		Subject:      subject,
		Status:       TaskPending,
		ExperimentID: experimentID,
		TaskParams:   taskParams,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
