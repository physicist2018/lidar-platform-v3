package domain

import "github.com/google/uuid"

// MatchStatus represents the result of matching a signal profile with its background.
type MatchStatus string

const (
	MatchOK            MatchStatus = "OK"
	NoBackground       MatchStatus = "NO_BACKGROUND"
	DataLengthMismatch MatchStatus = "MISMATCH"
)

// ProfileData contains the measurement data for a single LICEL profile.
type ProfileData struct {
	ProfileID   uuid.UUID
	LicelFileID uuid.UUID
	Data        []float64
	NumPoints   int32
	BinWidth    float32
}

// PairedProfile is a read model representing a signal profile paired with
// its matching background profile (by device_id, wavelength, polarization).
type PairedProfile struct {
	ExperimentID uuid.UUID
	DeviceID     string
	Wavelength   float32
	Polarization string
	Signal       ProfileData
	Background   *ProfileData // nil when no background exists
	MatchStatus  MatchStatus
}
