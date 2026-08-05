package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

const refreshTokenBytes = 32

// RefreshToken is a long-lived credential used to obtain new access tokens.
// Only the SHA-256 hash of the token is persisted; the plaintext lives
// exclusively on the client.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string // plaintext, only populated right after creation
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UserAgent string
	IP        string
}

// NewRefreshToken generates a random opaque token with the given TTL.
func NewRefreshToken(userID uuid.UUID, ttl time.Duration, userAgent, ip string) RefreshToken {
	b := make([]byte, refreshTokenBytes)
	rand.Read(b) // nolint: errcheck — rand.Read always returns len(b) and nil error
	token := hex.EncodeToString(b)
	now := time.Now()
	return RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		TokenHash: HashRefreshToken(token),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UserAgent: userAgent,
		IP:        ip,
	}
}

// HashRefreshToken computes the SHA-256 hex digest of a plaintext token.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsExpired reports whether the token has passed its expiry.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked reports whether the token was explicitly revoked.
func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// Revoke marks the token as revoked.
func (t *RefreshToken) Revoke() {
	now := time.Now()
	t.RevokedAt = &now
}
