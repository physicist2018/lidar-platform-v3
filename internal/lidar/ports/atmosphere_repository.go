package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// AtmosphereProfileRepository defines the persistence contract for atmosphere profiles.
type AtmosphereProfileRepository interface {
	Create(ctx context.Context, profile *domain.AtmosphereProfile) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AtmosphereProfile, error)
	FindAll(ctx context.Context, limit, offset int) ([]domain.AtmosphereProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
