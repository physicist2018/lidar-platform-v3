package molecular

import "sort"

// interpolate returns the linearly interpolated value of y at xv on the grid
// (x, y). When xv falls outside the grid, the value is clamped to the nearest
// edge of the grid and the returned flag reports the side (true for clamped).
func interpolate(x, y []float64, xv float64) (v float64, clamped bool) {
	n := len(x)
	if xv <= x[0] {
		return y[0], xv < x[0]
	}
	if xv >= x[n-1] {
		return y[n-1], xv > x[n-1]
	}
	// First index with x[i] >= xv; 1 <= i <= n-1 given the guards above.
	i := sort.SearchFloat64s(x, xv)
	t := (xv - x[i-1]) / (x[i] - x[i-1])
	return y[i-1] + t*(y[i]-y[i-1]), false
}
