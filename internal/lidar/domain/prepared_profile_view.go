package domain

import (
	"time"

	"github.com/google/uuid"
)

// PreparedExperimentItem is a lightweight read model for experiments
// that have prepared profiles, used for dropdowns in the frontend.
type PreparedExperimentItem struct {
	ID              uuid.UUID
	Title           string
	ExperimentStart time.Time
	ExperimentEnd   time.Time
}

// PreparedProfileView is a read model representing a prepared profile
// with metadata from the original licel profile and processing settings.
type PreparedProfileView struct {
	ID               uuid.UUID
	Wavelength       float32
	Polarization     string
	DeviceID         string
	BinWidth         float32
	Data             []float32
	MeasurementStart time.Time
	BackgroundType   BackgroundType
	BackgroundFrom   float32
	TrimFrom         float32
	CreatedAt        time.Time
}
