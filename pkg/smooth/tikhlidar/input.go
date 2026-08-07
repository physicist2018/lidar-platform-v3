package tikhlidar

// ProfileInput is the input for smoothing a single profile.
// All slices have the same length n.
type ProfileInput struct {
	// Range is the range grid in meters, strictly increasing.
	Range []float64

	// Signal is the measured backscatter signal corrected for range squared
	// (×r²), NOT corrected for transmission T². Negative values (e.g. after
	// background subtraction) are allowed.
	Signal []float64

	// Model is the modeled pure molecular backscatter signal (×r²).
	Model []float64

	// Weights is a per-point weight profile u(r) ≥ 0 that multiplies the fit
	// term of the objective. Use it to down-weight noisy or unreliable range
	// bins (e.g. by signal quality); uᵢ = 0 turns the fit off at that point.
	Weights []float64
}

// BatchInput is the input for smoothing a batch of profiles on a common
// range grid. Signals, Models and Weights are nt×n matrices (row-major:
// profile k is Signals[k], length n).
type BatchInput struct {
	// Time is the time of each profile in seconds, strictly increasing.
	Time []float64

	// Range is the common range grid in meters, strictly increasing.
	Range []float64

	// Signals[k] is the measured signal of profile k (length n).
	Signals [][]float64

	// Models[k] is the modeled molecular signal of profile k (length n).
	Models [][]float64

	// Weights[k] is the per-point weight profile of profile k (length n).
	Weights [][]float64
}
