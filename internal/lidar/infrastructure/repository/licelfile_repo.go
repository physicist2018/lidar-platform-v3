package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresLicelFileRepository implements ports.LicelFileRepository backed by sqlc.
type PostgresLicelFileRepository struct {
	q *db.Queries
}

// NewPostgresLicelFileRepository creates a new PostgresLicelFileRepository.
func NewPostgresLicelFileRepository(dbtx db.DBTX) *PostgresLicelFileRepository {
	return &PostgresLicelFileRepository{q: db.New(dbtx)}
}

// Create persists a new LICEL file.
func (r *PostgresLicelFileRepository) Create(ctx context.Context, file *domain.LicelFile) error {
	_, err := r.q.CreateLicelFile(ctx, db.CreateLicelFileParams{
		ExperimentID:     file.ExperimentID,
		MeasurementStart: file.MeasurementRange.Start,
		MeasurementStop:  file.MeasurementRange.End,
		NDatasets:        file.NDatasets,
		LaserFreq:        file.LaserFreq,
		IsBackground:     file.IsBackground,
		RawStorageID:     file.RawStorageID,
		Filename:         file.Filename,
	})
	return err
}

// FindByID looks up a LICEL file by ID.
func (r *PostgresLicelFileRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.LicelFile, error) {
	u, err := r.q.GetLicelFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapLicelFile(u), nil
}

// FindAllByExperimentID returns all non-deleted files for an experiment.
func (r *PostgresLicelFileRepository) FindAllByExperimentID(ctx context.Context, experimentID uuid.UUID) ([]domain.LicelFile, error) {
	rows, err := r.q.ListLicelFilesByExperiment(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	files := make([]domain.LicelFile, len(rows))
	for i, row := range rows {
		files[i] = *mapLicelFile(row)
	}
	return files, nil
}

// SoftDelete marks a LICEL file as deleted.
func (r *PostgresLicelFileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.q.SoftDeleteLicelFile(ctx, id)
}

// Restore removes the soft-delete mark from a LICEL file.
func (r *PostgresLicelFileRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.LicelFile, error) {
	u, err := r.q.RestoreLicelFile(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapLicelFile(u), nil
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapLicelFile(u db.LidarLicelfile) *domain.LicelFile {
	timeRange, err := domain.NewTimeRange(u.MeasurementStart, u.MeasurementStop)
	if err != nil {
		timeRange = domain.TimeRange{Start: u.MeasurementStart, End: u.MeasurementStop}
	}

	return &domain.LicelFile{
		ID:               u.ID,
		ExperimentID:     u.ExperimentID,
		MeasurementRange: timeRange,
		NDatasets:        u.NDatasets,
		LaserFreq:        u.LaserFreq,
		IsBackground:     u.IsBackground,
		RawStorageID:     u.RawStorageID,
		Filename:         u.Filename,
		CreatedAt:        u.CreatedAt,
		DeletedAt:        fromNullTime(u.DeletedAt),
	}
}
