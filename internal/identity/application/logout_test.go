package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

func TestLogoutUseCase_Success(t *testing.T) {
	frr := newFakeRefreshRepo()
	tok := domain.NewRefreshToken(uuid.New(), time.Hour, "", "")
	storeToken(t, frr, &tok)

	uc := NewLogoutUseCase(frr)
	require.NoError(t, uc.Execute(context.Background(), tok.Token))
	assert.True(t, tok.IsRevoked())
}

func TestLogoutUseCase_UnknownToken_IsIdempotent(t *testing.T) {
	frr := newFakeRefreshRepo()
	uc := NewLogoutUseCase(frr)
	assert.NoError(t, uc.Execute(context.Background(), "nonexistent"))
}

func TestLogoutUseCase_AlreadyRevoked_IsIdempotent(t *testing.T) {
	frr := newFakeRefreshRepo()
	tok := domain.NewRefreshToken(uuid.New(), time.Hour, "", "")
	storeToken(t, frr, &tok)
	tok.Revoke()

	uc := NewLogoutUseCase(frr)
	assert.NoError(t, uc.Execute(context.Background(), tok.Token))
}
