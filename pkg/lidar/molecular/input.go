package molecular

// AtmosphereModel is a vertical profile of the atmosphere. All slices have
// the same length and Altitude is strictly increasing.
type AtmosphereModel struct {
	Altitude    []float64 // km
	Temperature []float64 // °C
	Pressure    []float64 // hPa
}

// Input is the input for the molecular signal computation.
type Input struct {
	// Range is the line-of-sight range grid in meters, strictly increasing
	// and non-negative.
	Range []float64

	// ZenithAngle is the zenith angle in degrees (0 = pointing straight up,
	// 90 = horizontal).
	ZenithAngle float64

	// Wavelength is the laser wavelength in nanometers (must be > 160 nm for
	// the Edlén dispersion formula).
	Wavelength float64

	// Atmosphere is the atmosphere model (temperature and pressure profiles).
	Atmosphere AtmosphereModel
}

// Result holds the computed molecular profiles. All slices have the same
// length as Range.
type Result struct {
	// Backscatter is the molecular backscatter coefficient βm(r), m⁻¹·sr⁻¹.
	Backscatter []float64

	// Extinction is the molecular extinction coefficient αm(r), m⁻¹.
	Extinction []float64

	// Transmission is the two-way molecular transmission T_m²(r).
	Transmission []float64

	// Signal is the range-corrected molecular lidar signal
	// M(r) = βm(r)·T_m²(r), m⁻¹·sr⁻¹. The instrument calibration constant
	// is intentionally not included: it is estimated by the consumer (e.g.
	// the tikhlidar package calibrates it in its anchor range).
	Signal []float64
}
