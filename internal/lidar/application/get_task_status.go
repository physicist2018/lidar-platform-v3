package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// GetTaskStatusResponse is the response for a task status query.
type GetTaskStatusResponse struct {
	ID           uuid.UUID       `json:"id"`
	Subject      string          `json:"subject"`
	Status       string          `json:"status"`
	TaskParams   json.RawMessage `json:"task_params"`
	ErrorMessage string          `json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	StartedAt    *time.Time      `json:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at"`
}

// GetTaskStatusUseCase retrieves the status of an async task.
type GetTaskStatusUseCase struct {
	taskStatusRepo ports.TaskStatusRepository
}

// NewGetTaskStatusUseCase creates a new GetTaskStatusUseCase.
func NewGetTaskStatusUseCase(taskStatusRepo ports.TaskStatusRepository) *GetTaskStatusUseCase {
	return &GetTaskStatusUseCase{taskStatusRepo: taskStatusRepo}
}

// Execute retrieves the task status by task ID.
func (uc *GetTaskStatusUseCase) Execute(ctx context.Context, taskID uuid.UUID) (*GetTaskStatusResponse, error) {
	record, err := uc.taskStatusRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task status: %w", err)
	}

	return mapTaskStatusResponse(record), nil
}

func mapTaskStatusResponse(record *domain.TaskRecord) *GetTaskStatusResponse {
	return &GetTaskStatusResponse{
		ID:           record.ID,
		Subject:      record.Subject,
		Status:       string(record.Status),
		TaskParams:   record.TaskParams,
		ErrorMessage: record.ErrorMessage,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
		StartedAt:    record.StartedAt,
		FinishedAt:   record.FinishedAt,
	}
}
