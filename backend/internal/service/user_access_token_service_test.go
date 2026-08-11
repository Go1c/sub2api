//go:build unit

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type userAccessTokenRepoStub struct {
	byHash map[string]*UserAccessToken
	byID   map[int64]*UserAccessToken
	nextID int64
}

func newUserAccessTokenRepoStub() *userAccessTokenRepoStub {
	return &userAccessTokenRepoStub{
		byHash: make(map[string]*UserAccessToken),
		byID:   make(map[int64]*UserAccessToken),
		nextID: 1,
	}
}

func (r *userAccessTokenRepoStub) Create(_ context.Context, token *UserAccessToken) error {
	if token == nil {
		return ErrUserAccessTokenNotFound
	}
	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	cp := *token
	cp.ID = id
	cp.CreatedAt = now
	cp.UpdatedAt = now
	r.byID[id] = &cp
	r.byHash[cp.TokenHash] = &cp
	token.ID = id
	token.CreatedAt = now
	token.UpdatedAt = now
	return nil
}

func (r *userAccessTokenRepoStub) GetByID(_ context.Context, id int64) (*UserAccessToken, error) {
	t, ok := r.byID[id]
	if !ok {
		return nil, ErrUserAccessTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *userAccessTokenRepoStub) GetByTokenHash(_ context.Context, hash string) (*UserAccessToken, error) {
	t, ok := r.byHash[hash]
	if !ok {
		return nil, ErrUserAccessTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *userAccessTokenRepoStub) ListByUserID(_ context.Context, userID int64) ([]UserAccessToken, error) {
	out := make([]UserAccessToken, 0)
	for _, t := range r.byID {
		if t.UserID == userID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *userAccessTokenRepoStub) RevokeByIDForUser(_ context.Context, userID, id int64, revokedAt time.Time) error {
	t, ok := r.byID[id]
	if !ok || t.UserID != userID {
		return ErrUserAccessTokenNotFound
	}
	t.RevokedAt = &revokedAt
	t.UpdatedAt = revokedAt
	return nil
}

func (r *userAccessTokenRepoStub) TouchLastUsedAt(_ context.Context, id int64, usedAt time.Time) error {
	t, ok := r.byID[id]
	if !ok {
		return ErrUserAccessTokenNotFound
	}
	t.LastUsedAt = &usedAt
	t.UpdatedAt = usedAt
	return nil
}

func TestUserAccessTokenService_Create_DefaultExpiresIn7Days(t *testing.T) {
	repo := newUserAccessTokenRepoStub()
	svc := NewUserAccessTokenService(repo)
	before := time.Now().UTC()

	created, err := svc.Create(context.Background(), 11, CreateUserAccessTokenInput{Name: "ci"})
	require.NoError(t, err)
	require.NotEmpty(t, created.Token)
	require.True(t, strings.HasPrefix(created.Token, userAccessTokenPrefix))
	require.Equal(t, "ci", created.Name)
	require.Equal(t, int64(11), created.UserID)
	require.Empty(t, created.TokenHash) // response model should not expose hash by accident in Token field path
	require.NotEmpty(t, created.TokenPrefix)
	require.True(t, strings.HasPrefix(created.TokenPrefix, userAccessTokenPrefix))

	// expires ≈ now + 7d
	wantMin := before.Add(7 * 24 * time.Hour).Add(-2 * time.Second)
	wantMax := before.Add(7 * 24 * time.Hour).Add(5 * time.Second)
	require.True(t, !created.ExpiresAt.Before(wantMin) && !created.ExpiresAt.After(wantMax),
		"expires_at=%v want ~7d", created.ExpiresAt)

	// stored hash matches plaintext
	sum := sha256.Sum256([]byte(created.Token))
	hash := hex.EncodeToString(sum[:])
	stored, err := repo.GetByTokenHash(context.Background(), hash)
	require.NoError(t, err)
	require.Equal(t, created.ID, stored.ID)
	require.Equal(t, hash, stored.TokenHash)
}

func TestUserAccessTokenService_Create_ExpiresInDaysValidation(t *testing.T) {
	repo := newUserAccessTokenRepoStub()
	svc := NewUserAccessTokenService(repo)

	for _, days := range []int{0, -1, 31, 100} {
		d := days
		_, err := svc.Create(context.Background(), 1, CreateUserAccessTokenInput{
			Name:          "x",
			ExpiresInDays: &d,
		})
		require.Error(t, err, "days=%d", days)
		require.ErrorIs(t, err, ErrUserAccessTokenInvalidExpires)
	}

	for _, days := range []int{1, 7, 30} {
		d := days
		created, err := svc.Create(context.Background(), 1, CreateUserAccessTokenInput{
			Name:          "ok",
			ExpiresInDays: &d,
		})
		require.NoError(t, err, "days=%d", days)
		require.NotEmpty(t, created.Token)
	}
}

func TestUserAccessTokenService_Create_NameRequired(t *testing.T) {
	svc := NewUserAccessTokenService(newUserAccessTokenRepoStub())
	_, err := svc.Create(context.Background(), 1, CreateUserAccessTokenInput{Name: "  "})
	require.ErrorIs(t, err, ErrUserAccessTokenInvalidName)
}

func TestUserAccessTokenService_List_DoesNotExposeTokenOrHash(t *testing.T) {
	repo := newUserAccessTokenRepoStub()
	svc := NewUserAccessTokenService(repo)

	created, err := svc.Create(context.Background(), 5, CreateUserAccessTokenInput{Name: "a"})
	require.NoError(t, err)

	list, err := svc.List(context.Background(), 5)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Empty(t, list[0].Token)
	require.Empty(t, list[0].TokenHash)
	require.Equal(t, created.TokenPrefix, list[0].TokenPrefix)
	require.Equal(t, "a", list[0].Name)

	// other user cannot list
	other, err := svc.List(context.Background(), 99)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestUserAccessTokenService_Revoke_Ownership(t *testing.T) {
	repo := newUserAccessTokenRepoStub()
	svc := NewUserAccessTokenService(repo)

	created, err := svc.Create(context.Background(), 7, CreateUserAccessTokenInput{Name: "mine"})
	require.NoError(t, err)

	err = svc.Revoke(context.Background(), 99, created.ID)
	require.ErrorIs(t, err, ErrUserAccessTokenNotFound)

	err = svc.Revoke(context.Background(), 7, created.ID)
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.RevokedAt)
}

func TestUserAccessTokenService_Validate_ActiveRevokedExpired(t *testing.T) {
	repo := newUserAccessTokenRepoStub()
	svc := NewUserAccessTokenService(repo)

	created, err := svc.Create(context.Background(), 3, CreateUserAccessTokenInput{Name: "v"})
	require.NoError(t, err)

	// active
	got, err := svc.ValidateToken(context.Background(), created.Token)
	require.NoError(t, err)
	require.Equal(t, int64(3), got.UserID)
	require.Equal(t, created.ID, got.ID)

	// revoked
	require.NoError(t, svc.Revoke(context.Background(), 3, created.ID))
	_, err = svc.ValidateToken(context.Background(), created.Token)
	require.ErrorIs(t, err, ErrUserAccessTokenRevoked)

	// expired
	days := 1
	expired, err := svc.Create(context.Background(), 3, CreateUserAccessTokenInput{Name: "e", ExpiresInDays: &days})
	require.NoError(t, err)
	// force expires_at in past via stub
	sum := sha256.Sum256([]byte(expired.Token))
	hash := hex.EncodeToString(sum[:])
	stored := repo.byHash[hash]
	past := time.Now().UTC().Add(-time.Hour)
	stored.ExpiresAt = past

	_, err = svc.ValidateToken(context.Background(), expired.Token)
	require.ErrorIs(t, err, ErrUserAccessTokenExpired)

	// invalid format
	_, err = svc.ValidateToken(context.Background(), "sk-not-uat")
	require.ErrorIs(t, err, ErrUserAccessTokenInvalid)
	_, err = svc.ValidateToken(context.Background(), "")
	require.ErrorIs(t, err, ErrUserAccessTokenInvalid)
}
