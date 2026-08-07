package tikhlidar

import "errors"

// Sentinel errors returned by the package.
var (
	ErrNilInput            = errors.New("tikhlidar: nil input")
	ErrEmptyInput          = errors.New("tikhlidar: empty input")
	ErrLengthMismatch      = errors.New("tikhlidar: length mismatch")
	ErrProfileLength       = errors.New("tikhlidar: profile length mismatch")
	ErrTooFewPoints        = errors.New("tikhlidar: at least three range points are required")
	ErrRangeNotIncreasing  = errors.New("tikhlidar: range must be strictly increasing")
	ErrTimeNotIncreasing   = errors.New("tikhlidar: time must be strictly increasing")
	ErrNonFinite           = errors.New("tikhlidar: non-finite value")
	ErrInvalidParam        = errors.New("tikhlidar: invalid parameter")
	ErrAnchorRange         = errors.New("tikhlidar: anchor range must contain at least two points of the range grid")
	ErrModelNonPositive    = errors.New("tikhlidar: model must be positive")
	ErrNotPositiveDefinite = errors.New("tikhlidar: system matrix is not positive definite")
	ErrNotConverged        = errors.New("tikhlidar: solver did not converge")
)
