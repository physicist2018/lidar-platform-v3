package domain

import "errors"

var (
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrInvalidEmail         = errors.New("invalid email format")
	ErrWeakPassword         = errors.New("password must be at least 8 characters")
	ErrInvalidToken         = errors.New("invalid verification token")
	ErrTokenExpired         = errors.New("verification token expired")
	ErrEmailMismatch        = errors.New("email does not match token")
	ErrAlreadyVerified      = errors.New("user already verified")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrAccountNotVerified   = errors.New("account not verified")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrRefreshTokenExpired  = errors.New("refresh token expired")
)
