# Zeabur migration checksum mismatch during publish deploy

Date: 2026-04-23

## Symptom

Zeabur repeatedly restarted the `sub2api` container. Runtime logs showed:

```text
Failed to initialize application: migration 110_pending_auth_and_provider_default_grants.sql checksum mismatch
db=301e90405b3424967b7d1931568b7a244902148fa82802f362c115ae4e2ae2ef
file=57a196a9810fb478fa001dfff110f5c76a7d87fb04f15e12e513fcb75402d7a6
```

After restoring migration `110`, the next deployment progressed further and stopped on another modified migration:

```text
Failed to initialize application: migration 112_add_payment_order_provider_key_snapshot.sql checksum mismatch
db=d4476c67ceea871aa2d92ee2a603795a742d0379a58cf53938bb9aa559ff9caa
file=ab871fc02da1eabe0de6ca74a119ee3cea9c727caed30af2ae07a0cd1176d1b8
```

The `/app/data/logs/sub2api.log: permission denied` messages were secondary; the startup blocker was the migration checksum mismatch.

## Root Cause

`backend/migrations/110_pending_auth_and_provider_default_grants.sql` was modified after it had already been applied to production. The applied version used `auth_source_default_*_grant_on_signup = true`, while the deployed `publish` version changed those defaults to `false`.

This was part of a broader issue: several already-published migrations had been edited over time. The compatibility allowlist existed, but some entries used hashes that did not match the migration runner's actual `strings.TrimSpace(...)` SHA256 calculation, so production still rejected known historical versions.

The migration runner trims file content before hashing, so the relevant checksums were:

- Applied database checksum: `301e90405b3424967b7d1931568b7a244902148fa82802f362c115ae4e2ae2ef`
- Modified deployed file checksum: `57a196a9810fb478fa001dfff110f5c76a7d87fb04f15e12e513fcb75402d7a6`

## Fix

Restore migration `110` to the exact already-applied content. Do not update `schema_migrations` manually.

If the desired runtime behavior is to move untouched signup-grant defaults from `true` to `false`, implement that as a later forward-only migration. In `publish`, `123_fix_legacy_auth_source_grant_on_signup_defaults.sql` already performs that follow-up safely.

For the remaining already-edited migrations, keep compatibility narrowly scoped to exact known filename/checksum pairs. Do not add a broad bypass. The compatibility set must use the same trimmed-content checksum algorithm as the migration runner.

## Guardrail

Never edit a migration after it has been applied to any persistent environment. Add a new migration for behavior or data changes instead.
