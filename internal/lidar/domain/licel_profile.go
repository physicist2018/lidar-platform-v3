package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// LicelProfile is an entity representing a parsed profile from a LICEL file.
type LicelProfile struct {
	ID           uuid.UUID
	LicelFileID  uuid.UUID
	NDataPoints  int32
	HighVoltage  float32
	BinWidth     float32
	Wavelength   float32
	Polarization string
	DeviceID     string
	NShots       int32
	DiscrLevel   float32
	Data         []float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// LicelProfileOption is a functional option for creating a LicelProfile.
type LicelProfileOption func(*LicelProfile)

// NewLicelProfile creates a new LicelProfile, validating that len(Data) matches NDataPoints.
func NewLicelProfile(
	licelFileID uuid.UUID,
	nDataPoints int32,
	highVoltage, binWidth, wavelength float32,
	polarization, deviceID string,
	nShots int32,
	discrLevel float32,
	data []float64,
	opts ...LicelProfileOption,
) (LicelProfile, error) {
	if int32(len(data)) != nDataPoints {
		return LicelProfile{}, fmt.Errorf(
			"%w: nDataPoints=%d, len(data)=%d",
			ErrProfileDataMismatch, nDataPoints, len(data))
	}

	now := time.Now()
	p := LicelProfile{
		ID:           uuid.New(),
		LicelFileID:  licelFileID,
		NDataPoints:  nDataPoints,
		HighVoltage:  highVoltage,
		BinWidth:     binWidth,
		Wavelength:   wavelength,
		Polarization: polarization,
		DeviceID:     deviceID,
		NShots:       nShots,
		DiscrLevel:   discrLevel,
		Data:         data,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p, nil
}

// NumPoints returns the number of data points.
func (p *LicelProfile) NumPoints() int {
	return len(p.Data)
}

// PointAt returns the data value at the given index.
func (p *LicelProfile) PointAt(idx int) (float64, error) {
	if idx < 0 || idx >= len(p.Data) {
		return 0, fmt.Errorf("index %d out of range [0, %d)", idx, len(p.Data))
	}
	return p.Data[idx], nil
}

// SoftDelete marks the profile as deleted.
func (p *LicelProfile) SoftDelete() {
	now := time.Now()
	p.DeletedAt = &now
	p.UpdatedAt = now
}

// Restore removes the soft-delete mark.
func (p *LicelProfile) Restore() {
	p.DeletedAt = nil
	p.UpdatedAt = time.Now()
}

// IsDeleted returns true if the profile has been soft-deleted.
func (p *LicelProfile) IsDeleted() bool {
	return p.DeletedAt != nil
}
