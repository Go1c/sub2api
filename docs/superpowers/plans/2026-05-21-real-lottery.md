# Real Lottery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the frontend-only lottery mock with a backend-backed, multi-user lottery that delivers winning redeem codes through site messages.

**Architecture:** Add persistent lottery campaign, code, and draw records in the backend. The frontend asks the backend whether the authenticated user should see a popup and posts draw requests to the backend. The backend owns participant limits, duplicate-draw prevention, prize assignment, and site-message delivery.

**Tech Stack:** Go, Gin, Ent, PostgreSQL migrations, Vue 3, Pinia, Axios, Vitest, Go tests.

---

## File Map

- Create `backend/internal/domain/lottery.go` for lottery domain errors.
- Create `backend/ent/schema/lottery_campaign.go`, `backend/ent/schema/lottery_code.go`, and `backend/ent/schema/lottery_draw.go`.
- Create `backend/migrations/140_add_lottery.sql` for database tables and indexes.
- Create `backend/internal/service/lottery.go`, `backend/internal/service/lottery_service.go`, and `backend/internal/service/lottery_service_test.go`.
- Create `backend/internal/repository/lottery_repo.go`.
- Create `backend/internal/handler/dto/lottery.go`.
- Create `backend/internal/handler/lottery_handler.go` and `backend/internal/handler/admin/lottery_handler.go`.
- Modify `backend/internal/service/site_message.go` and `backend/internal/service/site_message_service.go` to add internal system/admin delivery.
- Modify `backend/internal/repository/wire.go`, `backend/internal/service/wire.go`, `backend/internal/handler/wire.go`, `backend/internal/server/routes/user.go`, and `backend/internal/server/routes/admin.go` to wire routes and dependencies.
- Regenerate `backend/ent/*` and `backend/cmd/server/wire_gen.go`.
- Create `frontend/src/api/lottery.ts` and `frontend/src/api/admin/lottery.ts`.
- Modify `frontend/src/api/admin/index.ts`, `frontend/src/stores/lottery.ts`, `frontend/src/components/lottery/LotteryPromptManager.vue`, `frontend/src/components/lottery/LotteryDialog.vue`, and `frontend/src/views/admin/LotteryView.vue`.
- Add or update tests under `frontend/src/stores/__tests__/lottery.spec.ts`, `frontend/src/components/lottery/__tests__/LotteryPromptManager.spec.ts`, and `frontend/src/views/admin/__tests__/LotteryView.spec.ts`.

---

## Task 1: Backend Service Contract and TDD

**Files:**
- Create: `backend/internal/domain/lottery.go`
- Create: `backend/internal/service/lottery.go`
- Create: `backend/internal/service/lottery_service.go`
- Create: `backend/internal/service/lottery_service_test.go`
- Modify: `backend/internal/service/site_message.go`
- Modify: `backend/internal/service/site_message_service.go`

- [ ] **Step 1: Write failing service tests**

Add tests that prove:

```go
func TestLotteryServiceCreateCampaignFinishesPreviousActive(t *testing.T)
func TestLotteryServiceCreateCampaignRejectsDuplicateCodes(t *testing.T)
func TestLotteryServiceCreateCampaignRejectsDisabledSiteMessages(t *testing.T)
func TestLotteryServiceGetActiveForUserReturnsNilAfterDraw(t *testing.T)
func TestLotteryServiceDrawWinSendsSiteMessage(t *testing.T)
func TestLotteryServiceDrawRejectsDuplicateUserDraw(t *testing.T)
func TestLotteryServiceDrawFinishesFullCampaign(t *testing.T)
```

Use in-memory stubs for `LotteryRepository`, `LotterySiteMessageSender`, and `LotterySettingsReader`.

- [ ] **Step 2: Verify RED**

Run:

```bash
cd backend
go test ./internal/service -run 'TestLotteryService' -count=1
```

Expected: FAIL because `LotteryService` and related types do not exist.

- [ ] **Step 3: Implement service types and validation**

Add constants, errors, models, request structs, repository interfaces, and a minimal `LotteryService` that compiles. Implement campaign creation validation first.

- [ ] **Step 4: Verify first GREEN**

Run the same service test command. Expected: campaign creation tests pass; draw tests still fail if not implemented.

- [ ] **Step 5: Implement draw logic**

Implement deterministic test injection:

```go
type LotteryService struct {
    repo LotteryRepository
    settings LotterySettingsReader
    siteMessages LotterySiteMessageSender
    now func() time.Time
    randFloat func() float64
}
```

Use `randFloat` in tests to force win or loss.

- [ ] **Step 6: Verify service GREEN**

Run:

```bash
cd backend
go test ./internal/service -run 'TestLotteryService' -count=1
```

Expected: PASS.

---

## Task 2: Persistence and Generated Ent Code

**Files:**
- Create: `backend/ent/schema/lottery_campaign.go`
- Create: `backend/ent/schema/lottery_code.go`
- Create: `backend/ent/schema/lottery_draw.go`
- Create: `backend/migrations/140_add_lottery.sql`
- Create: `backend/internal/repository/lottery_repo.go`
- Modify: `backend/internal/repository/wire.go`
- Generated: `backend/ent/*`

- [ ] **Step 1: Write failing repository tests**

Add tests to `backend/internal/repository/lottery_repo_test.go` for:

```go
func TestLotteryRepositoryCreateCampaignArchivesActive(t *testing.T)
func TestLotteryRepositoryDrawTransactionAssignsOneCode(t *testing.T)
func TestLotteryRepositoryGetActiveForUserExcludesDrawnUser(t *testing.T)
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd backend
go test ./internal/repository -run 'TestLotteryRepository' -count=1
```

Expected: FAIL because repository and generated Ent types do not exist.

- [ ] **Step 3: Add Ent schemas and migration**

Create the three schema files and `140_add_lottery.sql` with the tables, foreign keys, unique indexes, and lookup indexes from the design spec.

- [ ] **Step 4: Generate Ent code**

Run:

```bash
cd backend
go generate ./ent
```

Expected: generated lottery entity files appear under `backend/ent`.

- [ ] **Step 5: Implement repository**

Implement `NewLotteryRepository(client *ent.Client) service.LotteryRepository`, including a transaction method used by `LotteryService.Draw`.

- [ ] **Step 6: Verify repository GREEN**

Run:

```bash
cd backend
go test ./internal/repository -run 'TestLotteryRepository' -count=1
```

Expected: PASS.

---

## Task 3: Backend Handlers, Routes, and Wire

**Files:**
- Create: `backend/internal/handler/dto/lottery.go`
- Create: `backend/internal/handler/lottery_handler.go`
- Create: `backend/internal/handler/admin/lottery_handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/internal/repository/wire.go`
- Modify: `backend/internal/server/routes/user.go`
- Modify: `backend/internal/server/routes/admin.go`
- Generated: `backend/cmd/server/wire_gen.go`

- [ ] **Step 1: Write failing handler tests**

Add tests for:

```go
func TestLotteryHandlerGetActiveReturnsNullWhenNoCampaign(t *testing.T)
func TestLotteryHandlerDrawReturnsWinMessage(t *testing.T)
func TestAdminLotteryHandlerCreateCampaign(t *testing.T)
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd backend
go test ./internal/handler/... -run 'Lottery' -count=1
```

Expected: FAIL because handlers and routes do not exist.

- [ ] **Step 3: Implement DTOs and handlers**

Expose user endpoints:

```text
GET  /api/v1/lottery/active
POST /api/v1/lottery/:id/draw
```

Expose admin endpoints:

```text
GET  /api/v1/admin/lottery/campaigns
POST /api/v1/admin/lottery/campaigns
GET  /api/v1/admin/lottery/campaigns/:id
POST /api/v1/admin/lottery/campaigns/:id/finish
```

- [ ] **Step 4: Wire dependencies**

Add `Lottery` fields to `handler.Handlers` and `handler.AdminHandlers`. Register repository, service, and handlers in Wire provider sets.

- [ ] **Step 5: Generate Wire**

Run:

```bash
cd backend/cmd/server
go generate
```

Expected: `wire_gen.go` includes lottery dependencies.

- [ ] **Step 6: Verify backend handler GREEN**

Run:

```bash
cd backend
go test ./internal/handler/... -run 'Lottery' -count=1
```

Expected: PASS.

---

## Task 4: Frontend API Store and Popup

**Files:**
- Create: `frontend/src/api/lottery.ts`
- Create: `frontend/src/api/admin/lottery.ts`
- Modify: `frontend/src/api/admin/index.ts`
- Modify: `frontend/src/stores/lottery.ts`
- Modify: `frontend/src/components/lottery/LotteryPromptManager.vue`
- Modify: `frontend/src/components/lottery/LotteryDialog.vue`

- [ ] **Step 1: Write failing frontend tests**

Add tests that prove:

```ts
it('fetches active campaign after login')
it('does not open when campaign was dismissed in this session')
it('shows a site-message claim tip after winning')
it('does not render the raw code in the dialog result')
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd frontend
npm.cmd test -- src/stores/__tests__/lottery.spec.ts src/components/lottery/__tests__/LotteryPromptManager.spec.ts
```

Expected: FAIL because APIs and behavior are not implemented.

- [ ] **Step 3: Implement API-backed store**

Replace localStorage campaign persistence with `fetchActive` and `draw` calls. Keep only session dismissal in the prompt manager.

- [ ] **Step 4: Update popup dialog**

Keep wheel animation. Replace winner code display with the site-message claim tip and a `/site-messages` link.

- [ ] **Step 5: Verify frontend popup GREEN**

Run the same frontend test command. Expected: PASS.

---

## Task 5: Frontend Admin Page

**Files:**
- Modify: `frontend/src/views/admin/LotteryView.vue`
- Add or update: `frontend/src/views/admin/__tests__/LotteryView.spec.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/zh-Hant.ts`

- [ ] **Step 1: Write failing admin page tests**

Add tests that prove:

```ts
it('creates a campaign through the admin lottery API')
it('reloads campaign history after create')
it('shows a clear error when site messages are disabled')
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd frontend
npm.cmd test -- src/views/admin/__tests__/LotteryView.spec.ts
```

Expected: FAIL because the admin page still uses local store creation.

- [ ] **Step 3: Implement admin page API integration**

Load campaigns from `/admin/lottery/campaigns`, create through the backend, and show code and winner detail from backend detail responses.

- [ ] **Step 4: Verify admin page GREEN**

Run the same frontend admin test command. Expected: PASS.

---

## Task 6: Full Verification

**Files:**
- All changed files.

- [ ] **Step 1: Run backend focused tests**

```bash
cd backend
go test ./internal/service ./internal/repository ./internal/handler/... -run 'Lottery|SiteMessage' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run backend package tests likely affected by DI**

```bash
cd backend
go test ./cmd/server ./internal/server/... ./internal/service ./internal/repository
```

Expected: PASS.

- [ ] **Step 3: Run frontend focused tests**

```bash
cd frontend
npm.cmd test -- src/stores/__tests__/lottery.spec.ts src/components/lottery/__tests__/LotteryPromptManager.spec.ts src/views/admin/__tests__/LotteryView.spec.ts
```

Expected: PASS.

- [ ] **Step 4: Run frontend typecheck/build**

```bash
cd frontend
npm.cmd run typecheck
npm.cmd run build
```

Expected: both commands exit 0.

- [ ] **Step 5: Final status check**

```bash
git status --short
```

Expected: only intentional lottery-related changes.
