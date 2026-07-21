package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresAtmosphereProfileRepository implements ports.AtmosphereProfileRepository backed by sqlc.
type PostgresAtmosphereProfileRepository struct {
	q *db.Queries
}

// NewPostgresAtmosphereProfileRepository creates a new PostgresAtmosphereProfileRepository.
func NewPostgresAtmosphereProfileRepository(dbtx db.DBTX) *PostgresAtmosphereProfileRepository {
	return &PostgresAtmosphereProfileRepository{q: db.New(dbtx)}
}

// Create persists a new atmosphere profile.
func (r *PostgresAtmosphereProfileRepository) Create(ctx context.Context, profile *domain.AtmosphereProfile) error {
	_, err := r.q.CreateAtmosphereProfile(ctx, db.CreateAtmosphereProfileParams{
		ExperimentID: profile.ExperimentID,
		Altitude:     profile.Altitude,
		Temperature:  profile.Temperature,
		Pressure:     profile.Pressure,
	})
	return err
}

// FindByID looks up an atmosphere profile by ID.
func (r *PostgresAtmosphereProfileRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AtmosphereProfile, error) {
	u, err := r.q.GetAtmosphereProfileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapAtmosphereProfile(u), nil
}

// FindAll returns a paginated list of atmosphere profiles.
func (r *PostgresAtmosphereProfileRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.AtmosphereProfile, error) {
	rows, err := r.q.ListAtmosphereProfiles(ctx, db.ListAtmosphereProfilesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.AtmosphereProfile, len(rows))
	for i, row := range rows {
		profiles[i] = *mapAtmosphereProfile(row)
	}
	return profiles, nil
}

// Delete removes an atmosphere profile permanently.
func (r *PostgresAtmosphereProfileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteAtmosphereProfile(ctx, id)
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapAtmosphereProfile(u db.LidarAtmosphereProfile) *domain.AtmosphereProfile {
	return &domain.AtmosphereProfile{
		ID:           u.ID,
		ExperimentID: u.ExperimentID,
		Altitude:     u.Altitude,
		Temperature:  u.Temperature,
		Pressure:     u.Pressure,
		CreatedAt:    u.CreatedAt,
	}
}
