package tikhlidar

// ProfileResult is the result of smoothing a single profile.
type ProfileResult struct {
	// SmoothedSignal is the smoothed range-corrected backscatter signal Ŝ(r).
	// In the far zone (r ≫ Href) it equals the scaled molecular signal C·M(r).
	SmoothedSignal []float64

	// Calibration is the anchoring constant C computed in [r0, r1].
	Calibration float64

	// Residual is the relative residual of the fit term,
	// Σ (1−w)(S−Ŝ)² / Σ (1−w)S².
	Residual float64
}

// BatchResult is the result of smoothing a batch of profiles (nt×n).
type BatchResult struct {
	// SmoothedSignal[k] is the smoothed signal of profile k (length n).
	SmoothedSignal [][]float64

	// Calibration[k] is the anchoring constant of profile k.
	Calibration []float64

	// Residual[k] is the relative fit residual of profile k.
	Residual []float64
}
