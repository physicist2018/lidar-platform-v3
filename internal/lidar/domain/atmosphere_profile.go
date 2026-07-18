package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AtmosphereProfile is an entity representing a vertical profile of atmosphere measurements.
type AtmosphereProfile struct {
	ID          uuid.UUID
	Altitude    []float64 // km
	Temperature []float64 // °C
	Pressure    []float64 // hPa
	CreatedAt   time.Time
}

// NewAtmosphereProfile creates a new AtmosphereProfile with validated data arrays.
func NewAtmosphereProfile(altitude, temperature, pressure []float64) (AtmosphereProfile, error) {
	if len(altitude) != len(temperature) || len(altitude) != len(pressure) {
		return AtmosphereProfile{}, fmt.Errorf("%w: got alt=%d, temp=%d, pres=%d",
			ErrProfileDataMismatch, len(altitude), len(temperature), len(pressure))
	}
	return AtmosphereProfile{
		ID:          uuid.New(),
		Altitude:    altitude,
		Temperature: temperature,
		Pressure:    pressure,
		CreatedAt:   time.Now(),
	}, nil
}

// NumPoints returns the number of data points in the profile.
func (p *AtmosphereProfile) NumPoints() int {
	return len(p.Altitude)
}

// PointAt returns the altitude, temperature, and pressure at the given index.
func (p *AtmosphereProfile) PointAt(idx int) (altitude, temperature, pressure float64, err error) {
	if idx < 0 || idx >= len(p.Altitude) {
		return 0, 0, 0, fmt.Errorf("index %d out of range [0, %d)", idx, len(p.Altitude))
	}
	return p.Altitude[idx], p.Temperature[idx], p.Pressure[idx], nil
}
