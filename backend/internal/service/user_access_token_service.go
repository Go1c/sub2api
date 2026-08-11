package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	userAccessTokenPrefix       = "uat_"
	userAccessTokenSecretBytes  = 32 // 256-bit entropy
	userAccessTokenDefaultDays  = 7
	userAccessTokenMinDays      = 1
	userAccessTokenMaxDays      = 30
	userAccessTokenNameMaxLen   = 100
	userAccessTokenPrefixDispLen = 8 // chars of secret after prefix for display
)

var (
	ErrUserAccessTokenNotFound       = infraerrors.NotFound("USER_ACCESS_TOKEN_NOT_FOUND", "access token not found")
	ErrUserAccessTokenInvalid        = infraerrors.Unauthorized("INVALID_TOKEN", "invalid access token")
	ErrUserAccessTokenExpired        = infraerrors.Unauthorized("TOKEN_EXPIRED", "access token has expired")
	ErrUserAccessTokenRevoked        = infraerrors.Unauthorized("TOKEN_REVOKED", "access token has been revoked")
	ErrUserAccessTokenInvalidName    = infraerrors.BadRequest("INVALID_NAME", "name is required")
	ErrUserAccessTokenInvalidExpires = infraerrors.BadRequest("INVALID_EXPIRES_IN_DAYS", "expires_in_days must be between 1 and 30")
)

// UserAccessTokenRepository persists opaque user access tokens (hash only).
type UserAccessTokenRepository interface {
	Create(ctx context.Context, token *UserAccessToken) error
	GetByID(ctx context.Context, id int64) (*UserAccessToken, error)
	GetByTokenHash(ctx context.Context, hash string) (*UserAccessToken, error)
	ListByUserID(ctx context.Context, userID int64) ([]UserAccessToken, error)
	RevokeByIDForUser(ctx context.Context, userID, id int64, revokedAt time.Time) error
	TouchLastUsedAt(ctx context.Context, id int64, usedAt time.Time) error
}

// UserAccessTokenService manages creation, listing, revoke and validation of user access tokens.
type UserAccessTokenService struct {
	repo UserAccessTokenRepository
	now  func() time.Time
}

// NewUserAccessTokenService creates a UserAccessTokenService.
func NewUserAccessTokenService(repo UserAccessTokenRepository) *UserAccessTokenService {
	return &UserAccessTokenService{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// Create issues a new opaque token. Plaintext is returned only in the response model.
func (s *UserAccessTokenService) Create(ctx context.Context, userID int64, input CreateUserAccessTokenInput) (*UserAccessToken, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrUserAccessTokenInvalidName
	}
	if len(name) > userAccessTokenNameMaxLen {
		name = name[:userAccessTokenNameMaxLen]
	}

	days := userAccessTokenDefaultDays
	if input.ExpiresInDays != nil {
		days = *input.ExpiresInDays
		if days < userAccessTokenMinDays || days > userAccessTokenMaxDays {
			return nil, ErrUserAccessTokenInvalidExpires
		}
	}

	plaintext, prefix, hash, err := generateUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	now := s.now()
	record := &UserAccessToken{
		UserID:      userID,
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		ExpiresAt:   now.Add(time.Duration(days) * 24 * time.Hour),
	}
	if err := s.repo.Create(ctx, record); err != nil {
		return nil, err
	}

	// Return copy with plaintext for one-time display; hash stripped from API-facing fields.
	out := *record
	out.Token = plaintext
	out.TokenHash = ""
	return &out, nil
}

// List returns metadata for the user's tokens (no plaintext/hash).
func (s *UserAccessTokenService) List(ctx context.Context, userID int64) ([]UserAccessToken, error) {
	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Token = ""
		items[i].TokenHash = ""
	}
	return items, nil
}

// Revoke marks the token revoked if owned by userID.
func (s *UserAccessTokenService) Revoke(ctx context.Context, userID, id int64) error {
	return s.repo.RevokeByIDForUser(ctx, userID, id, s.now())
}

// ValidateToken authenticates an opaque Bearer token. Returns the stored record (no plaintext).
func (s *UserAccessTokenService) ValidateToken(ctx context.Context, raw string) (*UserAccessToken, error) {
	token := strings.TrimSpace(raw)
	if token == "" || !strings.HasPrefix(token, userAccessTokenPrefix) {
		return nil, ErrUserAccessTokenInvalid
	}
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])

	rec, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		if infraerrors.IsNotFound(err) || err == ErrUserAccessTokenNotFound {
			return nil, ErrUserAccessTokenInvalid
		}
		return nil, err
	}
	if rec.RevokedAt != nil {
		return nil, ErrUserAccessTokenRevoked
	}
	if !rec.ExpiresAt.After(s.now()) {
		return nil, ErrUserAccessTokenExpired
	}
	rec.Token = ""
	rec.TokenHash = ""
	return rec, nil
}

// TouchLastUsed updates last_used_at best-effort (caller may ignore errors).
func (s *UserAccessTokenService) TouchLastUsed(ctx context.Context, id int64) {
	if s == nil || s.repo == nil || id <= 0 {
		return
	}
	_ = s.repo.TouchLastUsedAt(ctx, id, s.now())
}

// HashUserAccessToken returns SHA-256 hex of the full token (for tests/helpers).
func HashUserAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsUserAccessTokenShape reports whether the bearer looks like a user access token (not JWT).
func IsUserAccessTokenShape(token string) bool {
	t := strings.TrimSpace(token)
	return strings.HasPrefix(t, userAccessTokenPrefix)
}

func generateUserAccessToken() (plaintext, prefix, hash string, err error) {
	secret := make([]byte, userAccessTokenSecretBytes)
	if _, err = rand.Read(secret); err != nil {
		return "", "", "", err
	}
	secretHex := hex.EncodeToString(secret)
	plaintext = userAccessTokenPrefix + secretHex
	// Display: uat_ + first 8 hex chars of secret
	disp := secretHex
	if len(disp) > userAccessTokenPrefixDispLen {
		disp = disp[:userAccessTokenPrefixDispLen]
	}
	prefix = userAccessTokenPrefix + disp
	hash = HashUserAccessToken(plaintext)
	return plaintext, prefix, hash, nil
}
