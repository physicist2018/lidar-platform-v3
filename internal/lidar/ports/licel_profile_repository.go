package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// LicelProfileRepository defines the persistence contract for LICEL profiles.
type LicelProfileRepository interface {
	Create(ctx context.Context, profile *domain.LicelProfile) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.LicelProfile, error)
	FindAllByLicelFileID(ctx context.Context, licelFileID uuid.UUID) ([]domain.LicelProfile, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*domain.LicelProfile, error)

	// FindProfilesWithBackground returns signal profiles paired with their
	// matching background profiles for a given experiment.
	FindProfilesWithBackground(ctx context.Context, experimentID uuid.UUID) ([]domain.PairedProfile, error)
}
