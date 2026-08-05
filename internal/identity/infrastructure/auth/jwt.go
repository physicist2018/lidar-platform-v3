package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

const defaultAccessTTL = time.Hour

// JWTTokenService implements ports.TokenService using HS256 JWTs.
type JWTTokenService struct {
	secret    []byte
	accessTTL time.Duration
}

// NewJWTTokenService creates a new JWTTokenService.
// If accessTTL is non-positive, a default of one hour is used.
func NewJWTTokenService(secret string, accessTTL time.Duration) *JWTTokenService {
	if accessTTL <= 0 {
		accessTTL = defaultAccessTTL
	}
	return &JWTTokenService{secret: []byte(secret), accessTTL: accessTTL}
}

type claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user ID.
func (s *JWTTokenService) GenerateToken(_ context.Context, userID string) (string, error) {
	now := time.Now()
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT, returning the embedded claims.
func (s *JWTTokenService) ValidateToken(tokenStr string) (*ports.TokenClaims, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt: parse: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}
	return &ports.TokenClaims{UserID: c.UserID}, nil
}
