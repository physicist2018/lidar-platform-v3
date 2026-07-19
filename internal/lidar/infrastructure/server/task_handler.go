package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
)

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	createTaskUC *application.CreateTaskUseCase
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(createTaskUC *application.CreateTaskUseCase) *TaskHandler {
	return &TaskHandler{createTaskUC: createTaskUC}
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
