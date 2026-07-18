package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// ExperimentRepository defines the persistence contract for experiments.
type ExperimentRepository interface {
	Create(ctx context.Context, experiment *domain.Experiment) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error)
	FindAll(ctx context.Context, limit, offset int) ([]domain.Experiment, error)
	Update(ctx context.Context, experiment *domain.Experiment) error
	UpdateStorageRefs(ctx context.Context, id uuid.UUID, refs domain.ExperimentStorageRefs) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*domain.Experiment, error)
}
