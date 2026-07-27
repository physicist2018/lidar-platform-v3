package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// PostgresExperimentRepository implements ports.ExperimentRepository backed by sqlc.
type PostgresExperimentRepository struct {
	q *db.Queries
}

// NewPostgresExperimentRepository creates a new PostgresExperimentRepository.
func NewPostgresExperimentRepository(dbtx db.DBTX) *PostgresExperimentRepository {
	return &PostgresExperimentRepository{q: db.New(dbtx)}
}

// Create persists a new experiment.
func (r *PostgresExperimentRepository) Create(ctx context.Context, experiment *domain.Experiment) error {
	_, err := r.q.CreateExperiment(ctx, db.CreateExperimentParams{
		ID:                   experiment.ID,
		Title:                experiment.Title,
		Comments:             toNullString(experiment.Comments),
		ZenithAngle:          experiment.ZenithAngle,
		ExperimentStart:      experiment.TimeRange.Start,
		ExperimentEnd:        experiment.TimeRange.End,
		Longitude:            experiment.GeoLocation.Longitude,
		Latitude:             experiment.GeoLocation.Latitude,
		ExperimentsStorageID: toNullUUID(experiment.StorageRefs.ExperimentDataID),
		BackgroundStorageID:  toNullUUID(experiment.StorageRefs.BackgroundID),
		MeteoStorageID:       toNullUUID(experiment.StorageRefs.MeteoID),
	})
	return err
}

// FindByID looks up an experiment by ID.
func (r *PostgresExperimentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error) {
	u, err := r.q.GetExperimentByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapExperiment(u), nil
}

// FindAll returns a paginated list of non-deleted experiments.
func (r *PostgresExperimentRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Experiment, error) {
	rows, err := r.q.ListExperiments(ctx, db.ListExperimentsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	experiments := make([]domain.Experiment, len(rows))
	for i, row := range rows {
		experiments[i] = *mapExperiment(row)
	}
	return experiments, nil
}

// Update updates mutable fields of an experiment.
func (r *PostgresExperimentRepository) Update(ctx context.Context, experiment *domain.Experiment) error {
	_, err := r.q.UpdateExperiment(ctx, db.UpdateExperimentParams{
		ID:              experiment.ID,
		Title:           experiment.Title,
		Comments:        toNullString(experiment.Comments),
		ZenithAngle:     experiment.ZenithAngle,
		Longitude:       experiment.GeoLocation.Longitude,
		Latitude:        experiment.GeoLocation.Latitude,
		ExperimentStart: experiment.TimeRange.Start,
		ExperimentEnd:   experiment.TimeRange.End,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrObjectNotFound
		}
		return err
	}
	return nil
}

// UpdateStorageRefs updates the storage references for an experiment.
func (r *PostgresExperimentRepository) UpdateStorageRefs(ctx context.Context, id uuid.UUID, refs domain.ExperimentStorageRefs) error {
	_, err := r.q.UpdateExperimentStorageRefs(ctx, db.UpdateExperimentStorageRefsParams{
		ID:                   id,
		ExperimentsStorageID: toNullUUID(refs.ExperimentDataID),
		BackgroundStorageID:  toNullUUID(refs.BackgroundID),
		MeteoStorageID:       toNullUUID(refs.MeteoID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrObjectNotFound
		}
		return err
	}
	return nil
}

// SoftDelete marks an experiment as deleted (sets deleted_at).
func (r *PostgresExperimentRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.q.SoftDeleteExperiment(ctx, id)
}

// Restore removes the soft-delete mark from an experiment.
func (r *PostgresExperimentRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.Experiment, error) {
	u, err := r.q.RestoreExperiment(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrObjectNotFound
		}
		return nil, err
	}
	return mapExperiment(u), nil
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func mapExperiment(u db.LidarExperiment) *domain.Experiment {
	timeRange, err := domain.NewTimeRange(u.ExperimentStart, u.ExperimentEnd)
	if err != nil {
		timeRange = domain.TimeRange{Start: u.ExperimentStart, End: u.ExperimentEnd}
	}

	geoLocation, err := domain.NewGeoLocation(u.Latitude, u.Longitude)
	if err != nil {
		geoLocation = domain.GeoLocation{Latitude: u.Latitude, Longitude: u.Longitude}
	}

	return &domain.Experiment{
		ID:          u.ID,
		Title:       u.Title,
		Comments:    u.Comments.String,
		ZenithAngle: u.ZenithAngle,
		TimeRange:   timeRange,
		GeoLocation: geoLocation,
		StorageRefs: domain.ExperimentStorageRefs{
			ExperimentDataID: fromNullUUID(u.ExperimentsStorageID),
			BackgroundID:     fromNullUUID(u.BackgroundStorageID),
			MeteoID:          fromNullUUID(u.MeteoStorageID),
		},
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		DeletedAt: fromNullTime(u.DeletedAt),
	}
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func toNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func fromNullUUID(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

func fromNullTime(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// FindByTimeRange returns non-deleted experiments within [startTime, endTime].
func (r *PostgresExperimentRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]domain.Experiment, error) {
	rows, err := r.q.ListExperimentsByTimeRange(ctx, db.ListExperimentsByTimeRangeParams{
		ExperimentStart: startTime,
		ExperimentEnd:   endTime,
		Limit:           int32(limit),
		Offset:          int32(offset),
	})
	if err != nil {
		return nil, err
	}
	experiments := make([]domain.Experiment, len(rows))
	for i, row := range rows {
		experiments[i] = *mapExperiment(row)
	}
	return experiments, nil
}
