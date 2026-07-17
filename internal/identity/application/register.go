package application

import (
	"context"
	"errors"
	"log"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// RegisterUseCase orchestrates user registration.
type RegisterUseCase struct {
	repo   ports.UserRepository
	mailer ports.MailSender
}

// NewRegisterUseCase creates a new RegisterUseCase.
func NewRegisterUseCase(repo ports.UserRepository, mailer ports.MailSender) *RegisterUseCase {
	return &RegisterUseCase{repo: repo, mailer: mailer}
}

// Execute registers a new user, saves to the database, and sends a verification email.
func (uc *RegisterUseCase) Execute(ctx context.Context, emailStr, passwordStr string) error {
	// 1. Validate email
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return err
	}

	// 2. Validate and hash password
	password, err := domain.NewPassword(passwordStr)
	if err != nil {
		return err
	}

	// 3. Check if email already exists
	existing, err := uc.repo.FindByEmail(ctx, email.String())
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return err
	}
	if existing != nil {
		return domain.ErrEmailAlreadyExists
	}

	// 4. Create domain user with pending status + verification token
	user := domain.NewPendingUser(email, password)

	// 5. Persist
	if err := uc.repo.Create(ctx, &user); err != nil {
		return err
	}

	// 6. Send verification email (best-effort: log failure but don't roll back)
	if err := uc.mailer.SendVerificationEmail(ctx, email.String(), user.VerificationToken.Token); err != nil {
		// In a production system you'd enqueue a retry. For now just log.
		log.Printf("failed to send verification email to %s: %v", email.String(), err)
	}
	log.Println("Succesful")
	return nil
}
