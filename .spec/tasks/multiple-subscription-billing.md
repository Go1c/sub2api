---
status: completed
---

# Multiple Subscription Billing

## Goal

Allow administrators to configure whether users can hold multiple active subscriptions, and when enabled bill usage against user subscriptions ordered by subscription time, falling through to later subscriptions when earlier ones cannot cover the charge because of remaining quota or daily / weekly limits.

## Acceptance

- Admin settings expose and persist a switch for multiple subscription purchases.
- Subscription purchase and fulfillment respect the setting.
- Usage billing consumes subscriptions by earliest purchase / creation time first.
- If the earliest subscription cannot fully cover the charge because of quota or window limits, remaining cost is attempted against later subscriptions before balance.
- Regression tests cover disabled and enabled settings, ordered consumption, and limit fallback.

## Verification

- `cd backend && go test -tags=unit ./...`
- `cd backend && go test -tags=integration ./...`
- `cd backend && go vet -tags integration ./...`
- `cd backend && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.0 run ./...`
- `cd frontend && pnpm test:run src/views/admin/__tests__/SettingsView.spec.ts`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- `curl -fsS -L http://127.0.0.1:5173/admin/settings`
