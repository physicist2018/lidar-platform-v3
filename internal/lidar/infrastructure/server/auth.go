package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
)

// tokenClaims represents the JWT claims expected from the identity service.
type tokenClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTAuthMiddleware validates Bearer JWTs and injects user_id into the request context.
// If the token is missing or invalid, it responds with 401 Unauthorized.
//
// If secret is empty, a random ephemeral key is generated — tokens signed with it
// will be invalidated on restart. In production, always set JWT_SECRET explicitly
// and ensure it matches the secret used by the identity service.
func JWTAuthMiddleware(secret string) func(http.Handler) http.Handler {
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Printf("WARNING: failed to generate random JWT secret: %v", err)
			return func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					writeUnauthorized(w, "JWT secret not configured")
				})
			}
		}
		secret = hex.EncodeToString(b)
		log.Printf("WARNING: JWT_SECRET not set, using ephemeral secret — tokens will be invalidated on restart")
	}

	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := extractBearerToken(r)
			if err != nil {
				writeUnauthorized(w, err.Error())
				return
			}

			claims := &tokenClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return secretBytes, nil
			})
			if err != nil {
				log.Printf("jwt: parse error: %v", err)
				writeUnauthorized(w, "invalid token")
				return
			}
			if !token.Valid {
				writeUnauthorized(w, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user ID from the request context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(UserIDKey).(string)
	return uid, ok
}

func extractBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid Authorization header format, expected 'Bearer <token>'")
	}

	if parts[1] == "" {
		return "", fmt.Errorf("empty token")
	}

	return parts[1], nil
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": message}) // nolint: errcheck
}
