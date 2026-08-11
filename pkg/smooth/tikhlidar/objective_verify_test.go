package tikhlidar

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/mat"
)

// TestSmoothProfile_MatchesObjective checks that the returned Ŝ satisfies the
// normal equations A·Ŝ = b derived from
//
//	Φ(Ŝ) = Σ (1−w)·u·(S−Ŝ)² + λ²·Σ (D²_r Ŝ)² + q·Σ w·(Ŝ−C·M)²,
//
// built independently (densely) from the formula.
func TestSmoothProfile_MatchesObjective(t *testing.T) {
	n := 401
	r := uniformRange(n, 5)
	m := make([]float64, n)
	s := make([]float64, n)
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		m[i] = 100*math.Exp(-r[i]/900) + 5
		ratio := 1 + 1.5*math.Exp(-math.Pow((r[i]-350)/120, 2))
		s[i] = 0.73 * m[i] * ratio
		u[i] = 1 + 0.5*math.Sin(float64(i)) // non-trivial weights
	}

	p := ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{1300, 1700},
		Href:            800,
		TransitionWidth: 60,
		AnchorStrength:  10,
		Lambda:          30,
	}

	res, err := SmoothProfile(context.Background(), ProfileInput{Range: r, Signal: s, Model: m, Weights: u}, p)
	require.NoError(t, err)

	// Independent dense assembly of A = diag((1−w)u + qw) + λ²·D2ᵀ·D2 and
	// b = (1−w)u·S + q·w·C·M directly from the formula.
	w := anchorWeights(r, p.Href, p.TransitionWidth)
	c, err := calibrationConstant(r, s, m, p.AnchorRange[0], p.AnchorRange[1])
	require.NoError(t, err)
	l2 := p.Lambda * p.Lambda

	d2 := mat.NewDense(n, n, nil)
	rows := secondDiffRows(r)
	for i := range rows {
		d2.Set(i, rows[i].cols[0], rows[i].vals[0])
		d2.Set(i, rows[i].cols[1], rows[i].vals[1])
		d2.Set(i, rows[i].cols[2], rows[i].vals[2])
	}
	var d2tD2 mat.Dense
	d2tD2.Mul(d2.T(), d2)

	am := mat.NewDense(n, n, nil)
	am.Scale(l2, &d2tD2)
	b := make([]float64, n)
	for i := 0; i < n; i++ {
		fitW := (1 - w[i]) * u[i]
		am.Set(i, i, am.At(i, i)+fitW+p.AnchorStrength*w[i])
		b[i] = fitW*s[i] + p.AnchorStrength*w[i]*c*m[i]
	}

	// Residual A·Ŝ − b must be ~0.
	sh := mat.NewVecDense(n, res.SmoothedSignal)
	bb := mat.NewVecDense(n, b)
	var ax mat.VecDense
	ax.MulVec(am, sh)
	ax.SubVec(&ax, bb)

	bnorm := bb.Norm(2)
	require.Greater(t, bnorm, 0.0)
	residual := ax.Norm(2) / bnorm
	assert.Less(t, residual, 1e-9, "relative residual of the normal equations")
}
