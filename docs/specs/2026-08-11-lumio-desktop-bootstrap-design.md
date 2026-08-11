# Lumio Desktop Bootstrap Design

## Goal

Provide the public, remotely managed bootstrap data required by Lumio Codex and make creation of the account-level `Lumio Codex Desktop` gateway key safe under retries and concurrent devices.

## Scope

This phase adds two backend capabilities only:

1. `GET /api/v1/desktop/config`, a public read-only endpoint with an explicit response whitelist.
2. Idempotent reuse or creation of the reserved desktop key through the existing authenticated `POST /api/v1/keys` flow.

Authentication UI, one-time payment handoff, desktop credential storage, Codex configuration takeover, and update downloads remain separate phases.

## Public desktop configuration

The desktop response is deliberately separate from `/api/v1/settings/public`. Reusing that large browser-oriented payload would couple desktop compatibility to unrelated site fields and increase the chance of exposing a future setting accidentally.

The endpoint returns only:

```json
{
  "default_model": "gpt-5.4",
  "payment_url": "/payment",
  "min_client_version": "0.0.0",
  "update_notice": "",
  "feature_flags": {
    "registration": true,
    "payment_handoff": true,
    "key_provisioning": true
  }
}
```

Configuration is stored under one settings key as JSON. A single document keeps related compatibility values atomic and allows the existing admin settings API to read and replace it without adding a new write surface. Parsing starts from built-in defaults, then overlays valid stored fields. Invalid JSON, invalid semantic versions, unsafe payment URLs, blank model names, and overlong notices fall back to safe values.

`payment_url` is restricted to a same-origin absolute path (leading `/`, never `//`, with no scheme or host). `registration` is additionally gated by the existing global registration setting; `payment_handoff` is gated by the existing payment-enabled setting. Therefore an older desktop configuration cannot override the operator's global kill switches.

Responses use an ETag derived from the whitelisted payload and `Cache-Control: public, max-age=300, stale-if-error=86400`. The desktop client remains responsible for persisting its last successful response and for its own embedded fallback.

### Alternatives considered

- Add the fields to `/settings/public`: fewer route changes, but poor compatibility isolation and a growing accidental-disclosure surface.
- Compile the values into the desktop: simplest server, but cannot enforce a minimum version or disable features without a client release.
- Separate endpoint backed by one typed settings document: selected for a narrow public contract, atomic administration, and safe normalization.

## Reserved desktop API key

The client continues to call `POST /api/v1/keys` with only:

```json
{"name":"Lumio Codex Desktop"}
```

The service treats that exact name as a reserved get-or-create request. It first queries the current user's active key by name. If none exists, it follows the normal validated creation path. A PostgreSQL partial unique index guarantees at most one non-deleted reserved row per user. If another device wins the insert race, the losing request resolves the unique conflict by reading and returning the winning row.

Soft-deleted keys do not participate in the unique index, so provisioning after revocation creates a new credential. Existing active duplicates are migrated without deleting or changing their credentials: the oldest row keeps the reserved name and later rows receive deterministic `legacy <id>` display names.

Ordinary key names retain their existing non-idempotent behavior. No device table or per-device key is introduced.

### Alternatives considered

- Client-side list then create: rejected because two devices can race between the operations.
- Process-local lock: rejected because it does not coordinate multiple server instances.
- Partial unique index plus conflict lookup: selected because the database is the shared serialization point and revoked rows remain recoverable.

## Error and security behavior

- Public configuration never returns arbitrary settings or secrets.
- Unsafe persisted desktop configuration degrades to built-in values instead of emitting an unsafe URL or blocking all clients.
- Admin updates with invalid desktop configuration are rejected before persistence.
- Reserved-key conflict recovery is limited to the exact fixed name and current user.
- Migration preserves every existing credential and only changes duplicate display names.
- API key responses follow the existing authenticated DTO; no key is placed in a URL or log.

## Verification

- Unit tests cover defaults, overlay behavior, global kill switches, unsafe persisted values, ETag/304, and the response whitelist.
- Unit tests cover existing reserved-key reuse, first creation, conflict recovery, ordinary-key conflict behavior, and revoked-key recreation semantics.
- Integration tests cover repository lookup and the database uniqueness guarantee.
- Migration tests assert duplicate-preserving rename and the partial unique index predicate.
- Full backend unit, integration, vet, formatting, and lint gates run before handoff.
