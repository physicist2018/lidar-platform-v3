package tikhlidar

import (
	"fmt"
	"sort"
)

// calibrationConstant computes the anchoring constant
//
//	C = median_{i: rᵢ ∈ [r0, r1]} Sᵢ/Mᵢ,
//
// the robust median of the measured-to-model signal ratio over the anchor
// range, where the aerosol backscatter is assumed absent (S ≈ C·M).
func calibrationConstant(r, s, m []float64, r0, r1 float64) (float64, error) {
	ratios := make([]float64, 0, len(r))
	for i := range r {
		if r[i] >= r0 && r[i] <= r1 {
			ratios = append(ratios, s[i]/m[i])
		}
	}
	if len(ratios) < 2 {
		return 0, fmt.Errorf("%w: [%g, %g]", ErrAnchorRange, r0, r1)
	}
	sort.Float64s(ratios)
	if len(ratios)%2 == 1 {
		return ratios[len(ratios)/2], nil
	}
	hi := len(ratios) / 2
	return (ratios[hi-1] + ratios[hi]) / 2, nil
}
