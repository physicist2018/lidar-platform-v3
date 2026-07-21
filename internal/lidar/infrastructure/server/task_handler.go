package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// GetTaskStatusUseCase is the interface for getting task status.
type GetTaskStatusUseCase interface {
	Execute(ctx context.Context, taskID uuid.UUID) (*application.GetTaskStatusResponse, error)
}

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	createTaskUC    *application.CreateTaskUseCase
	getTaskStatusUC GetTaskStatusUseCase
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(createTaskUC *application.CreateTaskUseCase, getTaskStatusUC GetTaskStatusUseCase) *TaskHandler {
	return &TaskHandler{
		createTaskUC:    createTaskUC,
		getTaskStatusUC: getTaskStatusUC,
	}
}

// HandleCreateTask handles POST /api/v1/experiments/task.
func (h *TaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req application.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.createTaskUC.Execute(r.Context(), &req)
	if err != nil {
		log.Printf("create task error: %v", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusCreated, result)
}

// HandleGetTaskStatus handles GET /api/v1/tasks/{taskID}.
func (h *TaskHandler) HandleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "taskID")
	if taskIDStr == "" {
		RespondWithError(w, http.StatusBadRequest, "taskID is required")
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	result, err := h.getTaskStatusUC.Execute(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, domain.ErrObjectNotFound) {
			RespondWithError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Printf("get task status error: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondWithJSON(w, http.StatusOK, result)
}
