package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/identity"
)

// PostgresUserRepository implements ports.UserRepository backed by sqlc.
type PostgresUserRepository struct {
	q *db.Queries
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(dbtx db.DBTX) *PostgresUserRepository {
	return &PostgresUserRepository{q: db.New(dbtx)}
}

// Create persists a new user.
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	token := sql.NullString{}
	expiresAt := sql.NullTime{}
	if user.VerificationToken != nil {
		token = sql.NullString{String: user.VerificationToken.Token, Valid: true}
		expiresAt = sql.NullTime{Time: user.VerificationToken.ExpiresAt, Valid: true}
	}
	_, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Email:             user.Email.String(),
		PasswordHash:      user.PasswordHash,
		Status:            string(user.Status),
		VerificationToken: token,
		TokenExpiresAt:    expiresAt,
	})
	return err
}

// FindByEmail looks up a user by email.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.q.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapUser(u), nil
}

// FindByID looks up a user by ID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	u, err := r.q.FindUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapUser(u), nil
}

// FindByVerificationToken looks up a user by their verification token.
// The underlying SQL query also verifies the token is not expired.
func (r *PostgresUserRepository) FindByVerificationToken(ctx context.Context, token string) (*domain.User, error) {
	u, err := r.q.FindByVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapUser(u), nil
}

// UpdateStatus changes a user's status and clears the verification token.
func (r *PostgresUserRepository) UpdateStatus(ctx context.Context, id string, status domain.UserStatus) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = r.q.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		ID:     uid,
		Status: string(status),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		return err
	}
	return nil
}

// mapUser converts a sqlc model to a domain User.
func mapUser(u db.IdentityUser) *domain.User {
	user := &domain.User{
		ID:           u.ID,
		Email:        domain.NewEmailUnsafe(u.Email),
		PasswordHash: u.PasswordHash,
		Status:       domain.UserStatus(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
	if u.VerificationToken.Valid && u.TokenExpiresAt.Valid {
		user.VerificationToken = &domain.VerificationToken{
			Token:     u.VerificationToken.String,
			ExpiresAt: u.TokenExpiresAt.Time,
		}
	}
	return user
}
