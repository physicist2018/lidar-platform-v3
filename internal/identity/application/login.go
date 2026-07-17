package application

import (
	"context"
	"errors"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// LoginUseCase handles user authentication.
type LoginUseCase struct {
	repo  ports.UserRepository
	token ports.TokenService
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(repo ports.UserRepository, token ports.TokenService) *LoginUseCase {
	return &LoginUseCase{repo: repo, token: token}
}

// LoginResult is returned on successful authentication.
type LoginResult struct {
	Token string `json:"token"`
}

// Execute authenticates a user and returns a JWT token.
func (uc *LoginUseCase) Execute(ctx context.Context, emailStr, password string) (*LoginResult, error) {
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

	// 5. Generate JWT
	tokenStr, err := uc.token.GenerateToken(ctx, user.ID.String())
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: tokenStr}, nil
}
