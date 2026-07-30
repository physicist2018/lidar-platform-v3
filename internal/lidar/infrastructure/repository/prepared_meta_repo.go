package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresPreparedMetaRepository implements ports.PreparedMetaRepository backed by sqlc.
type PostgresPreparedMetaRepository struct {
	q *db.Queries
}

// NewPostgresPreparedMetaRepository creates a new PostgresPreparedMetaRepository.
func NewPostgresPreparedMetaRepository(dbtx db.DBTX) *PostgresPreparedMetaRepository {
	return &PostgresPreparedMetaRepository{q: db.New(dbtx)}
}

// Create persists a new PreparedMeta and sets its ID from the DB-generated value.
func (r *PostgresPreparedMetaRepository) Create(ctx context.Context, meta *domain.PreparedMeta) error {
	u, err := r.q.CreatePreparedMeta(ctx, db.CreatePreparedMetaParams{
		ExperimentID:   meta.ExperimentID,
		BackgroundType: string(meta.BackgroundType),
		BackgroundFrom: meta.BackgroundFrom,
		TrimFrom:       meta.TrimFrom,
	})
	if err != nil {
		return err
	}
	meta.ID = u.ID
	return nil
}

// FindByExperimentID returns the PreparedMeta for a given experiment.
func (r *PostgresPreparedMetaRepository) FindByExperimentID(ctx context.Context, experimentID uuid.UUID) (*domain.PreparedMeta, error) {
	u, err := r.q.GetPreparedMetaByExperimentID(ctx, experimentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapPreparedMeta(u), nil
}

// DeleteByExperimentID permanently deletes PreparedMeta (and cascades to PreparedProfiles) for an experiment.
func (r *PostgresPreparedMetaRepository) DeleteByExperimentID(ctx context.Context, experimentID uuid.UUID) error {
	return r.q.DeletePreparedMetaByExperimentID(ctx, experimentID)
}

// ---------------------------------------------------------------------------
// Mapper
// ---------------------------------------------------------------------------

func mapPreparedMeta(u db.LidarPreparedMetum) *domain.PreparedMeta {
	return &domain.PreparedMeta{
		ID:             u.ID,
		ExperimentID:   u.ExperimentID,
		BackgroundType: domain.BackgroundType(u.BackgroundType),
		BackgroundFrom: u.BackgroundFrom,
		TrimFrom:       u.TrimFrom,
	}
}
