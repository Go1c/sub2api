# User Request Monitoring Design

Date: 2026-05-09
Branch: feature/ops-user-request-monitoring
Base branch: dev

## Goal

Add a risk-center feature that lets administrators monitor one user's future requests and capture the client's original request body for a limited time.

The first target user is `qokhi246487+luwnvj68kmze5@hotmail.com`, but the feature must work for any user selected by email.

## Confirmed Requirements

- Scope: capture only future requests after a monitor is created.
- Captured content: client original request body only.
- Excluded content: response body, upstream response body, upstream forwarded body, and request headers.
- Body handling: save raw body text without redaction.
- Size limit: save at most 256 KiB per request. Mark oversized bodies as truncated.
- Rate limit: each monitor has a per-minute maximum capture count.
- Sampling: after a request passes the per-minute cap gate, apply the monitor's sample rate.
- Duration: monitor runs for a configured duration and then expires automatically.
- Retention: captured records remain after monitor expiry and are deleted after a retention window. Default retention is 7 days.
- Permissions: only site owner if the project has an owner role. This project currently has only `admin` and `user`, so admin-only access is the initial behavior.
- Workflow: branch from `dev`, implement on a feature branch, open a PR, and merge back into `dev`.

## Non-Goals

- Do not backfill historical successful request bodies. They were not stored.
- Do not capture all users through a global debug log.
- Do not capture request headers or authorization tokens.
- Do not capture response bodies in this iteration.
- Do not add a new owner role in this iteration.

## Current System Behavior

Successful request metadata is stored in `usage_logs`. It includes user, API key, account, model, endpoint, token counts, cost, latency, IP address, and user agent. It does not include the request body.

Failed requests can be stored in `ops_error_logs` when ops monitoring is enabled. Those rows may include `request_body`, `request_headers`, error bodies, and upstream error context. That feature is error-oriented and does not provide targeted monitoring for successful requests.

The gateway has an environment-only debug body log (`SUB2API_DEBUG_GATEWAY_BODY`) that writes request snapshots to a file. It is global and not suitable for targeted user monitoring.

## Proposed Data Model

### `ops_user_request_monitors`

Stores one monitor task.

Fields:

- `id BIGSERIAL PRIMARY KEY`
- `user_id BIGINT NOT NULL`
- `target_email VARCHAR(255) NOT NULL`
- `status VARCHAR(20) NOT NULL` with values `active`, `expired`, `stopped`
- `duration_seconds INT NOT NULL`
- `max_captures_per_minute INT NOT NULL`
- `sample_rate_percent INT NOT NULL`
- `retention_days INT NOT NULL DEFAULT 7`
- `created_by BIGINT NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `ends_at TIMESTAMPTZ NOT NULL`
- `stopped_at TIMESTAMPTZ NULL`
- `last_capture_at TIMESTAMPTZ NULL`
- `capture_count BIGINT NOT NULL DEFAULT 0`

Indexes:

- `(user_id, status, starts_at, ends_at)` for hot-path active monitor lookup.
- `(created_at DESC)` for admin lists.
- `(ends_at)` for expiry cleanup.

### `ops_user_request_captures`

Stores captured request bodies.

Fields:

- `id BIGSERIAL PRIMARY KEY`
- `monitor_id BIGINT NOT NULL REFERENCES ops_user_request_monitors(id) ON DELETE CASCADE`
- `user_id BIGINT NOT NULL`
- `api_key_id BIGINT NULL`
- `account_id BIGINT NULL`
- `group_id BIGINT NULL`
- `request_id VARCHAR(64) NULL`
- `model VARCHAR(100) NULL`
- `inbound_endpoint VARCHAR(256) NULL`
- `method VARCHAR(16) NULL`
- `content_type VARCHAR(128) NULL`
- `body TEXT NOT NULL`
- `body_bytes INT NOT NULL`
- `body_truncated BOOLEAN NOT NULL DEFAULT false`
- `sample_rate_percent INT NOT NULL`
- `capture_minute TIMESTAMPTZ NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `expires_at TIMESTAMPTZ NOT NULL`

Indexes:

- `(monitor_id, created_at DESC)` for task detail pages.
- `(user_id, created_at DESC)` for user drill-down.
- `(expires_at)` for cleanup.
- `(request_id)` for correlation with `usage_logs` and `ops_error_logs`.

## Backend Architecture

Add a request monitoring service owned by the ops subsystem.

Responsibilities:

1. Resolve active monitors for a user.
2. Enforce per-minute capture limits.
3. Apply sample rate.
4. Truncate body to 256 KiB.
5. Persist captures asynchronously or with a short timeout so monitoring never blocks gateway responses.
6. Expire monitors and delete old captures.

Hot-path behavior:

- Gateway handlers read the request body as they do today.
- After authentication resolves the user and before forwarding upstream, the gateway calls `CaptureClientRequestIfEnabled` with:
  - user ID
  - API key ID
  - request ID
  - inbound endpoint
  - model when known
  - raw body bytes
  - content type
- The service checks an in-memory short TTL cache for active monitors by `user_id`.
- If a monitor exists, it checks a Redis per-minute counter when Redis is available.
- If Redis is unavailable, it skips capture rather than slowing or failing the user request.
- It then applies sample rate using a random integer from 1 to 100.
- It writes a row to `ops_user_request_captures`.
- Capture write failures are logged but do not affect the API response.

Rate limit semantics:

- The per-minute cap applies per monitor.
- The key format is `ops:user-request-monitor:{monitor_id}:{yyyyMMddHHmm}`.
- The counter TTL is at least 2 minutes.
- The gate runs before sampling, as requested.

Sampling semantics:

- `100` captures every candidate after the minute cap gate.
- `50` captures roughly half of eligible candidates.
- `1` captures roughly one percent.
- `0` is invalid for active monitors.

Expiry and cleanup:

- Listing endpoints compute expired status from `ends_at` even if the background worker has not updated the row.
- A lightweight worker periodically marks active monitors as expired when `ends_at <= now()`.
- The same worker deletes captures with `expires_at <= now()`.

## API Design

All endpoints live under `/api/v1/admin/ops/user-request-monitors` and require admin access.

### Create Monitor

`POST /admin/ops/user-request-monitors`

Request:

```json
{
  "user_id": 123,
  "duration_seconds": 1800,
  "max_captures_per_minute": 10,
  "sample_rate_percent": 100,
  "retention_days": 7
}
```

Rules:

- `user_id` must exist.
- `duration_seconds` must be positive and capped by a safe maximum, such as 24 hours.
- `max_captures_per_minute` must be positive and capped by a safe maximum, such as 120.
- `sample_rate_percent` must be 1 to 100.
- `retention_days` defaults to 7 and must be 1 to 30.
- Only one active monitor per user should exist by default. Creating another active monitor for the same user returns a conflict.

### List Monitors

`GET /admin/ops/user-request-monitors?page=1&page_size=20&status=active&user_query=email`

Returns monitor rows with target email, status, capture count, rate settings, start time, end time, and retention.

### Stop Monitor

`POST /admin/ops/user-request-monitors/:id/stop`

Sets `status=stopped` and `stopped_at=now()`.

### List Captures

`GET /admin/ops/user-request-monitors/:id/captures?page=1&page_size=20`

Returns capture summaries without the body by default.

### Capture Detail

`GET /admin/ops/user-request-monitors/:id/captures/:capture_id`

Returns one capture, including raw body text and truncation metadata.

### Delete Capture

`DELETE /admin/ops/user-request-monitors/:id/captures/:capture_id`

Optional but useful for immediate cleanup of sensitive content.

## Frontend Design

Add a card or tab in the existing ops dashboard for "User Request Monitoring".

Views:

1. Monitor list
   - Shows active and recent monitors.
   - Columns: target email, status, captured count, per-minute cap, sample rate, start time, end time, retention, actions.
   - Actions: view captures, stop active monitor, create monitor.

2. Create monitor dialog
   - Search user by email.
   - Duration input with presets: 5 minutes, 30 minutes, 1 hour, 24 hours.
   - Per-minute capture limit.
   - Sample rate percent.
   - Retention days defaulted to 7.
   - Warning that body is saved raw and unredacted.

3. Capture list
   - Columns: time, request ID, API key, model, endpoint, body bytes, truncated.
   - Row click opens detail.

4. Capture detail
   - Displays raw body in a code viewer.
   - Shows copied metadata.
   - Copy button.
   - Delete button if implemented.

## Security and Privacy

This feature intentionally saves raw request bodies. The UI must warn administrators before creation. The backend must not save request headers in this feature. The capture detail endpoint must remain admin-only.

Since the project does not have a site-owner role, all admins can access the feature. If a site-owner role is added later, the route middleware can be narrowed without changing the data model.

## Testing Strategy

Backend tests:

- Creating a monitor validates user, duration, per-minute limit, sample rate, and retention.
- Creating a second active monitor for the same user returns conflict.
- Active monitor lookup ignores expired or stopped monitors.
- Capture truncates bodies above 256 KiB.
- Rate limit runs before sampling.
- Sample rate 100 always captures after the cap gate.
- Capture write failures do not break gateway requests.
- Expired capture cleanup deletes only expired rows.

Frontend tests:

- Create dialog validates required fields.
- Monitor list renders status and actions.
- Capture detail displays raw body and truncation state.

Integration tests:

- A request by a monitored user creates a capture.
- A request by another user does not create a capture.
- A monitor with `max_captures_per_minute=1` captures at most one row in a minute.

## Rollout Plan

1. Add migrations.
2. Add repository and service methods.
3. Add admin ops API handlers and routes.
4. Wire gateway capture calls at request entry points that already have the raw body.
5. Add frontend API client and ops UI.
6. Add tests.
7. Run backend and frontend test suites.
8. Open a PR from `feature/ops-user-request-monitoring` into `dev`.

## Open Decisions

None. The current design uses all confirmed requirements.
