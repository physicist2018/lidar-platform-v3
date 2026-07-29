package domain

import (
	"time"

	"github.com/google/uuid"
)

// BackgroundType represents the method for background subtraction.
type BackgroundType string

const (
	BackgroundFromFile BackgroundType = "file" // subtract matching background profile
	BackgroundMean     BackgroundType = "mean" // subtract mean of the tail
)

// PreparedMeta represents experiment-level processing settings for background
// removal and profile trimming.
type PreparedMeta struct {
	ID             uuid.UUID
	ExperimentID   uuid.UUID
	BackgroundType BackgroundType
	BackgroundFrom float32 // meters, start of the tail for mean calculation
	TrimFrom       float32 // meters, profiles are trimmed to this distance
}

// PreparedProfile is a processed profile with background removed and trimmed.
type PreparedProfile struct {
	ID             uuid.UUID
	PreparedMetaID uuid.UUID
	LicelProfileID uuid.UUID
	Data           []float32 // processed data (REAL[] in DB)
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewPreparedMeta creates a new PreparedMeta.
func NewPreparedMeta(
	experimentID uuid.UUID,
	backgroundType BackgroundType,
	backgroundFrom float32,
	trimFrom float32,
) PreparedMeta {
	return PreparedMeta{
		ExperimentID:   experimentID,
		BackgroundType: backgroundType,
		BackgroundFrom: backgroundFrom,
		TrimFrom:       trimFrom,
	}
}
