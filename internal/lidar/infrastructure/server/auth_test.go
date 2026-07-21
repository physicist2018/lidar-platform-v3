package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testClaims is used to generate tokens in tests.
type testClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

const testJWTSecret = "test-secret-for-testing"

func generateTestToken(t *testing.T, secret string, userID string, exp time.Time) string {
	t.Helper()
	claims := testClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// echoHandler is a test handler that returns 200 and echoes the user_id from context.
func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "no user_id in context", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(uid)) //nolint: errcheck
	})
}

func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	token := generateTestToken(t, testJWTSecret, "user-123", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-123", w.Body.String())
}

func TestJWTAuthMiddleware_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_EmptyBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "NotBearer token123")
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_WrongSecret(t *testing.T) {
	token := generateTestToken(t, "different-secret", "user-123", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_ExpiredToken(t *testing.T) {
	token := generateTestToken(t, testJWTSecret, "user-123", time.Now().Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_EmptySecretFallback(t *testing.T) {
	// When secret is empty, middleware generates an ephemeral key.
	// A token signed with a known key won't match the ephemeral one.
	token := generateTestToken(t, "some-key", "user-123", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware := JWTAuthMiddleware("")
	middleware(echoHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserIDFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, "test-user")
	uid, ok := UserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "test-user", uid)
}

func TestUserIDFromContext_NotFound(t *testing.T) {
	_, ok := UserIDFromContext(context.Background())
	assert.False(t, ok)

	ctx := context.WithValue(context.Background(), UserIDKey, 42) // wrong type
	_, ok = UserIDFromContext(ctx)
	assert.False(t, ok)
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"valid", "Bearer mytoken", "mytoken", false},
		{"lowercase bearer", "bearer mytoken", "mytoken", false},
		{"missing", "", "", true},
		{"wrong format", "Basic dXNlcjpwYXNz", "", true},
		{"empty token", "Bearer ", "", true},
		{"no space", "Bearermytoken", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			got, err := extractBearerToken(req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
