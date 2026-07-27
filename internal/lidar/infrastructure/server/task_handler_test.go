package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockGetTaskStatusUC struct {
	executeFunc func(ctx context.Context, taskID uuid.UUID) (*application.GetTaskStatusResponse, error)
}

func (m *mockGetTaskStatusUC) Execute(ctx context.Context, taskID uuid.UUID) (*application.GetTaskStatusResponse, error) {
	if m.executeFunc == nil {
		panic("mockGetTaskStatusUC.Execute called unexpectedly")
	}
	return m.executeFunc(ctx, taskID)
}

func newTaskHandlerWithMocks(createTaskUC *application.CreateTaskUseCase, getTaskStatusFn func(ctx context.Context, taskID uuid.UUID) (*application.GetTaskStatusResponse, error)) *TaskHandler {
	var getTaskStatusUC GetTaskStatusUseCase
	if getTaskStatusFn != nil {
		getTaskStatusUC = &mockGetTaskStatusUC{executeFunc: getTaskStatusFn}
	}
	return NewTaskHandler(createTaskUC, getTaskStatusUC)
}

// ---------------------------------------------------------------------------
// HandleCreateTask
// ---------------------------------------------------------------------------

func TestHandleCreateTask_InvalidJSON(t *testing.T) {
	body := bytes.NewReader([]byte(`not-json`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/experiments/task", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := newTaskHandlerWithMocks(nil, nil)
	handler.HandleCreateTask(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "invalid JSON body", resp["error"])
}

func TestHandleCreateTask_EmptyTaskType(t *testing.T) {
	createTaskUC := application.NewCreateTaskUseCase(nil, nil)

	body := mustMarshal(t, map[string]any{
		"task_type": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/experiments/task", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := newTaskHandlerWithMocks(createTaskUC, nil)
	handler.HandleCreateTask(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Contains(t, resp["error"], "task_type must not be empty")
}

// ---------------------------------------------------------------------------
// HandleGetTaskStatus
// ---------------------------------------------------------------------------

func TestHandleGetTaskStatus_Success(t *testing.T) {
	taskID := uuid.New()
	mockFn := func(_ context.Context, id uuid.UUID) (*application.GetTaskStatusResponse, error) {
		assert.Equal(t, taskID, id)
		return &application.GetTaskStatusResponse{
			ID:      taskID,
			Subject: "lidar.task.parse_experiment",
			Status:  "completed",
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID.String(), nil)
	req = addChiURLParam(req, "taskID", taskID.String())
	w := httptest.NewRecorder()

	handler := newTaskHandlerWithMocks(nil, mockFn)
	handler.HandleGetTaskStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp application.GetTaskStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, taskID, resp.ID)
	assert.Equal(t, "lidar.task.parse_experiment", resp.Subject)
	assert.Equal(t, "completed", resp.Status)
}

func TestHandleGetTaskStatus_InvalidUUID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/invalid", nil)
	req = addChiURLParam(req, "taskID", "invalid")
	w := httptest.NewRecorder()

	handler := newTaskHandlerWithMocks(nil, nil)
	handler.HandleGetTaskStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "invalid task ID", resp["error"])
}

func TestHandleGetTaskStatus_NotFound(t *testing.T) {
	taskID := uuid.New()
	mockFn := func(_ context.Context, id uuid.UUID) (*application.GetTaskStatusResponse, error) {
		return nil, domain.ErrObjectNotFound
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID.String(), nil)
	req = addChiURLParam(req, "taskID", taskID.String())
	w := httptest.NewRecorder()

	handler := newTaskHandlerWithMocks(nil, mockFn)
	handler.HandleGetTaskStatus(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "task not found", resp["error"])
}

func TestHandleGetTaskStatus_MissingParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/", nil)
	w := httptest.NewRecorder()

	handler := newTaskHandlerWithMocks(nil, nil)
	handler.HandleGetTaskStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "taskID is required", resp["error"])
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return data
}

func addChiURLParam(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}
