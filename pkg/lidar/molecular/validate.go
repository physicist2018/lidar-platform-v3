package molecular

import (
	"fmt"
	"math"
)

// validate checks the input before computation.
func validate(in Input) error {
	if in.Range == nil || in.Atmosphere.Altitude == nil ||
		in.Atmosphere.Temperature == nil || in.Atmosphere.Pressure == nil {
		return ErrNilInput
	}
	n := len(in.Range)
	if n < 2 {
		return ErrTooFewPoints
	}
	for i := range in.Range {
		if math.IsNaN(in.Range[i]) || math.IsInf(in.Range[i], 0) {
			return fmt.Errorf("%w: range[%d]", ErrNonFinite, i)
		}
		if in.Range[i] < 0 {
			return fmt.Errorf("%w: range[%d]=%g", ErrInvalidParam, i, in.Range[i])
		}
		if i > 0 && in.Range[i] <= in.Range[i-1] {
			return fmt.Errorf("%w: range[%d]=%g <= range[%d]=%g", ErrRangeNotIncreasing, i, in.Range[i], i-1, in.Range[i-1])
		}
	}

	if in.Wavelength <= 160 {
		return fmt.Errorf("%w: wavelength=%g nm must be > 160 nm (Edlén validity range)", ErrInvalidParam, in.Wavelength)
	}
	if math.IsNaN(in.Wavelength) || math.IsInf(in.Wavelength, 0) {
		return fmt.Errorf("%w: wavelength", ErrNonFinite)
	}
	if in.ZenithAngle < 0 || in.ZenithAngle > 90 {
		return fmt.Errorf("%w: zenith angle=%g must be in [0, 90] degrees", ErrInvalidParam, in.ZenithAngle)
	}
	if math.IsNaN(in.ZenithAngle) || math.IsInf(in.ZenithAngle, 0) {
		return fmt.Errorf("%w: zenith angle", ErrNonFinite)
	}

	return validateModel(in.Atmosphere)
}

// validateModel checks the atmosphere model arrays.
func validateModel(m AtmosphereModel) error {
	na := len(m.Altitude)
	if na < 2 {
		return ErrTooFewPoints
	}
	if len(m.Temperature) != na || len(m.Pressure) != na {
		return fmt.Errorf("%w: altitude=%d, temperature=%d, pressure=%d",
			ErrLengthMismatch, na, len(m.Temperature), len(m.Pressure))
	}
	for i := range m.Altitude {
		if math.IsNaN(m.Altitude[i]) || math.IsInf(m.Altitude[i], 0) ||
			math.IsNaN(m.Temperature[i]) || math.IsInf(m.Temperature[i], 0) ||
			math.IsNaN(m.Pressure[i]) || math.IsInf(m.Pressure[i], 0) {
			return fmt.Errorf("%w: model level %d", ErrNonFinite, i)
		}
		if i > 0 && m.Altitude[i] <= m.Altitude[i-1] {
			return fmt.Errorf("%w: altitude[%d]=%g <= altitude[%d]=%g",
				ErrRangeNotIncreasing, i, m.Altitude[i], i-1, m.Altitude[i-1])
		}
		if m.Temperature[i] <= -273.15 {
			return fmt.Errorf("%w: temperature[%d]=%g °C is below absolute zero", ErrInvalidModel, i, m.Temperature[i])
		}
		if m.Pressure[i] <= 0 {
			return fmt.Errorf("%w: pressure[%d]=%g hPa must be positive", ErrInvalidModel, i, m.Pressure[i])
		}
	}
	return nil
}
