# User Request Monitoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add admin-only user request monitoring in the ops/risk center that captures future client request bodies for selected users without blocking gateway traffic.

**Architecture:** Add two ops tables for monitor tasks and captures, a focused `OpsUserRequestMonitorService` for admin CRUD plus best-effort hot-path capture, repository methods on the existing ops repository, and lightweight capture hooks in gateway handlers after auth and body read. The service launches capture work asynchronously, recovers panics, uses a per-minute Redis gate before sampling, truncates each raw request body at 256 KiB, and treats all monitor failures as record-only failures.

**Tech Stack:** Go 1.23 backend with Gin, PostgreSQL SQL migrations, Redis, Wire DI, Vue 3 + TypeScript frontend, Vitest where practical.

---

## File Map

- Create `backend/migrations/136_ops_user_request_monitoring.sql`: idempotent tables and indexes.
- Modify `backend/internal/service/ops_port.go`: repository interface additions and request monitor DTOs.
- Create `backend/internal/service/ops_user_request_monitor_service.go`: validation, admin APIs, capture gate, async non-blocking capture, cleanup worker.
- Create `backend/internal/service/ops_user_request_monitor_service_test.go`: TDD coverage for validation, truncation, rate-before-sampling, and non-blocking write failures.
- Create `backend/internal/repository/ops_user_request_monitor_repo.go`: SQL implementation for monitors and captures.
- Modify `backend/internal/service/ops_repo_mock_test.go`: mock method additions for the expanded interface.
- Modify `backend/internal/service/wire.go`: provide the monitor service.
- Modify `backend/internal/handler/admin/ops_handler.go`: inject monitor service.
- Create `backend/internal/handler/admin/ops_user_request_monitor_handler.go`: REST handlers.
- Modify `backend/internal/server/routes/admin.go`: add `/admin/ops/user-request-monitors` routes.
- Modify `backend/internal/handler/gateway_handler.go`, `backend/internal/handler/gateway_handler_chat_completions.go`, `backend/internal/handler/gateway_handler_responses.go`, `backend/internal/handler/gemini_v1beta_handler.go`: inject/call capture for Anthropic/Gemini-compatible paths.
- Modify `backend/internal/handler/openai_gateway_handler.go`, `backend/internal/handler/openai_chat_completions.go`, `backend/internal/handler/openai_images.go`: inject/call capture for OpenAI paths.
- Modify `backend/cmd/server/wire_gen.go` and `backend/cmd/server/wire_gen_test.go`: regenerate or align Wire output for the new dependency.
- Modify `frontend/src/api/admin/ops.ts`: TypeScript types and API calls.
- Create `frontend/src/views/admin/ops/components/OpsUserRequestMonitorCard.vue`: monitor list, create form, captures, detail view.
- Modify `frontend/src/views/admin/ops/OpsDashboard.vue`: mount the new card in ops dashboard.
- Modify `frontend/src/i18n/locales/zh.ts` and `frontend/src/i18n/locales/en.ts`: UI copy.

## Task 1: Backend Service Contract Tests

**Files:**
- Create: `backend/internal/service/ops_user_request_monitor_service_test.go`
- Modify: `backend/internal/service/ops_port.go`
- Modify: `backend/internal/service/ops_repo_mock_test.go`

- [ ] **Step 1: Write failing tests for create validation and defaults**

Add tests that construct `NewOpsUserRequestMonitorService(repo, userRepo, nil)` and assert:
- `retention_days` defaults to 7.
- invalid duration, max captures, sample rate, and retention return errors.
- second active monitor conflict returns an error.

Run: `go test ./internal/service -run TestOpsUserRequestMonitorService_Create -count=1`
Expected: FAIL because service/type does not exist.

- [ ] **Step 2: Add minimal service contracts**

Add DTOs to `ops_port.go`:
- `OpsUserRequestMonitor`, `OpsUserRequestCapture`, `OpsCreateUserRequestMonitorInput`, `OpsUserRequestMonitorFilter`, `OpsUserRequestCaptureFilter`, `OpsCaptureClientRequestInput`.
- Repository methods: create/list/get active/stop/insert/list/get/delete/cleanup.

Run the same test.
Expected: still FAIL because implementation is missing.

- [ ] **Step 3: Implement create/list/stop validation**

Implement `NewOpsUserRequestMonitorService`, `CreateMonitor`, `ListMonitors`, `StopMonitor`, `ListCaptures`, `GetCapture`, `DeleteCapture` with clear limits:
- duration: 1 second to 24 hours.
- max captures per minute: 1 to 120.
- sample rate percent: 1 to 100.
- retention days: default 7, allowed 1 to 30.
- one active monitor per user.

Run: `go test ./internal/service -run TestOpsUserRequestMonitorService_Create -count=1`
Expected: PASS.

## Task 2: Capture Semantics Tests

**Files:**
- Modify: `backend/internal/service/ops_user_request_monitor_service_test.go`
- Modify: `backend/internal/service/ops_user_request_monitor_service.go`

- [ ] **Step 1: Write failing tests for raw truncation**

Add a test where a monitor exists and body length is `256*1024+1`, then assert inserted capture body length is exactly `256*1024`, `body_bytes` is original length, and `body_truncated` is true.

Run: `go test ./internal/service -run TestOpsUserRequestMonitorService_CaptureTruncatesRawBody -count=1`
Expected: FAIL because capture is not implemented.

- [ ] **Step 2: Write failing tests for rate-before-sampling**

Use fake limiter and sampler hooks. Assert limiter is called before sampler and that limiter denial prevents sampler and insert.

Run: `go test ./internal/service -run TestOpsUserRequestMonitorService_CaptureAppliesRateBeforeSampling -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement capture path**

Implement:
- `CaptureClientRequestIfEnabled(ctx, input)` as async best-effort goroutine.
- `CaptureClientRequestSync(ctx, input)` for tests.
- Active monitor lookup by user.
- Redis limiter abstraction with skip-on-Redis-error behavior.
- `recover()` around async capture.
- write errors logged and swallowed.

Run: `go test ./internal/service -run TestOpsUserRequestMonitorService_Capture -count=1`
Expected: PASS.

## Task 3: Repository and Migration

**Files:**
- Create: `backend/migrations/136_ops_user_request_monitoring.sql`
- Create: `backend/internal/repository/ops_user_request_monitor_repo.go`
- Optional Test: `backend/internal/repository/ops_user_request_monitor_repo_test.go`

- [ ] **Step 1: Write migration**

Create idempotent SQL for the two tables, status checks, FK references, and indexes from the design.

- [ ] **Step 2: Write repository methods**

Use existing `opsRepository` and helper null functions. Implement list queries with joins to `users` and pagination.

- [ ] **Step 3: Verify compile**

Run: `go test ./internal/repository -run TestDoesNotExist -count=1`
Expected: PASS compile.

## Task 4: Admin HTTP Endpoints

**Files:**
- Modify: `backend/internal/handler/admin/ops_handler.go`
- Create: `backend/internal/handler/admin/ops_user_request_monitor_handler.go`
- Modify: `backend/internal/server/routes/admin.go`

- [ ] **Step 1: Add handlers**

Implement create/list/stop/list captures/detail/delete. Use `response.Success`, `response.Paginated`, and `response.ErrorFrom`. Resolve `created_by` from `GetAuthSubjectFromContext`.

- [ ] **Step 2: Add routes**

Add routes under `/api/v1/admin/ops/user-request-monitors`:
- `POST ""`
- `GET ""`
- `POST "/:id/stop"`
- `GET "/:id/captures"`
- `GET "/:id/captures/:capture_id"`
- `DELETE "/:id/captures/:capture_id"`

- [ ] **Step 3: Compile handlers**

Run: `go test ./internal/handler/admin ./internal/server/routes -run TestDoesNotExist -count=1`
Expected: PASS compile.

## Task 5: Gateway Capture Hooks

**Files:**
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/gateway_handler_chat_completions.go`
- Modify: `backend/internal/handler/gateway_handler_responses.go`
- Modify: `backend/internal/handler/gemini_v1beta_handler.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Modify: `backend/internal/handler/openai_chat_completions.go`
- Modify: `backend/internal/handler/openai_images.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/cmd/server/wire_gen.go`
- Modify: `backend/cmd/server/wire_gen_test.go`

- [ ] **Step 1: Inject service into gateway handlers**

Add `userRequestMonitorService *service.OpsUserRequestMonitorService` fields and constructor parameters to both gateway handlers.

- [ ] **Step 2: Add helper methods**

Add `captureClientRequest(c, model, body)` helpers that gather user ID, API key ID, group ID, content type, request ID, method, endpoint, and raw body. Return immediately if service is nil or user is unauthenticated.

- [ ] **Step 3: Call helper after body parse**

Call immediately after `setOpsRequestContext` in request-body handlers. Do not call for GET/model-list routes.

- [ ] **Step 4: Verify no blocking semantics**

Ensure helper only calls async service and never returns an error to handler.

Run: `go test ./internal/handler -run TestDoesNotExist -count=1`
Expected: PASS compile.

## Task 6: Frontend Ops UI

**Files:**
- Modify: `frontend/src/api/admin/ops.ts`
- Create: `frontend/src/views/admin/ops/components/OpsUserRequestMonitorCard.vue`
- Modify: `frontend/src/views/admin/ops/OpsDashboard.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Add API client types/functions**

Add monitor/capture types and methods to `opsAPI` for create, list, stop, list captures, detail, and delete.

- [ ] **Step 2: Add card UI**

Render:
- create form with user ID/email hint, duration minutes, per-minute cap, sample rate, retention default 7.
- warning that request body is stored raw and unredacted.
- monitor list with status, email, capture count, limits, end time, actions.
- capture list and detail dialog showing raw body and truncation state.

- [ ] **Step 3: Mount card**

Place card near the top of `OpsDashboard.vue` after the visual metrics rows so admins can access it from the ops center.

- [ ] **Step 4: Verify frontend types**

Run: `pnpm --dir frontend typecheck`
Expected: PASS.

## Task 7: Final Verification and PR

**Files:**
- All touched files.

- [ ] **Step 1: Run backend tests**

Run: `go test ./...` from `backend`.
Expected: PASS.

- [ ] **Step 2: Run frontend build checks**

Run: `pnpm --dir frontend typecheck` and `pnpm --dir frontend build`.
Expected: PASS.

- [ ] **Step 3: Commit implementation**

Run:
```bash
git status --short
git add backend frontend docs/superpowers/plans/2026-05-09-user-request-monitoring.md
git commit -m "feat: add user request monitoring"
```

- [ ] **Step 4: Push feature branch and open PR**

Run:
```bash
git push -u origin feature/ops-user-request-monitoring
gh pr create --base dev --head feature/ops-user-request-monitoring --title "feat: add user request monitoring" --body "Adds admin-only user request monitoring for future request bodies with rate limits, sampling, retention, and non-blocking gateway capture."
```

- [ ] **Step 5: Merge PR into dev when approved**

Use GitHub PR merge or `gh pr merge` only after checks pass and the user approves the PR merge step.
