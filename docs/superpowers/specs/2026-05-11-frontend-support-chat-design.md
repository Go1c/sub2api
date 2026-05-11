# Frontend AI Support Chat Design

Date: 2026-05-11
Status: Draft approved in conversation, pending final spec review

## Goal

Enable the existing AI support chat bubble on the main `frontend` application so it appears on `https://api.lumio.games/`, not only on `frontend-dashboard`.

The fix should preserve the current backend contract and support-gateway contract:

- public settings from `/api/v1/settings/public`
- gateway config from `/widget-config`
- streamed replies from `/chat/stream`

## Scope

In scope:

- move the existing support chat client and widget behavior into `frontend`
- mount the widget globally so it can appear on home, auth, and console pages
- preserve current per-locale behavior and logged-in user context
- add regression tests in `frontend`

Out of scope:

- redesigning the chat UI
- changing backend setting keys or API payloads
- changing `support-gateway`
- deduplicating chat code between `frontend` and `frontend-dashboard`

## Recommended Approach

Approach A: copy the current support chat implementation from `frontend-dashboard` into `frontend`, then wire it into the main app shell.

Why this approach:

- fastest path to restore expected production behavior
- lowest release risk because it reuses proven logic
- no backend or gateway changes
- clean rollback path if needed

Trade-off:

- support chat code will temporarily exist in two frontends
- shared-module cleanup can be handled later after production is stable

## Design

### 1. New frontend support chat module

Add `frontend` equivalents of the existing dashboard files:

- `frontend/src/api/supportChat.ts`
- `frontend/src/components/support/SupportChatWidget.vue`

Behavior should remain aligned with the current dashboard implementation:

- read public settings first
- treat chat as enabled only when the backend flag is true and the gateway URL is non-empty
- fetch gateway widget config using the current locale
- send logged-in user `id` and `email` when available
- render streamed answers and source links
- allow retry and clear conversation

### 2. Global mounting point

Mount `SupportChatWidget` in `frontend/src/App.vue`.

This is the simplest way to guarantee visibility across:

- marketing pages
- login and auth flows
- user console pages
- admin pages unless future product rules choose to hide it

### 3. i18n integration

Reuse existing `frontend` locale messages for support chat copy. The main frontend already contains the support chat translation keys in:

- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/zh-Hant.ts`
- `frontend/src/i18n/locales/en.ts`

No translation schema changes are required unless test coverage reveals missing keys.

### 4. Auth and settings integration

The new widget should use:

- `useAuthStore()` for logged-in user context
- existing fetch behavior against `/api/v1/settings/public`

No new store state is required. The widget can stay self-contained and fetch its own public settings exactly as the dashboard version does.

### 5. Styling strategy

Do not restyle the component for this release.

The production problem is missing functionality, not visual mismatch. Reusing the current widget styling reduces risk and keeps this release focused. A later follow-up can align the visuals with the main `frontend` brand if desired.

## Data Flow

1. `frontend/src/App.vue` mounts `SupportChatWidget`
2. widget loads `/settings/public`
3. widget checks:
   - `support_chat_enabled === true`
   - `support_chat_gateway_url` resolves to a non-empty absolute URL
4. widget fetches `GET <gateway>/widget-config?locale=<locale>`
5. user opens the bubble and sends a message
6. widget posts to `POST <gateway>/chat/stream`
7. widget renders streamed answer chunks, sources, and conversation id

## Error Handling

If public settings fail:

- fall back to env-based behavior exactly as the dashboard version does

If the chat is disabled or the gateway URL is blank:

- render nothing

If gateway config fails:

- keep the widget available when enabled
- show the existing config error copy

If streaming fails:

- show the existing error message
- keep retry behavior

## Testing

Use TDD in `frontend`.

Add tests that cover:

- widget stays hidden when public settings disable chat
- widget appears and loads config when enabled
- logged-in user context is attached to streamed requests
- active locale is sent to gateway requests

If the component depends on shared icons or frontend-specific rendering details, adapt the tests to `frontend` conventions without changing the external behavior.

## Release Plan

Implementation and release flow:

1. add failing tests in `frontend`
2. implement the copied support chat module in `frontend`
3. run targeted frontend tests
4. commit to `dev`
5. merge `dev` into `publish`
6. push `publish` to trigger the Zeabur deployment

## Risks

- slight visual mismatch between the reused widget and the main frontend
- duplicated support chat code across the two frontends until a later cleanup

These are acceptable for this release because restoring production functionality is the higher priority.

## Success Criteria

- `https://api.lumio.games/` shows the AI support chat bubble when the backend setting is enabled
- the widget uses `https://lumio-ai-support-chat.zeabur.app`
- locale and logged-in user context reach the gateway
- no backend or gateway changes are needed
