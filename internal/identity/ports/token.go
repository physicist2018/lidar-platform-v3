package ports

import "context"

// TokenService defines the contract for JWT token operations.
type TokenService interface {
	GenerateToken(ctx context.Context, userID string) (string, error)
	ValidateToken(token string) (*TokenClaims, error)
}

// TokenClaims represents the claims embedded in a token.
type TokenClaims struct {
	UserID string
}
