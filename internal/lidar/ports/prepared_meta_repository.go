package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// PreparedMetaRepository provides read/write access to processing metadata.
type PreparedMetaRepository interface {
	// Create persists a new PreparedMeta.
	Create(ctx context.Context, meta *domain.PreparedMeta) error

	// FindByExperimentID returns the PreparedMeta for a given experiment.
	FindByExperimentID(ctx context.Context, experimentID uuid.UUID) (*domain.PreparedMeta, error)
}
