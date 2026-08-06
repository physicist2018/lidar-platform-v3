package tikhlidar

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func batchInput(nt, n int, omega float64, timeNoise, seed int64) (BatchInput, BatchParams, []float64) {
	step := 5.0
	r := uniformRange(n, step)
	tm := uniformRange(nt, 60)
	href := 800.0

	// One shared true profile; each measured profile adds noise (and a
	// slowly varying temporal perturbation to test ω).
	rng := rand.New(rand.NewSource(seed))
	_, m, _, sTrue, _, _ := syntheticProfile(n, href, 0, seed)

	p := BatchParams{
		ProfileParams: ProfileParams{
			Wavelength:      532,
			AnchorRange:     [2]float64{1300, 1700},
			Href:            href,
			TransitionWidth: 60,
			AnchorStrength:  10,
			Lambda:          30,
		},
		Omega: omega,
	}

	signals := make([][]float64, nt)
	models := make([][]float64, nt)
	weights := make([][]float64, nt)
	for k := 0; k < nt; k++ {
		signals[k] = make([]float64, n)
		models[k] = make([]float64, n)
		weights[k] = make([]float64, n)
		copy(models[k], m)
		trend := 0.02 * math.Sin(float64(k)/float64(nt)*2*math.Pi)
		for i := 0; i < n; i++ {
			signals[k][i] = sTrue[i] * (1 + trend) * (1 + 0.03*rng.NormFloat64())
			weights[k][i] = 1
		}
	}
	return BatchInput{Time: tm, Range: r, Signals: signals, Models: models, Weights: weights}, p, sTrue
}

func TestSmoothBatch_OmegaZero_MatchesPerProfile(t *testing.T) {
	nt, n := 6, 401
	in, p, _ := batchInput(nt, n, 0, 0, 5)

	res, err := SmoothBatch(context.Background(), in, p)
	require.NoError(t, err)

	// Each batch result equals the independent per-profile smoothing.
	for k := 0; k < nt; k++ {
		single, err := SmoothProfile(context.Background(), ProfileInput{
			Range: in.Range, Signal: in.Signals[k], Model: in.Models[k], Weights: in.Weights[k],
		}, p.ProfileParams)
		require.NoError(t, err)
		for i := 0; i < n; i++ {
			assert.InDelta(t, single.SmoothedSignal[i], res.SmoothedSignal[k][i], 1e-8, "profile %d, r=%g", k, in.Range[i])
		}
		assert.InDelta(t, single.Calibration, res.Calibration[k], 1e-12)
	}
}

func TestSmoothBatch_OmegaReducesTemporalVariance(t *testing.T) {
	nt, n := 20, 401
	in, p, _ := batchInput(nt, n, 8, 0, 9)

	resNo, err := SmoothBatch(context.Background(), in, BatchParams{ProfileParams: p.ProfileParams, Omega: 0})
	require.NoError(t, err)
	resYes, err := SmoothBatch(context.Background(), in, p)
	require.NoError(t, err)

	// Temporal standard deviation of the smoothed signals at a fixed range
	// must drop when temporal smoothing is enabled.
	i := 150 // somewhere in the aerosol region
	std := func(res *BatchResult) float64 {
		var mean float64
		for k := 0; k < nt; k++ {
			mean += res.SmoothedSignal[k][i]
		}
		mean /= float64(nt)
		var v float64
		for k := 0; k < nt; k++ {
			d := res.SmoothedSignal[k][i] - mean
			v += d * d
		}
		return math.Sqrt(v / float64(nt))
	}
	assert.Less(t, std(resYes), std(resNo))
}
