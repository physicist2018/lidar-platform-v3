package molecular

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stdModel is a homogeneous standard atmosphere (15 °C, 1013.25 hPa at all
// altitudes), used for reference-value and transmission tests.
func stdModel() AtmosphereModel {
	return AtmosphereModel{
		Altitude:    []float64{0, 1, 5, 20},
		Temperature: []float64{15, 15, 15, 15},
		Pressure:    []float64{1013.25, 1013.25, 1013.25, 1013.25},
	}
}

func TestCompute_ReferenceValues532nm(t *testing.T) {
	// Standard atmosphere at sea level: βm(532) ≈ 1.57e-6 m⁻¹·sr⁻¹ and
	// αm(532) ≈ 1.31e-5 m⁻¹ (independent literature values, ~1% agreement).
	in := Input{
		Range:       []float64{0, 100},
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere:  stdModel(),
	}

	res, err := Compute(context.Background(), in)
	require.NoError(t, err)

	assert.InDelta(t, 1.568e-6, res.Backscatter[0], 1.6e-8, "βm(532) at sea level")
	assert.InDelta(t, 1.314e-5, res.Extinction[0], 1.3e-7, "αm(532) at sea level")

	// Molecular lidar ratio is exactly 8π/3.
	assert.InDelta(t, 8*math.Pi/3, res.Extinction[0]/res.Backscatter[0], 1e-9)

	// At r = 0 the transmission is 1, so the signal equals the backscatter.
	assert.InDelta(t, res.Backscatter[0], res.Signal[0], 1e-15)
	assert.InDelta(t, 1, res.Transmission[0], 1e-15)
}

func TestCompute_WavelengthScaling(t *testing.T) {
	// βm scales roughly as λ⁻⁴; the (n²−1)² factor adds ~5% between 355 and
	// 532 nm, giving an expected ratio of ≈ 5.32.
	compute := func(wvl float64) *Result {
		res, err := Compute(context.Background(), Input{
			Range:       []float64{0, 10},
			ZenithAngle: 0,
			Wavelength:  wvl,
			Atmosphere:  stdModel(),
		})
		require.NoError(t, err)
		return res
	}

	r355 := compute(355)
	r532 := compute(532)
	ratio := r355.Backscatter[0] / r532.Backscatter[0]

	assert.InDelta(t, 5.32, ratio, 0.06, "βm(355)/βm(532)")
}

func TestCompute_InterpolatedAtmosphere(t *testing.T) {
	// Model with a strong gradient: T is interpolated linearly, P in log
	// space (barometric).
	in := Input{
		Range:       []float64{0, 5000}, // α=0 → altitude = range
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere: AtmosphereModel{
			Altitude:    []float64{0, 10},
			Temperature: []float64{15, -50},   // °C
			Pressure:    []float64{1013, 265}, // hPa
		},
	}
	res, err := Compute(context.Background(), in)
	require.NoError(t, err)

	// At 5 km: T = -17.5 °C (linear), P = √(1013·265) ≈ 518.1 hPa (geometric
	// mean from log-space interpolation), not 639 hPa (linear mean).
	tK := -17.5 + 273.15
	pHPa := math.Sqrt(1013 * 265)
	expectedDensity := (pHPa * 100) / (boltzmann * tK)
	expectedBeta := expectedDensity * backscatterCrossSection(532)
	assert.InDelta(t, expectedBeta, res.Backscatter[1], expectedBeta*1e-9, "βm at 5 km")

	// Sanity: the log-space result differs from the linear one.
	linearP := (1013.0 + 265.0) / 2
	assert.Greater(t, linearP, pHPa)
}

func TestCompute_BarometricInterpolation(t *testing.T) {
	// Barometric (isothermal) atmosphere P(z) = P0·exp(−z/H): log-space
	// interpolation is exact at the midpoint, linear is not.
	const H = 8000.0 // scale height, m
	p0 := 1013.25
	p10 := p0 * math.Exp(-10000/H) // hPa at 10 km

	in := Input{
		Range:       []float64{0, 5000, 10000}, // α=0 → altitude = range
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere: AtmosphereModel{
			Altitude:    []float64{0, 10},
			Temperature: []float64{15, 15}, // isothermal
			Pressure:    []float64{p0, p10},
		},
	}
	res, err := Compute(context.Background(), in)
	require.NoError(t, err)

	// Exact barometric pressure at 5 km and the resulting βm.
	exactP := p0 * math.Exp(-5000/H)
	exactDensity := (exactP * 100) / (boltzmann * (15 + 273.15))
	exactBeta := exactDensity * backscatterCrossSection(532)
	assert.InDelta(t, exactBeta, res.Backscatter[1], exactBeta*1e-9, "βm at 5 km")

	// Linear interpolation would overestimate the pressure.
	linearP := (p0 + p10) / 2
	assert.Greater(t, linearP, exactP)
}

func TestCompute_HomogeneousTransmission(t *testing.T) {
	// Homogeneous atmosphere → αm constant → T_m²(r) = exp(−2·αm·r) exactly.
	n := 50
	r := make([]float64, n)
	for i := range r {
		r[i] = 10 * float64(i) // 0..490 m
	}
	res, err := Compute(context.Background(), Input{
		Range:       r,
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere:  stdModel(),
	})
	require.NoError(t, err)

	alpha := res.Extinction[0]
	for i := range r {
		want := math.Exp(-2 * alpha * r[i])
		assert.InDelta(t, want, res.Transmission[i], 1e-9, "T²(r=%g)", r[i])
		assert.InDelta(t, res.Backscatter[i]*res.Transmission[i], res.Signal[i], 1e-15)
	}
}

func TestCompute_ZenithGeometry(t *testing.T) {
	// α = 60° maps range to altitude as z = r·cos(60°) = r/2, so βm at range r
	// with α = 60° equals βm at range r/2 with α = 0°.
	model := AtmosphereModel{
		Altitude:    []float64{0, 2, 4, 6, 8, 10},
		Temperature: []float64{15, 2, -10, -25, -40, -55},
		Pressure:    []float64{1013, 795, 616, 472, 357, 264},
	}

	in0 := Input{
		Range:       []float64{0, 1000, 2000, 4000, 6000},
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere:  model,
	}
	straight, err := Compute(context.Background(), in0)
	require.NoError(t, err)

	tilted, err := Compute(context.Background(), Input{
		Range:       []float64{0, 2000, 4000, 8000, 12000},
		ZenithAngle: 60,
		Wavelength:  532,
		Atmosphere:  model,
	})
	require.NoError(t, err)

	for i := range in0.Range {
		assert.InDelta(t, straight.Backscatter[i], tilted.Backscatter[i], 1e-12, "point %d", i)
		assert.InDelta(t, straight.Extinction[i], tilted.Extinction[i], 1e-12, "point %d", i)
	}
}

func TestCompute_ClampsOutsideModel(t *testing.T) {
	// Range altitudes above the model top are clamped to the top values, so
	// βm stays constant at the value of the model's highest level.
	model := AtmosphereModel{
		Altitude:    []float64{0, 5},
		Temperature: []float64{15, -10},
		Pressure:    []float64{1013, 540},
	}

	ref, err := Compute(context.Background(), Input{
		Range:       []float64{0, 5000},
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere:  model,
	})
	require.NoError(t, err)

	aboveIn := Input{
		Range:       []float64{0, 5000, 10000, 15000},
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere:  model,
	}
	above, err := Compute(context.Background(), aboveIn)
	require.NoError(t, err)

	betaTop := ref.Backscatter[1] // βm at the model top (5 km)
	for i := 1; i < len(aboveIn.Range); i++ {
		assert.InDelta(t, betaTop, above.Backscatter[i], 1e-12, "point %d", i)
	}
}
