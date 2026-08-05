package ports

import (
	"context"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

// RefreshTokenRepository defines the persistence contract for refresh tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}
