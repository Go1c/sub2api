---
status: completed
---

# Reset Subscription Weekly Limit

## Goal

Add an administrator action on the subscription list that resets only the selected subscription's current weekly usage window, allowing it to use subscription credit again during the same week without changing total quota or cumulative quota usage.

## Acceptance

- Active subscription rows show a new action labeled “重置周限” alongside the existing adjust / reset quota / revoke actions.
- The new action has an explicit confirmation dialog explaining that only the weekly limit usage is reset.
- Confirming sends `POST /admin/subscriptions/:id/reset-quota` with exactly `{ daily: false, weekly: true, monthly: false }`.
- The action does not update `quota_limit_usd`, `quota_used_usd`, daily usage, subscription status, or expiration; it reuses the existing backend weekly-only reset behavior.
- Success closes the dialog, clears transient state, reloads the subscription list, and shows a localized success message; failure shows a localized error and preserves existing error-handling conventions.
- Chinese, English, and Traditional Chinese locale files remain structurally aligned.
- A frontend regression test first fails and then proves the weekly-only API request payload; existing backend weekly-only service coverage remains green.
- Update the subscription feature knowledge to record the administrator weekly-limit reset behavior.

## Relevant Files

- `frontend/src/views/admin/SubscriptionsView.vue`
- `frontend/src/api/admin/subscriptions.ts`
- `frontend/src/api/__tests__/admin.subscriptions.spec.ts` (new if appropriate)
- `frontend/src/i18n/locales/{zh,en,zh-Hant}.ts`
- `backend/internal/service/subscription_reset_quota_test.go`
- `.spec/knowledge/features/subscription-credit-pool.md`

## Verification

- `cd backend && go test -tags=unit ./internal/service -run TestAdminResetQuota_ResetWeeklyOnly`
- `cd frontend && pnpm test:run src/api/__tests__/admin.subscriptions.spec.ts`
- `cd frontend && pnpm test:run src/i18n/__tests__/localeIntegrity.spec.ts`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- Start the frontend dev server and verify `/admin/subscriptions` is reachable by curl or browser; visually confirm the new action and confirmation copy if an authenticated admin session is available.
