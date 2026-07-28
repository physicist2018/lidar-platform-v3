package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// TaskRequest is the JSON body for creating a processing task.
type TaskRequest struct {
	Subject  string          `json:"subject"`
	TaskType string          `json:"task_type"`
	TaskID   string          `json:"task_id,omitempty"` // optional; auto-generated if empty
	Payload  json.RawMessage `json:"payload"`
}

// TaskResponse is returned on successful task creation.
type TaskResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// CreateTaskUseCase creates a processing task and publishes it to NATS.
type CreateTaskUseCase struct {
	queue          ports.MessageQueue
	taskStatusRepo ports.TaskStatusRepository
}

// NewCreateTaskUseCase creates a new CreateTaskUseCase.
func NewCreateTaskUseCase(
	queue ports.MessageQueue,
	taskStatusRepo ports.TaskStatusRepository,
) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		queue:          queue,
		taskStatusRepo: taskStatusRepo,
	}
}

// Execute validates the request and publishes the task to NATS.
func (uc *CreateTaskUseCase) Execute(ctx context.Context, req *TaskRequest) (*TaskResponse, error) {
	if req.Subject == "" {
		return nil, fmt.Errorf("subject must not be empty")
	}
	if req.TaskType == "" {
		return nil, fmt.Errorf("task_type must not be empty")
	}

	taskID := req.TaskID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task_id: %w", err)
	}

	// Build the message body: original request + task_id.
	msg := map[string]any{
		"task_id":    taskID,
		"subject":    req.Subject,
		"task_type":  req.TaskType,
		"payload":    req.Payload,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal task: %w", err)
	}

	// Create task status record.
	params := map[string]any{"subject": req.Subject, "task_type": req.TaskType}
	if req.Payload != nil {
		params["payload"] = req.Payload
	}
	taskParams, _ := json.Marshal(params)
	taskRecord := domain.NewTaskRecord(
		taskUUID,
		req.Subject,
		taskParams,
	)
	if err := uc.taskStatusRepo.Create(ctx, &taskRecord); err != nil {
		log.Printf("create task status: %v", err)
	}

	// Publish with the task_id as dedup ID to prevent duplicates.
	subject := ports.Subject(req.Subject)
	if err := uc.queue.Publish(ctx, subject, data, taskID); err != nil {
		return nil, fmt.Errorf("publish task: %w", err)
	}

	return &TaskResponse{
		TaskID: taskID,
		Status: "queued",
	}, nil
}
