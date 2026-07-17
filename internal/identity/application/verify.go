package application

import (
	"context"
	"errors"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// VerifyUseCase orchestrates email verification.
type VerifyUseCase struct {
	repo ports.UserRepository
}

// NewVerifyUseCase creates a new VerifyUseCase.
func NewVerifyUseCase(repo ports.UserRepository) *VerifyUseCase {
	return &VerifyUseCase{repo: repo}
}

// Execute verifies a user by token and email.
func (uc *VerifyUseCase) Execute(ctx context.Context, token, email string) error {
	// 1. Find user by verification token
	user, err := uc.repo.FindByVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrInvalidToken
		}
		return err
	}

	// 2. Attempt verification on the domain entity
	if err := user.Verify(token, email); err != nil {
		return err
	}

	// 3. Persist status change
	if err := uc.repo.UpdateStatus(ctx, user.ID.String(), domain.UserStatusActive); err != nil {
		return err
	}

	return nil
}
