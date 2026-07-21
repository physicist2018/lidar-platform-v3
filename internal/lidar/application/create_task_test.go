package application

import (
	"context"
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
	createFunc func(ctx context.Context, record *domain.TaskRecord) error
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

func (m *mockTaskStatusRepoForCreate) FindByID(_ context.Context, _ uuid.UUID) (*domain.TaskRecord, error) {
	panic("mockTaskStatusRepo.FindByID not implemented")
}

func (m *mockTaskStatusRepoForCreate) FindByExperimentID(_ context.Context, _ uuid.UUID) ([]domain.TaskRecord, error) {
	panic("mockTaskStatusRepo.FindByExperimentID not implemented")
}

func (m *mockTaskStatusRepoForCreate) FindAll(_ context.Context) ([]domain.TaskRecord, error) {
	panic("mockTaskStatusRepo.FindAll not implemented")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateTask_EmptyProfileID(t *testing.T) {
	uc := NewCreateTaskUseCase(nil, nil)

	resp, err := uc.Execute(context.Background(), &TaskRequest{
		ProfileID: []string{},
		TaskType:  "gliding",
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile_id must not be empty")
}

func TestCreateTask_EmptyTaskType(t *testing.T) {
	uc := NewCreateTaskUseCase(nil, nil)

	resp, err := uc.Execute(context.Background(), &TaskRequest{
		ProfileID: []string{"profile-1"},
		TaskType:  "",
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

	uc := NewCreateTaskUseCase(queue, repo)
	resp, err := uc.Execute(context.Background(), &TaskRequest{
		ProfileID: []string{"profile-1", "profile-2"},
		TaskType:  "gliding",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEmpty(t, resp.TaskID)
	assert.Equal(t, "queued", resp.Status)

	// Verify task status record was created
	require.NotNil(t, createdRecord)
	assert.Equal(t, string(ports.SubjectProcessExperiment), createdRecord.Subject)
	assert.Equal(t, domain.TaskPending, createdRecord.Status)
	assert.Nil(t, createdRecord.ExperimentID)
	assert.Contains(t, string(createdRecord.TaskParams), "profile-1")

	// Verify message was published to NATS
	assert.Equal(t, ports.SubjectProcessExperiment, publishedSubject)
	assert.Equal(t, resp.TaskID, publishedDedupID)
	assert.Contains(t, string(publishedData), resp.TaskID)
	assert.Contains(t, string(publishedData), "gliding")
}
