package ports

import (
	"context"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
)

// UserRepository defines the persistence contract for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByVerificationToken(ctx context.Context, token string) (*domain.User, error)
	UpdateStatus(ctx context.Context, id string, status domain.UserStatus) error
}
