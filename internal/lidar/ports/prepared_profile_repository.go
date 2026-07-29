package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

// PreparedProfileRepository provides read/write access to processed profiles.
type PreparedProfileRepository interface {
	// Create persists a new PreparedProfile.
	Create(ctx context.Context, profile *domain.PreparedProfile) error

	// FindByMetaID returns all prepared profiles for a given meta, ordered by created_at.
	FindByMetaID(ctx context.Context, metaID uuid.UUID) ([]domain.PreparedProfile, error)

	// FindByExperiment returns prepared profiles with metadata, optionally filtered.
	FindByExperiment(ctx context.Context, experimentID uuid.UUID, wavelength *float32, polarization, deviceID *string) ([]domain.PreparedProfileView, error)

	// FindExperiments returns distinct experiment IDs that have prepared profiles.
	FindExperiments(ctx context.Context) ([]uuid.UUID, error)

	// FindWavelengths returns distinct wavelengths for an experiment.
	FindWavelengths(ctx context.Context, experimentID uuid.UUID) ([]float32, error)

	// FindPolarizations returns distinct polarizations for an experiment, optionally filtered by wavelength.
	FindPolarizations(ctx context.Context, experimentID uuid.UUID, wavelength *float32) ([]string, error)

	// FindDeviceIDs returns distinct device IDs for an experiment, optionally filtered by wavelength and polarization.
	FindDeviceIDs(ctx context.Context, experimentID uuid.UUID, wavelength *float32, polarization *string) ([]string, error)
}
