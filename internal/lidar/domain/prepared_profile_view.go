package domain

import (
	"time"

	"github.com/google/uuid"
)

// PreparedProfileView is a read model representing a prepared profile
// with metadata from the original licel profile and processing settings.
type PreparedProfileView struct {
	ID             uuid.UUID
	Wavelength     float32
	Polarization   string
	DeviceID       string
	BinWidth       float32
	Data           []float32
	BackgroundType BackgroundType
	BackgroundFrom float32
	TrimFrom       float32
	CreatedAt      time.Time
}
