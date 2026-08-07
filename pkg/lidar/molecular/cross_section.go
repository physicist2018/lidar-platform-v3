package molecular

import "math"

// Physical constants (SI).
const (
	boltzmann      = 1.380649e-23 // J/K
	standardTemp   = 288.15       // K (15 °C)
	standardPress  = 1013.25      // hPa (1013.25 hPa = 1 atm)
	depolarization = 0.0272       // depolarization ratio of air
)

// standardDensity returns the molecular number density at standard conditions
// (288.15 K, 1013.25 hPa), the Loschmidt constant, in m⁻³.
func standardDensity() float64 {
	return standardPress * 100 / (boltzmann * standardTemp)
}

// refractiveIndexMinus1 returns (n(λ) − 1) of dry air at standard conditions
// using the Edlén dispersion formula, with λ in nanometers:
//
//	(n−1)·10⁸ = 8342.13 + 2406030/(130 − σ²) + 15997/(38.9 − σ²),  σ = 1/λ[µm].
//
// The formula is valid for λ > 160 nm (σ² < 38.9).
func refractiveIndexMinus1(wvlNm float64) float64 {
	sigma := 1000 / wvlNm // 1/λ with λ in µm
	s2 := sigma * sigma
	edlen := 8342.13 + 2406030/(130-s2) + 15997/(38.9-s2)
	return edlen * 1e-8
}

// kingFactor returns the King correction factor F_k = (6+3ρ)/(6−7ρ) for the
// air depolarization ratio ρ.
func kingFactor() float64 {
	rho := depolarization
	return (6 + 3*rho) / (6 - 7*rho)
}

// backscatterCrossSection returns the Rayleigh backscatter cross-section per
// molecule σ_π(λ) in m²·sr⁻¹ (Bucholtz 1995 / Bodhaine et al. 1999):
//
//	σ_π(λ) = (π²/λ⁴)·[(n²−1)/N_s]²·F_k.
func backscatterCrossSection(wvlNm float64) float64 {
	lam := wvlNm * 1e-9 // m
	ns := standardDensity()
	n1 := refractiveIndexMinus1(wvlNm)
	n2m1 := n1 * (2 + n1) // n²−1
	ratio := n2m1 / ns
	return math.Pi * math.Pi / math.Pow(lam, 4) * ratio * ratio * kingFactor()
}

// extinctionCrossSection returns the Rayleigh extinction cross-section per
// molecule σ_s(λ) in m². The molecular lidar ratio is exactly 8π/3:
//
//	σ_s(λ) = (8π/3)·σ_π(λ).
func extinctionCrossSection(wvlNm float64) float64 {
	return (8 * math.Pi / 3) * backscatterCrossSection(wvlNm)
}
