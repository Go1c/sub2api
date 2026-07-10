---
status: in_progress
---

# Sync All Upstream Grok Changes — 2026-07-10

## Task 1 — Inventory the complete Grok change chain

- Scope: `origin/main` Grok/xAI commits not already represented in `origin/dev`, including OAuth, quota, scheduler, CLI compatibility, media routing, pricing/billing, Grok 4.5, and later correctness fixes.
- Acceptance:
  - Every Grok-related upstream PR/mainline follow-up is accounted for.
  - Non-Grok changes are excluded unless they are required to compile or preserve Grok behavior.
- Dependency: fork `main` synchronized to `upstream/main`.

## Task 2 — Port the Grok backend subsystem

- Scope: domain/constants, xAI client, OAuth/token refresh, quota probing, scheduler/gateway routing, media endpoints, usage and billing, migrations, wiring, admin APIs, and tests.
- Acceptance:
  - Grok accounts can be created/refreshed and scheduled.
  - OpenAI/Responses/Messages compatibility routes work for Grok.
  - Image/video generation routes and usage billing match the latest upstream behavior.
  - Existing fork payment/subscription/OpenAI customizations are preserved.
- Dependency: Task 1.

## Task 3 — Port the Grok frontend/admin integration

- Scope: Grok OAuth flows, account forms, quota display, platform badges/icons/colors, group/media pricing controls, model whitelist/mapping, and i18n.
- Acceptance:
  - Grok can be configured through the existing admin UI.
  - Frontend types and backend contracts agree.
  - Existing LumioAPI views and settings remain intact.
- Dependency: Task 2.

## Task 4 — Validate and merge to dev

- Scope: focused Grok tests plus repository-required backend/frontend gates; PR into `dev`.
- Acceptance:
  - `git diff --check` passes.
  - Backend unit/integration tests and `go vet -tags integration ./...` pass; lint/build gates pass where available.
  - Frontend typecheck/build pass.
  - PR is merged into `dev`; `publish` is not modified.
- Dependency: Tasks 2 and 3.
