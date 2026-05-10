# Site Messages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build configurable site messages with inbox, sent mail, replies, read state, unread red-dot reminders, admin user-table sending, 30-day default retention, and a configurable daily send limit for non-admin users.

**Architecture:** Add a dedicated `site_messages` Ent schema, repository, service, handlers, and routes. Reuse the existing settings pipeline for feature switch, daily send limit, and retention. Frontend adds typed API modules, a user mailbox view, sidebar unread indicator, settings controls, and an admin send dialog in the users table.

**Tech Stack:** Go, Gin, Ent, Wire, Vue 3, Pinia, Vue Router, TypeScript, Vitest, pnpm.

---

## Files

- Create: `backend/ent/schema/site_message.go`
- Create: `backend/internal/domain/site_message.go`
- Create: `backend/internal/service/site_message.go`
- Create: `backend/internal/service/site_message_service.go`
- Create: `backend/internal/service/site_message_service_test.go`
- Create: `backend/internal/repository/site_message_repo.go`
- Create: `backend/internal/handler/dto/site_message.go`
- Create: `backend/internal/handler/site_message_handler.go`
- Create: `backend/internal/handler/admin/site_message_handler.go`
- Create: `backend/migrations/137_add_site_messages.sql`
- Create: `frontend/src/api/siteMessages.ts`
- Create: `frontend/src/api/admin/siteMessages.ts`
- Create: `frontend/src/stores/siteMessages.ts`
- Create: `frontend/src/views/user/SiteMessagesView.vue`
- Create: `frontend/src/components/admin/user/UserSiteMessageModal.vue`
- Modify: `backend/ent/schema/user.go`
- Modify generated Ent files under `backend/ent/` after `go generate ./ent`
- Modify: `backend/internal/repository/wire.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify generated Wire files after `go generate ./cmd/server`
- Modify: `backend/internal/server/routes/user.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/settings_view.go`
- Modify: `backend/internal/service/setting_service.go`
- Modify: `backend/internal/handler/dto/settings.go`
- Modify: `backend/internal/handler/setting_handler.go`
- Modify: `backend/internal/handler/admin/setting_handler.go`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/admin/index.ts`
- Modify: `frontend/src/api/admin/settings.ts`
- Modify: `frontend/src/stores/app.ts`
- Modify: `frontend/src/utils/featureFlags.ts`
- Modify: `frontend/src/router/meta.d.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/views/admin/UsersView.vue`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/zh-Hant.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

---

### Task 1: Backend Service Behavior

**Files:**
- Create: `backend/internal/domain/site_message.go`
- Create: `backend/internal/service/site_message.go`
- Create: `backend/internal/service/site_message_service.go`
- Test: `backend/internal/service/site_message_service_test.go`

- [ ] **Step 1: Write failing service tests**

Cover:
- disabled feature returns `ErrSiteMessagesDisabled`.
- regular users resolve only exact user ID or exact email.
- regular users hit the configurable daily send limit.
- admins bypass the daily send limit.
- only sender or recipient can read message detail.
- opening a received message marks it read.
- replies require access to the parent and preserve `parent_id`.
- expired messages are ignored by unread counts and list/detail reads.

Run:

```bash
cd backend
go test ./internal/service -run 'TestSiteMessage' -count=1
```

Expected: FAIL because site message types and service do not exist.

- [ ] **Step 2: Implement minimal service/domain code**

Create domain errors:

```go
var (
	ErrSiteMessageNotFound = infraerrors.NotFound("SITE_MESSAGE_NOT_FOUND", "site message not found")
	ErrSiteMessagesDisabled = infraerrors.Forbidden("SITE_MESSAGES_DISABLED", "site messages are disabled")
	ErrSiteMessageRecipientNotFound = infraerrors.NotFound("SITE_MESSAGE_RECIPIENT_NOT_FOUND", "recipient not found")
	ErrSiteMessageDailyLimitExceeded = infraerrors.Forbidden("SITE_MESSAGE_DAILY_LIMIT_EXCEEDED", "daily site message send limit exceeded")
)
```

Create service input/output structs and repository interfaces. Keep the service independent from Ent so tests can use in-memory stubs.

- [ ] **Step 3: Run service tests**

Run:

```bash
cd backend
go test ./internal/service -run 'TestSiteMessage' -count=1
```

Expected: PASS.

---

### Task 2: Backend Persistence and Generated Code

**Files:**
- Create: `backend/ent/schema/site_message.go`
- Create: `backend/migrations/137_add_site_messages.sql`
- Create: `backend/internal/repository/site_message_repo.go`
- Modify: `backend/ent/schema/user.go`
- Modify: `backend/internal/repository/wire.go`
- Modify generated Ent files.

- [ ] **Step 1: Add Ent schema and SQL migration**

The SQL migration creates `site_messages` with `sender_id`, `recipient_id`, nullable `parent_id`, `subject`, `content`, nullable `read_at`, timestamps, foreign keys to `users(id)`, self parent FK, and the indexes from the design spec.

- [ ] **Step 2: Generate Ent code**

Run:

```bash
cd backend
go generate ./ent
```

Expected: generated `backend/ent/sitemessage*` files and updated `backend/ent/client.go`, `backend/ent/migrate/schema.go`, and predicate/runtime files.

- [ ] **Step 3: Implement repository**

Implement the `SiteMessageRepository` interface with Ent queries and pagination helpers.

- [ ] **Step 4: Run backend package tests**

Run:

```bash
cd backend
go test ./internal/service ./internal/repository -run 'TestSiteMessage|TestAnnouncement' -count=1
```

Expected: PASS.

---

### Task 3: Backend HTTP API and Settings

**Files:**
- Create: `backend/internal/handler/dto/site_message.go`
- Create: `backend/internal/handler/site_message_handler.go`
- Create: `backend/internal/handler/admin/site_message_handler.go`
- Modify: `backend/internal/server/routes/user.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/service/wire.go`
- Modify generated Wire files.
- Modify settings files listed in the Files section.

- [ ] **Step 1: Write failing handler/settings tests where practical**

Add focused tests for public settings injection schema and route handler basics when existing test seams are available.

Run:

```bash
cd backend
go test ./internal/handler ./internal/handler/admin ./internal/handler/dto -run 'TestSiteMessage|TestPublicSettingsInjectionPayload' -count=1
```

Expected: FAIL before handler/settings fields exist.

- [ ] **Step 2: Add settings pipeline**

Add keys:
- `site_messages_enabled`, default `false`.
- `site_messages_daily_send_limit`, default `10`.
- `site_messages_retention_days`, default `30`.

Expose `site_messages_enabled` in public settings and SSR injection. Expose all three in admin settings and update audit diff.

- [ ] **Step 3: Add user/admin handlers and routes**

User routes:
- `GET /api/v1/site-messages/inbox`
- `GET /api/v1/site-messages/sent`
- `GET /api/v1/site-messages/unread-count`
- `GET /api/v1/site-messages/:id`
- `POST /api/v1/site-messages`
- `POST /api/v1/site-messages/:id/reply`
- `POST /api/v1/site-messages/:id/read`
- `GET /api/v1/site-messages/recipient/resolve`

Admin routes:
- `POST /api/v1/admin/site-messages/users/:id`
- `GET /api/v1/admin/site-messages/recipients`

- [ ] **Step 4: Generate Wire code**

Run:

```bash
cd backend
go generate ./cmd/server
```

Expected: generated `cmd/server/wire_gen.go` includes the new repository, service, and handlers.

- [ ] **Step 5: Run backend tests**

Run:

```bash
cd backend
go test ./internal/service ./internal/handler ./internal/handler/admin ./internal/handler/dto ./internal/repository ./internal/server/routes -count=1
```

Expected: PASS.

---

### Task 4: Frontend APIs, Store, Navigation, and Routing

**Files:**
- Create: `frontend/src/api/siteMessages.ts`
- Create: `frontend/src/api/admin/siteMessages.ts`
- Create: `frontend/src/stores/siteMessages.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/admin/index.ts`
- Modify: `frontend/src/stores/app.ts`
- Modify: `frontend/src/utils/featureFlags.ts`
- Modify: `frontend/src/router/meta.d.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`

- [ ] **Step 1: Add failing frontend tests for feature flag and unread state where practical**

Use existing router/sidebar tests as a guide.

Run:

```bash
cd frontend
pnpm test:run src/router/__tests__/guards.spec.ts src/views/__tests__/HomeViewNav.spec.ts
```

Expected: existing tests pass before new assertions are added; new assertions fail until implementation.

- [ ] **Step 2: Add typed API clients and store**

The store fetches unread count only when authenticated and `site_messages_enabled` is true. It exposes `unreadCount`, `hasUnread`, and `refreshUnreadCount()`.

- [ ] **Step 3: Add route and feature gate**

Add `/site-messages` with `requiresSiteMessages: true`. When disabled, redirect users to `/dashboard`.

- [ ] **Step 4: Add sidebar item and red dot**

Add `nav.siteMessages`, use the existing `mail` icon, and render a small red indicator for the item when unread count is greater than zero.

---

### Task 5: Frontend User Mailbox

**Files:**
- Create: `frontend/src/views/user/SiteMessagesView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify locale files.

- [ ] **Step 1: Implement mailbox view**

Include:
- inbox/sent tabs.
- compose form with exact-recipient resolve.
- list rows with unread styling.
- detail panel.
- reply form.

- [ ] **Step 2: Verify with typecheck**

Run:

```bash
cd frontend
pnpm typecheck
```

Expected: PASS.

---

### Task 6: Admin Send Dialog and Settings UI

**Files:**
- Create: `frontend/src/components/admin/user/UserSiteMessageModal.vue`
- Modify: `frontend/src/views/admin/UsersView.vue`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/api/admin/settings.ts`
- Modify locale files.

- [ ] **Step 1: Add settings form fields**

Defaults:
- `site_messages_enabled: false`
- `site_messages_daily_send_limit: 10`
- `site_messages_retention_days: 30`

- [ ] **Step 2: Add admin user-table action**

In the "更多" menu, add "发送站内信". The modal recipient is fixed to the selected user and sends `{ subject, content }`.

- [ ] **Step 3: Run frontend verification**

Run:

```bash
cd frontend
pnpm typecheck
pnpm test:run src/views/admin/__tests__/UsersView.spec.ts src/router/__tests__/guards.spec.ts
```

Expected: PASS.

---

### Task 7: Final Verification

**Files:**
- All modified files.

- [ ] **Step 1: Format Go code**

Run:

```bash
cd backend
gofmt -w $(git ls-files '*.go')
```

- [ ] **Step 2: Backend verification**

Run:

```bash
cd backend
go test ./internal/service ./internal/handler ./internal/handler/admin ./internal/handler/dto ./internal/repository ./internal/server/routes ./cmd/server -count=1
```

Expected: PASS.

- [ ] **Step 3: Frontend verification**

Run:

```bash
cd frontend
pnpm typecheck
pnpm test:run
```

Expected: PASS or report pre-existing failures with exact output.

- [ ] **Step 4: Review diff**

Run:

```bash
git status --short
git diff --stat
```

Confirm the branch only contains the site-message feature and plan/spec docs.
