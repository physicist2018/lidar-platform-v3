package tikhlidar

import (
	"context"
	"fmt"
)

// SmoothBatch performs Tikhonov smoothing of a batch of range-corrected
// backscatter signals on a common range grid, with additional temporal
// smoothing controlled by ω. Each profile gets its own anchoring constant
// C_k from its own anchor range [r0, r1].
//
// When the batch has fewer than three profiles, or ω = 0, the profiles are
// smoothed independently.
func SmoothBatch(ctx context.Context, in BatchInput, p BatchParams) (*BatchResult, error) {
	if err := validateBatch(in, p); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	nt, n := len(in.Time), len(in.Range)

	// Sigmoid anchor weights (common grid, common parameters).
	w := anchorWeights(in.Range, p.Href, p.TransitionWidth)

	// Anchoring constant per profile.
	cs := make([]float64, nt)
	for k := 0; k < nt; k++ {
		c, err := calibrationConstant(in.Range, in.Signals[k], in.Models[k], p.AnchorRange[0], p.AnchorRange[1])
		if err != nil {
			return nil, fmt.Errorf("profile %d: %w", k, err)
		}
		cs[k] = c
	}

	// Per-profile diagonal and right-hand side.
	diag := make([][]float64, nt)
	b := make([]float64, nt*n)
	for k := 0; k < nt; k++ {
		diag[k] = make([]float64, n)
		for i := 0; i < n; i++ {
			fitW := (1 - w[i]) * in.Weights[k][i]
			diag[k][i] = fitW + p.AnchorStrength*w[i]
			b[k*n+i] = fitW*in.Signals[k][i] + p.AnchorStrength*w[i]*cs[k]*in.Models[k][i]
		}
	}

	res := &BatchResult{
		SmoothedSignal: make([][]float64, nt),
		Calibration:    cs,
		Residual:       make([]float64, nt),
	}

	// No temporal coupling: solve nt independent pentadiagonal systems.
	if nt < 3 || p.Omega <= 0 {
		d2Rows := secondDiffRows(in.Range)
		for k := 0; k < nt; k++ {
			band := assembleProfileSystem(n, diag[k], p.Lambda, d2Td2(d2Rows))
			xk, err := solveBanded(band, b[k*n:(k+1)*n])
			if err != nil {
				return nil, fmt.Errorf("profile %d: %w", k, err)
			}
			res.SmoothedSignal[k] = xk
			res.Residual[k] = fitResidual(w, in.Weights[k], in.Signals[k], xk)
		}
		return res, nil
	}

	// Full batch: matrix-free conjugate gradients on the block-structured
	// system A = I⊗P + ω²·(D2_tᵀD2_t)⊗I.
	op := newBatchOperator(nt, n, diag, p.Lambda, p.Omega, secondDiffRows(in.Range), secondDiffRows(in.Time))
	x, _, err := conjugateGradient(ctx, op.MulVec, nt*n, b, batchMaxIter, batchTol)
	if err != nil {
		return nil, fmt.Errorf("smooth batch: %w", err)
	}
	for k := 0; k < nt; k++ {
		res.SmoothedSignal[k] = x[k*n : (k+1)*n]
		res.Residual[k] = fitResidual(w, in.Weights[k], in.Signals[k], res.SmoothedSignal[k])
	}
	return res, nil
}
