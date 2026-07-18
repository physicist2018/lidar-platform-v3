package domain

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Value objects
// ---------------------------------------------------------------------------

// TimeRange represents a time interval for an experiment.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange validates and creates a TimeRange.
func NewTimeRange(start, end time.Time) (TimeRange, error) {
	if !start.Before(end) {
		return TimeRange{}, ErrInvalidTimeRange
	}
	return TimeRange{Start: start, End: end}, nil
}

// Duration returns the duration of the time range.
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// GeoLocation represents geographic coordinates.
type GeoLocation struct {
	Latitude  float32
	Longitude float32
}

// NewGeoLocation validates and creates a GeoLocation.
func NewGeoLocation(lat, lng float32) (GeoLocation, error) {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return GeoLocation{}, ErrInvalidGeoLocation
	}
	return GeoLocation{Latitude: lat, Longitude: lng}, nil
}

// ExperimentStorageRefs groups optional references to storage objects.
type ExperimentStorageRefs struct {
	ExperimentDataID *uuid.UUID // experiments_storage_id
	BackgroundID     *uuid.UUID // background_storage_id
	MeteoID          *uuid.UUID // meteo_storage_id
}

// ---------------------------------------------------------------------------
// Entity
// ---------------------------------------------------------------------------

// Experiment is the central domain entity representing a lidar measurement session.
type Experiment struct {
	ID                  uuid.UUID
	Title               string
	Comments            string
	ZenithAngle         float32
	TimeRange           TimeRange
	GeoLocation         GeoLocation
	AtmosphereProfileID uuid.UUID
	StorageRefs         ExperimentStorageRefs
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}

// ExperimentOption is a functional option for creating an Experiment.
type ExperimentOption func(*Experiment)

// WithComments sets the comments field.
func WithComments(comments string) ExperimentOption {
	return func(e *Experiment) { e.Comments = comments }
}

// WithStorageRefs sets the storage references.
func WithStorageRefs(refs ExperimentStorageRefs) ExperimentOption {
	return func(e *Experiment) { e.StorageRefs = refs }
}

// NewExperiment creates a new Experiment with a generated ID and the current timestamp.
func NewExperiment(
	title string,
	zenithAngle float32,
	timeRange TimeRange,
	location GeoLocation,
	atmosphereProfileID uuid.UUID,
	opts ...ExperimentOption,
) Experiment {
	now := time.Now()
	e := Experiment{
		ID:                  uuid.New(),
		Title:               title,
		ZenithAngle:         zenithAngle,
		TimeRange:           timeRange,
		GeoLocation:         location,
		AtmosphereProfileID: atmosphereProfileID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// UpdateDetails updates the mutable fields of the experiment.
func (e *Experiment) UpdateDetails(title string, zenithAngle float32, timeRange TimeRange, location GeoLocation) error {
	e.Title = title
	e.ZenithAngle = zenithAngle
	e.TimeRange = timeRange
	e.GeoLocation = location
	e.UpdatedAt = time.Now()
	return nil
}

// SetStorageRefs sets the storage references.
func (e *Experiment) SetStorageRefs(refs ExperimentStorageRefs) {
	e.StorageRefs = refs
	e.UpdatedAt = time.Now()
}

// SoftDelete marks the experiment as deleted.
func (e *Experiment) SoftDelete() {
	now := time.Now()
	e.DeletedAt = &now
	e.UpdatedAt = now
}

// Restore removes the soft-delete mark.
func (e *Experiment) Restore() {
	e.DeletedAt = nil
	e.UpdatedAt = time.Now()
}

// IsDeleted returns true if the experiment has been soft-deleted.
func (e *Experiment) IsDeleted() bool {
	return e.DeletedAt != nil
}
