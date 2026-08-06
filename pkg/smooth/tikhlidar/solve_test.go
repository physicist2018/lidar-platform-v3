package tikhlidar

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/mat"
)

// denseProfileSystem builds the same pentadiagonal matrix as
// assembleProfileSystem but as a dense symmetric matrix, for use as a
// reference in tests.
func denseProfileSystem(n int, diag []float64, lambda float64, m *banded5) *mat.SymDense {
	l2 := lambda * lambda
	a := mat.NewSymDense(n, nil)
	for i := 0; i < n; i++ {
		a.SetSym(i, i, diag[i]+l2*m.d0[i])
		if i+1 < n {
			a.SetSym(i, i+1, l2*m.d1[i])
		}
		if i+2 < n {
			a.SetSym(i, i+2, l2*m.d2[i])
		}
	}
	return a
}

func uniformRange(n int, step float64) []float64 {
	r := make([]float64, n)
	for i := range r {
		r[i] = step * float64(i)
	}
	return r
}

func TestSecondDiffRows_ConstantVector(t *testing.T) {
	n := 20
	r := uniformRange(n, 5)
	rows := secondDiffRows(r)

	// D2 applied to a constant vector is zero.
	var maxAbs float64
	for i := range rows {
		v := rows[i].vals[0] + rows[i].vals[1] + rows[i].vals[2]
		maxAbs = math.Max(maxAbs, math.Abs(v))
	}
	assert.Less(t, maxAbs, 1e-12)
}

func TestSecondDiffRows_Quadratic(t *testing.T) {
	// The second derivative of x(r) = r² is exactly 2.
	n := 50
	r := uniformRange(n, 7)
	rows := secondDiffRows(r)
	x := make([]float64, n)
	for i := range x {
		x[i] = r[i] * r[i]
	}
	for i, row := range rows {
		v := row.vals[0]*x[row.cols[0]] + row.vals[1]*x[row.cols[1]] + row.vals[2]*x[row.cols[2]]
		assert.InDelta(t, 2, v, 2e-6, "row %d", i)
	}
}

func TestSecondDiffRows_NonUniform(t *testing.T) {
	// On a non-uniform grid the second derivative of r² must still be 2.
	r := []float64{0, 3, 9, 22, 40, 70, 100}
	x := make([]float64, len(r))
	for i := range x {
		x[i] = r[i] * r[i]
	}
	rows := secondDiffRows(r)
	for i, row := range rows {
		v := row.vals[0]*x[row.cols[0]] + row.vals[1]*x[row.cols[1]] + row.vals[2]*x[row.cols[2]]
		assert.InDelta(t, 2, v, 1e-6, "row %d", i)
	}
}

func TestD2Td2_MatchesDense(t *testing.T) {
	n := 30
	r := uniformRange(n, 5)
	rows := secondDiffRows(r)
	m := d2Td2(rows)

	// Dense D2 and M = D2ᵀ·D2.
	d2 := mat.NewDense(n, n, nil)
	for i := range rows {
		d2.Set(i, rows[i].cols[0], rows[i].vals[0])
		d2.Set(i, rows[i].cols[1], rows[i].vals[1])
		d2.Set(i, rows[i].cols[2], rows[i].vals[2])
	}
	var mDense mat.Dense
	mDense.Mul(d2.T(), d2)

	for i := 0; i < n; i++ {
		assert.InDelta(t, m.d0[i], mDense.At(i, i), 1e-10, "d0[%d]", i)
		if i+1 < n {
			assert.InDelta(t, m.d1[i], mDense.At(i, i+1), 1e-10, "d1[%d]", i)
		}
		if i+2 < n {
			assert.InDelta(t, m.d2[i], mDense.At(i, i+2), 1e-10, "d2[%d]", i)
		}
	}
}

func TestApplyD2T_MatchesDense(t *testing.T) {
	n := 25
	r := uniformRange(n, 5)
	rows := secondDiffRows(r)
	m := d2Td2(rows)

	rng := rand.New(rand.NewSource(7))
	x := make([]float64, n)
	diag := make([]float64, n)
	for i := range x {
		x[i] = rng.Float64()
		diag[i] = 1 + rng.Float64()
	}
	scale := 3.5

	y := make([]float64, n)
	applyD2T(rows, x, scale, diag, y)

	for i := 0; i < n; i++ {
		want := diag[i]*x[i] + scale*m.d0[i]*x[i]
		if i+1 < n {
			want += scale * m.d1[i] * x[i+1]
		}
		if i+2 < n {
			want += scale * m.d2[i] * x[i+2]
		}
		if i-1 >= 0 {
			want += scale * m.d1[i-1] * x[i-1]
		}
		if i-2 >= 0 {
			want += scale * m.d2[i-2] * x[i-2]
		}
		assert.InDelta(t, want, y[i], 1e-10, "row %d", i)
	}
}

func TestSolveBanded_MatchesDenseCholesky(t *testing.T) {
	n := 60
	r := uniformRange(n, 5)
	rng := rand.New(rand.NewSource(11))
	diag := make([]float64, n)
	for i := range diag {
		diag[i] = 0.5 + rng.Float64()
	}
	lambda := 2.5
	m := d2Td2(secondDiffRows(r))
	band := assembleProfileSystem(n, diag, lambda, m)
	b := make([]float64, n)
	for i := range b {
		b[i] = rng.Float64()
	}

	x, err := solveBanded(band, b)
	require.NoError(t, err)

	a := denseProfileSystem(n, diag, lambda, m)
	var chol mat.Cholesky
	require.True(t, chol.Factorize(a))
	xRef := mat.NewVecDense(n, nil)
	require.NoError(t, chol.SolveVecTo(xRef, mat.NewVecDense(n, b)))

	for i := range x {
		assert.InDelta(t, xRef.AtVec(i), x[i], 1e-8, "row %d", i)
	}
}

func TestConjugateGradient_MatchesDenseCholesky(t *testing.T) {
	// Small batch solved with CG must match the dense Cholesky solution.
	nt, n := 4, 6
	r := uniformRange(n, 5)
	time := uniformRange(nt, 10)
	rng := rand.New(rand.NewSource(21))

	diagFlat := make([]float64, n)
	for i := range diagFlat {
		diagFlat[i] = 0.5 + rng.Float64()
	}
	lambda, omega := 1.5, 2.0

	diag := make([][]float64, nt)
	for k := range diag {
		diag[k] = diagFlat
	}

	op := newBatchOperator(nt, n, diag, lambda, omega, secondDiffRows(r), secondDiffRows(time))
	nn := nt * n
	b := make([]float64, nn)
	for i := range b {
		b[i] = rng.Float64()
	}

	x, iters, err := conjugateGradient(context.Background(), op.MulVec, nn, b, 1000, 1e-10)
	require.NoError(t, err)
	require.Less(t, iters, 100)

	// Dense reference: A = blkdiag(P_k) + ω²·(T⊗I) built explicitly.
	a := mat.NewSymDense(nn, nil)
	spatial := denseProfileSystem(n, diagFlat, lambda, d2Td2(secondDiffRows(r)))
	tMat := d2Td2(secondDiffRows(time))
	o2 := omega * omega
	for k := 0; k < nt; k++ {
		for i := 0; i < n; i++ {
			for j := i; j < n; j++ {
				a.SetSym(k*n+i, k*n+j, spatial.At(i, j))
			}
		}
	}
	// Temporal coupling, upper triangle only (each pair visited once).
	for k := 0; k < nt; k++ {
		for l := k; l < nt && l-k <= 2; l++ {
			tv := 0.0
			switch l - k {
			case 0:
				tv = tMat.d0[k]
			case 1:
				tv = tMat.d1[k]
			case 2:
				tv = tMat.d2[k]
			}
			for i := 0; i < n; i++ {
				a.SetSym(k*n+i, l*n+i, a.At(k*n+i, l*n+i)+o2*tv)
			}
		}
	}

	var chol mat.Cholesky
	require.True(t, chol.Factorize(a))
	xRef := mat.NewVecDense(nn, nil)
	require.NoError(t, chol.SolveVecTo(xRef, mat.NewVecDense(nn, b)))

	for i := range x {
		assert.InDelta(t, xRef.AtVec(i), x[i], 1e-6, "row %d", i)
	}
}
