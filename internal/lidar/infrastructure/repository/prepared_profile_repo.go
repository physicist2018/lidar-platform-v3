package repository

import (
	"context"

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
