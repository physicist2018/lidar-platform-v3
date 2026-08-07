package tikhlidar

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validParams() ProfileParams {
	return ProfileParams{
		Wavelength:      532,
		AnchorRange:     [2]float64{600, 800},
		Href:            500,
		TransitionWidth: 50,
		AnchorStrength:  10,
		Lambda:          20,
	}
}

func validInput() ProfileInput {
	n := 100
	r := uniformRange(n, 10) // 0..990
	s := make([]float64, n)
	m := make([]float64, n)
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		m[i] = 100 + float64(i)
		s[i] = 0.7 * m[i]
		u[i] = 1
	}
	return ProfileInput{Range: r, Signal: s, Model: m, Weights: u}
}

func TestValidateProfile_OK(t *testing.T) {
	assert.NoError(t, validateProfile(validInput(), validParams()))
	assert.NoError(t, validateCommon(validParams(), validInput().Range))
}

func TestValidateProfile_Errors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ProfileInput, *ProfileParams)
		wantErr error
	}{
		{"nil input", func(in *ProfileInput, p *ProfileParams) { *in = ProfileInput{} }, ErrNilInput},
		{"too few points", func(in *ProfileInput, p *ProfileParams) { in.Range = in.Range[:2] }, ErrTooFewPoints},
		{"length mismatch", func(in *ProfileInput, p *ProfileParams) { in.Signal = in.Signal[:5] }, ErrLengthMismatch},
		{"non-increasing range", func(in *ProfileInput, p *ProfileParams) { in.Range[3] = in.Range[2] }, ErrRangeNotIncreasing},
		{"NaN in signal", func(in *ProfileInput, p *ProfileParams) { in.Signal[7] = nan() }, ErrNonFinite},
		{"non-positive model", func(in *ProfileInput, p *ProfileParams) { in.Model[7] = 0 }, ErrModelNonPositive},
		{"nil weights", func(in *ProfileInput, p *ProfileParams) { in.Weights = nil }, ErrNilInput},
		{"weights length mismatch", func(in *ProfileInput, p *ProfileParams) { in.Weights = in.Weights[:5] }, ErrLengthMismatch},
		{"negative weight", func(in *ProfileInput, p *ProfileParams) { in.Weights[7] = -1 }, ErrInvalidParam},
		{"NaN weight", func(in *ProfileInput, p *ProfileParams) { in.Weights[7] = nan() }, ErrNonFinite},
		{"negative lambda", func(in *ProfileInput, p *ProfileParams) { p.Lambda = -1 }, ErrInvalidParam},
		{"negative q", func(in *ProfileInput, p *ProfileParams) { p.AnchorStrength = -1 }, ErrInvalidParam},
		{"negative L", func(in *ProfileInput, p *ProfileParams) { p.TransitionWidth = -1 }, ErrInvalidParam},
		{"r0 >= r1", func(in *ProfileInput, p *ProfileParams) { p.AnchorRange = [2]float64{800, 600} }, ErrInvalidParam},
		{"anchor outside grid", func(in *ProfileInput, p *ProfileParams) { p.AnchorRange = [2]float64{10000, 11000} }, ErrAnchorRange},
		{"anchor one point", func(in *ProfileInput, p *ProfileParams) { p.AnchorRange = [2]float64{0, 0.5} }, ErrAnchorRange},
		{"href outside", func(in *ProfileInput, p *ProfileParams) { p.Href = -10 }, ErrInvalidParam},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, p := validInput(), validParams()
			tt.mutate(&in, &p)
			err := validateProfile(in, p)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateBatch_Errors(t *testing.T) {
	base := func() BatchInput {
		nt, n := 5, 40
		r := uniformRange(n, 10)
		tm := uniformRange(nt, 60)
		signals := make([][]float64, nt)
		models := make([][]float64, nt)
		weights := make([][]float64, nt)
		for k := 0; k < nt; k++ {
			signals[k] = make([]float64, n)
			models[k] = make([]float64, n)
			weights[k] = make([]float64, n)
			for i := 0; i < n; i++ {
				models[k][i] = 100 + float64(i)
				signals[k][i] = 0.7 * models[k][i]
				weights[k][i] = 1
			}
		}
		return BatchInput{Time: tm, Range: r, Signals: signals, Models: models, Weights: weights}
	}
	p := BatchParams{ProfileParams: validParams(), Omega: 2}
	// The batch grid in base() is 0..390 m; validParams() targets the
	// 401-point grid, so override grid-dependent parameters.
	p.Href = 200
	p.AnchorRange = [2]float64{300, 350}

	tests := []struct {
		name    string
		mutate  func(*BatchInput, *BatchParams)
		wantErr error
	}{
		{"nil input", func(in *BatchInput, p *BatchParams) { *in = BatchInput{} }, ErrNilInput},
		{"non-increasing time", func(in *BatchInput, p *BatchParams) { in.Time[2] = in.Time[1] }, ErrTimeNotIncreasing},
		{"profile length mismatch", func(in *BatchInput, p *BatchParams) { in.Signals[1] = in.Signals[1][:10] }, ErrProfileLength},
		{"weights length mismatch", func(in *BatchInput, p *BatchParams) { in.Weights[1] = in.Weights[1][:10] }, ErrProfileLength},
		{"negative omega", func(in *BatchInput, p *BatchParams) { p.Omega = -1 }, ErrInvalidParam},
		{"signals count mismatch", func(in *BatchInput, p *BatchParams) { in.Signals = in.Signals[:2] }, ErrLengthMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, pp := base(), p
			tt.mutate(&in, &pp)
			err := validateBatch(in, pp)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func nan() float64 {
	return math.NaN()
}
