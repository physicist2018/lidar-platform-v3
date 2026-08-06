package tikhlidar

import (
	"context"
	"fmt"
	"math"

	"gonum.org/v1/gonum/blas/blas64"
	"gonum.org/v1/gonum/mat"
)

// Default iterative solver parameters for the batch case.
const (
	batchMaxIter = 2000
	batchTol     = 1e-8
)

// assembleProfileSystem builds the symmetric pentadiagonal matrix
//
//	A = diag((1−w) + q·w) + λ²·D2ᵀ·D2
//
// in gonum's packed symmetric band format (upper bandwidth 2) and returns it.
func assembleProfileSystem(n int, diag []float64, lambda float64, m *banded5) *mat.SymBandDense {
	band := mat.NewSymBandDense(n, 2, nil)
	data := band.RawSymBand().Data
	l2 := lambda * lambda
	for i := 0; i < n; i++ {
		row := data[i*3 : i*3+3]
		row[0] = diag[i] + l2*m.d0[i]
		if i+1 < n {
			row[1] = l2 * m.d1[i]
		}
		if i+2 < n {
			row[2] = l2 * m.d2[i]
		}
	}
	return band
}

// solveBanded solves the SPD pentadiagonal system A·x = b with gonum's banded
// Cholesky factorization.
func solveBanded(band *mat.SymBandDense, b []float64) ([]float64, error) {
	var chol mat.BandCholesky
	if !chol.Factorize(band) {
		return nil, ErrNotPositiveDefinite
	}
	x := mat.NewVecDense(len(b), nil)
	if err := chol.SolveVecTo(x, mat.NewVecDense(len(b), b)); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotPositiveDefinite, err)
	}
	return x.RawVector().Data, nil
}

// batchOperator applies the batch system matrix
//
//	A = blkdiag(P_k) + ω²·(D2_tᵀ·D2_t) ⊗ I_n,
//
// where P_k = diag((1−w)·u_k + q·w) + λ²·D2ᵀ·D2 is the per-profile
// pentadiagonal operator (the diagonal part differs between profiles through
// the weight profile u_k) and D2_t is the temporal second-difference
// operator. The matrix is never stored; MulVec performs a matrix-free
// product, which is all that conjugate gradients require.
type batchOperator struct {
	nt, n    int
	diag     [][]float64 // per-profile diagonal, length nt×n
	lambda   float64
	omega    float64
	d2Rows   []triRow  // spatial operator rows, length n
	d2tRows  []triRow  // temporal operator rows, length nt
	d2x      []float64 // scratch, length n
	col, tmp []float64 // scratch, length nt
}

func newBatchOperator(nt, n int, diag [][]float64, lambda, omega float64, d2Rows, d2tRows []triRow) *batchOperator {
	return &batchOperator{
		nt: nt, n: n, diag: diag, lambda: lambda, omega: omega,
		d2Rows: d2Rows, d2tRows: d2tRows,
		d2x: make([]float64, n), col: make([]float64, nt), tmp: make([]float64, nt),
	}
}

// MulVec computes y = A·x for a flat vector of length nt·n.
func (op *batchOperator) MulVec(x []float64) []float64 {
	y := make([]float64, op.nt*op.n)
	l2 := op.lambda * op.lambda

	// Spatial part: block-diagonal P_k applied per profile.
	for k := 0; k < op.nt; k++ {
		base := k * op.n
		xk := x[base : base+op.n]
		yk := y[base : base+op.n]
		applyD2T(op.d2Rows, xk, l2, op.diag[k], yk)
	}

	// Temporal part: ω²·D2_tᵀ·D2_t applied along the time axis.
	if op.omega > 0 && op.nt >= 3 {
		o2 := op.omega * op.omega
		for i := 0; i < op.n; i++ {
			for k := 0; k < op.nt; k++ {
				op.col[k] = x[k*op.n+i]
			}
			applyD2T(op.d2tRows, op.col, o2, nil, op.tmp)
			for k := 0; k < op.nt; k++ {
				y[k*op.n+i] += op.tmp[k]
			}
		}
	}
	return y
}

// conjugateGradient solves the SPD system A·x = b using the conjugate
// gradient method with the matrix-free operator mulVec. Vector operations use
// gonum's blas64. It returns the solution and the number of iterations used.
func conjugateGradient(ctx context.Context, mulVec func([]float64) []float64, n int, b []float64, maxIter int, tol float64) ([]float64, int, error) {
	x := make([]float64, n)
	r := make([]float64, n)
	p := make([]float64, n)
	copy(r, b)
	copy(p, b)

	vx := blas64.Vector{N: n, Inc: 1, Data: x}
	vr := blas64.Vector{N: n, Inc: 1, Data: r}
	vp := blas64.Vector{N: n, Inc: 1, Data: p}
	vap := blas64.Vector{N: n, Inc: 1, Data: make([]float64, n)}

	rsold := blas64.Dot(vr, vr)
	bnorm := math.Sqrt(rsold)
	if bnorm == 0 {
		return x, 0, nil
	}

	for iter := 1; iter <= maxIter; iter++ {
		if err := ctx.Err(); err != nil {
			return nil, iter, err
		}
		ap := mulVec(p)
		vap.Data = ap
		pAp := blas64.Dot(vp, vap)
		if pAp <= 0 {
			return nil, iter, ErrNotPositiveDefinite
		}
		alpha := rsold / pAp
		blas64.Axpy(alpha, vp, vx)
		blas64.Axpy(-alpha, vap, vr)
		rsnew := blas64.Dot(vr, vr)
		if math.Sqrt(rsnew) <= tol*bnorm {
			return x, iter, nil
		}
		beta := rsnew / rsold
		blas64.Scal(beta, vp)
		blas64.Axpy(1, vr, vp)
		rsold = rsnew
	}
	return nil, maxIter, fmt.Errorf("%w: after %d iterations", ErrNotConverged, maxIter)
}
