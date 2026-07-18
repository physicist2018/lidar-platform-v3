package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresLicelProfileRepository implements ports.LicelProfileRepository backed by sqlc.
type PostgresLicelProfileRepository struct {
	q *db.Queries
}

// NewPostgresLicelProfileRepository creates a new PostgresLicelProfileRepository.
func NewPostgresLicelProfileRepository(dbtx db.DBTX) *PostgresLicelProfileRepository {
	return &PostgresLicelProfileRepository{q: db.New(dbtx)}
}

// Create persists a new LICEL profile.
func (r *PostgresLicelProfileRepository) Create(ctx context.Context, profile *domain.LicelProfile) error {
	_, err := r.q.CreateLicelProfile(ctx, db.CreateLicelProfileParams{
		LicelfileID:  profile.LicelFileID,
		NDataPoints:  profile.NDataPoints,
		HighVoltage:  profile.HighVoltage,
		BinWidth:     profile.BinWidth,
		Wavelength:   profile.Wavelength,
		Polarization: profile.Polarization,
		DeviceID:     profile.DeviceID,
		NShots:       profile.NShots,
		DiscrLevel:   profile.DiscrLevel,
		Data:         profile.Data,
	})
	return err
}

// FindByID looks up a LICEL profile by ID.
func (r *PostgresLicelProfileRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.LicelProfile, error) {
	u, err := r.q.GetLicelProfileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapLicelProfile(u), nil
}

// FindAllByLicelFileID returns all non-deleted profiles for a LICEL file.
func (r *PostgresLicelProfileRepository) FindAllByLicelFileID(ctx context.Context, licelFileID uuid.UUID) ([]domain.LicelProfile, error) {
	rows, err := r.q.ListLicelProfilesByLicelFile(ctx, licelFileID)
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.LicelProfile, len(rows))
	for i, row := range rows {
		profiles[i] = *mapLicelProfile(row)
	}
	return profiles, nil
}

// SoftDelete marks a LICEL profile as deleted.
func (r *PostgresLicelProfileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.q.SoftDeleteLicelProfile(ctx, id)
}

// Restore removes the soft-delete mark from a LICEL profile.
func (r *PostgresLicelProfileRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.LicelProfile, error) {
	u, err := r.q.RestoreLicelProfile(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapLicelProfile(u), nil
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapLicelProfile(u db.LidarLicelProfile) *domain.LicelProfile {
	return &domain.LicelProfile{
		ID:           u.ID,
		LicelFileID:  u.LicelfileID,
		NDataPoints:  u.NDataPoints,
		HighVoltage:  u.HighVoltage,
		BinWidth:     u.BinWidth,
		Wavelength:   u.Wavelength,
		Polarization: u.Polarization,
		DeviceID:     u.DeviceID,
		NShots:       u.NShots,
		DiscrLevel:   u.DiscrLevel,
		Data:         u.Data,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		DeletedAt:    fromNullTime(u.DeletedAt),
	}
}
