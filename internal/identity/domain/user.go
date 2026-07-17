package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"

	tokenBytes    = 32
	tokenDuration = 24 * time.Hour
)

// UserStatus represents the lifecycle status of a user.
type UserStatus string

// VerificationToken is a value object for email verification.
type VerificationToken struct {
	Token     string
	ExpiresAt time.Time
}

// NewVerificationToken generates a random hex token with a 24-hour expiry.
func NewVerificationToken() VerificationToken {
	b := make([]byte, tokenBytes)
	rand.Read(b) // nolint: errcheck — rand.Read always returns len(b) and nil error
	return VerificationToken{
		Token:     hex.EncodeToString(b),
		ExpiresAt: time.Now().Add(tokenDuration),
	}
}

// IsExpired checks whether the token has expired.
func (vt VerificationToken) IsExpired() bool {
	return time.Now().After(vt.ExpiresAt)
}

// User is the central domain entity.
type User struct {
	ID                uuid.UUID
	Email             Email
	PasswordHash      string
	Status            UserStatus
	VerificationToken *VerificationToken
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewPendingUser creates a new user with pending status and a verification token.
func NewPendingUser(email Email, password Password) User {
	token := NewVerificationToken()
	return User{
		ID:                uuid.New(),
		Email:             email,
		PasswordHash:      password.Hash(),
		Status:            UserStatusPending,
		VerificationToken: &token,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// ComparePassword checks whether the given plain-text password matches the stored hash.
func (u *User) ComparePassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plain)) == nil
}

// Verify transitions the user from pending to active.
// Returns an error if the token is invalid, expired, or the user is already active.
func (u *User) Verify(token string, email string) error {
	if u.Status == UserStatusActive {
		return ErrAlreadyVerified
	}

	if u.VerificationToken == nil || u.VerificationToken.Token != token {
		return ErrInvalidToken
	}

	if u.VerificationToken.IsExpired() {
		return ErrTokenExpired
	}

	if !u.Email.Equals(NewEmailUnsafe(email)) {
		return ErrEmailMismatch
	}

	u.Status = UserStatusActive
	u.VerificationToken = nil
	u.UpdatedAt = time.Now()
	return nil
}
