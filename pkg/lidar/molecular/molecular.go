package molecular

import (
	"context"
	"log"
	"math"
)

// Compute calculates the pure molecular (Rayleigh) backscatter and extinction
// coefficient profiles and the range-corrected molecular lidar signal:
//
//	βm(r) = N(z(r))·σ_π(λ),     N = P/(k_B·T),
//	αm(r) = N(z(r))·σ_s(λ)  = (8π/3)·βm(r),
//	T_m²(r) = exp(−2·∫₀^r αm(s)ds),
//	M(r) = βm(r)·T_m²(r),
//
// where z(r) = r·cos(α) is the altitude along the beam (flat atmosphere) and
// T(z) is linearly interpolated from the atmosphere model while P(z) is
// interpolated in log space (barometric profile): the linear interpolation of
// ln P is exponentiated. Both are clamped to the model edges when the beam
// goes outside the model altitude range (a warning is logged in that case).
func Compute(ctx context.Context, in Input) (*Result, error) {
	if err := validate(in); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	n := len(in.Range)

	// Cross-sections are constant for the given wavelength.
	betaCS := backscatterCrossSection(in.Wavelength)
	alphaCS := extinctionCrossSection(in.Wavelength)

	// Altitude from range and zenith angle.
	cosZ := math.Cos(in.ZenithAngle * math.Pi / 180)
	alt := make([]float64, n)
	for i := range alt {
		alt[i] = in.Range[i] * cosZ / 1000 // km
	}

	// Interpolate temperature and pressure (clamped at the model edges).
	// Pressure follows an approximately barometric (exponential) profile, so
	// it is interpolated in log space: linear interpolation of ln P, then
	// exponentiated.
	logP := make([]float64, len(in.Atmosphere.Pressure))
	for i := range logP {
		logP[i] = math.Log(in.Atmosphere.Pressure[i])
	}
	tK := make([]float64, n)
	pPa := make([]float64, n)
	clamped := 0
	for i := range alt {
		tv, lo := interpolate(in.Atmosphere.Altitude, in.Atmosphere.Temperature, alt[i])
		lv, hi := interpolate(in.Atmosphere.Altitude, logP, alt[i])
		if lo || hi {
			clamped++
		}
		tK[i] = tv + 273.15
		pPa[i] = math.Exp(lv) * 100
	}
	if clamped > 0 {
		model := in.Atmosphere
		log.Printf("molecular: %d of %d range points outside model altitude [%g, %g] km — clamped to the edges",
			clamped, n, model.Altitude[0], model.Altitude[len(model.Altitude)-1])
	}

	res := &Result{
		Backscatter:  make([]float64, n),
		Extinction:   make([]float64, n),
		Transmission: make([]float64, n),
		Signal:       make([]float64, n),
	}

	// Number density → backscatter and extinction coefficients.
	for i := range n {
		density := pPa[i] / (boltzmann * tK[i])
		res.Backscatter[i] = density * betaCS
		res.Extinction[i] = density * alphaCS
	}

	// Two-way molecular transmission via the trapezoidal integral along the path.
	res.Transmission[0] = 1
	var opticalDepth float64 // ∫₀^r αm(s) ds
	for i := 1; i < n; i++ {
		opticalDepth += (res.Extinction[i-1] + res.Extinction[i]) / 2 * (in.Range[i] - in.Range[i-1])
		res.Transmission[i] = math.Exp(-2 * opticalDepth)
	}

	// Range-corrected molecular signal.
	for i := range n {
		res.Signal[i] = res.Backscatter[i] * res.Transmission[i]
	}

	return res, nil
}
