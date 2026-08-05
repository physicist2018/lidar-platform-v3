package application

import (
	"context"
	"errors"
	"time"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// LoginUseCase handles user authentication.
type LoginUseCase struct {
	repo        ports.UserRepository
	refreshRepo ports.RefreshTokenRepository
	token       ports.TokenService
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(
	repo ports.UserRepository,
	refreshRepo ports.RefreshTokenRepository,
	token ports.TokenService,
	accessTTL, refreshTTL time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		repo:        repo,
		refreshRepo: refreshRepo,
		token:       token,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

// Execute authenticates a user and returns an access token pair
// (JWT access token + opaque refresh token).
func (uc *LoginUseCase) Execute(ctx context.Context, emailStr, password, userAgent, ip string) (*TokenPair, error) {
	// 1. Validate email format
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, domain.ErrInvalidEmail
	}

	// 2. Look up the user
	user, err := uc.repo.FindByEmail(ctx, email.String())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Return a generic message to avoid leaking whether the email exists.
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	// 3. Verify password
	if !user.ComparePassword(password) {
		return nil, domain.ErrInvalidCredentials
	}

	// 4. Check user is active
	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrAccountNotVerified
	}

	// 5. Generate access token
	access, err := uc.token.GenerateToken(ctx, user.ID.String())
	if err != nil {
		return nil, err
	}

	// 6. Issue a refresh token
	refresh := domain.NewRefreshToken(user.ID, uc.refreshTTL, userAgent, ip)
	if err := uc.refreshRepo.Create(ctx, &refresh); err != nil {
		return nil, err
	}

	return &TokenPair{
		Token:        access,
		RefreshToken: refresh.Token,
		ExpiresIn:    int64(uc.accessTTL.Seconds()),
	}, nil
}
