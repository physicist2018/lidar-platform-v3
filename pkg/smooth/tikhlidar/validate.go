package tikhlidar

import (
	"fmt"
	"math"
)

// validateRangeGrid checks that r is strictly increasing, finite and has at
// least three points.
func validateRangeGrid(r []float64) error {
	if len(r) < 3 {
		return ErrTooFewPoints
	}
	for i := range r {
		if math.IsNaN(r[i]) || math.IsInf(r[i], 0) {
			return fmt.Errorf("%w: range[%d]", ErrNonFinite, i)
		}
		if i > 0 && r[i] <= r[i-1] {
			return fmt.Errorf("%w: range[%d]=%g <= range[%d]=%g", ErrRangeNotIncreasing, i, r[i], i-1, r[i-1])
		}
	}
	return nil
}

func validateProfile(in ProfileInput, p ProfileParams) error {
	if in.Range == nil || in.Signal == nil || in.Model == nil || in.Weights == nil {
		return ErrNilInput
	}
	if len(in.Range) == 0 {
		return ErrEmptyInput
	}
	if err := validateRangeGrid(in.Range); err != nil {
		return err
	}
	n := len(in.Range)
	if len(in.Signal) != n || len(in.Model) != n || len(in.Weights) != n {
		return fmt.Errorf("%w: range=%d, signal=%d, model=%d, weights=%d", ErrLengthMismatch, n, len(in.Signal), len(in.Model), len(in.Weights))
	}
	for i := range in.Signal {
		if math.IsNaN(in.Signal[i]) || math.IsInf(in.Signal[i], 0) {
			return fmt.Errorf("%w: signal[%d]", ErrNonFinite, i)
		}
		if in.Signal[i] < 0 {
			return fmt.Errorf("%w: signal[%d]=%g", ErrSignalNegative, i, in.Signal[i])
		}
	}
	for i := range in.Model {
		if math.IsNaN(in.Model[i]) || math.IsInf(in.Model[i], 0) {
			return fmt.Errorf("%w: model[%d]", ErrNonFinite, i)
		}
		if in.Model[i] <= 0 {
			return fmt.Errorf("%w: model[%d]=%g", ErrModelNonPositive, i, in.Model[i])
		}
	}
	for i := range in.Weights {
		if math.IsNaN(in.Weights[i]) || math.IsInf(in.Weights[i], 0) {
			return fmt.Errorf("%w: weights[%d]", ErrNonFinite, i)
		}
		if in.Weights[i] < 0 {
			return fmt.Errorf("%w: weights[%d]=%g", ErrInvalidParam, i, in.Weights[i])
		}
	}
	if err := validateParams(p); err != nil {
		return err
	}
	return validateCommon(p, in.Range)
}

// validateParams checks the profile parameters.
func validateParams(p ProfileParams) error {
	if p.Lambda < 0 || p.AnchorStrength < 0 || p.TransitionWidth < 0 {
		return fmt.Errorf("%w: lambda, q and L must be >= 0", ErrInvalidParam)
	}
	return nil
}

// validateCommon checks parameters that depend on the grid.
func validateCommon(p ProfileParams, r []float64) error {
	r0, r1 := p.AnchorRange[0], p.AnchorRange[1]
	if r0 >= r1 {
		return fmt.Errorf("%w: r0=%g >= r1=%g", ErrInvalidParam, r0, r1)
	}
	if p.Href < r[0] || p.Href > r[len(r)-1] {
		return fmt.Errorf("%w: href=%g outside range [%g, %g]", ErrInvalidParam, p.Href, r[0], r[len(r)-1])
	}
	count := 0
	for i := range r {
		if r[i] >= r0 && r[i] <= r1 {
			count++
		}
	}
	if count < 2 {
		return fmt.Errorf("%w: [%g, %g]", ErrAnchorRange, r0, r1)
	}
	return nil
}

func validateBatch(in BatchInput, p BatchParams) error {
	if in.Time == nil || in.Range == nil || in.Signals == nil || in.Models == nil || in.Weights == nil {
		return ErrNilInput
	}
	if len(in.Time) == 0 || len(in.Range) == 0 {
		return ErrEmptyInput
	}
	if err := validateRangeGrid(in.Range); err != nil {
		return err
	}
	nt, n := len(in.Time), len(in.Range)
	if len(in.Signals) != nt || len(in.Models) != nt || len(in.Weights) != nt {
		return fmt.Errorf("%w: time=%d, signals=%d, models=%d, weights=%d", ErrLengthMismatch, nt, len(in.Signals), len(in.Models), len(in.Weights))
	}
	for k := 0; k < nt; k++ {
		if math.IsNaN(in.Time[k]) || math.IsInf(in.Time[k], 0) {
			return fmt.Errorf("%w: time[%d]", ErrNonFinite, k)
		}
		if k > 0 && in.Time[k] <= in.Time[k-1] {
			return fmt.Errorf("%w: time[%d]=%g <= time[%d]=%g", ErrTimeNotIncreasing, k, in.Time[k], k-1, in.Time[k-1])
		}
		if len(in.Signals[k]) != n || len(in.Models[k]) != n || len(in.Weights[k]) != n {
			return fmt.Errorf("%w: profile %d: got %d, %d and %d points, want %d", ErrProfileLength, k, len(in.Signals[k]), len(in.Models[k]), len(in.Weights[k]), n)
		}
		for i := range in.Signals[k] {
			if math.IsNaN(in.Signals[k][i]) || math.IsInf(in.Signals[k][i], 0) {
				return fmt.Errorf("%w: signals[%d][%d]", ErrNonFinite, k, i)
			}
			if in.Signals[k][i] < 0 {
				return fmt.Errorf("%w: signals[%d][%d]=%g", ErrSignalNegative, k, i, in.Signals[k][i])
			}
		}
		for i := range in.Models[k] {
			if math.IsNaN(in.Models[k][i]) || math.IsInf(in.Models[k][i], 0) {
				return fmt.Errorf("%w: models[%d][%d]", ErrNonFinite, k, i)
			}
			if in.Models[k][i] <= 0 {
				return fmt.Errorf("%w: models[%d][%d]=%g", ErrModelNonPositive, k, i, in.Models[k][i])
			}
		}
		for i := range in.Weights[k] {
			if math.IsNaN(in.Weights[k][i]) || math.IsInf(in.Weights[k][i], 0) {
				return fmt.Errorf("%w: weights[%d][%d]", ErrNonFinite, k, i)
			}
			if in.Weights[k][i] < 0 {
				return fmt.Errorf("%w: weights[%d][%d]=%g", ErrInvalidParam, k, i, in.Weights[k][i])
			}
		}
	}
	if p.Lambda < 0 || p.AnchorStrength < 0 || p.TransitionWidth < 0 || p.Omega < 0 {
		return fmt.Errorf("%w: lambda, q, L and omega must be >= 0", ErrInvalidParam)
	}
	return validateCommon(p.ProfileParams, in.Range)
}
