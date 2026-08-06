package tikhlidar

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syntheticProfile builds a test scenario: a smooth molecular signal, a
// Gaussian aerosol layer below Href (R = 1 above it), a noisy measured
// signal S = C·M·R, and unit weight profile.
func syntheticProfile(n int, href float64, noise float64, seed int64) (r, m, s, sTrue, w []float64, cTrue float64) {
	step := 5.0
	r = uniformRange(n, step)
	m = make([]float64, n)
	s = make([]float64, n)
	sTrue = make([]float64, n)
	w = make([]float64, n)
	cTrue = 0.73
	rng := rand.New(rand.NewSource(seed))

	for i := 0; i < n; i++ {
		m[i] = 500*math.Exp(-r[i]/900) + 20
		ratio := 1 + 1.5*math.Exp(-math.Pow((r[i]-350)/120, 2))
		sTrue[i] = cTrue * m[i] * ratio
		s[i] = sTrue[i] * (1 + noise*rng.NormFloat64())
		w[i] = 1
	}
	return r, m, s, sTrue, w, cTrue
}

func TestSmoothProfile_SyntheticRecovery(t *testing.T) {
	n := 401 // 0..2000 m
	href := 800.0
	r, m, s, sTrue, w, cTrue := syntheticProfile(n, href, 0.02, 42)

	p := ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{1300, 1700}, // molecular region above href+L
		Href:            href,
		TransitionWidth: 60,
		AnchorStrength:  10,
		Lambda:          30,
	}

	res, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: w}, p)
	require.NoError(t, err)

	// Calibration constant recovered from the anchor range.
	assert.InDelta(t, cTrue, res.Calibration, 0.01)

	// Far zone (r > href + 3L): the signal is replaced by the molecular one.
	for i := 0; i < n; i++ {
		if r[i] > href+3*p.TransitionWidth {
			assert.InDelta(t, cTrue*m[i], res.SmoothedSignal[i], 0.02*cTrue*m[i], "far zone r=%g", r[i])
		}
	}

	// Aerosol zone (below the transition): the smoothed signal follows the
	// true signal within the noise + smoothing budget.
	var maxRel float64
	for i := 0; i < n; i++ {
		if r[i] < href-p.TransitionWidth {
			rel := math.Abs(res.SmoothedSignal[i]-sTrue[i]) / sTrue[i]
			maxRel = math.Max(maxRel, rel)
		}
	}
	assert.Less(t, maxRel, 0.08, "aerosol zone relative error too large")

	// Smoothness: the second difference of the result is far below the
	// second difference of the raw noisy signal in the aerosol zone.
	rows := secondDiffRows(r)
	applyD2 := func(x []float64) float64 {
		var v float64
		for i := 40; i < n-10; i++ {
			d := rows[i].vals[0]*x[rows[i].cols[0]] + rows[i].vals[1]*x[rows[i].cols[1]] + rows[i].vals[2]*x[rows[i].cols[2]]
			v = math.Max(v, math.Abs(d))
		}
		return v
	}
	assert.Less(t, applyD2(res.SmoothedSignal), applyD2(s))
}

func TestSmoothProfile_LambdaSmoothsNoise(t *testing.T) {
	n := 401
	href := 800.0
	r, m, s, sTrue, w, _ := syntheticProfile(n, href, 0.05, 1)

	base := ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{1300, 1700},
		Href:            href,
		TransitionWidth: 60,
		AnchorStrength:  10,
	}

	std := func(res *ProfileResult) float64 {
		var sumSq float64
		count := 0
		for i := 0; i < n; i++ {
			if r[i] > 200 && r[i] < href-100 { // aerosol zone away from boundaries
				d := res.SmoothedSignal[i] - sTrue[i]
				sumSq += d * d
				count++
			}
		}
		return math.Sqrt(sumSq / float64(count))
	}

	// λ = 0: the system is diagonal, so the solution is the pointwise blend
	// of measurement and model: Ŝ = [(1−w)·u·S + q·w·C·M] / [(1−w)·u + q·w].
	resNoSmooth, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: w}, base)
	require.NoError(t, err)
	q := base.AnchorStrength
	for i := 0; i < n; i++ {
		if r[i] < href-100 {
			wi := anchorWeight(r[i], href, base.TransitionWidth)
			fitW := (1 - wi) * w[i]
			want := (fitW*s[i] + q*wi*resNoSmooth.Calibration*m[i]) / (fitW + q*wi)
			assert.InDelta(t, want, resNoSmooth.SmoothedSignal[i], 1e-6, "r=%g", r[i])
		}
	}

	// λ > 0: noise variance is reduced.
	base.Lambda = 40
	resSmoothed, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: w}, base)
	require.NoError(t, err)
	assert.Less(t, std(resSmoothed), std(resNoSmooth))
}

func TestSmoothProfile_StrongAnchorPullsToMolecular(t *testing.T) {
	n := 401
	href := 800.0
	r, m, s, _, w, _ := syntheticProfile(n, href, 0.01, 3)

	p := ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{1300, 1700},
		Href:            href,
		TransitionWidth: 40,
		AnchorStrength:  1e6, // hard anchor
		Lambda:          10,
	}

	res, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: w}, p)
	require.NoError(t, err)

	// The far zone is pulled to the calibrated molecular signal Ĉ·M; the
	// comparison uses Ĉ (the returned constant) to decouple from the
	// calibration noise budget.
	for i := 0; i < n; i++ {
		if r[i] > href+2*p.TransitionWidth {
			want := res.Calibration * m[i]
			assert.InDelta(t, want, res.SmoothedSignal[i], 1e-6*want, "r=%g", r[i])
		}
	}
}

func TestSmoothProfile_ValidationErrors(t *testing.T) {
	n := 100
	r := uniformRange(n, 10)
	m := make([]float64, n)
	s := make([]float64, n)
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		m[i] = 100 + float64(i)
		s[i] = 0.7 * m[i]
		u[i] = 1
	}
	p := validParams()

	// Anchor range with a single point.
	p.AnchorRange = [2]float64{0, 0.5}
	_, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: u}, p)
	assert.ErrorIs(t, err, ErrAnchorRange)

	// Non-positive model.
	m[5] = 0
	_, err = SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: u}, p)
	assert.ErrorIs(t, err, ErrModelNonPositive)
}

func TestSmoothProfile_ZeroWeightIgnoresOutlier(t *testing.T) {
	// A single outlier is ignored where its weight is zero: the smoothed
	// signal stays close to the local trend instead of chasing the spike.
	n := 101
	r := uniformRange(n, 1) // 0..100 m
	m := make([]float64, n)
	s := make([]float64, n)
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		m[i] = 100
		s[i] = 100
		u[i] = 1
	}
	spike := 50
	s[spike] = 1000

	p := ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{95, 100}, // molecular region at the top
		Href:            90,
		TransitionWidth: 10,
		AnchorStrength:  5,
		Lambda:          2,
	}

	// All weights one: the fit pulls the solution towards the spike.
	uAll := make([]float64, n)
	copy(uAll, u)
	resAll, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: uAll}, p)
	require.NoError(t, err)

	// Zero weight at the spike: the fit is off there, so the smoothing keeps
	// the solution near the local level.
	u[spike] = 0
	resZero, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: u}, p)
	require.NoError(t, err)

	assert.Greater(t, resAll.SmoothedSignal[spike], resZero.SmoothedSignal[spike]+50,
		"the outlier must affect the solution when weighted")
	assert.InDelta(t, 100, resZero.SmoothedSignal[spike], 5,
		"the outlier must be ignored where its weight is zero")
}
