package domain

import "errors"

var (
	ErrInvalidPath         = errors.New("invalid storage path: bucket and path must not be empty")
	ErrObjectNotFound      = errors.New("storage object not found")
	ErrInvalidTimeRange    = errors.New("invalid time range: start must be before end")
	ErrInvalidGeoLocation  = errors.New("invalid geolocation: latitude must be -90..90, longitude -180..180")
	ErrProfileDataMismatch = errors.New("altitude, temperature and pressure arrays must have the same length")
)
