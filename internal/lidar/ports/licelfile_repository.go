package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// LicelFileRepository defines the persistence contract for LICEL files.
type LicelFileRepository interface {
	Create(ctx context.Context, file *domain.LicelFile) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.LicelFile, error)
	FindAllByExperimentID(ctx context.Context, experimentID uuid.UUID) ([]domain.LicelFile, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*domain.LicelFile, error)
}
