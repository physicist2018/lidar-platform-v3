package tikhlidar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnchorWeight_Asymptotes(t *testing.T) {
	href, L := 1000.0, 200.0

	assert.InDelta(t, 0, anchorWeight(href-10*L, href, L), 1e-12)
	assert.InDelta(t, 1, anchorWeight(href+10*L, href, L), 1e-12)
	assert.InDelta(t, 0.5, anchorWeight(href, href, L), 1e-12)

	// Transition width: σ(±2) with s = L/4 → ~0.12 / ~0.88.
	assert.InDelta(t, 0.119, anchorWeight(href-L/2, href, L), 1e-3)
	assert.InDelta(t, 0.881, anchorWeight(href+L/2, href, L), 1e-3)
}

func TestAnchorWeight_Monotonic(t *testing.T) {
	href, L := 1000.0, 200.0
	prev := 0.0
	for r := 500.0; r <= 1500; r += 10 {
		w := anchorWeight(r, href, L)
		assert.GreaterOrEqual(t, w, prev)
		prev = w
	}
}

func TestAnchorWeight_ZeroWidthIsStep(t *testing.T) {
	href := 800.0
	assert.Equal(t, 0.0, anchorWeight(href-1, href, 0))
	assert.Equal(t, 0.5, anchorWeight(href, href, 0))
	assert.Equal(t, 1.0, anchorWeight(href+1, href, 0))
}
