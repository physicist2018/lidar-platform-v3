package tikhlidar

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// asinhTestData builds a scenario with a large dynamic range and a
// calibration constant far from one: signal in "instrument counts"
// (C₀ = 1e5) while the model is in physical units (~10..100).
func asinhTestData(n int, seed int64) (r, m, s, sTrue []float64, c0 float64) {
	r = uniformRange(n, 5)
	m = make([]float64, n)
	sTrue = make([]float64, n)
	c0 = 1e5
	rng := rand.New(rand.NewSource(seed))

	for i := 0; i < n; i++ {
		m[i] = 100*math.Exp(-r[i]/900) + 5
		ratio := 1 + 1.5*math.Exp(-math.Pow((r[i]-350)/120, 2))
		sTrue[i] = c0 * m[i] * ratio
	}
	s = make([]float64, n)
	for i := range s {
		s[i] = sTrue[i] * (1 + 0.02*rng.NormFloat64())
	}
	return r, m, s, sTrue, c0
}

func asinhParams() ProfileParams {
	return ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{1300, 1700},
		Href:            800,
		TransitionWidth: 60,
		AnchorStrength:  10,
		Lambda:          30,
	}
}

func TestAsinhRoundTrip(t *testing.T) {
	// eps·sinh(asinh(x/eps)) = x.
	eps := 1e-3
	for _, x := range []float64{0, 1e-9, 1e-3, 0.5, 10, 1e4} {
		assert.InDelta(t, x, eps*math.Sinh(math.Asinh(x/eps)), math.Abs(x)*1e-12, "x=%g", x)
	}
}

func TestSmoothProfileAsinh_FarZone(t *testing.T) {
	n := 401
	r, m, s, sTrue, c0 := asinhTestData(n, 7)

	res, err := SmoothProfileAsinh(context.Background(),
		ProfileInput{Range: r, Signal: s, Model: m}, asinhParams(), 1e-3)
	require.NoError(t, err)

	// The calibration constant is recovered in the original domain.
	assert.InDelta(t, c0, res.Calibration, c0*0.01, "C")

	// Far zone (molecular region): the smoothed signal equals C·M.
	for i := 0; i < n; i++ {
		if r[i] > 800+3*60 {
			assert.InDelta(t, c0*m[i], res.SmoothedSignal[i], c0*m[i]*0.03, "far zone r=%g", r[i])
		}
	}

	// Aerosol zone: the smoothed signal follows the true signal.
	var maxRel float64
	for i := 0; i < n; i++ {
		if r[i] < 800-60 {
			rel := math.Abs(res.SmoothedSignal[i]-sTrue[i]) / sTrue[i]
			maxRel = math.Max(maxRel, rel)
		}
	}
	assert.Less(t, maxRel, 0.08, "aerosol zone relative error")
}

func TestSmoothProfileAsinh_EquivalentToManual(t *testing.T) {
	n := 401
	r, m, s, _, _ := asinhTestData(n, 11)
	eps := 1e-3
	p := asinhParams()

	// Helper result.
	res, err := SmoothProfileAsinh(context.Background(), ProfileInput{Range: r, Signal: s, Model: m}, p, eps)
	require.NoError(t, err)

	// Manual: normalize, transform, smooth with weights |S_t|, invert.
	c := medianRatio(r, s, m, p.AnchorRange[0], p.AnchorRange[1])
	sT := make([]float64, n)
	mT := make([]float64, n)
	w := make([]float64, n)
	for i := 0; i < n; i++ {
		sT[i] = math.Asinh(s[i] / c / eps)
		mT[i] = math.Asinh(m[i] / eps)
		w[i] = math.Abs(sT[i])
	}
	manual, err := SmoothProfile(context.Background(),
		ProfileInput{Range: r, Signal: sT, Model: mT, Weights: w}, p)
	require.NoError(t, err)

	for i := 0; i < n; i++ {
		want := c * eps * math.Sinh(manual.SmoothedSignal[i])
		assert.InDelta(t, want, res.SmoothedSignal[i], math.Abs(want)*1e-9, "point %d", i)
	}
	assert.InDelta(t, c, res.Calibration, 1e-12)
}

// medianRatio is a local copy of the calibration computation used to double
// check the helper independently of the package internals.
func medianRatio(r, s, m []float64, r0, r1 float64) float64 {
	ratios := make([]float64, 0)
	for i := range r {
		if r[i] >= r0 && r[i] <= r1 {
			ratios = append(ratios, s[i]/m[i])
		}
	}
	sort.Float64s(ratios)
	if len(ratios)%2 == 1 {
		return ratios[len(ratios)/2]
	}
	hi := len(ratios) / 2
	return (ratios[hi-1] + ratios[hi]) / 2
}

func TestSmoothProfileAsinh_NegativeSignal(t *testing.T) {
	n := 401
	r, m, s, _, c0 := asinhTestData(n, 3)
	// Force a few far-zone points to be slightly negative (noise after
	// background subtraction).
	for i := 380; i < 401; i++ {
		s[i] = -10 * (float64(i) - 380)
	}

	res, err := SmoothProfileAsinh(context.Background(),
		ProfileInput{Range: r, Signal: s, Model: m}, asinhParams(), 1e-3)
	require.NoError(t, err)
	assert.InDelta(t, c0, res.Calibration, c0*0.01)
	for i := range res.SmoothedSignal {
		assert.False(t, math.IsNaN(res.SmoothedSignal[i]), "NaN at %d", i)
		assert.False(t, math.IsInf(res.SmoothedSignal[i], 0), "Inf at %d", i)
	}
}

func TestSmoothProfileAsinh_EpsValidation(t *testing.T) {
	r, m, s, _, _ := asinhTestData(50, 1)
	in := ProfileInput{Range: r, Signal: s, Model: m}

	_, err := SmoothProfileAsinh(context.Background(), in, asinhParams(), 0)
	assert.ErrorIs(t, err, ErrInvalidParam)

	_, err = SmoothProfileAsinh(context.Background(), in, asinhParams(), -1)
	assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestSmoothProfileAsinh_ZeroCalibration(t *testing.T) {
	// A signal that is (essentially) zero in the anchor range yields a
	// non-positive calibration constant → error.
	n := 100
	r := uniformRange(n, 10)
	m := make([]float64, n)
	s := make([]float64, n)
	for i := 0; i < n; i++ {
		m[i] = 100 + float64(i)
	}
	p := asinhParams()
	p.AnchorRange = [2]float64{0, 200}

	_, err := SmoothProfileAsinh(context.Background(), ProfileInput{Range: r, Signal: s, Model: m}, p, 1e-3)
	assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestSmoothBatchAsinh_OmegaZero_MatchesPerProfile(t *testing.T) {
	nt, n := 6, 401
	rng := rand.New(rand.NewSource(5))
	r, m, _, sTrue, c0 := asinhTestData(n, 5)

	p := BatchParams{ProfileParams: asinhParams(), Omega: 0}
	signals := make([][]float64, nt)
	models := make([][]float64, nt)
	for k := 0; k < nt; k++ {
		signals[k] = make([]float64, n)
		models[k] = make([]float64, n)
		copy(models[k], m)
		trend := 0.02 * math.Sin(float64(k)/float64(nt)*2*math.Pi)
		for i := 0; i < n; i++ {
			signals[k][i] = sTrue[i] * (1 + trend) * (1 + 0.03*rng.NormFloat64())
		}
	}
	in := BatchInput{
		Time: uniformRange(nt, 60), Range: r, Signals: signals, Models: models,
	}

	res, err := SmoothBatchAsinh(context.Background(), in, p, 1e-3)
	require.NoError(t, err)

	for k := 0; k < nt; k++ {
		// The 2% temporal trend scales the per-profile signal, so C_k deviates
		// from c₀ by roughly the trend amplitude.
		assert.InDelta(t, c0, res.Calibration[k], c0*0.03, "profile %d calibration", k)
		single, err := SmoothProfileAsinh(context.Background(),
			ProfileInput{Range: in.Range, Signal: in.Signals[k], Model: in.Models[k]}, p.ProfileParams, 1e-3)
		require.NoError(t, err)
		for i := 0; i < n; i++ {
			assert.InDelta(t, single.SmoothedSignal[i], res.SmoothedSignal[k][i], math.Abs(single.SmoothedSignal[i])*1e-8, "profile %d, r=%g", k, in.Range[i])
		}
	}
}
