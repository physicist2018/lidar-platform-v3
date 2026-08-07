// Package molecular computes the pure molecular (Rayleigh) backscatter and
// extinction coefficient profiles and the corresponding range-corrected lidar
// signal for a given line-of-sight range grid, zenith angle, wavelength and
// atmosphere model.
//
// # Model
//
// The altitude along the beam is z(r) = r·cos(α) (flat-atmosphere
// approximation), where α is the zenith angle. Temperature T(z) and pressure
// P(z) are linearly interpolated from the atmosphere model; range points
// outside the model altitude range are clamped to the model edges (a warning
// is logged).
//
// The molecular number density follows the ideal gas law:
//
//	N(z) = P(z) / (k_B·T(z)).
//
// The Rayleigh cross-sections per molecule (Bucholtz 1995; Bodhaine et al.
// 1999) are computed from the dry-air refractive index n(λ) (Edlén dispersion
// formula, valid for λ > 160 nm) and the Loschmidt constant N_s:
//
//	σ_π(λ) = (π²/λ⁴)·[(n²−1)/N_s]²·F_k,        backscatter, m²·sr⁻¹
//	σ_s(λ) = (8π/3)·σ_π(λ),                    extinction, m²
//
// where F_k = (6+3ρ)/(6−7ρ) is the King correction factor with the air
// depolarization ratio ρ = 0.0272. The molecular lidar ratio is therefore
// exactly S_m = αm/βm = 8π/3 ≈ 8.38 sr.
//
// The output profiles are:
//
//	βm(r)  = N(z(r))·σ_π(λ),                        m⁻¹·sr⁻¹
//	αm(r)  = N(z(r))·σ_s(λ) = (8π/3)·βm(r),         m⁻¹
//	T_m²(r) = exp(−2·∫₀^r αm(s) ds),                dimensionless
//	M(r)   = βm(r)·T_m²(r),                         m⁻¹·sr⁻¹
//
// M(r) is the range-corrected pure molecular lidar signal (the lidar equation
// P(r) = C·β(r)·T²(r)/r², range-corrected by multiplying by r²). The
// instrument calibration constant C is intentionally not included: consumers
// estimate it themselves (e.g. the pkg/smooth/tikhlidar package calibrates it
// in its anchor range, so M(r) can be passed directly as its Model input).
//
// # Units
//
// Inputs follow the conventions of the lidar domain: range in meters, zenith
// angle in degrees, wavelength in nanometers, model altitude in km,
// temperature in °C and pressure in hPa. Outputs are in SI (m⁻¹·sr⁻¹, m⁻¹).
package molecular
