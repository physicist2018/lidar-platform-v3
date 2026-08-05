package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

const (
	testAccessTTL  = time.Hour
	testRefreshTTL = 30 * 24 * time.Hour
)

func newRefreshUC(frr *fakeRefreshRepo, fur *fakeUserRepo) *RefreshUseCase {
	return NewRefreshUseCase(fur, frr, fakeTokenSvc{}, testAccessTTL, testRefreshTTL)
}

func storeToken(t *testing.T, frr *fakeRefreshRepo, tok *domain.RefreshToken) {
	t.Helper()
	require.NoError(t, frr.Create(context.Background(), tok))
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRefreshUseCase_Success(t *testing.T) {
	frr := newFakeRefreshRepo()
	user := &domain.User{ID: uuid.New(), Status: domain.UserStatusActive}
	uc := newRefreshUC(frr, &fakeUserRepo{users: map[string]*domain.User{user.ID.String(): user}})

	old := domain.NewRefreshToken(user.ID, testRefreshTTL, "ua", "ip")
	storeToken(t, frr, &old)

	pair, err := uc.Execute(context.Background(), old.Token, "ua2", "ip2")
	require.NoError(t, err)

	assert.Equal(t, "new-access-token", pair.Token)
	assert.Equal(t, int64(testAccessTTL.Seconds()), pair.ExpiresIn)

	// Old token revoked, new token created and returned.
	require.Len(t, frr.revoked, 1)
	assert.Equal(t, old.ID.String(), frr.revoked[0])
	require.Len(t, frr.created, 2)
	newTok := frr.created[1]
	assert.Equal(t, pair.RefreshToken, newTok.Token)
	assert.NotEqual(t, old.Token, newTok.Token)
	assert.Equal(t, "ua2", newTok.UserAgent)
	assert.Equal(t, "ip2", newTok.IP)
	assert.Empty(t, frr.revokedBy)
}

func TestRefreshUseCase_UnknownToken(t *testing.T) {
	frr := newFakeRefreshRepo()
	uc := newRefreshUC(frr, &fakeUserRepo{})

	_, err := uc.Execute(context.Background(), "nonexistent", "", "")
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	assert.Empty(t, frr.created)
	assert.Empty(t, frr.revoked)
}

func TestRefreshUseCase_RevokedToken_RevokesFamily(t *testing.T) {
	frr := newFakeRefreshRepo()
	user := &domain.User{ID: uuid.New(), Status: domain.UserStatusActive}
	uc := newRefreshUC(frr, &fakeUserRepo{users: map[string]*domain.User{user.ID.String(): user}})

	old := domain.NewRefreshToken(user.ID, testRefreshTTL, "", "")
	storeToken(t, frr, &old)
	old.Revoke()

	_, err := uc.Execute(context.Background(), old.Token, "", "")
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)

	// Reuse detection revokes the whole family and issues nothing.
	require.Len(t, frr.revokedBy, 1)
	assert.Equal(t, user.ID.String(), frr.revokedBy[0])
	assert.Len(t, frr.created, 1)
	assert.Empty(t, frr.revoked)
}

func TestRefreshUseCase_ExpiredToken(t *testing.T) {
	frr := newFakeRefreshRepo()
	user := &domain.User{ID: uuid.New(), Status: domain.UserStatusActive}
	uc := newRefreshUC(frr, &fakeUserRepo{users: map[string]*domain.User{user.ID.String(): user}})

	old := domain.NewRefreshToken(user.ID, testRefreshTTL, "", "")
	old.ExpiresAt = time.Now().Add(-time.Minute)
	storeToken(t, frr, &old)

	_, err := uc.Execute(context.Background(), old.Token, "", "")
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	assert.Empty(t, frr.revoked)
	assert.Len(t, frr.created, 1)
}

func TestRefreshUseCase_DisabledUser(t *testing.T) {
	frr := newFakeRefreshRepo()
	user := &domain.User{ID: uuid.New(), Status: domain.UserStatusDisabled}
	uc := newRefreshUC(frr, &fakeUserRepo{users: map[string]*domain.User{user.ID.String(): user}})

	old := domain.NewRefreshToken(user.ID, testRefreshTTL, "", "")
	storeToken(t, frr, &old)

	_, err := uc.Execute(context.Background(), old.Token, "", "")
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	assert.Empty(t, frr.revoked)
	assert.Len(t, frr.created, 1)
}

func TestRefreshUseCase_UnknownUser(t *testing.T) {
	frr := newFakeRefreshRepo()
	uc := newRefreshUC(frr, &fakeUserRepo{}) // empty user repo

	orphan := domain.NewRefreshToken(uuid.New(), testRefreshTTL, "", "")
	storeToken(t, frr, &orphan)

	_, err := uc.Execute(context.Background(), orphan.Token, "", "")
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}
