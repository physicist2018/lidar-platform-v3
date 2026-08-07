// Package tikhlidar performs Tikhonov smoothing of range-corrected lidar
// backscatter signals with a smooth (sigmoid) anchor to the molecular signal
// in the far zone where the aerosol contribution is absent.
//
// # Single profile
//
// Given the measured range-corrected signal S(r) and the modeled pure
// molecular signal M(r), the package produces a smoothed signal Ŝ(r) such that
//
//	Ŝ ≈ S            in the aerosol region (near zone),
//	Ŝ ≈ C·M          in the molecular region (far zone),
//
// with a smooth sigmoid transition of width L around the reference height
// Href. C is the anchoring (calibration) constant computed in the anchor
// range [r0, r1], where the aerosol backscatter is absent (S ≈ C·M).
//
// The signals are NOT corrected for transmission T²: the measured far-zone
// signal is noisy and contaminated by unknown aerosol transmission, so it is
// replaced by the molecular model there. The constant C, estimated in [r0, r1]
// (where the backscatter ratio equals one), absorbs the approximately constant
// aerosol transmission over the window.
//
// # Objective
//
// The smoothed signal minimizes
//
//	Φ(Ŝ) = Σᵢ (1−wᵢ)·uᵢ·(Sᵢ − Ŝᵢ)² + λ²·Σᵢ (D²_r Ŝ)ᵢ² + q·Σᵢ wᵢ·(Ŝᵢ − C·Mᵢ)²,
//
// where w(r) = 1/(1 + exp(−(r − Href)/s)), s = L/4, is the logistic anchor
// weight (w→0 in the aerosol region, w→1 in the molecular region), u(r) is a
// user-provided per-point weight profile (a multiplier of the fit term;
// u ≥ 0, u = 0 turns the fit off at that point), D²_r is the second
// derivative with respect to range (one-sided differences at the boundaries),
// λ is the distance smoothing parameter, and q is the anchor strength.
//
// The objective is quadratic, so the solution is a linear symmetric
// positive-definite pentadiagonal system, solved with gonum's banded Cholesky
// factorization.
//
// Signals may contain negative values (e.g. after background subtraction);
// the model must stay positive.
//
// # asinh (log-like) helpers
//
// SmoothProfileAsinh and SmoothBatchAsinh wrap the smoothing for signals with
// a large dynamic range. The signal is first normalized by the calibration
// constant C = median(S/M) over the anchor range (so the model and the signal
// share a scale and the log-domain calibration offset vanishes), then both
// the signal and the model are transformed with the variance-stabilizing
// transform x → asinh(x/eps) (≈ ln(x) for x ≫ eps, linear for x ≪ eps), the
// smoothing runs in the transformed domain with weights |S_t|, and the result
// is transformed back with Ŝ = C·eps·sinh(Ŝ_t). eps is a small parameter
// (typically 1e-6 .. 1e-3) relative to the model scale.
//
// # Batch of profiles
//
// SmoothBatch processes a set of profiles on a common range grid and adds
// temporal smoothing:
//
//   - ω²·Σₖ (D²_t Ŝ)ₖ²,
//
// where D²_t is the second derivative with respect to time and ω is the
// temporal smoothing parameter. The resulting block-structured system is
// solved with a conjugate-gradient solver built on gonum's blas64 primitives.
// For nt < 3 profiles (or ω = 0) the profiles are processed independently.
package tikhlidar
