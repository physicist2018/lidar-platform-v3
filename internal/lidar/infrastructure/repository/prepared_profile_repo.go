package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresPreparedProfileRepository implements ports.PreparedProfileRepository backed by sqlc.
type PostgresPreparedProfileRepository struct {
	q *db.Queries
}

// NewPostgresPreparedProfileRepository creates a new PostgresPreparedProfileRepository.
func NewPostgresPreparedProfileRepository(dbtx db.DBTX) *PostgresPreparedProfileRepository {
	return &PostgresPreparedProfileRepository{q: db.New(dbtx)}
}

// Create persists a new PreparedProfile and sets its ID from the DB-generated value.
func (r *PostgresPreparedProfileRepository) Create(ctx context.Context, profile *domain.PreparedProfile) error {
	u, err := r.q.CreatePreparedProfile(ctx, db.CreatePreparedProfileParams{
		PreparedMetaID: profile.PreparedMetaID,
		LicelProfileID: profile.LicelProfileID,
		Data:           profile.Data,
	})
	if err != nil {
		return err
	}
	profile.ID = u.ID
	profile.CreatedAt = u.CreatedAt
	profile.UpdatedAt = u.UpdatedAt
	return nil
}

// FindByMetaID returns all prepared profiles for a given meta, ordered by created_at.
func (r *PostgresPreparedProfileRepository) FindByMetaID(ctx context.Context, metaID uuid.UUID) ([]domain.PreparedProfile, error) {
	rows, err := r.q.ListPreparedProfilesByMetaID(ctx, metaID)
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.PreparedProfile, len(rows))
	for i, row := range rows {
		profiles[i] = *mapPreparedProfile(row)
	}
	return profiles, nil
}

// ---------------------------------------------------------------------------
// Filters
// ---------------------------------------------------------------------------

// FindExperiments returns distinct experiment IDs that have prepared profiles.
func (r *PostgresPreparedProfileRepository) FindExperiments(ctx context.Context) ([]uuid.UUID, error) {
	return r.q.ListPreparedExperiments(ctx)
}

// FindWavelengths returns distinct wavelengths for an experiment.
func (r *PostgresPreparedProfileRepository) FindWavelengths(ctx context.Context, experimentID uuid.UUID) ([]float32, error) {
	return r.q.ListPreparedProfileWavelengths(ctx, experimentID)
}

// FindPolarizations returns distinct polarizations for an experiment, optionally filtered by wavelength.
func (r *PostgresPreparedProfileRepository) FindPolarizations(ctx context.Context, experimentID uuid.UUID, wavelength *float32) ([]string, error) {
	return r.q.ListPreparedProfilePolarizations(ctx, db.ListPreparedProfilePolarizationsParams{
		ExperimentID: experimentID,
		Wavelength:   toNullFloat64(wavelength),
	})
}

// FindDeviceIDs returns distinct device IDs for an experiment, optionally filtered by wavelength and polarization.
func (r *PostgresPreparedProfileRepository) FindDeviceIDs(ctx context.Context, experimentID uuid.UUID, wavelength *float32, polarization *string) ([]string, error) {
	return r.q.ListPreparedProfileDeviceIDs(ctx, db.ListPreparedProfileDeviceIDsParams{
		ExperimentID: experimentID,
		Wavelength:   toNullFloat64(wavelength),
		Polarization: toNullStringPtr(polarization),
	})
}

// ---------------------------------------------------------------------------
// Query: prepared profiles with metadata
// ---------------------------------------------------------------------------

// FindByExperiment returns prepared profiles with metadata, optionally filtered.
func (r *PostgresPreparedProfileRepository) FindByExperiment(
	ctx context.Context,
	experimentID uuid.UUID,
	wavelength *float32,
	polarization, deviceID *string,
) ([]domain.PreparedProfileView, error) {
	params := db.ListPreparedProfilesByExperimentParams{
		ExperimentID: experimentID,
		Wavelength:   toNullFloat64(wavelength),
		Polarization: toNullStringPtr(polarization),
		DeviceID:     toNullStringPtr(deviceID),
	}
	rows, err := r.q.ListPreparedProfilesByExperiment(ctx, params)
	if err != nil {
		return nil, err
	}
	views := make([]domain.PreparedProfileView, len(rows))
	for i, row := range rows {
		views[i] = domain.PreparedProfileView{
			ID:               row.ID,
			Wavelength:       row.Wavelength,
			Polarization:     row.Polarization,
			DeviceID:         row.DeviceID,
			BinWidth:         row.BinWidth,
			Data:             row.Data,
			MeasurementStart: row.MeasurementStart,
			BackgroundType:   domain.BackgroundType(row.BackgroundType),
			BackgroundFrom:   row.BackgroundFrom,
			TrimFrom:         row.TrimFrom,
			CreatedAt:        row.CreatedAt,
		}
	}
	return views, nil
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapPreparedProfile(u db.LidarPreparedProfile) *domain.PreparedProfile {
	return &domain.PreparedProfile{
		ID:             u.ID,
		PreparedMetaID: u.PreparedMetaID,
		LicelProfileID: u.LicelProfileID,
		Data:           u.Data,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toNullFloat64(v *float32) sql.NullFloat64 {
	if v == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: float64(*v), Valid: true}
}

func toNullStringPtr(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *v, Valid: true}
}

func float32OrZero(v *float32) float32 {
	if v == nil {
		return 0
	}
	return *v
}

func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
