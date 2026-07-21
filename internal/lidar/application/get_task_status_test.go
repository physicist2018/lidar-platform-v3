package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

type mockTaskStatusRepoForGet struct {
	findByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error)
}

func (m *mockTaskStatusRepoForGet) Create(_ context.Context, _ *domain.TaskRecord) error {
	panic("mockTaskStatusRepo.Create not implemented")
}

func (m *mockTaskStatusRepoForGet) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.TaskStatus, _ string) error {
	panic("mockTaskStatusRepo.UpdateStatus not implemented")
}

func (m *mockTaskStatusRepoForGet) FindByID(ctx context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
	if m.findByIDFunc == nil {
		panic("mockTaskStatusRepo.FindByID called unexpectedly")
	}
	return m.findByIDFunc(ctx, id)
}

func (m *mockTaskStatusRepoForGet) FindByExperimentID(_ context.Context, _ uuid.UUID) ([]domain.TaskRecord, error) {
	panic("mockTaskStatusRepo.FindByExperimentID not implemented")
}

func (m *mockTaskStatusRepoForGet) FindAll(_ context.Context) ([]domain.TaskRecord, error) {
	panic("mockTaskStatusRepo.FindAll not implemented")
}

func TestGetTaskStatus_Found(t *testing.T) {
	taskID := uuid.New()
	now := time.Now()
	expID := uuid.New()

	repo := &mockTaskStatusRepoForGet{
		findByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
			assert.Equal(t, taskID, id)
			return &domain.TaskRecord{
				ID:           taskID,
				Subject:      "lidar.task.parse_experiment",
				Status:       domain.TaskCompleted,
				ExperimentID: &expID,
				CreatedAt:    now,
				UpdatedAt:    now,
			}, nil
		},
	}

	uc := NewGetTaskStatusUseCase(repo)
	resp, err := uc.Execute(context.Background(), taskID)

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, taskID, resp.ID)
	assert.Equal(t, "lidar.task.parse_experiment", resp.Subject)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, &expID, resp.ExperimentID)
}

func TestGetTaskStatus_NotFound(t *testing.T) {
	taskID := uuid.New()

	repo := &mockTaskStatusRepoForGet{
		findByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
			return nil, domain.ErrObjectNotFound
		},
	}

	uc := NewGetTaskStatusUseCase(repo)
	resp, err := uc.Execute(context.Background(), taskID)

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrObjectNotFound))
}

func TestGetTaskStatus_WithParams(t *testing.T) {
	taskID := uuid.New()
	params := []byte(`{"profile_ids":["p1","p2"],"task_type":"gliding"}`)

	repo := &mockTaskStatusRepoForGet{
		findByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.TaskRecord, error) {
			return &domain.TaskRecord{
				ID:           taskID,
				Subject:      "lidar.task.process_experiment",
				Status:       domain.TaskFailed,
				TaskParams:   params,
				ErrorMessage: "something went wrong",
			}, nil
		},
	}

	uc := NewGetTaskStatusUseCase(repo)
	resp, err := uc.Execute(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, "failed", resp.Status)
	assert.Equal(t, "something went wrong", resp.ErrorMessage)
	assert.JSONEq(t, `{"profile_ids":["p1","p2"],"task_type":"gliding"}`, string(resp.TaskParams))
}
