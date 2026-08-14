package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type desktopHandoffStoreStub struct {
	storedHash   string
	storedData   DesktopPaymentHandoffData
	ttl          time.Duration
	storeErr     error
	consumeHash  string
	consumeData  *DesktopPaymentHandoffData
	consumeErr   error
	consumeCalls int
}

func (s *desktopHandoffStoreStub) Store(
	_ context.Context,
	tokenHash string,
	data DesktopPaymentHandoffData,
	ttl time.Duration,
) error {
	s.storedHash = tokenHash
	s.storedData = data
	s.ttl = ttl
	return s.storeErr
}

func (s *desktopHandoffStoreStub) Consume(
	_ context.Context,
	tokenHash string,
) (*DesktopPaymentHandoffData, error) {
	s.consumeCalls++
	s.consumeHash = tokenHash
	return s.consumeData, s.consumeErr
}

type desktopHandoffUserStub struct {
	user     *User
	err      error
	lastID   int64
	getCalls int
}

func (s *desktopHandoffUserStub) GetByID(_ context.Context, id int64) (*User, error) {
	s.getCalls++
	s.lastID = id
	return s.user, s.err
}

type desktopHandoffConfigStub struct {
	cfg   *LumioDesktopConfig
	err   error
	calls int
}

func (s *desktopHandoffConfigStub) GetLumioDesktopConfig(_ context.Context) (*LumioDesktopConfig, error) {
	s.calls++
	return s.cfg, s.err
}

func TestDesktopPaymentHandoffIssueStoresOnlyHash(t *testing.T) {
	store := &desktopHandoffStoreStub{}
	users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
	svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

	ticket, err := svc.Issue(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 60, ticket.ExpiresIn)
	require.True(t, len(ticket.Token) > len(DesktopPaymentHandoffTokenPrefix))
	require.Equal(t, DesktopPaymentHandoffTokenPrefix, ticket.Token[:len(DesktopPaymentHandoffTokenPrefix)])
	decoded, err := base64.RawURLEncoding.DecodeString(ticket.Token[len(DesktopPaymentHandoffTokenPrefix):])
	require.NoError(t, err)
	require.Len(t, decoded, DesktopPaymentHandoffTokenBytes)
	digest := sha256.Sum256([]byte(ticket.Token))
	require.Equal(t, hex.EncodeToString(digest[:]), store.storedHash)
	require.NotEqual(t, ticket.Token, store.storedHash)
	require.Equal(t, DesktopPaymentHandoffData{UserID: 42}, store.storedData)
	require.Equal(t, time.Minute, store.ttl)
	require.Equal(t, int64(42), users.lastID)
}

func TestDesktopPaymentHandoffIssueFailsClosed(t *testing.T) {
	t.Run("feature disabled", func(t *testing.T) {
		store := &desktopHandoffStoreStub{}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
		cfg := activeDesktopHandoffConfig("/payment")
		cfg.FeatureFlags.PaymentHandoff = false
		svc, _ := newDesktopPaymentHandoffTestService(store, users, cfg)

		_, err := svc.Issue(context.Background(), 42)

		require.ErrorIs(t, err, ErrDesktopPaymentHandoffUnavailable)
		require.Zero(t, users.getCalls)
		require.Empty(t, store.storedHash)
	})

	t.Run("inactive user", func(t *testing.T) {
		store := &desktopHandoffStoreStub{}
		users := &desktopHandoffUserStub{user: &User{ID: 42, Status: StatusDisabled}}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Issue(context.Background(), 42)

		require.ErrorIs(t, err, ErrUserNotActive)
		require.Empty(t, store.storedHash)
	})

	t.Run("storage failure", func(t *testing.T) {
		store := &desktopHandoffStoreStub{storeErr: errors.New("redis unavailable")}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Issue(context.Background(), 42)

		require.ErrorIs(t, err, ErrDesktopPaymentHandoffUnavailable)
	})
}

func TestDesktopPaymentHandoffConsumeUsesStoredUserAndSafeRedirect(t *testing.T) {
	rawToken := desktopPaymentHandoffTestToken(0x42)
	store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
	users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
	svc, auth := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("https://evil.example/payment"))

	session, err := svc.Consume(context.Background(), rawToken)

	require.NoError(t, err)
	digest := sha256.Sum256([]byte(rawToken))
	require.Equal(t, hex.EncodeToString(digest[:]), store.consumeHash)
	require.Equal(t, int64(42), users.lastID)
	require.Equal(t, "/payment?desktop_handoff=1", session.RedirectURL)
	require.Equal(t, 900, session.ExpiresIn)
	claims, err := auth.ValidateToken(session.AccessToken)
	require.NoError(t, err)
	require.Equal(t, int64(42), claims.UserID)
}

func TestDesktopPaymentHandoffConsumePreservesSafeConfiguredURL(t *testing.T) {
	rawToken := desktopPaymentHandoffTestToken(0x24)
	store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 7}}
	users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(7)}
	svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/purchase?source=config#plans"))

	session, err := svc.Consume(context.Background(), rawToken)

	require.NoError(t, err)
	require.Equal(t, "/purchase?desktop_handoff=1&source=config#plans", session.RedirectURL)
}

func TestDesktopPaymentHandoffConsumeRejectsInvalidOrReusedToken(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		store := &desktopHandoffStoreStub{}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Consume(context.Background(), "dph_short")

		require.ErrorIs(t, err, ErrDesktopPaymentHandoffInvalid)
		require.Zero(t, store.consumeCalls)
	})

	t.Run("reused", func(t *testing.T) {
		store := &desktopHandoffStoreStub{consumeErr: ErrDesktopPaymentHandoffInvalid}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Consume(context.Background(), desktopPaymentHandoffTestToken(0x11))

		require.ErrorIs(t, err, ErrDesktopPaymentHandoffInvalid)
		require.Zero(t, users.getCalls)
	})
}

func TestDesktopPaymentHandoffConsumeFailsClosedAfterAtomicConsume(t *testing.T) {
	t.Run("feature disabled", func(t *testing.T) {
		store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
		cfg := activeDesktopHandoffConfig("/payment")
		cfg.FeatureFlags.PaymentHandoff = false
		svc, _ := newDesktopPaymentHandoffTestService(store, users, cfg)

		_, err := svc.Consume(context.Background(), desktopPaymentHandoffTestToken(0x12))

		require.ErrorIs(t, err, ErrDesktopPaymentHandoffUnavailable)
		require.Equal(t, 1, store.consumeCalls)
		require.Zero(t, users.getCalls)
	})

	t.Run("inactive bound user", func(t *testing.T) {
		store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
		users := &desktopHandoffUserStub{user: &User{ID: 42, Status: StatusDisabled}}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Consume(context.Background(), desktopPaymentHandoffTestToken(0x13))

		require.ErrorIs(t, err, ErrUserNotActive)
		require.Equal(t, 1, store.consumeCalls)
	})

	t.Run("repository returns wrong user", func(t *testing.T) {
		store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(99)}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Consume(context.Background(), desktopPaymentHandoffTestToken(0x15))

		require.ErrorIs(t, err, ErrUserNotActive)
		require.Equal(t, int64(42), users.lastID)
	})

	t.Run("storage failure", func(t *testing.T) {
		store := &desktopHandoffStoreStub{consumeErr: errors.New("redis unavailable")}
		users := &desktopHandoffUserStub{user: activeDesktopHandoffUser(42)}
		svc, _ := newDesktopPaymentHandoffTestService(store, users, activeDesktopHandoffConfig("/payment"))

		_, err := svc.Consume(context.Background(), desktopPaymentHandoffTestToken(0x14))

		require.ErrorIs(t, err, ErrDesktopPaymentHandoffUnavailable)
		require.Zero(t, users.getCalls)
	})
}

func newDesktopPaymentHandoffTestService(
	store DesktopPaymentHandoffStore,
	users DesktopPaymentHandoffUserReader,
	desktopConfig *LumioDesktopConfig,
) (*DesktopPaymentHandoffService, *AuthService) {
	auth := &AuthService{cfg: &config.Config{JWT: config.JWTConfig{
		Secret:                   "desktop-payment-handoff-test-secret",
		AccessTokenExpireMinutes: 15,
	}}}
	return NewDesktopPaymentHandoffService(
		store,
		users,
		auth,
		&desktopHandoffConfigStub{cfg: desktopConfig},
	), auth
}

func activeDesktopHandoffConfig(paymentURL string) *LumioDesktopConfig {
	cfg := DefaultLumioDesktopConfig()
	cfg.PaymentURL = paymentURL
	cfg.FeatureFlags.PaymentHandoff = true
	return &cfg
}

func activeDesktopHandoffUser(id int64) *User {
	return &User{
		ID:           id,
		Email:        "desktop@example.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 3,
	}
}

func desktopPaymentHandoffTestToken(fill byte) string {
	return DesktopPaymentHandoffTokenPrefix + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{fill}, DesktopPaymentHandoffTokenBytes))
}
