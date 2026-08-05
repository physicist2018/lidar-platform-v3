package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/ports"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeRefreshRepo struct {
	byHash    map[string]*domain.RefreshToken
	created   []*domain.RefreshToken
	revoked   []string // token IDs passed to Revoke
	revokedBy []string // user IDs passed to RevokeAllForUser
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{byHash: map[string]*domain.RefreshToken{}}
}

func (f *fakeRefreshRepo) Create(_ context.Context, token *domain.RefreshToken) error {
	f.created = append(f.created, token)
	f.byHash[token.TokenHash] = token
	return nil
}

func (f *fakeRefreshRepo) FindByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	t, ok := f.byHash[hash]
	if !ok {
		return nil, domain.ErrRefreshTokenNotFound
	}
	return t, nil
}

func (f *fakeRefreshRepo) Revoke(_ context.Context, id string) error {
	for _, t := range f.byHash {
		if t.ID.String() == id {
			t.Revoke()
			f.revoked = append(f.revoked, id)
			return nil
		}
	}
	return nil
}

func (f *fakeRefreshRepo) RevokeAllForUser(_ context.Context, userID string) error {
	f.revokedBy = append(f.revokedBy, userID)
	return nil
}

type fakeUserRepo struct {
	users map[string]*domain.User
}

func (f *fakeUserRepo) Create(context.Context, *domain.User) error { return nil }
func (f *fakeUserRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (f *fakeUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}
func (f *fakeUserRepo) FindByVerificationToken(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (f *fakeUserRepo) UpdateStatus(context.Context, string, domain.UserStatus) error { return nil }

type fakeTokenSvc struct{}

func (fakeTokenSvc) GenerateToken(_ context.Context, _ string) (string, error) {
	return "new-access-token", nil
}
func (fakeTokenSvc) ValidateToken(string) (*ports.TokenClaims, error) { return nil, nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newRefreshHandler builds a RefreshHandler backed by fake repos and an
// active user, returning the fakes for assertions.
func newRefreshHandler(t *testing.T) (*RefreshHandler, *fakeRefreshRepo, *domain.User) {
	t.Helper()
	user := &domain.User{ID: uuid.New(), Status: domain.UserStatusActive}
	frr := newFakeRefreshRepo()
	fur := &fakeUserRepo{users: map[string]*domain.User{user.ID.String(): user}}
	uc := application.NewRefreshUseCase(fur, frr, fakeTokenSvc{}, time.Hour, 30*24*time.Hour)
	return NewRefreshHandler(uc), frr, user
}

// doRefreshRequest performs a POST /refresh with a JSON body and standard
// request metadata (User-Agent, RemoteAddr).
func doRefreshRequest(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "203.0.113.7:12345"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func storeToken(t *testing.T, frr *fakeRefreshRepo, tok *domain.RefreshToken) {
	t.Helper()
	require.NoError(t, frr.Create(context.Background(), tok))
}

type refreshResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// ---------------------------------------------------------------------------
// POST /refresh
// ---------------------------------------------------------------------------

func TestRefreshHandler_Success(t *testing.T) {
	h, frr, user := newRefreshHandler(t)
	old := domain.NewRefreshToken(user.ID, time.Hour, "", "")
	storeToken(t, frr, &old)

	body, err := json.Marshal(map[string]string{"refresh_token": old.Token})
	require.NoError(t, err)

	w := doRefreshRequest(h, string(body))
	require.Equal(t, http.StatusOK, w.Code)

	var resp refreshResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "new-access-token", resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, old.Token, resp.RefreshToken)
	assert.Equal(t, int64(time.Hour.Seconds()), resp.ExpiresIn)

	// Old token revoked, new token issued with UA/IP captured from the request.
	require.Len(t, frr.revoked, 1)
	assert.Equal(t, old.ID.String(), frr.revoked[0])
	require.Len(t, frr.created, 2)
	newTok := frr.created[1]
	assert.Equal(t, resp.RefreshToken, newTok.Token)
	assert.Equal(t, "test-agent", newTok.UserAgent)
	assert.Equal(t, "203.0.113.7", newTok.IP)
	assert.Empty(t, frr.revokedBy)
}

func TestRefreshHandler_InvalidJSON(t *testing.T) {
	h, _, _ := newRefreshHandler(t)

	w := doRefreshRequest(h, "{not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestRefreshHandler_MissingToken(t *testing.T) {
	h, _, _ := newRefreshHandler(t)

	w := doRefreshRequest(h, `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "refresh_token is required")
}

func TestRefreshHandler_UnknownToken(t *testing.T) {
	h, _, _ := newRefreshHandler(t)

	w := doRefreshRequest(h, `{"refresh_token": "nonexistent"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid refresh token")
}

func TestRefreshHandler_RevokedToken_RevokesFamily(t *testing.T) {
	h, frr, user := newRefreshHandler(t)
	old := domain.NewRefreshToken(user.ID, time.Hour, "", "")
	storeToken(t, frr, &old)
	old.Revoke()

	w := doRefreshRequest(h, `{"refresh_token": "`+old.Token+`"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid refresh token")

	require.Len(t, frr.revokedBy, 1)
	assert.Equal(t, user.ID.String(), frr.revokedBy[0])
}

func TestRefreshHandler_ExpiredToken(t *testing.T) {
	h, frr, user := newRefreshHandler(t)
	old := domain.NewRefreshToken(user.ID, time.Hour, "", "")
	old.ExpiresAt = time.Now().Add(-time.Minute)
	storeToken(t, frr, &old)

	w := doRefreshRequest(h, `{"refresh_token": "`+old.Token+`"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid refresh token")
}

// ---------------------------------------------------------------------------
// clientIP
// ---------------------------------------------------------------------------

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:8080"
	assert.Equal(t, "10.0.0.5", clientIP(req))

	req.RemoteAddr = "no-port"
	assert.Equal(t, "no-port", clientIP(req))
}
