package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	userAccessTokenPrefix           = "uat_"
	userAccessTokenSecretBytes      = 32 // 256-bit entropy
	userAccessTokenDefaultDays      = 7
	userAccessTokenMinDays          = 1
	userAccessTokenMaxDays          = 30
	userAccessTokenNameMaxLen       = 100
	userAccessTokenPrefixDispLen    = 8 // chars of secret after prefix for display
	userAccessTokenMaxActivePerUser = 10
	userAccessTokenValidateCacheTTL = 30 * time.Second
	userAccessTokenTouchMinInterval = 60 * time.Second
)

type userAccessTokenCacheEntry struct {
	token     UserAccessToken
	expiresAt time.Time
}

var (
	ErrUserAccessTokenNotFound       = infraerrors.NotFound("USER_ACCESS_TOKEN_NOT_FOUND", "access token not found")
	ErrUserAccessTokenInvalid        = infraerrors.Unauthorized("INVALID_TOKEN", "invalid access token")
	ErrUserAccessTokenExpired        = infraerrors.Unauthorized("TOKEN_EXPIRED", "access token has expired")
	ErrUserAccessTokenRevoked        = infraerrors.Unauthorized("TOKEN_REVOKED", "access token has been revoked")
	ErrUserAccessTokenInvalidName    = infraerrors.BadRequest("INVALID_NAME", "name is required")
	ErrUserAccessTokenInvalidExpires = infraerrors.BadRequest("INVALID_EXPIRES_IN_DAYS", "expires_in_days must be between 1 and 30")
	ErrUserAccessTokenLimitReached   = infraerrors.BadRequest("USER_ACCESS_TOKEN_LIMIT", "maximum active access tokens reached (10)")
)

// UserAccessTokenRepository persists opaque user access tokens (hash only).
type UserAccessTokenRepository interface {
	Create(ctx context.Context, token *UserAccessToken) error
	GetByID(ctx context.Context, id int64) (*UserAccessToken, error)
	GetByTokenHash(ctx context.Context, hash string) (*UserAccessToken, error)
	ListByUserID(ctx context.Context, userID int64) ([]UserAccessToken, error)
	CountActiveByUserID(ctx context.Context, userID int64, now time.Time) (int, error)
	RevokeByIDForUser(ctx context.Context, userID, id int64, revokedAt time.Time) error
	TouchLastUsedAt(ctx context.Context, id int64, usedAt time.Time) error
}

// UserAccessTokenService manages creation, listing, revoke and validation of user access tokens.
type UserAccessTokenService struct {
	repo UserAccessTokenRepository
	now  func() time.Time

	// Short-lived positive validation cache (hash -> token metadata) to cut DB load.
	validateCache sync.Map // map[string]userAccessTokenCacheEntry
	// last touch times to throttle last_used_at writes (token id -> unix nano)
	touchThrottle sync.Map // map[int64]int64
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

	now := s.now()
	active, err := s.repo.CountActiveByUserID(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	if active >= userAccessTokenMaxActivePerUser {
		return nil, ErrUserAccessTokenLimitReached
	}

	plaintext, prefix, hash, err := generateUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

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
	// Drop any positive cache entries for this user on revoke (best-effort scan).
	// Keys are token hashes; we only know id, so clear all entries matching id.
	s.validateCache.Range(func(key, value any) bool {
		entry, ok := value.(userAccessTokenCacheEntry)
		if ok && entry.token.ID == id {
			s.validateCache.Delete(key)
		}
		return true
	})
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
	now := s.now()

	if cached, ok := s.validateCache.Load(hash); ok {
		entry := cached.(userAccessTokenCacheEntry)
		if entry.expiresAt.After(now) {
			cp := entry.token
			if cp.RevokedAt != nil {
				return nil, ErrUserAccessTokenRevoked
			}
			if !cp.ExpiresAt.After(now) {
				s.validateCache.Delete(hash)
				return nil, ErrUserAccessTokenExpired
			}
			return &cp, nil
		}
		s.validateCache.Delete(hash)
	}

	rec, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		if infraerrors.IsNotFound(err) || err == ErrUserAccessTokenNotFound {
			return nil, ErrUserAccessTokenInvalid
		}
		return nil, err
	}
	if rec.RevokedAt != nil {
		s.validateCache.Delete(hash)
		return nil, ErrUserAccessTokenRevoked
	}
	if !rec.ExpiresAt.After(now) {
		s.validateCache.Delete(hash)
		return nil, ErrUserAccessTokenExpired
	}
	rec.Token = ""
	rec.TokenHash = ""

	// Cache positive result briefly. Revoke still hits DB until TTL elapses (max 30s).
	s.validateCache.Store(hash, userAccessTokenCacheEntry{
		token:     *rec,
		expiresAt: now.Add(userAccessTokenValidateCacheTTL),
	})
	return rec, nil
}

// TouchLastUsed updates last_used_at best-effort (caller may ignore errors).
// Writes are throttled to at most once per minute per token to avoid write amplification.
func (s *UserAccessTokenService) TouchLastUsed(ctx context.Context, id int64) {
	if s == nil || s.repo == nil || id <= 0 {
		return
	}
	now := s.now()
	if lastRaw, ok := s.touchThrottle.Load(id); ok {
		if last, ok := lastRaw.(int64); ok {
			if now.UnixNano()-last < int64(userAccessTokenTouchMinInterval) {
				return
			}
		}
	}
	s.touchThrottle.Store(id, now.UnixNano())
	_ = s.repo.TouchLastUsedAt(ctx, id, now)
}

// InvalidateValidateCache clears cached validation for a token hash (tests / revoke helpers).
func (s *UserAccessTokenService) InvalidateValidateCache(tokenPlaintext string) {
	if s == nil {
		return
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(tokenPlaintext)))
	s.validateCache.Delete(hex.EncodeToString(sum[:]))
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
