# Support Chat Gateway Fix and Brand Refresh Design

**Date:** 2026-05-11

## Goal

Fix the production support-chat failure for logged-in users and restyle the widget so it visually matches the homepage's blue-indigo-purple brand system instead of the older teal primary palette.

## Root Cause

The support gateway accepts anonymous chat requests and string user identifiers, but rejects logged-in requests when `user.id` is sent as a number. The current frontend forwards `auth.user.id` as-is, so authenticated users hit `POST /chat/stream -> 400 invalid json body` and the widget falls back to the generic unavailable state.

### Evidence

- `GET https://lumio-ai-support-chat.zeabur.app/widget-config?locale=zh-CN` returns `200`.
- `POST /chat/stream` with `{ "message": "hello", "locale": "en-US" }` returns `200`.
- `POST /chat/stream` with `{ "message": "hello", "locale": "en-US", "user": { "id": 123, "email": "test@example.com" } }` returns `400 invalid json body`.
- The same request with `user.id` changed to `"123"` returns `200`.
- The browser console log provided by the user shows the same `400` on the production bundle.

## Scope

### In Scope

- Normalize support-chat user IDs to strings before sending browser requests to the gateway.
- Preserve anonymous chat behavior.
- Keep existing locale mapping and conversation handling unchanged.
- Refresh the widget's actionable accents to match the homepage's blue-indigo-purple palette.
- Update unit tests around the request payload and visible UI classes.

### Out of Scope

- Changes to the separate `lumio-ai-support-chat` gateway repository.
- Backend settings schema changes.
- Reworking the support-chat interaction model or message layout.

## Design

### Request Normalization

The frontend support-chat payload should keep the existing shape, but `user.id` must be serialized as a string whenever user context is present. This is the smallest safe fix because it changes only the gateway-facing contract without altering application auth types.

### Visual Refresh

The widget should keep its white/glass conversation surfaces for readability, while switching interactive emphasis from teal to the homepage palette already used in `HomeView.vue`:

- toggle bubble: blue-indigo-purple gradient with cool glow
- send button: same gradient family for visual consistency
- user messages: gradient accent instead of solid teal
- source pills and focus states: indigo/purple tinted surfaces
- support links hover/focus: blue-purple brand accents instead of primary teal

### UX Constraints

- Do not reduce contrast in assistant messages or body copy.
- Keep mobile layout and widget width behavior unchanged.
- Preserve the existing retry/error flow so failures remain understandable.

## Files

- `frontend/src/api/supportChat.ts`
- `frontend/src/api/supportChat.spec.ts`
- `frontend/src/components/support/SupportChatWidget.vue`
- `frontend/src/components/support/SupportChatWidget.spec.ts`

## Testing Strategy

- Add an API-level unit test that proves numeric `user.id` values are sent as strings.
- Update the widget spec to assert logged-in requests carry a string ID.
- Add a UI assertion for the new brand classes on the toggle or send button.
- Run targeted Vitest specs, `vue-tsc --noEmit`, and a production build.
