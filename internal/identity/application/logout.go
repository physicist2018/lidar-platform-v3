package application

import (
	"context"
	"errors"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// LogoutUseCase revokes a refresh token, ending the session.
type LogoutUseCase struct {
	refreshRepo ports.RefreshTokenRepository
}

// NewLogoutUseCase creates a new LogoutUseCase.
func NewLogoutUseCase(refreshRepo ports.RefreshTokenRepository) *LogoutUseCase {
	return &LogoutUseCase{refreshRepo: refreshRepo}
}

// Execute revokes the given refresh token. Unknown or already-revoked
// tokens are treated as success so logout is idempotent.
func (uc *LogoutUseCase) Execute(ctx context.Context, refreshTokenStr string) error {
	hash := domain.HashRefreshToken(refreshTokenStr)

	stored, err := uc.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil
		}
		return err
	}

	if stored.IsRevoked() {
		return nil
	}
	return uc.refreshRepo.Revoke(ctx, stored.ID.String())
}
