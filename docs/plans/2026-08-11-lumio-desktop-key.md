# Lumio Desktop Key Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development to implement this plan task-by-task (hosts without subagents: its Inline Fallback section). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the existing API-key create endpoint return one account-level `Lumio Codex Desktop` key across retries, devices, and server instances.

**Architecture:** The service recognizes the exact reserved name, queries the active row, then uses the normal creation path. A PostgreSQL partial unique index serializes concurrent inserts; a loser reads and returns the winning row. Soft deletion releases the reserved name. Migration renames historical duplicate display names without deleting credentials.

**Tech Stack:** Go 1.25.7, Ent, PostgreSQL partial indexes, SQL migrations, unit and integration tests.

## Global Constraints

- Reuse the existing authenticated `POST /api/v1/keys`; do not add device management or a second key endpoint.
- Reserved name is exactly `Lumio Codex Desktop`.
- At most one non-deleted reserved row exists per user.
- Historical duplicate credentials must remain valid and must not be deleted or rotated.
- Soft-deleted/revoked reserved keys do not block a replacement.

---

### Task 1: Database invariant and repository lookup

**Files:**
- Modify: `backend/ent/schema/api_key.go`
- Modify: generated Ent files via `go generate ./ent`
- Create: `backend/migrations/923_lumio_desktop_api_key_unique.sql`
- Create: `backend/migrations/lumio_desktop_api_key_migration_test.go`
- Modify: `backend/internal/repository/api_key_repo.go`
- Modify: `backend/internal/repository/api_key_repo_integration_test.go`

**Interfaces:**
- Produces: `GetByUserIDAndName(context.Context, int64, string) (*service.APIKey, error)`

- [ ] **Step 1: Write failing migration and repository tests**

```go
func TestMigration923PreservesDuplicateCredentialsAndAddsPartialUniqueIndex(t *testing.T) {
    raw, err := FS.ReadFile("923_lumio_desktop_api_key_unique.sql")
    require.NoError(t, err)
    sql := string(raw)
    require.Contains(t, sql, "ROW_NUMBER() OVER")
    require.Contains(t, sql, "Lumio Codex Desktop (legacy ")
    require.Contains(t, sql, "deleted_at IS NULL")
    require.NotContains(t, strings.ToUpper(sql), "DELETE FROM API_KEYS")
}

func (s *APIKeyRepoSuite) TestGetByUserIDAndName() {
    user := s.mustCreateUser("lumio-lookup@test.com")
    expected := s.mustCreateApiKey(user.ID, "sk-lumio-lookup", service.LumioDesktopAPIKeyName, nil)
    got, err := s.repo.GetByUserIDAndName(s.ctx, user.ID, service.LumioDesktopAPIKeyName)
    s.Require().NoError(err)
    s.Require().Equal(expected.ID, got.ID)
}

func (s *APIKeyRepoSuite) TestReservedDesktopNameIsUniquePerActiveUser() {
    user := s.mustCreateUser("lumio-unique@test.com")
    first := &service.APIKey{UserID: user.ID, Key: "sk-lumio-first", Name: service.LumioDesktopAPIKeyName, Status: service.StatusActive}
    second := &service.APIKey{UserID: user.ID, Key: "sk-lumio-second", Name: service.LumioDesktopAPIKeyName, Status: service.StatusActive}
    s.Require().NoError(s.repo.Create(s.ctx, first))
    s.Require().ErrorIs(s.repo.Create(s.ctx, second), service.ErrAPIKeyExists)
    s.Require().NoError(s.repo.Delete(s.ctx, first.ID))
    s.Require().NoError(s.repo.Create(s.ctx, second))
}
```

- [ ] **Step 2: Run focused tests and verify RED**

Run: `cd backend && go test -tags=unit ./migrations -run 'Migration923' && go test -tags=integration ./internal/repository -run 'APIKeyRepoSuite'`

Expected: migration test fails because file 923 is absent; repository test fails because the method/index is absent.

- [ ] **Step 3: Implement migration, Ent index, and lookup**

Use a `row_number() over (partition by user_id order by created_at, id)` CTE. Keep row 1 unchanged; rename rows `rn > 1` to `Lumio Codex Desktop (legacy <id>)`. Add a named partial unique index on `(user_id, name)` with `deleted_at IS NULL AND name = 'Lumio Codex Desktop'`. Mirror it in Ent with `StorageKey` and `entsql.IndexWhere`.

- [ ] **Step 4: Generate Ent and verify GREEN**

Run: `cd backend && go generate ./ent && go test -tags=unit ./migrations -run 'Migration923' && go test -tags=integration ./internal/repository -run 'APIKeyRepoSuite'`

Expected: PASS.

### Task 2: Service get-or-create behavior

**Files:**
- Modify: `backend/internal/service/api_key_service.go`
- Create: `backend/internal/service/api_key_service_lumio_desktop_test.go`
- Modify: API-key repository test stubs required by the interface.

**Interfaces:**
- Consumes: `APIKeyRepository.GetByUserIDAndName`
- Produces: idempotent behavior for `Create(ctx, userID, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})`

- [ ] **Step 1: Write failing service tests**

```go
func TestAPIKeyServiceCreateLumioDesktopReusesExisting(t *testing.T) {
    existing := &APIKey{ID: 7, UserID: 42, Key: "sk-existing", Name: LumioDesktopAPIKeyName, Status: StatusActive}
    repo := newLumioDesktopAPIKeyRepoStub(existing)
    svc := newLumioDesktopAPIKeyService(repo)
    got, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})
    require.NoError(t, err)
    require.Same(t, existing, got)
    require.Zero(t, repo.createCalls)
}

func TestAPIKeyServiceCreateLumioDesktopCreatesFirstKey(t *testing.T) {
    repo := newLumioDesktopAPIKeyRepoStub(nil)
    svc := newLumioDesktopAPIKeyService(repo)
    got, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})
    require.NoError(t, err)
    require.Equal(t, LumioDesktopAPIKeyName, got.Name)
    require.Equal(t, 1, repo.createCalls)
}

func TestAPIKeyServiceCreateLumioDesktopResolvesConcurrentUniqueConflict(t *testing.T) {
    winner := &APIKey{ID: 9, UserID: 42, Key: "sk-winner", Name: LumioDesktopAPIKeyName, Status: StatusActive}
    repo := newLumioDesktopAPIKeyConflictRepoStub(winner)
    svc := newLumioDesktopAPIKeyService(repo)
    got, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: LumioDesktopAPIKeyName})
    require.NoError(t, err)
    require.Equal(t, winner.ID, got.ID)
    require.Equal(t, 2, repo.lookupCalls)
}

func TestAPIKeyServiceCreateOrdinaryNameKeepsConflict(t *testing.T) {
    repo := newOrdinaryAPIKeyConflictRepoStub()
    svc := newLumioDesktopAPIKeyService(repo)
    _, err := svc.Create(context.Background(), 42, CreateAPIKeyRequest{Name: "ordinary"})
    require.ErrorIs(t, err, ErrAPIKeyExists)
}
```

- [ ] **Step 2: Run focused test and verify RED**

Run: `cd backend && go test -tags=unit ./internal/service -run 'LumioDesktop'`

Expected: existing reserved key is not reused.

- [ ] **Step 3: Implement minimal conflict-safe get-or-create**

Add `LumioDesktopAPIKeyName`. Before generating a key, owner-scope lookup the reserved active row. On a create error matching `ErrAPIKeyExists`, repeat the lookup and return that row; ordinary names return the original error unchanged. Preserve all existing validation and cache invalidation for the first successful insert.

- [ ] **Step 4: Run focused and package tests**

Run: `cd backend && go test -tags=unit ./internal/service ./internal/repository ./internal/handler`

Expected: PASS.

- [ ] **Step 5: Review the task diff and commit**

Run: `git diff --check && git diff --stat`

Commit: `feat(keys): provision one Lumio desktop key per account`
