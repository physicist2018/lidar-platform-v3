package tikhlidar

import (
	"context"
	"fmt"
)

// SmoothProfile performs Tikhonov smoothing of a single range-corrected
// backscatter signal with a sigmoid anchor to the molecular profile.
//
// # The result Ŝ minimizes the objective
//
// Φ(Ŝ) = Σᵢ (1−wᵢ)·uᵢ·(Sᵢ − Ŝᵢ)² + λ²·Σᵢ (D²_r Ŝ)ᵢ² + q·Σᵢ wᵢ·(Ŝᵢ − C·Mᵢ)²,
//
// where w is the logistic sigmoid around Href with transition width L and C is
// the anchoring constant estimated in [r0, r1].
func SmoothProfile(ctx context.Context, in ProfileInput, p ProfileParams) (*ProfileResult, error) {
	if err := validateProfile(in, p); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	n := len(in.Range)

	// Sigmoid anchor weights.
	w := anchorWeights(in.Range, p.Href, p.TransitionWidth)

	// Anchoring constant from [r0, r1].
	c, err := calibrationConstant(in.Range, in.Signal, in.Model, p.AnchorRange[0], p.AnchorRange[1])
	if err != nil {
		return nil, err
	}

	// Assemble the linear system.
	diag := make([]float64, n)
	b := make([]float64, n)
	for i := 0; i < n; i++ {
		fitW := (1 - w[i]) * in.Weights[i]
		diag[i] = fitW + p.AnchorStrength*w[i]
		b[i] = fitW*in.Signal[i] + p.AnchorStrength*w[i]*c*in.Model[i]
	}

	band := assembleProfileSystem(n, diag, p.Lambda, d2Td2(secondDiffRows(in.Range)))
	x, err := solveBanded(band, b)
	if err != nil {
		return nil, fmt.Errorf("smooth profile: %w", err)
	}

	return &ProfileResult{
		SmoothedSignal: x,
		Calibration:    c,
		Residual:       fitResidual(w, in.Weights, in.Signal, x),
	}, nil
}

// fitResidual computes the relative fit residual
// Σ (1−w)·u·(S−Ŝ)² / Σ (1−w)·u·S².
func fitResidual(w, u, s, shat []float64) float64 {
	var num, den float64
	for i := range s {
		fw := (1 - w[i]) * u[i]
		d := s[i] - shat[i]
		num += fw * d * d
		den += fw * s[i] * s[i]
	}
	if den == 0 {
		return 0
	}
	return num / den
}
