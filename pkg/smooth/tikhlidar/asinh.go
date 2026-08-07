package tikhlidar

import (
	"context"
	"fmt"
	"math"
)

// SmoothProfileAsinh smooths a single profile in the asinh (log-like) domain.
//
// The signal and the molecular model are first brought to a common scale by
// the calibration constant C = median(S/M) over the anchor range [r0, r1]
// (computed in the original domain), then transformed with the
// variance-stabilizing transform
//
//	S_t = asinh((S/C)/eps),   M_t = asinh(M/eps),
//
// smoothed by SmoothProfile with the weight profile set to |S_t|, and finally
// transformed back to the original calibrated units:
//
//	Ŝ = C·eps·sinh(Ŝ_t).
//
// For |x| ≫ eps the transform behaves as ln(x), compressing the dynamic range
// of the signal; for |x| ≪ eps it is linear, so weak and slightly negative
// signals (e.g. after background subtraction) are handled gracefully. eps is a
// small regularization parameter (typically 1e-6 .. 1e-3) relative to the
// model scale.
//
// Any caller-provided Weights are ignored: the helper derives them from the
// transformed signal (|S_t|, which equals S_t for non-negative signals).
// The returned Calibration is the original-domain constant C, and the
// SmoothedSignal is in the original (calibrated) units; the Residual refers to
// the transformed-domain fit.
func SmoothProfileAsinh(ctx context.Context, in ProfileInput, p ProfileParams, eps float64) (*ProfileResult, error) {
	if eps <= 0 {
		return nil, fmt.Errorf("%w: eps=%g must be positive", ErrInvalidParam, eps)
	}
	// validateProfile requires a weight profile; the helper derives it from
	// the transformed signal, so validate with a placeholder profile.
	check := in
	check.Weights = make([]float64, len(in.Range))
	if err := validateProfile(check, p); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := len(in.Range)

	// Calibration constant in the original domain: brings the signal to the
	// model scale, so the log-domain calibration offset vanishes.
	c, err := calibrationConstant(in.Range, in.Signal, in.Model, p.AnchorRange[0], p.AnchorRange[1])
	if err != nil {
		return nil, err
	}
	if c <= 0 {
		return nil, fmt.Errorf("%w: calibration constant C=%g must be positive", ErrInvalidParam, c)
	}

	sT := make([]float64, n)
	mT := make([]float64, n)
	w := make([]float64, n)
	for i := 0; i < n; i++ {
		sT[i] = math.Asinh(in.Signal[i] / c / eps)
		mT[i] = math.Asinh(in.Model[i] / eps)
		w[i] = math.Abs(sT[i])
	}

	res, err := SmoothProfile(ctx, ProfileInput{Range: in.Range, Signal: sT, Model: mT, Weights: w}, p)
	if err != nil {
		return nil, err
	}

	for i := range res.SmoothedSignal {
		res.SmoothedSignal[i] = c * eps * math.Sinh(res.SmoothedSignal[i])
	}
	res.Calibration = c
	return res, nil
}

// SmoothBatchAsinh is the batch counterpart of SmoothProfileAsinh: each
// profile is normalized with its own calibration constant C_k =
// median(S_k/M_k) over [r0, r1], transformed with asinh, smoothed by
// SmoothBatch (with per-profile weights |S_t,k|) and transformed back.
func SmoothBatchAsinh(ctx context.Context, in BatchInput, p BatchParams, eps float64) (*BatchResult, error) {
	if eps <= 0 {
		return nil, fmt.Errorf("%w: eps=%g must be positive", ErrInvalidParam, eps)
	}
	check := in
	check.Weights = make([][]float64, len(in.Time))
	for k := range check.Weights {
		check.Weights[k] = make([]float64, len(in.Range))
	}
	if err := validateBatch(check, p); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	nt, n := len(in.Time), len(in.Range)

	cs := make([]float64, nt)
	for k := 0; k < nt; k++ {
		c, err := calibrationConstant(in.Range, in.Signals[k], in.Models[k], p.AnchorRange[0], p.AnchorRange[1])
		if err != nil {
			return nil, fmt.Errorf("profile %d: %w", k, err)
		}
		if c <= 0 {
			return nil, fmt.Errorf("profile %d: %w: calibration constant C=%g must be positive", k, ErrInvalidParam, c)
		}
		cs[k] = c
	}

	sT := make([][]float64, nt)
	mT := make([][]float64, nt)
	w := make([][]float64, nt)
	for k := 0; k < nt; k++ {
		sT[k] = make([]float64, n)
		mT[k] = make([]float64, n)
		w[k] = make([]float64, n)
		for i := 0; i < n; i++ {
			sT[k][i] = math.Asinh(in.Signals[k][i] / cs[k] / eps)
			mT[k][i] = math.Asinh(in.Models[k][i] / eps)
			w[k][i] = math.Abs(sT[k][i])
		}
	}

	res, err := SmoothBatch(ctx, BatchInput{
		Time: in.Time, Range: in.Range, Signals: sT, Models: mT, Weights: w,
	}, p)
	if err != nil {
		return nil, err
	}

	for k := 0; k < nt; k++ {
		for i := 0; i < n; i++ {
			res.SmoothedSignal[k][i] = cs[k] * eps * math.Sinh(res.SmoothedSignal[k][i])
		}
		res.Calibration[k] = cs[k]
	}
	return res, nil
}
