# Login Failure Email Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Require an additional email verification code after any user enters the wrong password 5 consecutive times, while still requiring the correct password.

**Architecture:** Add a Redis-backed login protection cache for account-scoped failure counters, challenge flags, and login-only email codes. Keep `AuthService.Login` for existing callers and route `/api/v1/auth/login` through a new `LoginWithEmailCode` method that enforces the challenge before the existing TOTP step. Extend the upstream Vue login page to reveal a code field only after the backend returns `EMAIL_CODE_REQUIRED`.

**Tech Stack:** Go, Gin, Redis, existing `infraerrors` response model, Vue 3, Pinia auth store, Vitest.

---

## File Structure

- Create: `backend/internal/service/login_protection.go`
  - Defines login challenge constants, `LoginProtectionCache`, new errors, `AuthService.SetLoginProtectionCache`, `AuthService.LoginWithEmailCode`, and `AuthService.SendLoginEmailCode`.
- Create: `backend/internal/repository/login_protection_cache.go`
  - Redis implementation for login failure counters, challenge keys, and login-only verification-code storage.
- Create: `backend/internal/service/login_protection_test.go`
  - Service-level tests with in-memory stubs.
- Modify: `backend/internal/handler/auth_handler.go`
  - Adds `email_code` to login request and `SendLoginEmailCode`.
- Modify: `backend/internal/server/routes/auth.go`
  - Registers `POST /api/v1/auth/login/send-email-code` with fail-close rate limiting.
- Modify: `backend/cmd/server/wire_gen.go`
  - Instantiates `repository.NewLoginProtectionCache(redisClient)` and injects it with `authService.SetLoginProtectionCache(...)`.
- Modify: `backend/internal/repository/wire.go`
  - Adds `NewLoginProtectionCache` to provider set for generated builds.
- Modify: `frontend/src/types/index.ts`
  - Adds optional `email_code` to `LoginRequest`.
- Modify: `frontend/src/api/auth.ts`
  - Adds `sendLoginEmailCode`.
- Modify: `frontend/src/views/auth/LoginView.vue`
  - Adds the conditional email-code field, send-code button, and request payload.
- Modify: `frontend/src/views/auth/__tests__/LoginView.spec.ts`
  - Adds frontend regression tests for the challenge flow.
- Modify: `frontend/src/i18n/locales/zh.ts`, `frontend/src/i18n/locales/zh-Hant.ts`, `frontend/src/i18n/locales/en.ts`
  - Adds login challenge labels and messages.
- Modify: `doc/plan/login-failure-email-verification-design.md`
  - Mark implementation decisions that differ from the initial design only if implementation discovers a necessary adjustment.

---

### Task 1: Redis Login Protection Cache

**Files:**
- Create: `backend/internal/service/login_protection.go`
- Create: `backend/internal/repository/login_protection_cache.go`
- Test: `backend/internal/repository/login_protection_cache_test.go`

- [ ] **Step 1: Write the failing repository tests**

Create `backend/internal/repository/login_protection_cache_test.go` with focused key behavior tests:

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLoginProtectionCacheFailureAndChallenge(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	cache := NewLoginProtectionCache(rdb)

	_, err := cache.IncrementFailure(ctx, "user@example.com", 15*time.Minute)
	require.Error(t, err)
}

func TestLoginProtectionCacheKeysNormalizeEmail(t *testing.T) {
	require.Equal(t, "login_failures:user@example.com", loginFailuresKey(" User@Example.COM "))
	require.Equal(t, "login_email_challenge:user@example.com", loginChallengeKey(" User@Example.COM "))
	require.Equal(t, "login_email_code:user@example.com", loginEmailCodeKey(" User@Example.COM "))
}

func TestLoginProtectionCodeRoundTripWithStubbedRedis(t *testing.T) {
	_ = service.VerificationCodeData{
		Code:      "123456",
		Attempts:  0,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
}
```

- [ ] **Step 2: Run the repository test to confirm it fails on missing symbols**

Run:

```bash
cd backend && go test ./internal/repository -run 'TestLoginProtectionCache' -count=1
```

Expected: build failure mentioning `NewLoginProtectionCache`, `loginFailuresKey`, `loginChallengeKey`, or `loginEmailCodeKey`.

- [ ] **Step 3: Add service interface and constants**

Create `backend/internal/service/login_protection.go` with this initial content:

```go
package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	loginFailureThreshold = 5
	loginChallengeTTL     = 15 * time.Minute
	loginEmailCodeTTL     = 15 * time.Minute
	loginEmailCodeCooldown = time.Minute
)

var (
	ErrLoginEmailCodeRequired = infraerrors.BadRequest("EMAIL_CODE_REQUIRED", "additional verification required")
	ErrLoginProtectionUnavailable = infraerrors.ServiceUnavailable("LOGIN_PROTECTION_UNAVAILABLE", "login protection temporarily unavailable")
)

type LoginProtectionCache interface {
	IncrementFailure(ctx context.Context, email string, ttl time.Duration) (int64, error)
	SetChallenge(ctx context.Context, email string, ttl time.Duration) error
	IsChallengeRequired(ctx context.Context, email string) (bool, error)
	Clear(ctx context.Context, email string) error
	GetEmailCode(ctx context.Context, email string) (*VerificationCodeData, error)
	SetEmailCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error
	DeleteEmailCode(ctx context.Context, email string) error
}

func normalizeLoginEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func (s *AuthService) SetLoginProtectionCache(cache LoginProtectionCache) {
	if s != nil {
		s.loginProtectionCache = cache
	}
}

func (s *AuthService) loginProtectionAvailable() bool {
	return s != nil && s.loginProtectionCache != nil
}
```

Also add this field to `AuthService` in `backend/internal/service/auth_service.go`:

```go
	loginProtectionCache LoginProtectionCache
```

- [ ] **Step 4: Add Redis implementation**

Create `backend/internal/repository/login_protection_cache.go`:

```go
package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	loginFailuresKeyPrefix  = "login_failures:"
	loginChallengeKeyPrefix = "login_email_challenge:"
	loginEmailCodeKeyPrefix = "login_email_code:"
)

type loginProtectionCache struct {
	rdb *redis.Client
}

func NewLoginProtectionCache(rdb *redis.Client) service.LoginProtectionCache {
	return &loginProtectionCache{rdb: rdb}
}

func normalizedLoginProtectionEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func loginFailuresKey(email string) string {
	return loginFailuresKeyPrefix + normalizedLoginProtectionEmail(email)
}

func loginChallengeKey(email string) string {
	return loginChallengeKeyPrefix + normalizedLoginProtectionEmail(email)
}

func loginEmailCodeKey(email string) string {
	return loginEmailCodeKeyPrefix + normalizedLoginProtectionEmail(email)
}

func (c *loginProtectionCache) IncrementFailure(ctx context.Context, email string, ttl time.Duration) (int64, error) {
	key := loginFailuresKey(email)
	count, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return count, err
	}
	return count, nil
}

func (c *loginProtectionCache) SetChallenge(ctx context.Context, email string, ttl time.Duration) error {
	return c.rdb.Set(ctx, loginChallengeKey(email), "1", ttl).Err()
}

func (c *loginProtectionCache) IsChallengeRequired(ctx context.Context, email string) (bool, error) {
	n, err := c.rdb.Exists(ctx, loginChallengeKey(email)).Result()
	return n > 0, err
}

func (c *loginProtectionCache) Clear(ctx context.Context, email string) error {
	return c.rdb.Del(ctx, loginFailuresKey(email), loginChallengeKey(email), loginEmailCodeKey(email)).Err()
}

func (c *loginProtectionCache) GetEmailCode(ctx context.Context, email string) (*service.VerificationCodeData, error) {
	val, err := c.rdb.Get(ctx, loginEmailCodeKey(email)).Result()
	if err != nil {
		return nil, err
	}
	var data service.VerificationCodeData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *loginProtectionCache) SetEmailCode(ctx context.Context, email string, data *service.VerificationCodeData, ttl time.Duration) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, loginEmailCodeKey(email), val, ttl).Err()
}

func (c *loginProtectionCache) DeleteEmailCode(ctx context.Context, email string) error {
	return c.rdb.Del(ctx, loginEmailCodeKey(email)).Err()
}
```

- [ ] **Step 5: Run the repository tests**

Run:

```bash
cd backend && go test ./internal/repository -run 'TestLoginProtectionCache' -count=1
```

Expected: PASS for key normalization, and the unavailable Redis test returns an error without panic.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/login_protection.go backend/internal/service/auth_service.go backend/internal/repository/login_protection_cache.go backend/internal/repository/login_protection_cache_test.go
git commit -m "feat: add login protection cache"
```

---

### Task 2: AuthService Password Failure Challenge

**Files:**
- Modify: `backend/internal/service/login_protection.go`
- Create: `backend/internal/service/login_protection_test.go`

- [ ] **Step 1: Write failing service tests**

Create `backend/internal/service/login_protection_test.go` with an in-memory cache and tests:

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type loginProtectionCacheStub struct {
	failures  map[string]int64
	challenge map[string]bool
	code      map[string]*VerificationCodeData
	err       error
}

func newLoginProtectionCacheStub() *loginProtectionCacheStub {
	return &loginProtectionCacheStub{
		failures:  map[string]int64{},
		challenge: map[string]bool{},
		code:      map[string]*VerificationCodeData{},
	}
}

func (s *loginProtectionCacheStub) IncrementFailure(_ context.Context, email string, _ time.Duration) (int64, error) {
	if s.err != nil { return 0, s.err }
	email = normalizeLoginEmail(email)
	s.failures[email]++
	return s.failures[email], nil
}

func (s *loginProtectionCacheStub) SetChallenge(_ context.Context, email string, _ time.Duration) error {
	if s.err != nil { return s.err }
	s.challenge[normalizeLoginEmail(email)] = true
	return nil
}

func (s *loginProtectionCacheStub) IsChallengeRequired(_ context.Context, email string) (bool, error) {
	if s.err != nil { return false, s.err }
	return s.challenge[normalizeLoginEmail(email)], nil
}

func (s *loginProtectionCacheStub) Clear(_ context.Context, email string) error {
	if s.err != nil { return s.err }
	email = normalizeLoginEmail(email)
	delete(s.failures, email)
	delete(s.challenge, email)
	delete(s.code, email)
	return nil
}

func (s *loginProtectionCacheStub) GetEmailCode(_ context.Context, email string) (*VerificationCodeData, error) {
	if s.err != nil { return nil, s.err }
	data := s.code[normalizeLoginEmail(email)]
	if data == nil { return nil, errors.New("missing code") }
	return data, nil
}

func (s *loginProtectionCacheStub) SetEmailCode(_ context.Context, email string, data *VerificationCodeData, _ time.Duration) error {
	if s.err != nil { return s.err }
	s.code[normalizeLoginEmail(email)] = data
	return nil
}

func (s *loginProtectionCacheStub) DeleteEmailCode(_ context.Context, email string) error {
	if s.err != nil { return s.err }
	delete(s.code, normalizeLoginEmail(email))
	return nil
}

func TestLoginWithEmailCodeRequiresChallengeAfterFiveFailures(t *testing.T) {
	ctx := context.Background()
	repo := newAuthLoginUserRepoStub(t, "secret-password")
	cache := newLoginProtectionCacheStub()
	svc := newAuthLoginProtectionService(repo, cache)

	for i := 1; i <= 4; i++ {
		_, _, err := svc.LoginWithEmailCode(ctx, "USER@example.com", "wrong-password", "")
		require.ErrorIs(t, err, ErrInvalidCredentials)
		require.False(t, cache.challenge["user@example.com"])
	}

	_, _, err := svc.LoginWithEmailCode(ctx, "user@example.com", "wrong-password", "")
	require.ErrorIs(t, err, ErrLoginEmailCodeRequired)
	require.True(t, cache.challenge["user@example.com"])
}

func TestLoginWithEmailCodeRequiresPasswordAndEmailCodeWhenChallenged(t *testing.T) {
	ctx := context.Background()
	repo := newAuthLoginUserRepoStub(t, "secret-password")
	cache := newLoginProtectionCacheStub()
	cache.challenge["user@example.com"] = true
	cache.code["user@example.com"] = &VerificationCodeData{
		Code:      "123456",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(loginEmailCodeTTL),
	}
	svc := newAuthLoginProtectionService(repo, cache)

	_, _, err := svc.LoginWithEmailCode(ctx, "user@example.com", "wrong-password", "123456")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.NotNil(t, cache.code["user@example.com"], "wrong password must not consume email code")

	_, _, err = svc.LoginWithEmailCode(ctx, "user@example.com", "secret-password", "")
	require.ErrorIs(t, err, ErrLoginEmailCodeRequired)

	token, user, err := svc.LoginWithEmailCode(ctx, "user@example.com", "secret-password", "123456")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, int64(1), user.ID)
	require.False(t, cache.challenge["user@example.com"])
	require.Nil(t, cache.code["user@example.com"])
}
```

Add helper functions in the same file using the existing `UserRepository` interface methods needed by `AuthService.LoginWithEmailCode`. Return zero values for unrelated methods and `panic("unexpected call")` only for methods that the tests should not hit.

- [ ] **Step 2: Run service tests and confirm failure**

Run:

```bash
cd backend && go test ./internal/service -run 'TestLoginWithEmailCode' -count=1
```

Expected: build failure because `LoginWithEmailCode` does not exist.

- [ ] **Step 3: Implement password challenge helpers**

Append these methods to `backend/internal/service/login_protection.go`:

```go
func (s *AuthService) LoginWithEmailCode(ctx context.Context, email, password, emailCode string) (string, *User, error) {
	normalizedEmail := normalizeLoginEmail(email)
	user, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error during login: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return "", nil, s.recordLoginPasswordFailure(ctx, normalizedEmail)
	}

	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	if err := s.requireLoginEmailCodeIfChallenged(ctx, normalizedEmail, emailCode); err != nil {
		return "", nil, err
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	s.clearLoginProtection(ctx, normalizedEmail)
	return token, user, nil
}

func (s *AuthService) recordLoginPasswordFailure(ctx context.Context, email string) error {
	if !s.loginProtectionAvailable() {
		return ErrInvalidCredentials
	}
	count, err := s.loginProtectionCache.IncrementFailure(ctx, email, loginChallengeTTL)
	if err != nil {
		slog.Error("login protection failure counter unavailable", "email", email, "error", err)
		return ErrLoginProtectionUnavailable
	}
	if count >= loginFailureThreshold {
		if err := s.loginProtectionCache.SetChallenge(ctx, email, loginChallengeTTL); err != nil {
			slog.Error("login protection challenge unavailable", "email", email, "error", err)
			return ErrLoginProtectionUnavailable
		}
		return ErrLoginEmailCodeRequired
	}
	return ErrInvalidCredentials
}
```

Import `errors` and `logger` in `login_protection.go`.

- [ ] **Step 4: Implement challenge code verification**

Append this code to `backend/internal/service/login_protection.go`:

```go
func (s *AuthService) requireLoginEmailCodeIfChallenged(ctx context.Context, email, code string) error {
	if !s.loginProtectionAvailable() {
		return nil
	}
	required, err := s.loginProtectionCache.IsChallengeRequired(ctx, email)
	if err != nil {
		slog.Error("login protection challenge check unavailable", "email", email, "error", err)
		return ErrLoginProtectionUnavailable
	}
	if !required {
		return nil
	}
	if strings.TrimSpace(code) == "" {
		return ErrLoginEmailCodeRequired
	}
	return s.verifyLoginEmailCode(ctx, email, code)
}

func (s *AuthService) verifyLoginEmailCode(ctx context.Context, email, code string) error {
	data, err := s.loginProtectionCache.GetEmailCode(ctx, email)
	if err != nil || data == nil {
		return ErrInvalidVerifyCode
	}
	if data.Attempts >= maxVerifyCodeAttempts {
		return ErrVerifyCodeMaxAttempts
	}
	if subtle.ConstantTimeCompare([]byte(data.Code), []byte(strings.TrimSpace(code))) != 1 {
		data.Attempts++
		remaining := time.Until(data.ExpiresAt)
		if remaining <= 0 {
			return ErrInvalidVerifyCode
		}
		if err := s.loginProtectionCache.SetEmailCode(ctx, email, data, remaining); err != nil {
			slog.Error("failed to update login email code attempts", "email", email, "error", err)
		}
		if data.Attempts >= maxVerifyCodeAttempts {
			return ErrVerifyCodeMaxAttempts
		}
		return ErrInvalidVerifyCode
	}
	if err := s.loginProtectionCache.DeleteEmailCode(ctx, email); err != nil {
		slog.Error("failed to delete login email code after verification", "email", email, "error", err)
	}
	return nil
}

func (s *AuthService) clearLoginProtection(ctx context.Context, email string) {
	if s.loginProtectionAvailable() {
		if err := s.loginProtectionCache.Clear(ctx, email); err != nil {
			slog.Error("failed to clear login protection state", "email", email, "error", err)
		}
	}
}
```

- [ ] **Step 5: Run service tests**

Run:

```bash
cd backend && go test ./internal/service -run 'TestLoginWithEmailCode' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/login_protection.go backend/internal/service/login_protection_test.go
git commit -m "feat: require email code after login failures"
```

---

### Task 3: Login Email Code Send Endpoint

**Files:**
- Modify: `backend/internal/service/login_protection.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/internal/server/routes/auth.go`

- [ ] **Step 1: Add service tests for send behavior**

Extend `backend/internal/service/login_protection_test.go`:

```go
func TestSendLoginEmailCodeOnlySendsWhenChallengeExists(t *testing.T) {
	ctx := context.Background()
	repo := newAuthLoginUserRepoStub(t, "secret-password")
	cache := newLoginProtectionCacheStub()
	svc := newAuthLoginProtectionService(repo, cache)

	result, err := svc.SendLoginEmailCode(ctx, "user@example.com")
	require.NoError(t, err)
	require.Equal(t, 60, result.Countdown)
	require.Nil(t, cache.code["user@example.com"])

	cache.challenge["user@example.com"] = true
	result, err = svc.SendLoginEmailCode(ctx, "user@example.com")
	require.NoError(t, err)
	require.Equal(t, 60, result.Countdown)
	require.NotNil(t, cache.code["user@example.com"])
	require.Len(t, cache.code["user@example.com"].Code, 6)
}
```

- [ ] **Step 2: Implement `SendLoginEmailCode`**

Append to `backend/internal/service/login_protection.go`:

```go
func (s *AuthService) SendLoginEmailCode(ctx context.Context, email string) (*SendVerifyCodeResult, error) {
	normalizedEmail := normalizeLoginEmail(email)
	result := &SendVerifyCodeResult{Countdown: int(loginEmailCodeCooldown.Seconds())}
	if !s.loginProtectionAvailable() {
		return nil, ErrLoginProtectionUnavailable
	}
	required, err := s.loginProtectionCache.IsChallengeRequired(ctx, normalizedEmail)
	if err != nil {
		slog.Error("login email code challenge check unavailable", "email", normalizedEmail, "error", err)
		return nil, ErrLoginProtectionUnavailable
	}
	if !required {
		return result, nil
	}
	if _, err := s.userRepo.GetByEmail(ctx, normalizedEmail); err != nil {
		return result, nil
	}
	existing, err := s.loginProtectionCache.GetEmailCode(ctx, normalizedEmail)
	if err == nil && existing != nil && time.Since(existing.CreatedAt) < loginEmailCodeCooldown {
		return nil, ErrVerifyCodeTooFrequent
	}
	code, err := s.emailService.GenerateVerifyCode()
	if err != nil {
		return nil, fmt.Errorf("generate login email code: %w", err)
	}
	data := &VerificationCodeData{
		Code:      code,
		Attempts:  0,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(loginEmailCodeTTL),
	}
	if err := s.loginProtectionCache.SetEmailCode(ctx, normalizedEmail, data, loginEmailCodeTTL); err != nil {
		return nil, fmt.Errorf("save login email code: %w", err)
	}
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}
	subject := fmt.Sprintf("[%s] Login Verification Code", siteName)
	body := fmt.Sprintf(`<p>Your login verification code is:</p><h2>%s</h2><p>This code expires in 15 minutes.</p>`, code)
	if err := s.emailService.SendEmail(ctx, normalizedEmail, subject, body); err != nil {
		return nil, fmt.Errorf("send login email code: %w", err)
	}
	return result, nil
}
```

- [ ] **Step 3: Update handler request and add handler method**

In `backend/internal/handler/auth_handler.go`, change `LoginRequest`:

```go
type LoginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	EmailCode      string `json:"email_code"`
	TurnstileToken string `json:"turnstile_token"`
}
```

Change `Login` to call:

```go
token, user, err := h.authService.LoginWithEmailCode(c.Request.Context(), req.Email, req.Password, req.EmailCode)
```

Add a request type and method:

```go
type SendLoginEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) SendLoginEmailCode(c *gin.Context) {
	var req SendLoginEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.authService.SendLoginEmailCode(c.Request.Context(), req.Email)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, SendVerifyCodeResponse{
		Message:   "Verification code sent successfully",
		Countdown: result.Countdown,
	})
}
```

- [ ] **Step 4: Register the route**

In `backend/internal/server/routes/auth.go`, add after `/login`:

```go
auth.POST("/login/send-email-code", rateLimiter.LimitWithOptions("auth-login-send-email-code", 5, time.Minute, middleware.RateLimitOptions{
	FailureMode: middleware.RateLimitFailClose,
}), h.Auth.SendLoginEmailCode)
```

- [ ] **Step 5: Run backend auth tests**

Run:

```bash
cd backend && go test ./internal/service ./internal/handler ./internal/server/routes -run 'Login|AuthRoutes|SendLoginEmailCode' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/login_protection.go backend/internal/service/login_protection_test.go backend/internal/handler/auth_handler.go backend/internal/server/routes/auth.go
git commit -m "feat: add login email code endpoint"
```

---

### Task 4: Production Wiring

**Files:**
- Modify: `backend/cmd/server/wire_gen.go`
- Modify: `backend/internal/repository/wire.go`

- [ ] **Step 1: Inject cache in generated server setup**

In `backend/cmd/server/wire_gen.go`, after `refreshTokenCache := repository.NewRefreshTokenCache(redisClient)`, add:

```go
loginProtectionCache := repository.NewLoginProtectionCache(redisClient)
```

After `authService := service.NewAuthService(...)`, add:

```go
authService.SetLoginProtectionCache(loginProtectionCache)
```

- [ ] **Step 2: Add provider to repository wire set**

In `backend/internal/repository/wire.go`, add `NewLoginProtectionCache` to the provider set near other Redis caches.

- [ ] **Step 3: Run compile check**

Run:

```bash
cd backend && go test ./cmd/server ./internal/repository ./internal/service -run 'TestNonExistent' -count=1
```

Expected: package compile succeeds with `testing: warning: no tests to run`.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/wire_gen.go backend/internal/repository/wire.go
git commit -m "chore: wire login protection cache"
```

---

### Task 5: Frontend Login Challenge UI

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/auth.ts`
- Modify: `frontend/src/views/auth/LoginView.vue`
- Modify: `frontend/src/views/auth/__tests__/LoginView.spec.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/zh-Hant.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Write failing frontend test**

In `frontend/src/views/auth/__tests__/LoginView.spec.ts`, extend the `@/api/auth` mock:

```ts
const sendLoginEmailCode = vi.fn()

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => getPublicSettings(...args),
  sendLoginEmailCode: (...args: any[]) => sendLoginEmailCode(...args),
  isTotp2FARequired: (response: any) => response?.requires_2fa === true,
  isWeChatWebOAuthEnabled: () => false,
}))
```

Add reset in `beforeEach`:

```ts
sendLoginEmailCode.mockReset()
sendLoginEmailCode.mockResolvedValue({ message: 'ok', countdown: 60 })
```

Add this test:

```ts
it('shows login email code challenge and submits the code after EMAIL_CODE_REQUIRED', async () => {
  login
    .mockRejectedValueOnce({ response: { data: { error: { code: 'EMAIL_CODE_REQUIRED' } } } })
    .mockResolvedValueOnce({
      access_token: 'login-token',
      token_type: 'Bearer',
      user: {
        id: 1,
        username: 'test',
        email: 'test@example.com',
        role: 'user',
        balance: 0,
        concurrency: 5,
        status: 'active',
        allowed_groups: null,
        created_at: '',
        updated_at: '',
      },
    })

  const wrapper = mountLoginView()
  await submitLogin(wrapper)

  expect(wrapper.find('[data-testid="login-email-code"]').exists()).toBe(true)

  await wrapper.find('[data-testid="login-email-code-send"]').trigger('click')
  await flushPromises()
  expect(sendLoginEmailCode).toHaveBeenCalledWith({ email: 'test@example.com' })

  await wrapper.find('[data-testid="login-email-code"]').setValue('123456')
  await wrapper.find('form').trigger('submit')
  await flushPromises()

  expect(login).toHaveBeenLastCalledWith({
    email: 'test@example.com',
    password: 'password123',
    email_code: '123456',
    turnstile_token: undefined,
  })
  expect(routerPush).toHaveBeenCalledWith('/dashboard')
})
```

- [ ] **Step 2: Run frontend test and confirm failure**

Run:

```bash
cd frontend && pnpm test -- LoginView.spec.ts --run
```

Expected: failure because `sendLoginEmailCode` export or `data-testid="login-email-code"` does not exist.

- [ ] **Step 3: Update types and API**

In `frontend/src/types/index.ts`, change `LoginRequest`:

```ts
export interface LoginRequest {
  email: string
  password: string
  email_code?: string
  turnstile_token?: string
}
```

In `frontend/src/api/auth.ts`, add:

```ts
export async function sendLoginEmailCode(
  request: Pick<SendVerifyCodeRequest, 'email'>
): Promise<SendVerifyCodeResponse> {
  const { data } = await apiClient.post<SendVerifyCodeResponse>('/auth/login/send-email-code', request)
  return data
}
```

Add `sendLoginEmailCode` to the exported `authAPI` object.

- [ ] **Step 4: Add LoginView state and error detection**

In `frontend/src/views/auth/LoginView.vue`, import `sendLoginEmailCode` from `@/api/auth`.

Add state near other form state:

```ts
const loginEmailCodeRequired = ref(false)
const loginEmailCode = ref('')
const loginEmailCodeSending = ref(false)
const loginEmailCodeCountdown = ref(0)
let loginEmailCodeTimer: number | undefined
```

Add helper:

```ts
function isEmailCodeRequiredError(error: unknown): boolean {
  const err = error as { response?: { data?: { error?: { code?: string }, code?: string } } }
  return err.response?.data?.error?.code === 'EMAIL_CODE_REQUIRED' || err.response?.data?.code === 'EMAIL_CODE_REQUIRED'
}
```

Add send-code method:

```ts
async function handleSendLoginEmailCode(): Promise<void> {
  if (!formData.email || loginEmailCodeCountdown.value > 0) return
  loginEmailCodeSending.value = true
  try {
    const result = await sendLoginEmailCode({ email: formData.email })
    loginEmailCodeCountdown.value = result.countdown || 60
    window.clearInterval(loginEmailCodeTimer)
    loginEmailCodeTimer = window.setInterval(() => {
      loginEmailCodeCountdown.value -= 1
      if (loginEmailCodeCountdown.value <= 0) {
        window.clearInterval(loginEmailCodeTimer)
        loginEmailCodeTimer = undefined
      }
    }, 1000)
    appStore.showSuccess(t('auth.codeSentSuccess'))
  } catch {
    appStore.showError(t('auth.sendCodeFailed'))
  } finally {
    loginEmailCodeSending.value = false
  }
}
```

Add `onBeforeUnmount` cleanup if not already imported:

```ts
onBeforeUnmount(() => {
  window.clearInterval(loginEmailCodeTimer)
})
```

- [ ] **Step 5: Add UI field**

In the form, place this block before the Turnstile widget:

```vue
<div v-if="loginEmailCodeRequired" class="space-y-2">
  <label for="login-email-code" class="form-label">
    {{ t('auth.loginEmailCode') }}
  </label>
  <div class="flex gap-2">
    <input
      id="login-email-code"
      v-model="loginEmailCode"
      data-testid="login-email-code"
      type="text"
      inputmode="numeric"
      autocomplete="one-time-code"
      maxlength="6"
      class="input flex-1"
      :placeholder="t('auth.loginEmailCodePlaceholder')"
    />
    <button
      type="button"
      data-testid="login-email-code-send"
      class="btn btn-secondary whitespace-nowrap"
      :disabled="loginEmailCodeSending || loginEmailCodeCountdown > 0"
      @click="handleSendLoginEmailCode"
    >
      {{ loginEmailCodeCountdown > 0 ? t('auth.resendCountdown', { countdown: loginEmailCodeCountdown }) : t('auth.sendCode') }}
    </button>
  </div>
  <p class="text-xs text-gray-500 dark:text-dark-400">
    {{ t('auth.loginEmailCodeHint') }}
  </p>
</div>
```

- [ ] **Step 6: Include email code in login payload and detect challenge**

In `handleLogin`, change the login payload:

```ts
const response = await authStore.login({
  email: formData.email,
  password: formData.password,
  email_code: loginEmailCodeRequired.value ? loginEmailCode.value : undefined,
  turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined
})
```

In the `catch`, before generic error extraction:

```ts
if (isEmailCodeRequiredError(error)) {
  loginEmailCodeRequired.value = true
  errorMessage.value = t('auth.loginEmailCodeRequired')
  appStore.showWarning(errorMessage.value)
  return
}
```

On successful password login or successful 2FA, reset:

```ts
loginEmailCodeRequired.value = false
loginEmailCode.value = ''
```

- [ ] **Step 7: Add i18n keys**

Add these keys under `auth` in all three locale files:

```ts
loginEmailCode: '邮箱验证码',
loginEmailCodePlaceholder: '输入6位邮箱验证码',
loginEmailCodeHint: '此账号需要额外邮箱验证。请先发送验证码，再重新登录。',
loginEmailCodeRequired: '当前账号需要邮箱验证码后才能继续登录。',
```

Use English equivalents in `en.ts` and Traditional Chinese equivalents in `zh-Hant.ts`.

- [ ] **Step 8: Run frontend tests**

Run:

```bash
cd frontend && pnpm test -- LoginView.spec.ts --run
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/auth.ts frontend/src/views/auth/LoginView.vue frontend/src/views/auth/__tests__/LoginView.spec.ts frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/zh-Hant.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: add login email challenge UI"
```

---

### Task 6: Full Verification and Docs

**Files:**
- Modify: `doc/plan/login-failure-email-verification-design.md` if implementation details changed.

- [ ] **Step 1: Run backend targeted tests**

Run:

```bash
cd backend && go test ./internal/service ./internal/repository ./internal/handler ./internal/server/routes
```

Expected: PASS.

- [ ] **Step 2: Run frontend checks**

Run:

```bash
cd frontend && pnpm typecheck && pnpm build
```

Expected: PASS.

- [ ] **Step 3: Manual API smoke test**

Start the backend using the project’s normal dev command. Then use an existing test account:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"wrong"}'
```

Repeat until the 5th failure. Expected response includes `EMAIL_CODE_REQUIRED`. Then request a code:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/auth/login/send-email-code \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com"}'
```

Expected: `200` with a countdown and no token.

- [ ] **Step 4: Review docs**

If implementation matches the design exactly, no docs changes are needed. If any path or error code changes, update:

```bash
doc/plan/login-failure-email-verification-design.md
```

- [ ] **Step 5: Final commit**

If docs changed:

```bash
git add doc/plan/login-failure-email-verification-design.md
git commit -m "docs: update login challenge design"
```

If docs did not change, skip this commit.

---

## Self-Review

- Spec coverage: account-wide failure counter, 5 failures, 15-minute challenge, password plus email code, manual code sending, no frontend-dashboard changes, and TOTP preservation are covered by Tasks 2, 3, and 5.
- Placeholder scan: no placeholders remain in this plan; every step has concrete files, commands, and expected outcomes.
- Type consistency: backend uses `email_code`, `EMAIL_CODE_REQUIRED`, and `LoginProtectionCache`; frontend uses the same `email_code` request field and error code.
