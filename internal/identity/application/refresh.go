package application

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// RefreshUseCase exchanges a valid refresh token for a new token pair.
// The old refresh token is rotated (revoked) and a new one is issued.
type RefreshUseCase struct {
	repo        ports.UserRepository
	refreshRepo ports.RefreshTokenRepository
	tokenSvc    ports.TokenService
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewRefreshUseCase creates a new RefreshUseCase.
func NewRefreshUseCase(
	repo ports.UserRepository,
	refreshRepo ports.RefreshTokenRepository,
	tokenSvc ports.TokenService,
	accessTTL, refreshTTL time.Duration,
) *RefreshUseCase {
	return &RefreshUseCase{
		repo:        repo,
		refreshRepo: refreshRepo,
		tokenSvc:    tokenSvc,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

// Execute validates the refresh token, rotates it, and issues a new access token.
func (uc *RefreshUseCase) Execute(ctx context.Context, refreshTokenStr, userAgent, ip string) (*TokenPair, error) {
	hash := domain.HashRefreshToken(refreshTokenStr)

	stored, err := uc.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil, domain.ErrInvalidRefreshToken
		}
		return nil, err
	}

	// Reuse of an already-revoked token suggests theft: revoke the whole family.
	if stored.IsRevoked() {
		if revokeErr := uc.refreshRepo.RevokeAllForUser(ctx, stored.UserID.String()); revokeErr != nil {
			log.Printf("refresh: failed to revoke token family for user %s: %v", stored.UserID, revokeErr)
		}
		return nil, domain.ErrInvalidRefreshToken
	}

	if stored.IsExpired() {
		return nil, domain.ErrInvalidRefreshToken
	}

	// Only active users may refresh their session.
	user, err := uc.repo.FindByID(ctx, stored.UserID.String())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidRefreshToken
		}
		return nil, err
	}
	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrInvalidRefreshToken
	}

	// Rotate: revoke the old token, issue a new one.
	if err := uc.refreshRepo.Revoke(ctx, stored.ID.String()); err != nil {
		return nil, err
	}

	newRefresh := domain.NewRefreshToken(user.ID, uc.refreshTTL, userAgent, ip)
	if err := uc.refreshRepo.Create(ctx, &newRefresh); err != nil {
		return nil, err
	}

	access, err := uc.tokenSvc.GenerateToken(ctx, user.ID.String())
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		Token:        access,
		RefreshToken: newRefresh.Token,
		ExpiresIn:    int64(uc.accessTTL.Seconds()),
	}, nil
}
