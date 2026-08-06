package tikhlidar

import "math"

// anchorWeight returns the logistic sigmoid anchor weight
//
//	w(r) = 1 / (1 + exp(−(r − Href)/s)),   s = L/4.
//
// w→0 well below Href (aerosol region, free retrieval) and w→1 well above
// Href (molecular region, pull towards the molecular profile). With L = 0 the
// weight degenerates to a step function at Href.
func anchorWeight(r, href, width float64) float64 {
	if width <= 0 {
		switch {
		case r < href:
			return 0
		case r > href:
			return 1
		default:
			return 0.5
		}
	}
	s := width / 4
	return 1 / (1 + math.Exp(-(r-href)/s))
}

// anchorWeights computes the anchor weights for a range grid.
func anchorWeights(r []float64, href, width float64) []float64 {
	w := make([]float64, len(r))
	for i := range r {
		w[i] = anchorWeight(r[i], href, width)
	}
	return w
}
