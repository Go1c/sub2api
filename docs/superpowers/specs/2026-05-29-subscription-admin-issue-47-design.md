# Subscription Admin Issue 47 Design

## Goal

Implement GitHub issue #47 with minimal upstream-facing changes:

- Revoked subscriptions do not count toward subscription waste statistics.
- Subscription daily and weekly quota windows refresh at an admin-configured UTC offset and hour.
- Subscription plan management appears under Subscription Management, not Order Management.

## Current State

Waste statistics read `subscription_credit_ledger` rows joined to `user_subscriptions`. The shared query builder filters by ledger time, plan, and user, but it does not filter subscription status. Ledger rows from subscriptions later marked `revoked` still appear in summary, plan, and time-series waste reports.

Subscription billing computes quota windows with fixed UTC helpers in `backend/internal/repository/usage_billing_repo.go`. Daily windows start at `00:00 UTC`, and weekly windows start on Monday at `00:00 UTC`. No subscription-specific reset setting exists in `SettingService`, the admin settings DTO, or the frontend settings form.

The admin subscription plan page uses `frontend/src/views/admin/orders/AdminPaymentPlansView.vue`, but the route and sidebar entry live under `/admin/orders/plans`. Subscription Management already contains user subscriptions and waste statistics.

## Product Rules

- A subscription with status `revoked` is excluded from all waste-stat views.
- Other inactive statuses, such as expired subscriptions, remain in waste statistics until a separate product requirement changes that rule.
- The subscription quota reset configuration has two admin fields:
  - UTC offset, stored as minutes from UTC.
  - Reset hour, stored as an integer from `0` to `23`.
- Defaults preserve existing behavior: `UTC+00:00` and hour `0`.
- Daily quota windows refresh every day at the configured hour in the configured UTC offset.
- Weekly quota windows refresh every Monday at the configured hour in the configured UTC offset.
- The setting affects new billing decisions immediately after the admin saves settings. Existing usage counters reset only when the next billed request observes that its stored window start is before the current configured window start.
- The old subscription plan URL redirects to the new URL so bookmarks and external links continue to work.

## Backend Design

### Waste Statistics

Add `SubscriptionStatusRevoked = "revoked"` beside the existing subscription status constants, then add the revoked-status exclusion once in `wasteStatsWhere()` in `backend/internal/repository/subscription_waste_stats_repo.go`. The summary, by-plan, and time-series queries all call this helper, so one condition keeps the reports consistent.

The SQL condition compares `us.status` with `service.SubscriptionStatusRevoked`. The filter belongs beside the existing common predicates so future waste-stat queries inherit it automatically.

### Settings Model

Add two system setting keys in `backend/internal/service/domain_constants.go`:

- `subscription_quota_reset_utc_offset_minutes`
- `subscription_quota_reset_hour`

Add matching fields to the admin settings model and DTO:

- `SubscriptionQuotaResetUTCOffsetMinutes int`
- `SubscriptionQuotaResetHour int`

The settings service parses missing or invalid values to defaults:

- offset: `0`
- hour: `0`

Validation accepts offsets from `-720` through `840` minutes and requires a 15-minute increment. This covers real UTC offsets from `UTC-12:00` through `UTC+14:00`, including half-hour and quarter-hour zones. Validation accepts hours from `0` through `23`.

Only the admin settings API needs these fields. Public settings do not expose quota reset time.

### Billing Window Calculation

Add a small subscription reset config value in the service layer:

```go
type SubscriptionQuotaResetConfig struct {
	UTCOffsetMinutes int
	ResetHour        int
}
```

Add a helper that normalizes this config to the defaults and bounds above. Gateway billing code reads the config from `SettingService` when building a `UsageBillingCommand`; if `SettingService` is unavailable, it uses the default config.

Add the normalized config to `UsageBillingCommand`. The repository then replaces `startOfUTCDay(now)` and `startOfUTCWeek(now)` with helpers that compute:

- start of the current daily window in the configured fixed offset,
- start of the current weekly window, with Monday as the first day, in the configured fixed offset.

The helpers return UTC timestamps for database storage. With the default config, they return the same values as the current UTC helpers.

## Frontend Design

### Admin Settings

Add two fields under the existing admin settings card that controls the user subscription entry:

- UTC offset selector, such as `UTC+00:00`, `UTC+08:00`, and other valid offsets.
- Reset hour numeric input from `0` to `23`.

The UI copy should make the weekly rule explicit: weekly quota refreshes every Monday at the selected hour. The frontend sends the two numeric fields through `frontend/src/api/admin/settings.ts`.

### Subscription Plan Navigation

Move the subscription plan menu item from the Order Management group to the Subscription Management group in `frontend/src/components/layout/AppSidebar.vue`.

Add a new route:

- `/admin/subscriptions/plans`

This route can continue to render `AdminPaymentPlansView.vue` and keep the existing `requiresPayment` guard because the page still uses payment plan APIs. Replace `/admin/orders/plans` with a redirect to `/admin/subscriptions/plans`.

The existing `nav.paymentPlans` label can remain because it already reads as subscription plans in Chinese.

## Data Flow

1. An admin updates the subscription reset offset and hour in Settings.
2. The admin settings handler validates and persists the two values through `SettingService`.
3. A gateway request that bills a subscription builds a `UsageBillingCommand`.
4. The command carries the current subscription reset config.
5. `UsageBillingRepository.Apply` locks the subscription row, computes current daily and weekly window starts, and resets usage when the stored window start is older.
6. Ledger entries for waste and consumption keep the same shape; only the window boundary changes.

## Error Handling

- Invalid admin settings return the existing settings-validation error response.
- A missing setting key uses the default value and does not fail billing.
- A temporary settings read failure during gateway billing uses the default config and logs the failure through the existing gateway logging path.
- Waste statistics never fail because of a revoked status filter; revoked rows simply disappear from the result set.

## Testing

Backend tests:

- A repository-level waste-stat test proves revoked subscriptions are excluded from all waste-stat SQL paths.
- Subscription window tests prove the default config matches current UTC behavior.
- Subscription window tests prove `UTC+08:00` with reset hour `4` starts the daily window at `20:00 UTC` on the previous calendar day for local times before `04:00`.
- Subscription window tests prove the weekly window starts on Monday at the configured offset and hour.
- Setting service tests prove defaults, valid updates, invalid offsets, and invalid hours.
- Admin settings handler tests prove the new fields round-trip through the admin API.

Frontend checks:

- Type checking proves the settings DTO and form model stay aligned.
- Build verification proves the moved route and sidebar item compile.
- If a local frontend target is available, browser verification confirms the settings fields appear under User Subscription Entry and the plan page appears under Subscription Management.

## Out Of Scope

- Per-plan reset schedules.
- Per-user reset schedules.
- Configurable weekly day. The weekly window stays Monday-based.
- Cron expressions or arbitrary reset minutes.
- Changing how expired subscriptions appear in waste statistics.
- Renaming the payment-plan API or component.

## Self Review

- The design covers all three issue #47 requirements.
- Defaults preserve existing billing behavior.
- The settings API stays admin-only for quota reset time.
- The billing change uses fixed UTC offsets, not time zone names or daylight-saving rules.
- The route change keeps backward compatibility through a redirect.
