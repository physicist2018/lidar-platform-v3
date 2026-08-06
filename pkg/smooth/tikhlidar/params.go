package tikhlidar

// ProfileParams holds the smoothing parameters for a single profile.
type ProfileParams struct {
	// Wavelength is the laser wavelength in nanometers. It is currently
	// informational: the molecular model signal M(r) is expected to be
	// provided ready-made by the caller.
	Wavelength float64

	// AnchorRange is the range interval [r0, r1] in meters used to compute
	// the anchoring (calibration) constant C. Inside the interval the
	// aerosol backscatter is assumed absent, so S ≈ C·M.
	AnchorRange [2]float64

	// Href is the reference height in meters: the center of the sigmoid
	// transition. Below it the aerosol region is retrieved freely; above it
	// the signal is pulled towards the molecular profile.
	Href float64

	// TransitionWidth is the width L in meters of the sigmoid transition
	// zone around Href. Zero yields a step function at Href.
	TransitionWidth float64

	// AnchorStrength is the strength q of the pull towards the molecular
	// profile in the far zone (q >= 0).
	AnchorStrength float64

	// Lambda is the distance smoothing parameter (λ >= 0). It scales the
	// penalty on the second derivative of the smoothed signal.
	Lambda float64
}

// BatchParams holds the smoothing parameters for a batch of profiles.
type BatchParams struct {
	ProfileParams
	// Omega is the temporal smoothing parameter (ω >= 0). It scales the
	// penalty on the second time derivative across consecutive profiles.
	// Ignored when the batch has fewer than three profiles.
	Omega float64
}
