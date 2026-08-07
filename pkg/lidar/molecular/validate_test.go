package molecular

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validInput() Input {
	return Input{
		Range:       []float64{0, 100, 200},
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere: AtmosphereModel{
			Altitude:    []float64{0, 5},
			Temperature: []float64{15, -20},
			Pressure:    []float64{1013, 540},
		},
	}
}

func TestValidate_OK(t *testing.T) {
	assert.NoError(t, validate(validInput()))
}

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Input)
		wantErr error
	}{
		{"nil range", func(in *Input) { in.Range = nil }, ErrNilInput},
		{"nil model", func(in *Input) { in.Atmosphere.Temperature = nil }, ErrNilInput},
		{"too few points", func(in *Input) { in.Range = in.Range[:1] }, ErrTooFewPoints},
		{"non-increasing range", func(in *Input) { in.Range[2] = in.Range[1] }, ErrRangeNotIncreasing},
		{"negative range", func(in *Input) { in.Range[0] = -1 }, ErrInvalidParam},
		{"NaN range", func(in *Input) { in.Range[1] = math.NaN() }, ErrNonFinite},
		{"wavelength too small", func(in *Input) { in.Wavelength = 100 }, ErrInvalidParam},
		{"NaN wavelength", func(in *Input) { in.Wavelength = math.NaN() }, ErrNonFinite},
		{"zenith negative", func(in *Input) { in.ZenithAngle = -1 }, ErrInvalidParam},
		{"zenith above 90", func(in *Input) { in.ZenithAngle = 91 }, ErrInvalidParam},
		{"model length mismatch", func(in *Input) { in.Atmosphere.Pressure = in.Atmosphere.Pressure[:1] }, ErrLengthMismatch},
		{"model altitude not increasing", func(in *Input) { in.Atmosphere.Altitude[1] = 0 }, ErrRangeNotIncreasing},
		{"zero pressure", func(in *Input) { in.Atmosphere.Pressure[0] = 0 }, ErrInvalidModel},
		{"temperature below absolute zero", func(in *Input) { in.Atmosphere.Temperature[1] = -300 }, ErrInvalidModel},
		{"NaN temperature", func(in *Input) { in.Atmosphere.Temperature[0] = math.NaN() }, ErrNonFinite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.mutate(&in)
			err := validate(in)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCompute_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Compute(ctx, validInput())
	assert.ErrorIs(t, err, context.Canceled)
}
