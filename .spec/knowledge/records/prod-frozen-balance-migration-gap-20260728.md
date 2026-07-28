---
name: prod-frozen-balance-migration-gap-20260728
description: 2026-07-28 生产发布 74cf5de 引入 users.frozen_balance Ent 字段但未附带迁移，登录与余额接口 503 全站不可用；手动 ALTER 恢复后补 920 与 CI 护栏。
metadata:
  type: record
  date: 2026-07-28
  status: 归档
---

# Production outage: `users.frozen_balance` missing migration (2026-07-28)

## Symptom

After deploy of image `sha-74cf5dedbe8e3f048fe7ef73cbfb4e46b58e9f3e` (publish merge of webhook-only notify, PR #261), all user-balance-related APIs including `/api/v1/auth/login` returned **503** with:

```text
pq: column users.frozen_balance does not exist
```

Reported call sites (Ent `SELECT` scanning full User rows):

- `internal/service/auth_service.go` (login path via `userRepo.GetByEmail`)
- `internal/service/billing_cache_service.go` (balance eligibility / cache load)

Last successful production migration recorded: `919_webhook_only_drop_websocket.sql` (2026-07-28 16:03:02). That file adds webhook sub-toggles and **drops** `websocket_*` columns; it does **not** add `frozen_balance`.

## Immediate mitigation (ops)

Manual production DDL (already applied before this fix landed in git):

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS frozen_balance numeric(20,8) NOT NULL DEFAULT 0;
```

Service recovered while still running image `sha-74cf5de…`. Manual DDL is **not** recorded in `schema_migrations` and has **no** app-managed checksum until a forward migration runs.

## Root cause

### What shipped in 74cf5de / 8de4569c8

Commit `8de4569c8` (`feat(user): Webhook-only notify`) correctly added:

- `backend/migrations/919_webhook_only_drop_websocket.sql` for webhook/websocket column changes
- Ent schema edits to remove `websocket_*` and add `webhook_site_message_notify_enabled` / `webhook_announcement_notify_enabled`

It also **introduced** `field.Float("frozen_balance")` on `backend/ent/schema/user.go` and regenerated ent code so every User query selects `users.frozen_balance`.

There was **no** migration for `frozen_balance` in that commit or in the published image’s embedded `backend/migrations/*.sql` set.

### Why the field appeared without product intent

`frozen_balance` is **not** part of the webhook-only feature. It comes from upstream batch-image hold accounting (origin/main / origin/dev):

| Lineage | Migration | Notes |
|---------|-----------|--------|
| `origin/dev` / `origin/main` | `160_add_user_frozen_balance.sql` | `ADD COLUMN IF NOT EXISTS frozen_balance DECIMAL(20,8) NOT NULL DEFAULT 0` |
| publish image at outage | *(missing)* | publish never applied 160; 9xx notify work ran on a thinner schema history |

`frozen_balance` had been on `origin/dev`’s ent schema since the batch-image foundation merge (`b122a76f5`, 2026-07-15). The webhook-only work regenerated / merged ent schema while keeping that field, but the **release/publish path** did not carry `160_…sql` (and did not add an equivalent 9xx migration).

### Why CI / deploy did not catch it

1. **Migrations are embedded** (`//go:embed *.sql`) and applied at process start. Unit/integration tests that do not assert “ent fields ⊆ migration SQL” pass without the column existing in prod.
2. **`lumio-production.yml`** builds the image and deploys; it does **not** dry-run migrations against a production-like schema snapshot, nor compare ent schema to migrations.
3. **Publish ⊆ partial feature promotion**: webhook PRs were promoted via `release/webhook-only-to-publish-*` rather than a full `dev → publish` snapshot. Schema fields from other topics (batch image / 160) can appear in generated ent if schema was shared, while their migrations stay only on `dev`.
4. No automated check for “new column referenced by code / ent must have a migration in the same change set.”

## Fix in repository

1. **`backend/migrations/920_add_user_frozen_balance.sql`**  
   Idempotent `ADD COLUMN IF NOT EXISTS` matching the manual hotfix and `160_…`.  
   - Fresh full-dev installs: 160 applies first; 920 is a no-op and still gets a `schema_migrations` row.  
   - Publish / prod after manual ALTER: 920 is a no-op and **records checksum**.  
   - Publish without manual ALTER: 920 creates the column.

2. **`backend/migrations/user_schema_columns_migration_test.go`**  
   CI guard: every `field.X("name")` in `ent/schema/user.go` must appear in some embedded `*.sql`.  
   Plus assertions that 160 and 920 both cover `frozen_balance` with `IF NOT EXISTS`.

3. **Process / docs** (see `operations/deployment.md` and this record):  
   - Prefer promoting a **full dev snapshot** for schema-heavy releases; avoid accidental ent field bleed from unrelated topics.  
   - Destructive migrations (DROP COLUMN) should lag code removal by several release cycles.  
   - Recommended follow-up: deploy dry-run migrate on a prod schema clone; expand column-coverage tests beyond `users`.

## Other fields in the same release (74cf5de audit)

Relative to parent of the webhook-only merge, **new DB-facing user columns** were:

| Column | Migration in release | Status |
|--------|----------------------|--------|
| `webhook_site_message_notify_enabled` | 919 | OK |
| `webhook_announcement_notify_enabled` | 919 | OK |
| `frozen_balance` | **missing** (should have been 160 or 920) | **Outage** |
| `signup_source` allows `dingtalk` (Go validate only) | DB check constraint update lives on dev as `136_add_dingtalk_provider_type.sql` | Not the 503 cause; only bites if someone signs up with dingtalk before 136 is applied |

Dropped columns (`websocket_*`) were covered by 919. No other **new** user columns in that diff lacked a migration besides `frozen_balance`.

Affiliate `aff_frozen_quota` / ledger `frozen_until` are **unrelated** (migrations 133/134) and were not involved.

## Destructive migration note (919)

`919_webhook_only_drop_websocket.sql` **drops** five `websocket_*` columns. Forward-only runner means:

- Rolling **code** back to a build that still SELECTs those columns will fail once 919 has run.
- Best practice: ship “stop reading/writing column” first; drop columns only after several production cycles (or keep columns unused).

Treat this as a process reminder for future drop migrations, not a change to 919 (already applied; immutability rule).

## Verification checklist (post-deploy of 920)

```sql
-- column present
SELECT column_name, data_type, column_default, is_nullable
FROM information_schema.columns
WHERE table_name = 'users' AND column_name = 'frozen_balance';

-- migration recorded
SELECT filename, checksum, applied_at
FROM schema_migrations
WHERE filename IN (
  '160_add_user_frozen_balance.sql',
  '920_add_user_frozen_balance.sql',
  '919_webhook_only_drop_websocket.sql'
)
ORDER BY filename;
```

App checks: login + any balance endpoint should return 200 (not 503).

## Guardrails going forward

| Guard | Owner |
|-------|--------|
| Never ship ent schema column without a same-PR (or already-applied) SQL migration | Authors / review |
| `TestUserEntFieldsCoveredByMigrations` must stay green in CI | backend-ci |
| Prefer full `dev → publish` for mixed schema; selective topic release must include **all** migrations for every ent field in the built binary | release |
| Destructive DROP COLUMN: lag several releases after code stops using the column | release / ops |
| Optional: pre-deploy migrate dry-run against anonymized prod clone | ops / CI enhancement |

## Related

- Migrations README immutability: `backend/migrations/README.md`
- Prior checksum incident: [`bugwall-zeabur-migration-checksum-20260423.md`](./bugwall-zeabur-migration-checksum-20260423.md)
- Deploy overview: [`../operations/deployment.md`](../operations/deployment.md)
