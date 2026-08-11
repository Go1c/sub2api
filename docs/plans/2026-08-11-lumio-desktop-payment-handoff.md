# Lumio Desktop Payment Handoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development to implement this plan task-by-task (hosts without subagents: its Inline Fallback section). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Open the current Lumio Codex user's website payment page through a 60-second, single-use opaque handoff without putting a JWT or API key in the URL.

**Architecture:** Redis stores only `SHA-256(raw handoff token) -> user_id` and atomically consumes it with `GETDEL`. The browser consumer issues a normal access JWT into a host-only HttpOnly cookie, then the Vue router restores `/auth/me` from that cookie and removes a non-secret bootstrap marker.

**Tech Stack:** Go 1.25.7, Gin, go-redis v9, Redis 6.2+, JWT HS256, Vue 3, Pinia, Vue Router, Vitest.

## Global Constraints

- Issue TTL is exactly 60 seconds; raw tokens use the exact `dph_` prefix plus 32 random bytes encoded with raw base64url.
- Cookie name is exactly `lumio_web_session`; it is host-only, `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` for HTTPS.
- Issue requires JWT auth; opaque user access tokens keep their existing path restriction.
- The consume URL may contain only the opaque handoff token, never a JWT, refresh token, gateway API key, or caller-controlled redirect.
- Redirect targets come only from normalized `LumioDesktopConfig.PaymentURL`; unsafe data falls back to `/payment`.
- No new dependency, device table, OAuth flow, refresh-cookie flow, or payment UI.

---

### Task 1: Hash-only Redis store and handoff service

**Files:**
- Create: `backend/internal/service/desktop_payment_handoff.go`
- Create: `backend/internal/service/desktop_payment_handoff_test.go`
- Create: `backend/internal/repository/desktop_payment_handoff_store.go`
- Create: `backend/internal/repository/desktop_payment_handoff_store_test.go`
- Modify: `backend/internal/repository/wire.go`
- Modify: `backend/internal/service/wire.go`

**Interfaces:**
- Produces: `DesktopPaymentHandoffStore.Store(context.Context, string, DesktopPaymentHandoffData, time.Duration) error`
- Produces: `DesktopPaymentHandoffStore.Consume(context.Context, string) (*DesktopPaymentHandoffData, error)`
- Produces: `(*DesktopPaymentHandoffService).Issue(context.Context, int64) (*DesktopPaymentHandoffTicket, error)`
- Produces: `(*DesktopPaymentHandoffService).Consume(context.Context, string) (*DesktopPaymentHandoffSession, error)`
- Consumes: `(*SettingService).GetLumioDesktopConfig`

- [x] **Step 1: Write failing Redis store tests**

```go
func TestDesktopPaymentHandoffStoreConsumesOnceAndExpires(t *testing.T) {
    mr := miniredis.RunT(t)
    client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    store := NewDesktopPaymentHandoffStore(client)
    ctx := context.Background()

    require.NoError(t, store.Store(ctx, "hash-only", service.DesktopPaymentHandoffData{UserID: 42}, time.Minute))
    require.True(t, mr.Exists("desktop_payment_handoff:hash-only"))

    got, err := store.Consume(ctx, "hash-only")
    require.NoError(t, err)
    require.Equal(t, int64(42), got.UserID)
    _, err = store.Consume(ctx, "hash-only")
    require.ErrorIs(t, err, service.ErrDesktopPaymentHandoffInvalid)

    require.NoError(t, store.Store(ctx, "expires", service.DesktopPaymentHandoffData{UserID: 7}, time.Minute))
    mr.FastForward(time.Minute)
    _, err = store.Consume(ctx, "expires")
    require.ErrorIs(t, err, service.ErrDesktopPaymentHandoffInvalid)
}
```

- [x] **Step 2: Run the store test and verify RED**

Run: `cd backend && go test -tags=unit -count=1 ./internal/repository -run 'DesktopPaymentHandoffStore'`

Expected: compile failure because the store and service contract do not exist.

- [x] **Step 3: Implement the Redis store**

```go
const desktopPaymentHandoffKeyPrefix = "desktop_payment_handoff:"

func (s *desktopPaymentHandoffStore) Store(ctx context.Context, tokenHash string, data service.DesktopPaymentHandoffData, ttl time.Duration) error {
    raw, err := json.Marshal(data)
    if err != nil { return fmt.Errorf("marshal desktop payment handoff: %w", err) }
    return s.client.Set(ctx, desktopPaymentHandoffKeyPrefix+tokenHash, raw, ttl).Err()
}

func (s *desktopPaymentHandoffStore) Consume(ctx context.Context, tokenHash string) (*service.DesktopPaymentHandoffData, error) {
    raw, err := s.client.GetDel(ctx, desktopPaymentHandoffKeyPrefix+tokenHash).Bytes()
    if errors.Is(err, redis.Nil) { return nil, service.ErrDesktopPaymentHandoffInvalid }
    if err != nil { return nil, err }
    var data service.DesktopPaymentHandoffData
    if err := json.Unmarshal(raw, &data); err != nil { return nil, fmt.Errorf("unmarshal desktop payment handoff: %w", err) }
    return &data, nil
}
```

- [x] **Step 4: Write failing service tests**

```go
func TestDesktopPaymentHandoffIssueStoresOnlyHash(t *testing.T) {
    store := &desktopHandoffStoreStub{}
    svc := newDesktopPaymentHandoffTestService(store, activeDesktopHandoffConfig("/payment"))
    ticket, err := svc.Issue(context.Background(), 42)
    require.NoError(t, err)
    require.Equal(t, 60, ticket.ExpiresIn)
    require.True(t, strings.HasPrefix(ticket.Token, "dph_"))
    require.NotEqual(t, ticket.Token, store.storedHash)
    require.Regexp(t, `^[a-f0-9]{64}$`, store.storedHash)
    require.Equal(t, time.Minute, store.ttl)
}

func TestDesktopPaymentHandoffConsumeUsesStoredUserAndSafeRedirect(t *testing.T) {
    store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
    svc := newDesktopPaymentHandoffTestService(store, activeDesktopHandoffConfig("https://evil.example/payment"))
    session, err := svc.Consume(context.Background(), "dph_valid")
    require.NoError(t, err)
    require.Equal(t, "/payment?desktop_handoff=1", session.RedirectURL)
    claims, err := svc.tokenIssuer.(*AuthService).ValidateToken(session.AccessToken)
    require.NoError(t, err)
    require.Equal(t, int64(42), claims.UserID)
}

func TestDesktopPaymentHandoffConsumeRejectsReusedToken(t *testing.T) {
    store := &desktopHandoffStoreStub{consumeErr: ErrDesktopPaymentHandoffInvalid}
    svc := newDesktopPaymentHandoffTestService(store, activeDesktopHandoffConfig("/payment"))
    _, err := svc.Consume(context.Background(), "dph_reused")
    require.ErrorIs(t, err, ErrDesktopPaymentHandoffInvalid)
}

func TestDesktopPaymentHandoffConsumeFailsClosedWhenDisabled(t *testing.T) {
    store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
    cfg := activeDesktopHandoffConfig("/payment")
    cfg.FeatureFlags.PaymentHandoff = false
    svc := newDesktopPaymentHandoffTestService(store, cfg)
    _, err := svc.Consume(context.Background(), "dph_valid")
    require.ErrorIs(t, err, ErrDesktopPaymentHandoffUnavailable)
}

func TestDesktopPaymentHandoffConsumeRejectsInactiveBoundUser(t *testing.T) {
    store := &desktopHandoffStoreStub{consumeData: &DesktopPaymentHandoffData{UserID: 42}}
    svc := newDesktopPaymentHandoffTestService(store, activeDesktopHandoffConfig("/payment"))
    svc.users = &desktopHandoffUserStub{user: &User{ID: 42, Status: StatusDisabled}}
    _, err := svc.Consume(context.Background(), "dph_valid")
    require.ErrorIs(t, err, ErrUserNotActive)
}
```

The test file defines four narrow stubs matching the service ports: `desktopHandoffStoreStub` records `storedHash`/`ttl` and returns `consumeData`/`consumeErr`; `desktopHandoffUserStub` returns one user; `desktopHandoffTokenIssuerStub` delegates to an `AuthService` configured with a test JWT secret; and `desktopHandoffConfigStub` returns the supplied typed config.

- [x] **Step 5: Run service tests and verify RED**

Run: `cd backend && go test -tags=unit -count=1 ./internal/service -run 'DesktopPaymentHandoff'`

Expected: compile failure because the service types and methods do not exist.

- [x] **Step 6: Implement the minimal service**

```go
const (
    DesktopPaymentHandoffTTL = time.Minute
    DesktopPaymentHandoffTokenPrefix = "dph_"
    LumioWebSessionCookieName = "lumio_web_session"
)

type DesktopPaymentHandoffData struct { UserID int64 `json:"user_id"` }
type DesktopPaymentHandoffTicket struct { Token string; ExpiresIn int }
type DesktopPaymentHandoffSession struct { AccessToken, RedirectURL string; ExpiresIn int }

func (s *DesktopPaymentHandoffService) Issue(ctx context.Context, userID int64) (*DesktopPaymentHandoffTicket, error) {
    if err := s.ensureEnabled(ctx); err != nil { return nil, err }
    user, err := s.users.GetByID(ctx, userID)
    if err != nil || !user.IsActive() { return nil, ErrUserNotActive }
    raw, err := newDesktopPaymentHandoffToken()
    if err != nil { return nil, ErrServiceUnavailable }
    if err := s.store.Store(ctx, hashDesktopPaymentHandoffToken(raw), DesktopPaymentHandoffData{UserID: userID}, DesktopPaymentHandoffTTL); err != nil {
        return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err)
    }
    return &DesktopPaymentHandoffTicket{Token: raw, ExpiresIn: int(DesktopPaymentHandoffTTL.Seconds())}, nil
}

func (s *DesktopPaymentHandoffService) Consume(ctx context.Context, raw string) (*DesktopPaymentHandoffSession, error) {
    if !validDesktopPaymentHandoffToken(raw) { return nil, ErrDesktopPaymentHandoffInvalid }
    data, err := s.store.Consume(ctx, hashDesktopPaymentHandoffToken(raw))
    if err != nil { return nil, normalizeDesktopHandoffConsumeError(err) }
    cfg, err := s.config.GetLumioDesktopConfig(ctx)
    if err != nil || !cfg.FeatureFlags.PaymentHandoff { return nil, ErrDesktopPaymentHandoffUnavailable }
    user, err := s.users.GetByID(ctx, data.UserID)
    if err != nil || !user.IsActive() { return nil, ErrUserNotActive }
    accessToken, err := s.tokens.GenerateToken(ctx, user)
    if err != nil { return nil, ErrDesktopPaymentHandoffUnavailable.WithCause(err) }
    return &DesktopPaymentHandoffSession{AccessToken: accessToken, RedirectURL: desktopPaymentRedirectURL(cfg.PaymentURL), ExpiresIn: s.tokens.GetAccessTokenExpiresIn()}, nil
}
```

- [x] **Step 7: Register providers and verify GREEN**

Run: `cd backend && go test -tags=unit -count=1 ./internal/repository ./internal/service -run 'DesktopPaymentHandoff'`

Expected: PASS.

### Task 2: HTTP endpoints, Cookie authentication, logout, and Wire

**Files:**
- Create: `backend/internal/handler/desktop_payment_handoff_handler.go`
- Create: `backend/internal/handler/desktop_payment_handoff_handler_test.go`
- Create: `backend/internal/server/routes/desktop.go`
- Create: `backend/internal/server/routes/desktop_test.go`
- Modify: `backend/internal/server/router.go`
- Modify: `backend/internal/server/middleware/jwt_auth.go`
- Modify: `backend/internal/server/middleware/jwt_auth_test.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/internal/handler/auth_session_revocation_test.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: generated `backend/cmd/server/wire_gen.go` via `go generate ./cmd/server`

**Interfaces:**
- Consumes: Task 1 service contract
- Produces: `POST /api/v1/desktop/payment-handoff`
- Produces: `GET /api/v1/desktop/payment-handoff/consume`
- Produces: JWT middleware fallback to `lumio_web_session`

- [x] **Step 1: Write failing handler tests**

```go
func TestDesktopPaymentHandoffHandlerIssueReturnsOpaqueRelativeURL(t *testing.T) {
    stub := &desktopPaymentHandoffServiceStub{ticket: &service.DesktopPaymentHandoffTicket{Token: "dph_opaque", ExpiresIn: 60}}
    h := newDesktopPaymentHandoffHandlerForTest(stub)
    recorder := invokeIssueWithSubject(t, h, 42)
    require.Equal(t, http.StatusOK, recorder.Code)
    require.Contains(t, recorder.Body.String(), `/api/v1/desktop/payment-handoff/consume?token=dph_opaque`)
    require.NotContains(t, recorder.Body.String(), "Bearer")
    require.NotContains(t, recorder.Body.String(), "sk-")
}

func TestDesktopPaymentHandoffHandlerConsumeSetsCookieAndIgnoresRedirect(t *testing.T) {
    stub := &desktopPaymentHandoffServiceStub{session: &service.DesktopPaymentHandoffSession{AccessToken: "jwt-secret", RedirectURL: "/payment?desktop_handoff=1", ExpiresIn: 900}}
    h := newDesktopPaymentHandoffHandlerForTest(stub)
    recorder := invokeConsume(t, h, "/api/v1/desktop/payment-handoff/consume?token=dph_ok&redirect=https://evil.example")
    require.Equal(t, http.StatusSeeOther, recorder.Code)
    require.Equal(t, "/payment?desktop_handoff=1", recorder.Header().Get("Location"))
    cookie := findCookie(recorder.Result().Cookies(), service.LumioWebSessionCookieName)
    require.True(t, cookie.HttpOnly)
    require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
    require.Equal(t, "/", cookie.Path)
}
```

- [x] **Step 2: Run handler tests and verify RED**

Run: `cd backend && go test -tags=unit -count=1 ./internal/handler -run 'DesktopPaymentHandoff'`

Expected: compile failure because the handler does not exist.

- [x] **Step 3: Implement handler and security headers**

```go
func (h *DesktopPaymentHandoffHandler) Issue(c *gin.Context) {
    subject, ok := middleware.GetAuthSubjectFromContext(c)
    if !ok { response.Unauthorized(c, "User not authenticated"); return }
    ticket, err := h.service.Issue(c.Request.Context(), subject.UserID)
    if err != nil { response.ErrorFrom(c, err); return }
    response.Success(c, DesktopPaymentHandoffIssueResponse{
        HandoffURL: "/api/v1/desktop/payment-handoff/consume?token=" + url.QueryEscape(ticket.Token),
        ExpiresIn: ticket.ExpiresIn,
    })
}

func (h *DesktopPaymentHandoffHandler) Consume(c *gin.Context) {
    c.Header("Cache-Control", "no-store")
    c.Header("Referrer-Policy", "no-referrer")
    session, err := h.service.Consume(c.Request.Context(), c.Query("token"))
    if err != nil { response.ErrorFrom(c, err); return }
    setLumioWebSessionCookie(c, session.AccessToken, session.ExpiresIn)
    c.Redirect(http.StatusSeeOther, session.RedirectURL)
}
```

- [x] **Step 4: Write failing middleware, logout, and route tests**

```go
func TestJWTAuthUsesLumioWebSessionCookieWhenHeaderMissing(t *testing.T) {
    user := &service.User{ID: 1, Email: "cookie@example.com", Role: service.RoleUser, Status: service.StatusActive, TokenVersion: 1}
    router, authSvc := newJWTTestEnv(map[int64]*service.User{1: user})
    token, err := authSvc.GenerateToken(context.Background(), user)
    require.NoError(t, err)
    recorder := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    req.AddCookie(&http.Cookie{Name: service.LumioWebSessionCookieName, Value: token})
    router.ServeHTTP(recorder, req)
    require.Equal(t, http.StatusOK, recorder.Code)
}

func TestJWTAuthAuthorizationHeaderWinsOverCookie(t *testing.T) {
    headerUser := &service.User{ID: 1, Email: "header@example.com", Role: service.RoleUser, Status: service.StatusActive, TokenVersion: 1}
    cookieUser := &service.User{ID: 2, Email: "cookie@example.com", Role: service.RoleUser, Status: service.StatusActive, TokenVersion: 1}
    router, authSvc := newJWTTestEnv(map[int64]*service.User{1: headerUser, 2: cookieUser})
    headerToken, err := authSvc.GenerateToken(context.Background(), headerUser)
    require.NoError(t, err)
    cookieToken, err := authSvc.GenerateToken(context.Background(), cookieUser)
    require.NoError(t, err)
    recorder := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+headerToken)
    req.AddCookie(&http.Cookie{Name: service.LumioWebSessionCookieName, Value: cookieToken})
    router.ServeHTTP(recorder, req)
    require.Contains(t, recorder.Body.String(), `"user_id":1`)
}

func TestAuthHandlerLogoutClearsLumioWebSessionCookie(t *testing.T) {
    h := &AuthHandler{}
    recorder := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(recorder)
    c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
    h.Logout(c)
    cookie := findCookie(recorder.Result().Cookies(), service.LumioWebSessionCookieName)
    require.NotNil(t, cookie)
    require.Less(t, cookie.MaxAge, 0)
    require.True(t, cookie.HttpOnly)
}

func TestRegisterDesktopRoutesSeparatesIssueAuthAndPublicConsume(t *testing.T) {
    handlers := &handler.Handlers{DesktopPaymentHandoff: handler.NewDesktopPaymentHandoffHandler(&desktopRouteHandoffStub{})}
    router := gin.New()
    deny := middleware.JWTAuthMiddleware(func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) })
    RegisterDesktopRoutes(router.Group("/api/v1"), handlers, deny, nil, nil)
    issue := httptest.NewRecorder()
    router.ServeHTTP(issue, httptest.NewRequest(http.MethodPost, "/api/v1/desktop/payment-handoff", nil))
    require.Equal(t, http.StatusUnauthorized, issue.Code)
    consume := httptest.NewRecorder()
    router.ServeHTTP(consume, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/payment-handoff/consume?token=dph_ok", nil))
    require.NotEqual(t, http.StatusUnauthorized, consume.Code)
}
```

- [x] **Step 5: Run middleware/route tests and verify RED**

Run: `cd backend && go test -tags=unit -count=1 ./internal/server/middleware ./internal/server/routes ./internal/handler -run 'LumioWebSession|DesktopPaymentHandoff'`

Expected: cookie auth and desktop routes are absent.

- [x] **Step 6: Implement cookie fallback, logout clearing, and routes**

```go
if authHeader == "" {
    if cookieToken, err := c.Cookie(service.LumioWebSessionCookieName); err == nil {
        tokenString = strings.TrimSpace(cookieToken)
    }
}
```

Register the consume route before the authenticated issue group, apply existing fail-close IP rate limiting, and keep `BackendModeUserGuard` on issue. Always clear `lumio_web_session` in `AuthHandler.Logout`.

- [x] **Step 7: Wire, generate, and verify GREEN**

Run: `cd backend && go generate ./cmd/server && go test -tags=unit -count=1 ./internal/handler ./internal/server/middleware ./internal/server/routes ./internal/server`

Expected: PASS and `wire_gen.go` includes the new store, service, handler, and `Handlers` field.

- [x] **Step 8: Commit backend handoff**

Run: `git diff --check && git diff --stat`

Commit: `feat(desktop): add one-time payment handoff`

### Task 3: Cookie-session frontend bootstrap and `/payment` alias

**Files:**
- Modify: `frontend/src/api/auth.ts`
- Create: `frontend/src/api/__tests__/auth-cookie-session.spec.ts`
- Modify: `frontend/src/stores/auth.ts`
- Modify: `frontend/src/stores/__tests__/auth.spec.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/__tests__/integration/navigation.spec.ts`

**Interfaces:**
- Consumes: Task 2 HttpOnly cookie and `/auth/me`
- Consumes: `desktop_handoff=1`
- Produces: `authStore.restoreCookieSession({ replaceClientSession?: boolean })`
- Produces: `/payment` alias for `PaymentView.vue`

- [x] **Step 1: Write failing API/store tests**

```ts
it('probes the current user with credentials without persisting a token', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: 0, data: user }), { status: 200 })))
  const got = await probeCookieSession()
  expect(got.id).toBe(user.id)
  expect(fetch).toHaveBeenCalledWith('/api/v1/auth/me', expect.objectContaining({ credentials: 'include' }))
  expect(localStorage.getItem('auth_token')).toBeNull()
})

it('replaces stale local auth with the cookie session', async () => {
  localStorage.setItem('auth_token', 'old-user-token')
  localStorage.setItem('auth_user', JSON.stringify(oldUser))
  vi.mocked(authAPI.probeCookieSession).mockResolvedValue(cookieUser)
  const restored = await store.restoreCookieSession({ replaceClientSession: true })
  expect(restored).toBe(true)
  expect(store.user?.id).toBe(cookieUser.id)
  expect(store.token).toBeNull()
  expect(localStorage.getItem('auth_token')).toBeNull()
  expect(store.isAuthenticated).toBe(true)
})

it('always calls backend logout for a cookie-only session', async () => {
  await authAPI.logout()
  expect(apiClient.post).toHaveBeenCalledWith('/auth/logout', {})
})
```

- [x] **Step 2: Run frontend tests and verify RED**

Run: `cd frontend && pnpm test --run src/stores/__tests__/auth.spec.ts src/api/__tests__/auth-cookie-session.spec.ts`

Expected: missing cookie-session probe and restore action.

- [x] **Step 3: Implement probe and Pinia cookie-session state**

```ts
export async function probeCookieSession(): Promise<CurrentUserResponse> {
  const response = await fetch(buildApiUrl('/auth/me'), {
    method: 'GET',
    credentials: 'include',
    headers: { Accept: 'application/json' }
  })
  const envelope = await response.json() as ApiResponse<CurrentUserResponse>
  if (!response.ok || envelope.code !== 0 || !envelope.data) throw { status: response.status, message: envelope.message }
  return envelope.data
}

const cookieSessionActive = ref(false)
const isAuthenticated = computed(() => !!user.value && (!!token.value || cookieSessionActive.value))

async function restoreCookieSession(options?: { replaceClientSession?: boolean }): Promise<boolean> {
  if (options?.replaceClientSession) clearAuth()
  try {
    const current = await authAPI.probeCookieSession()
    const { run_mode, ...userData } = current
    runMode.value = run_mode || 'standard'
    user.value = userData
    cookieSessionActive.value = true
    startAutoRefresh()
    return true
  } catch {
    cookieSessionActive.value = false
    return false
  }
}
```

Allow `refreshUser` and auto-refresh when either a local token or the cookie-session flag is active. Make `authAPI.logout` always POST `/auth/logout`, using `{ refresh_token }` only when present.

- [x] **Step 4: Write failing router tests**

```ts
it('resolves /payment to the existing purchase view', () => {
  expect(router.resolve('/payment').name).toBe('PurchaseSubscription')
})

it('forces cookie account restore and strips only desktop_handoff', async () => {
  restoreCookieSession.mockResolvedValue(true)
  await router.push('/payment?desktop_handoff=1&source=desktop#top')
  expect(restoreCookieSession).toHaveBeenCalledWith({ replaceClientSession: true })
  expect(router.currentRoute.value.fullPath).toBe('/payment?source=desktop#top')
})
```

- [x] **Step 5: Run router tests and verify RED**

Run: `cd frontend && pnpm test --run src/__tests__/integration/navigation.spec.ts`

Expected: `/payment` does not resolve and the guard redirects to login.

- [x] **Step 6: Implement alias and guard bootstrap**

Add `alias: '/payment'` to `PurchaseSubscription`. Before the protected-route rejection:

```ts
const desktopHandoff = to.query.desktop_handoff === '1'
if (requiresAuth && (desktopHandoff || !authStore.isAuthenticated)) {
  const restored = await authStore.restoreCookieSession({ replaceClientSession: desktopHandoff })
  if (restored && desktopHandoff) {
    const query = { ...to.query }
    delete query.desktop_handoff
    next({ path: to.path, query, hash: to.hash, replace: true })
    return
  }
}
```

- [x] **Step 7: Verify frontend and commit**

Run: `cd frontend && pnpm typecheck && pnpm test --run src/stores/__tests__/auth.spec.ts src/api/__tests__/auth-cookie-session.spec.ts src/__tests__/integration/navigation.spec.ts && pnpm build`

Commit: `feat(payment): restore desktop handoff session`

### Task 4: Activate feature flag, synchronize knowledge, and verify

**Files:**
- Modify: `backend/internal/service/lumio_desktop_config.go`
- Modify: `backend/internal/service/lumio_desktop_config_test.go`
- Modify: `backend/internal/server/api_contract_test.go`
- Modify: `.spec/knowledge/features/lumio-desktop-client.md`
- Modify: `.spec/tasks/lumio-desktop-payment-handoff-*.md`
- Modify: `docs/specs/2026-08-11-lumio-desktop-payment-handoff-design.md`
- Modify: `docs/plans/2026-08-11-lumio-desktop-payment-handoff.md`

**Interfaces:**
- Consumes: completed Tasks 1-3
- Produces: built-in `PaymentHandoff: true`, still gated by `SettingPaymentEnabled`

- [x] **Step 1: Change default expectations first and verify RED**

Change the default service/API contract tests to expect `payment_handoff: true` while retaining the test where global payment is false.

Run: `cd backend && go test -tags=unit -count=1 ./internal/service ./internal/server -run 'LumioDesktop|APIContracts'`

Expected: default-config assertions fail because production still returns false.

- [x] **Step 2: Enable the default and verify GREEN**

```go
FeatureFlags: LumioDesktopFeatureFlags{
    Registration: true,
    PaymentHandoff: true,
    KeyProvisioning: true,
},
```

Run: `cd backend && go test -tags=unit -count=1 ./internal/service ./internal/server -run 'LumioDesktop|APIContracts'`

Expected: PASS; the global-payment-off case still returns false.

- [x] **Step 3: Synchronize knowledge and task status**

Document the issue/consume endpoints, hash-only Redis storage, access-only Cookie, frontend bootstrap marker, and Docker integration-test limitation in `.spec/knowledge/features/lumio-desktop-client.md`. Mark all three task cards completed only after verification.

- [x] **Step 4: Run full verification**

Backend:

```bash
cd backend
go test -tags=unit -count=1 ./...
go test -tags=integration -count=1 ./...
go vet -tags integration ./...
/Users/cui/go/bin/golangci-lint run --new-from-rev ef620b39a ./...
```

Frontend:

```bash
cd frontend
pnpm typecheck
pnpm test --run
pnpm build
```

Also run `git diff --check`. If Docker remains unavailable, record the repository integration harness skip explicitly; do not claim database-backed integration execution.

Verification result:

- Backend unit/integration/vet passed and incremental golangci-lint reported `0 issues`.
- The Docker client was present but its Colima daemon was unavailable. Integration-tag code ran, while container-backed database/Redis harnesses followed the repository's existing skip path; no external-container execution is claimed.
- Frontend typecheck, production build, `/payment` dev-server HTTP 200, and the task-related route/auth tests passed, including custom protected payment-path account replacement.
- The final full frontend suite reported 1212 passed, 7 failed, and two unrelated unhandled rejections. The same baseline failures remain in unchanged model-whitelist, account modal, sidebar, risk-control, ops-log locale, and prompt-audit tests; the unhandled rejections came from unchanged dashboard/profile tests. They are outside this task and were not modified.

- [x] **Step 5: Review and commit activation/docs**

Commit: `docs(desktop): finalize payment handoff contract`
