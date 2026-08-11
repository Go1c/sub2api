//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type lumioDesktopAPIKeyRepoStub struct {
	*apiKeyRepoStub
	existing               *APIKey
	createErr              error
	winnerAfterCreateError *APIKey
	createCalls            int
	lookupCalls            int
	lastLookupUserID       int64
	lastLookupReservedName string
}

func newLumioDesktopAPIKeyRepoStub(existing *APIKey) *lumioDesktopAPIKeyRepoStub {
	return &lumioDesktopAPIKeyRepoStub{
		apiKeyRepoStub: &apiKeyRepoStub{},
		existing:       existing,
	}
}

func (s *lumioDesktopAPIKeyRepoStub) GetByUserIDAndName(_ context.Context, userID int64, name string) (*APIKey, error) {
	s.lookupCalls++
	s.lastLookupUserID = userID
	s.lastLookupReservedName = name
	if s.existing == nil {
		return nil, ErrAPIKeyNotFound
	}
	return s.existing, nil
}

func (s *lumioDesktopAPIKeyRepoStub) Create(_ context.Context, key *APIKey) error {
	s.createCalls++
	if s.createErr != nil {
		s.existing = s.winnerAfterCreateError
		return s.createErr
	}
	key.ID = 101
	s.existing = key
	return nil
}

func newLumioDesktopAPIKeyService(repo APIKeyRepository) *APIKeyService {
	return &APIKeyService{
		apiKeyRepo: repo,
		userRepo: &userRepoStub{user: &User{
			ID:     42,
			Status: StatusActive,
		}},
		cfg: &config.Config{},
	}
}

func TestAPIKeyServiceCreateLumioDesktopReusesExisting(t *testing.T) {
	existing := &APIKey{
		ID:     7,
		UserID: 42,
		Key:    "sk-existing",
		Name:   LumioDesktopAPIKeyName,
		Status: StatusActive,
	}
	repo := newLumioDesktopAPIKeyRepoStub(existing)
	svc := newLumioDesktopAPIKeyService(repo)

	got, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})

	require.NoError(t, err)
	require.Same(t, existing, got)
	require.Zero(t, repo.createCalls)
	require.Equal(t, int64(42), repo.lastLookupUserID)
	require.Equal(t, LumioDesktopAPIKeyName, repo.lastLookupReservedName)
}

func TestAPIKeyServiceCreateLumioDesktopCreatesFirstKey(t *testing.T) {
	repo := newLumioDesktopAPIKeyRepoStub(nil)
	svc := newLumioDesktopAPIKeyService(repo)

	got, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})

	require.NoError(t, err)
	require.Equal(t, LumioDesktopAPIKeyName, got.Name)
	require.NotEmpty(t, got.Key)
	require.Equal(t, 1, repo.createCalls)
	require.Equal(t, 1, repo.lookupCalls)
}

func TestAPIKeyServiceCreateLumioDesktopResolvesConcurrentUniqueConflict(t *testing.T) {
	winner := &APIKey{
		ID:     9,
		UserID: 42,
		Key:    "sk-winner",
		Name:   LumioDesktopAPIKeyName,
		Status: StatusActive,
	}
	repo := newLumioDesktopAPIKeyRepoStub(nil)
	repo.createErr = ErrAPIKeyExists
	repo.winnerAfterCreateError = winner
	svc := newLumioDesktopAPIKeyService(repo)

	got, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})

	require.NoError(t, err)
	require.Same(t, winner, got)
	require.Equal(t, 1, repo.createCalls)
	require.Equal(t, 2, repo.lookupCalls)
}

func TestAPIKeyServiceCreateOrdinaryNameKeepsConflict(t *testing.T) {
	repo := newLumioDesktopAPIKeyRepoStub(nil)
	repo.createErr = ErrAPIKeyExists
	svc := newLumioDesktopAPIKeyService(repo)

	_, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: "ordinary"})

	require.ErrorIs(t, err, ErrAPIKeyExists)
	require.Equal(t, 1, repo.createCalls)
	require.Zero(t, repo.lookupCalls)
}
