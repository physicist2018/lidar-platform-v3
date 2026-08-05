package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/identity"
)

// PostgresRefreshTokenRepository implements ports.RefreshTokenRepository backed by sqlc.
type PostgresRefreshTokenRepository struct {
	q *db.Queries
}

// NewPostgresRefreshTokenRepository creates a new PostgresRefreshTokenRepository.
func NewPostgresRefreshTokenRepository(dbtx db.DBTX) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{q: db.New(dbtx)}
}

// Create persists a new refresh token.
func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		UserAgent: sql.NullString{String: token.UserAgent, Valid: token.UserAgent != ""},
		Ip:        sql.NullString{String: token.IP, Valid: token.IP != ""},
	})
	return err
}

// FindByHash looks up a refresh token by its SHA-256 hash.
func (r *PostgresRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	t, err := r.q.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return mapRefreshToken(t), nil
}

// Revoke marks a refresh token as revoked. Revoking an already-revoked
// token is treated as success.
func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = r.q.RevokeRefreshToken(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

// RevokeAllForUser revokes every active refresh token of the user.
func (r *PostgresRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.RevokeAllUserTokens(ctx, uid)
}

// mapRefreshToken converts a sqlc model to a domain RefreshToken.
func mapRefreshToken(t db.IdentityRefreshToken) *domain.RefreshToken {
	token := &domain.RefreshToken{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
	if t.RevokedAt.Valid {
		revoked := t.RevokedAt.Time
		token.RevokedAt = &revoked
	}
	if t.UserAgent.Valid {
		token.UserAgent = t.UserAgent.String
	}
	if t.Ip.Valid {
		token.IP = t.Ip.String
	}
	return token
}
