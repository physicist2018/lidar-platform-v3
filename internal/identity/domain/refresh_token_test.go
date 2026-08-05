package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefreshToken(t *testing.T) {
	userID := uuid.New()
	ttl := 30 * 24 * time.Hour

	token := NewRefreshToken(userID, ttl, "test-agent", "127.0.0.1")

	require.NotEmpty(t, token.Token)
	assert.Equal(t, HashRefreshToken(token.Token), token.TokenHash)
	assert.NotEqual(t, token.Token, token.TokenHash)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, "test-agent", token.UserAgent)
	assert.Equal(t, "127.0.0.1", token.IP)
	assert.False(t, token.IsRevoked())
	assert.False(t, token.IsExpired())
	assert.WithinDuration(t, time.Now().Add(ttl), token.ExpiresAt, time.Minute)
	assert.NotEqual(t, uuid.Nil, token.ID)
}

func TestNewRefreshToken_GeneratesDistinctTokens(t *testing.T) {
	a := NewRefreshToken(uuid.New(), time.Hour, "", "")
	b := NewRefreshToken(uuid.New(), time.Hour, "", "")

	assert.NotEqual(t, a.Token, b.Token)
	assert.NotEqual(t, a.TokenHash, b.TokenHash)
}

func TestHashRefreshToken(t *testing.T) {
	assert.Equal(t, HashRefreshToken("abc"), HashRefreshToken("abc"))
	assert.NotEqual(t, HashRefreshToken("abc"), HashRefreshToken("abd"))
	assert.Len(t, HashRefreshToken("abc"), 64) // SHA-256 hex digest
}

func TestRefreshToken_IsExpired(t *testing.T) {
	future := NewRefreshToken(uuid.New(), time.Hour, "", "")
	future.ExpiresAt = time.Now().Add(time.Hour)
	assert.False(t, future.IsExpired())

	past := NewRefreshToken(uuid.New(), time.Hour, "", "")
	past.ExpiresAt = time.Now().Add(-time.Minute)
	assert.True(t, past.IsExpired())
}

func TestRefreshToken_Revoke(t *testing.T) {
	token := NewRefreshToken(uuid.New(), time.Hour, "", "")
	assert.False(t, token.IsRevoked())

	token.Revoke()
	assert.True(t, token.IsRevoked())
	assert.NotNil(t, token.RevokedAt)
}
