package molecular

import "errors"

// Sentinel errors returned by the package.
var (
	ErrNilInput           = errors.New("molecular: nil input")
	ErrEmptyInput         = errors.New("molecular: empty input")
	ErrTooFewPoints       = errors.New("molecular: at least two points are required")
	ErrLengthMismatch     = errors.New("molecular: length mismatch")
	ErrRangeNotIncreasing = errors.New("molecular: range and model altitude must be strictly increasing")
	ErrNonFinite          = errors.New("molecular: non-finite value")
	ErrInvalidParam       = errors.New("molecular: invalid parameter")
	ErrInvalidModel       = errors.New("molecular: invalid atmosphere model")
)
