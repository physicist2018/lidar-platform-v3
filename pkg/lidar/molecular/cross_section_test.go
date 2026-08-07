package molecular

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterpolate(t *testing.T) {
	x := []float64{0, 10}
	y := []float64{0, 100}

	v, clamped := interpolate(x, y, 5)
	assert.False(t, clamped)
	assert.InDelta(t, 50, v, 1e-12)

	v, clamped = interpolate(x, y, 2.5)
	assert.False(t, clamped)
	assert.InDelta(t, 25, v, 1e-12)

	// Exact node values.
	v, clamped = interpolate(x, y, 0)
	assert.False(t, clamped)
	assert.InDelta(t, 0, v, 1e-12)

	v, clamped = interpolate(x, y, 10)
	assert.False(t, clamped)
	assert.InDelta(t, 100, v, 1e-12)
}

func TestInterpolate_Clamps(t *testing.T) {
	x := []float64{1, 10}
	y := []float64{10, 100}

	v, clamped := interpolate(x, y, 0)
	assert.True(t, clamped)
	assert.InDelta(t, 10, v, 1e-12)

	v, clamped = interpolate(x, y, 20)
	assert.True(t, clamped)
	assert.InDelta(t, 100, v, 1e-12)
}

func TestRefractiveIndex(t *testing.T) {
	// Edlén: n−1 ≈ 2.78e-4 at 532 nm, ≈ 2.86e-4 at 355 nm.
	assert.InDelta(t, 2.7819e-4, refractiveIndexMinus1(532), 1e-8)
	assert.InDelta(t, 2.8570e-4, refractiveIndexMinus1(355), 1e-8)
}

func TestKingFactor(t *testing.T) {
	// F_k = (6+3·0.0272)/(6−7·0.0272) ≈ 1.0468.
	assert.InDelta(t, 1.0468, kingFactor(), 1e-4)
}

func TestCrossSections(t *testing.T) {
	// σ_π(532) ≈ 6.2e-32 m²·sr⁻¹; σ_s = (8π/3)·σ_π.
	bcs := backscatterCrossSection(532)
	assert.InDelta(t, 6.16e-32, bcs, 1e-34)

	ecs := extinctionCrossSection(532)
	assert.InDelta(t, (8*math.Pi/3)*bcs, ecs, 1e-40)
}
