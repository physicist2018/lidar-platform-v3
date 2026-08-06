package tikhlidar

// triRow holds the three non-zero entries of one row of the second-difference
// operator D2. Each row has exactly three consecutive non-zero columns.
type triRow struct {
	cols [3]int
	vals [3]float64
}

// secondDiffRows builds the rows of the n×n second-difference operator D2 on
// the grid r: centered divided differences in the interior, one-sided
// differences at the boundaries (forward at the start, backward at the end).
// All rows use the same divided-difference formula, which reduces to
// [1, −2, 1]/h² on a uniform grid. The operator approximates the second
// derivative, so entries scale as 1/Δr². Requires len(r) >= 3.
func secondDiffRows(r []float64) []triRow {
	n := len(r)
	rows := make([]triRow, n)

	// Forward one-sided at i = 0 (points 0, 1, 2).
	rows[0] = dividedDiffRow(r[1]-r[0], r[2]-r[1], 0, 1, 2)

	// Centered divided differences in the interior.
	for i := 1; i < n-1; i++ {
		rows[i] = dividedDiffRow(r[i]-r[i-1], r[i+1]-r[i], i-1, i, i+1)
	}

	// Backward one-sided at i = n-1 (points n-3, n-2, n-1).
	rows[n-1] = dividedDiffRow(r[n-2]-r[n-3], r[n-1]-r[n-2], n-3, n-2, n-1)

	return rows
}

// dividedDiffRow builds one row of the second divided difference at points
// (cols[0], cols[1], cols[2]) with left spacing hl and right spacing hr:
//
//	2·[(x₂−x₁)/hr − (x₁−x₀)/hl] / (hl+hr)
func dividedDiffRow(hl, hr float64, c0, c1, c2 int) triRow {
	sum := hl + hr
	return triRow{
		cols: [3]int{c0, c1, c2},
		vals: [3]float64{
			2 / (hl * sum),
			-2 / (hl * hr),
			2 / (hr * sum),
		},
	}
}

// banded5 stores a symmetric pentadiagonal matrix (bandwidth 2) by its upper
// diagonals.
type banded5 struct {
	d0 []float64 // main diagonal,   length n
	d1 []float64 // first  upper,    length n-1
	d2 []float64 // second upper,    length n-2
}

func newBanded5(n int) *banded5 {
	return &banded5{
		d0: make([]float64, n),
		d1: make([]float64, n-1),
		d2: make([]float64, n-2),
	}
}

// add accumulates v into the symmetric entry (i, j) with |i-j| <= 2.
func (m *banded5) add(i, j int, v float64) {
	if i > j {
		i, j = j, i
	}
	switch j - i {
	case 0:
		m.d0[i] += v
	case 1:
		m.d1[i] += v
	case 2:
		m.d2[i] += v
	default:
		panic("tikhlidar: banded5.add out of band")
	}
}

// d2Td2 computes the symmetric pentadiagonal matrix M = D2ᵀ·D2 by accumulating
// the outer products of the rows of D2.
func d2Td2(rows []triRow) *banded5 {
	m := newBanded5(len(rows))
	for k := range rows {
		for a := 0; a < 3; a++ {
			for b := a; b < 3; b++ {
				m.add(rows[k].cols[a], rows[k].cols[b], rows[k].vals[a]*rows[k].vals[b])
			}
		}
	}
	return m
}

// applyD2T computes y = diag∘x + scale·D2ᵀ·D2·x. If diag is nil the diagonal
// part is skipped. The result is written into y (which must have length n).
func applyD2T(rows []triRow, x []float64, scale float64, diag []float64, y []float64) {
	n := len(x)
	for i := 0; i < n; i++ {
		y[i] = 0
		if diag != nil {
			y[i] = diag[i] * x[i]
		}
	}
	// d = D2·x
	d := make([]float64, n)
	for i := 0; i < n; i++ {
		d[i] = rows[i].vals[0]*x[rows[i].cols[0]] +
			rows[i].vals[1]*x[rows[i].cols[1]] +
			rows[i].vals[2]*x[rows[i].cols[2]]
	}
	// y += scale·D2ᵀ·d
	for k := 0; k < n; k++ {
		for a := 0; a < 3; a++ {
			y[rows[k].cols[a]] += scale * rows[k].vals[a] * d[k]
		}
	}
}
