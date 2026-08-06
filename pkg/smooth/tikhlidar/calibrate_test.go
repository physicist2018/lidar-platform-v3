package tikhlidar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalibrationConstant_Median(t *testing.T) {
	r := uniformRange(10, 5) // 0..45
	s := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	m := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	// Ratios in [10, 30] are {3, 4, 5, 6, 7} → median 5.
	c, err := calibrationConstant(r, s, m, 10, 30)
	require.NoError(t, err)
	assert.InDelta(t, 5, c, 1e-12)
}

func TestCalibrationConstant_Noisy(t *testing.T) {
	// S = C·M with multiplicative noise in the anchor range → robust median.
	n := 101
	r := uniformRange(n, 10)
	m := make([]float64, n)
	cTrue := 0.73
	for i := range m {
		m[i] = 500 - 100*float64(i)/float64(n)
	}
	s := make([]float64, n)
	for i := range s {
		ratio := 1.0
		switch i % 5 {
		case 0:
			ratio = 1.2 // outlier
		case 1:
			ratio = 0.8
		case 2:
			ratio = 1.05
		case 3:
			ratio = 0.95
		case 4:
			ratio = 1.0
		}
		s[i] = cTrue * m[i] * ratio
	}

	c, err := calibrationConstant(r, s, m, 300, 800)
	require.NoError(t, err)
	assert.InDelta(t, cTrue, c, 0.01)
}

func TestCalibrationConstant_TooFewPoints(t *testing.T) {
	r := uniformRange(5, 5)
	s := []float64{1, 2, 3, 4, 5}
	m := []float64{1, 1, 1, 1, 1}

	_, err := calibrationConstant(r, s, m, 0, 0) // no points
	assert.ErrorIs(t, err, ErrAnchorRange)

	_, err = calibrationConstant(r, s, m, 5, 5.01) // one point only
	assert.ErrorIs(t, err, ErrAnchorRange)
}
