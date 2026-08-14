package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrDesktopPaymentHandoffInvalid = infraerrors.New(
		http.StatusGone,
		"DESKTOP_PAYMENT_HANDOFF_INVALID",
		"payment handoff is invalid or expired",
	)
	ErrDesktopPaymentHandoffUnavailable = infraerrors.ServiceUnavailable(
		"DESKTOP_PAYMENT_HANDOFF_UNAVAILABLE",
		"payment handoff is temporarily unavailable",
	)
)

const (
	DesktopPaymentHandoffTTL         = time.Minute
	DesktopPaymentHandoffTokenPrefix = "dph_"
	DesktopPaymentHandoffTokenBytes  = 32
	LumioWebSessionCookieName        = "lumio_web_session"
)

type DesktopPaymentHandoffData struct {
	UserID int64 `json:"user_id"`
}

type DesktopPaymentHandoffStore interface {
	Store(ctx context.Context, tokenHash string, data DesktopPaymentHandoffData, ttl time.Duration) error
	Consume(ctx context.Context, tokenHash string) (*DesktopPaymentHandoffData, error)
}

type DesktopPaymentHandoffUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type DesktopPaymentHandoffTokenIssuer interface {
	GenerateToken(ctx context.Context, user *User) (string, error)
	GetAccessTokenExpiresIn() int
}

type DesktopPaymentHandoffConfigReader interface {
	GetLumioDesktopConfig(ctx context.Context) (*LumioDesktopConfig, error)
}

type DesktopPaymentHandoffTicket struct {
	Token     string
	ExpiresIn int
}

type DesktopPaymentHandoffSession struct {
	AccessToken string
	RedirectURL string
	ExpiresIn   int
}

type DesktopPaymentHandoffService struct {
	store  DesktopPaymentHandoffStore
	users  DesktopPaymentHandoffUserReader
	tokens DesktopPaymentHandoffTokenIssuer
	config DesktopPaymentHandoffConfigReader
}

func NewDesktopPaymentHandoffService(
	store DesktopPaymentHandoffStore,
	users DesktopPaymentHandoffUserReader,
	tokens DesktopPaymentHandoffTokenIssuer,
	config DesktopPaymentHandoffConfigReader,
) *DesktopPaymentHandoffService {
	return &DesktopPaymentHandoffService{
		store:  store,
		users:  users,
		tokens: tokens,
		config: config,
	}
}

func (s *DesktopPaymentHandoffService) Issue(
	ctx context.Context,
	userID int64,
) (*DesktopPaymentHandoffTicket, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotActive
		}
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}
	if user == nil || user.ID != userID || !user.IsActive() {
		return nil, ErrUserNotActive
	}

	rawToken, err := newDesktopPaymentHandoffToken()
	if err != nil {
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}
	if err := s.store.Store(
		ctx,
		hashDesktopPaymentHandoffToken(rawToken),
		DesktopPaymentHandoffData{UserID: userID},
		DesktopPaymentHandoffTTL,
	); err != nil {
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}

	return &DesktopPaymentHandoffTicket{
		Token:     rawToken,
		ExpiresIn: int(DesktopPaymentHandoffTTL.Seconds()),
	}, nil
}

func (s *DesktopPaymentHandoffService) Consume(
	ctx context.Context,
	rawToken string,
) (*DesktopPaymentHandoffSession, error) {
	if !validDesktopPaymentHandoffToken(rawToken) {
		return nil, ErrDesktopPaymentHandoffInvalid
	}

	data, err := s.store.Consume(ctx, hashDesktopPaymentHandoffToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrDesktopPaymentHandoffInvalid) {
			return nil, ErrDesktopPaymentHandoffInvalid
		}
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}
	if data == nil || data.UserID <= 0 {
		return nil, ErrDesktopPaymentHandoffInvalid
	}

	cfg, err := s.enabledConfig(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, data.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotActive
		}
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}
	if user == nil || user.ID != data.UserID || !user.IsActive() {
		return nil, ErrUserNotActive
	}

	accessToken, err := s.tokens.GenerateToken(ctx, user)
	if err != nil {
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}

	return &DesktopPaymentHandoffSession{
		AccessToken: accessToken,
		RedirectURL: desktopPaymentRedirectURL(cfg.PaymentURL),
		ExpiresIn:   s.tokens.GetAccessTokenExpiresIn(),
	}, nil
}

func (s *DesktopPaymentHandoffService) ensureEnabled(ctx context.Context) error {
	_, err := s.enabledConfig(ctx)
	return err
}

func (s *DesktopPaymentHandoffService) enabledConfig(ctx context.Context) (*LumioDesktopConfig, error) {
	cfg, err := s.config.GetLumioDesktopConfig(ctx)
	if err != nil {
		return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
	}
	if cfg == nil || !cfg.FeatureFlags.PaymentHandoff {
		return nil, ErrDesktopPaymentHandoffUnavailable
	}
	return cfg, nil
}

func newDesktopPaymentHandoffToken() (string, error) {
	random := make([]byte, DesktopPaymentHandoffTokenBytes)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return DesktopPaymentHandoffTokenPrefix + base64.RawURLEncoding.EncodeToString(random), nil
}

func validDesktopPaymentHandoffToken(rawToken string) bool {
	if !strings.HasPrefix(rawToken, DesktopPaymentHandoffTokenPrefix) {
		return false
	}
	encoded := strings.TrimPrefix(rawToken, DesktopPaymentHandoffTokenPrefix)
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(decoded) != DesktopPaymentHandoffTokenBytes {
		return false
	}
	return base64.RawURLEncoding.EncodeToString(decoded) == encoded
}

func hashDesktopPaymentHandoffToken(rawToken string) string {
	digest := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(digest[:])
}

func desktopPaymentRedirectURL(rawURL string) string {
	value := strings.TrimSpace(rawURL)
	if !isSafeLumioDesktopPaymentURL(value) {
		value = LumioDesktopDefaultPaymentURL
	}
	parsed, err := url.Parse(value)
	if err != nil {
		parsed = &url.URL{Path: LumioDesktopDefaultPaymentURL}
	}
	query := parsed.Query()
	query.Set("desktop_handoff", "1")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
