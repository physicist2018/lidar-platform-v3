package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// DeleteTaskUseCase deletes a task and its associated results.
type DeleteTaskUseCase struct {
	taskStatusRepo   ports.TaskStatusRepository
	preparedMetaRepo ports.PreparedMetaRepository
}

// NewDeleteTaskUseCase creates a new DeleteTaskUseCase.
func NewDeleteTaskUseCase(
	taskStatusRepo ports.TaskStatusRepository,
	preparedMetaRepo ports.PreparedMetaRepository,
) *DeleteTaskUseCase {
	return &DeleteTaskUseCase{
		taskStatusRepo:   taskStatusRepo,
		preparedMetaRepo: preparedMetaRepo,
	}
}

// Execute deletes a task and its associated results.
func (uc *DeleteTaskUseCase) Execute(ctx context.Context, taskID uuid.UUID) error {
	// 1. Find the task to determine its type.
	task, err := uc.taskStatusRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// 2. For prepare_experiment tasks, delete the prepared data first.
	if task.Subject == "lidar.task.prepare_experiment" {
		experimentID := extractExperimentID(task.TaskParams)
		if experimentID != nil {
			if err := uc.preparedMetaRepo.DeleteByExperimentID(ctx, *experimentID); err != nil {
				return fmt.Errorf("delete prepared data: %w", err)
			}
		}
	}

	// 3. For parse_experiment tasks, the task_id equals the experiment_id.
	//    We only delete the task record — parsed data (licel_files, profiles)
	//    is kept so the user doesn't need to re-upload the archive.
	//    If they want to re-parse, they can delete the experiment itself.

	// 4. Delete the task record.
	if err := uc.taskStatusRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

// extractExperimentID tries to parse experiment_id from task_params JSON.
func extractExperimentID(params []byte) *uuid.UUID {
	if params == nil {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(params, &data); err != nil {
		return nil
	}
	// Check top-level
	if raw, ok := data["experiment_id"]; ok {
		if s, ok := raw.(string); ok {
			id, err := uuid.Parse(s)
			if err == nil {
				return &id
			}
		}
	}
	// Check nested payload
	if payload, ok := data["payload"]; ok {
		if m, ok := payload.(map[string]any); ok {
			if raw, ok := m["experiment_id"]; ok {
				if s, ok := raw.(string); ok {
					id, err := uuid.Parse(s)
					if err == nil {
						return &id
					}
				}
			}
		}
	}
	return nil
}
