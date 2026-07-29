package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockQueue struct {
	publishFunc func(ctx context.Context, subject ports.Subject, data []byte, dedupID string) error
}

func (m *mockQueue) Publish(ctx context.Context, subject ports.Subject, data []byte, dedupID string) error {
	if m.publishFunc == nil {
		panic("mockQueue.Publish called unexpectedly")
	}
	return m.publishFunc(ctx, subject, data, dedupID)
}

func (m *mockQueue) Subscribe(_ context.Context, _ ports.Subject, _ string, _ ports.MessageHandler) (ports.Subscription, error) {
	panic("mockQueue.Subscribe not implemented")
}

func (m *mockQueue) Close() error {
	panic("mockQueue.Close not implemented")
}

type mockTaskStatusRepoForCreate struct {
	createFunc   func(ctx context.Context, record *domain.TaskRecord) error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error)
}

func (m *mockTaskStatusRepoForCreate) Create(ctx context.Context, record *domain.TaskRecord) error {
	if m.createFunc == nil {
		panic("mockTaskStatusRepo.Create called unexpectedly")
	}
	return m.createFunc(ctx, record)
}

func (m *mockTaskStatusRepoForCreate) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.TaskStatus, _ string) error {
	panic("mockTaskStatusRepo.UpdateStatus not implemented")
}

func (m *mockTaskStatusRepoForCreate) FindByID(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domain.ErrObjectNotFound
}

func (m *mockTaskStatusRepoForCreate) FindAll(_ context.Context) ([]domain.TaskRecord, error) {
	panic("mockTaskStatusRepo.FindAll not implemented")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateTask_EmptySubject(t *testing.T) {
	uc := NewCreateTaskUseCase(nil, nil)

	resp, err := uc.Execute(context.Background(), &TaskRequest{
		Subject:  "",
		TaskType: "gliding",
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject must not be empty")
}

func TestCreateTask_EmptyTaskType(t *testing.T) {
	uc := NewCreateTaskUseCase(nil, nil)

	resp, err := uc.Execute(context.Background(), &TaskRequest{
		Subject:  "lidar.task.test",
		TaskType: "",
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task_type must not be empty")
}

func TestCreateTask_Success(t *testing.T) {
	var createdRecord *domain.TaskRecord
	var publishedSubject ports.Subject
	var publishedData []byte
	var publishedDedupID string

	queue := &mockQueue{
		publishFunc: func(_ context.Context, subject ports.Subject, data []byte, dedupID string) error {
			publishedSubject = subject
			publishedData = data
			publishedDedupID = dedupID
			return nil
		},
	}

	repo := &mockTaskStatusRepoForCreate{
		createFunc: func(_ context.Context, record *domain.TaskRecord) error {
			createdRecord = record
			return nil
		},
	}

	payload := json.RawMessage(`{"profile_ids":["p1","p2"]}`)
	uc := NewCreateTaskUseCase(queue, repo)
	resp, err := uc.Execute(context.Background(), &TaskRequest{
		Subject:  string(ports.SubjectProcessExperiment),
		TaskType: "gliding",
		Payload:  payload,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEmpty(t, resp.TaskID)
	assert.Equal(t, "queued", resp.Status)

	// Verify task status record was created
	require.NotNil(t, createdRecord)
	assert.Equal(t, string(ports.SubjectProcessExperiment), createdRecord.Subject)
	assert.Equal(t, domain.TaskPending, createdRecord.Status)
	assert.Contains(t, string(createdRecord.TaskParams), "gliding")
	assert.Contains(t, string(createdRecord.TaskParams), "p1")

	// Verify message was published to NATS
	assert.Equal(t, ports.SubjectProcessExperiment, publishedSubject)
	assert.Equal(t, resp.TaskID, publishedDedupID)
	assert.Contains(t, string(publishedData), resp.TaskID)
	assert.Contains(t, string(publishedData), "gliding")
	assert.Contains(t, string(publishedData), "p1")
}

func TestCreateTask_Idempotent(t *testing.T) {
	taskID := uuid.New()
	var publishCalled bool

	queue := &mockQueue{
		publishFunc: func(_ context.Context, _ ports.Subject, _ []byte, _ string) error {
			publishCalled = true
			return nil
		},
	}

	repo := &mockTaskStatusRepoForCreate{
		findByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
			return &domain.TaskRecord{
				ID:     id,
				Status: domain.TaskCompleted,
			}, nil
		},
	}

	uc := NewCreateTaskUseCase(queue, repo)
	resp, err := uc.Execute(context.Background(), &TaskRequest{
		Subject:  "lidar.task.test",
		TaskType: "test",
		TaskID:   taskID.String(),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, taskID.String(), resp.TaskID)
	assert.Equal(t, "completed", resp.Status)
	assert.False(t, publishCalled, "NATS publish should not be called for existing task")
}
