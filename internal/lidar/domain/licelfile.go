package domain

import (
	"time"

	"github.com/google/uuid"
)

// LicelFile is an entity representing a raw LICEL data file from an experiment.
type LicelFile struct {
	ID               uuid.UUID
	ExperimentID     uuid.UUID
	MeasurementRange TimeRange
	NDatasets        int32
	LaserFreq        int32
	IsBackground     bool
	RawStorageID     uuid.UUID
	Filename         string
	CreatedAt        time.Time
	DeletedAt        *time.Time
}

// LicelFileOption is a functional option for creating a LicelFile.
type LicelFileOption func(*LicelFile)

// WithFilename sets the filename.
func WithFilename(name string) LicelFileOption {
	return func(f *LicelFile) { f.Filename = name }
}

// NewLicelFile creates a new LicelFile with a generated ID.
func NewLicelFile(
	experimentID uuid.UUID,
	measurementRange TimeRange,
	nDatasets int32,
	laserFreq int32,
	isBackground bool,
	rawStorageID uuid.UUID,
	opts ...LicelFileOption,
) LicelFile {
	f := LicelFile{
		ID:               uuid.New(),
		ExperimentID:     experimentID,
		MeasurementRange: measurementRange,
		NDatasets:        nDatasets,
		LaserFreq:        laserFreq,
		IsBackground:     isBackground,
		RawStorageID:     rawStorageID,
		CreatedAt:        time.Now(),
	}
	for _, opt := range opts {
		opt(&f)
	}
	return f
}

// SoftDelete marks the file as deleted.
func (f *LicelFile) SoftDelete() {
	now := time.Now()
	f.DeletedAt = &now
}

// Restore removes the soft-delete mark.
func (f *LicelFile) Restore() {
	f.DeletedAt = nil
}

// IsDeleted returns true if the file has been soft-deleted.
func (f *LicelFile) IsDeleted() bool {
	return f.DeletedAt != nil
}
