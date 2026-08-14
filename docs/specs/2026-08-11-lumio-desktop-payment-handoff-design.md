# Lumio Desktop Payment Handoff Design

Implementation status: the server issue/consume flow, website Cookie bootstrap, and safe-on desktop feature default are implemented. Native Lumio Codex button/browser integration remains in the client phase.

## Goal

Let an authenticated Lumio Codex client open the current account's website payment page without exposing its JWT or API key in a URL and without asking the user to sign in again.

## Scope

This phase adds one server-mediated browser handoff and the minimum website support needed to consume it:

1. An authenticated desktop endpoint issues a short-lived opaque handoff URL.
2. A public browser endpoint atomically consumes that opaque token, creates an HttpOnly website session, and redirects to the configured same-origin payment path.
3. The website can restore the current user from that HttpOnly session when no localStorage token exists.

The desktop UI button, OS browser launch, native credential storage, and payment UI itself remain in their existing client/payment phases. No OAuth provider, device table, or separate user database is introduced.

## API contract

### Issue

`POST /api/v1/desktop/payment-handoff` requires normal JWT authentication. Opaque user access tokens remain rejected by their existing path scope.

Successful response:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "handoff_url": "/api/v1/desktop/payment-handoff/consume?token=dph_<opaque>",
    "expires_in": 60
  }
}
```

The response is a same-origin relative path. It never contains an access JWT, refresh token, gateway API key, user ID, or arbitrary redirect supplied by the caller.

### Consume

`GET /api/v1/desktop/payment-handoff/consume?token=...` is public because the opaque token is the browser's one-time credential. On success it:

1. atomically consumes the token;
2. reloads the token's server-owned user ID and verifies the account is active;
3. rechecks the desktop payment feature/global payment switch;
4. issues a normal access JWT into the host-only `lumio_web_session` cookie;
5. returns `303 See Other` to the configured payment path with `desktop_handoff=1` added as a non-secret bootstrap marker.

The cookie is `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` for TLS or trusted `X-Forwarded-Proto: https` requests. Its lifetime is the configured access-token lifetime. It is an access-only website session; when it expires, the user signs in again. Logout clears it.

Expired, malformed, already-consumed, or unknown handoff tokens return the same `410 DESKTOP_PAYMENT_HANDOFF_INVALID` response. Disabled payment handoff and storage failures return a service-unavailable response without setting a cookie.

## Token storage and one-time semantics

The raw token is `dph_` plus 32 cryptographically random bytes encoded as unpadded base64url. The service computes SHA-256 and passes only the 64-character hex digest to the store.

Redis stores:

```text
desktop_payment_handoff:<sha256(raw token)> -> {"user_id":123}
TTL = 60 seconds
```

Consumption uses Redis `GETDEL`, so concurrent browser requests have one winner across all server instances. Redis never stores the raw handoff token, a JWT, a refresh token, or an API key.

## Redirect safety

The client cannot submit a redirect target. Issue and consume both use `SettingService.GetLumioDesktopConfig`; consume derives the final target from its normalized `payment_url` and applies the same-origin path validation again as defense in depth. Unsafe data falls back to `/payment`.

The frontend adds `/payment` as an alias for the existing `/purchase` payment view. Any `redirect`, `return_to`, or similarly named query parameter on the consume request is ignored. The only server-added query value is `desktop_handoff=1`, which contains no credential and is removed by the router after session bootstrap.

The router handles that marker on any authenticated destination route, not only routes tagged `requiresPayment`, because the server-owned `payment_url` may select another same-origin protected payment page.

## Website session bootstrap

JWT middleware continues to prefer an explicit `Authorization: Bearer` header. If the header is absent, it accepts `lumio_web_session` as the JWT source and performs the same signature, expiry, token-version, account-status, and optional IP/UA binding checks.

The router handles two cases before rejecting a protected route:

- With `desktop_handoff=1`, it clears any previous localStorage credentials first, probes `/auth/me` using cookies, then removes the marker. This prevents an older website login for another account from overriding the desktop-selected account.
- Without the marker and without local credentials, it probes the cookie session once for the requested protected navigation. This keeps a reloaded `/payment` page signed in without making HttpOnly data readable to JavaScript.

The probe uses a small credentialed request path that does not invoke the global 401 redirect interceptor. Successful bootstrap stores only the public current-user DTO in memory; it never copies the cookie JWT into localStorage. The existing Axios client already sends cookies on same-origin requests.

## Feature gates and failure behavior

`feature_flags.payment_handoff` now defaults on because both backend and frontend support are present. A persisted desktop override can disable it, and the effective value remains false whenever the global payment switch is off. Both issue and consume re-evaluate that effective flag, so an operator can stop new and outstanding handoffs without a client release.

Redis errors fail closed. No token is issued when storage cannot guarantee one-time semantics. A token is consumed before creating the website session; if the user was disabled or payment was switched off meanwhile, that token cannot be retried.

## Alternatives considered

- Put the current JWT in a payment URL: rejected because URLs leak through history, logs, referrers, screenshots, and support tooling.
- Store a plaintext one-time token in Redis: simpler lookup, but unnecessary secret retention. Hash-keyed storage is selected.
- Set access and refresh JWT cookies: gives a long-running browser login but widens cookie refresh and CSRF scope. An access-only cookie is sufficient for payment and keeps the first release smaller.
- Add a new server-side general session system: strong revocation control, but duplicates the existing JWT/token-version model and adds user-session lifecycle work. Reusing the existing JWT validator is selected.
- Return an arbitrary `redirect` supplied by the desktop: rejected as an open-redirect surface. The server-owned typed desktop config is selected.

## Verification

- Service tests cover hash-only storage, 60-second TTL, single consumption, expired/reused tokens, disabled payment, inactive/wrong-account protection, and unsafe redirect fallback.
- Redis tests cover TTL and atomic `GETDEL` behavior without storing the raw token.
- Handler tests cover JWT-authenticated issue, `303`, cookie attributes, no JWT/API key in the URL, and ignored hostile redirect parameters.
- Middleware tests cover cookie fallback, Authorization precedence, expiry, token-version checks, and logout cookie clearing.
- Frontend tests cover forced account replacement on the bootstrap marker, cookie-only `/auth/me` restore, marker removal, `/payment` route resolution, and cookie-session logout.
- Full backend unit/integration/vet/lint and frontend typecheck/test/build gates run before handoff.
- Integration-tag tests and compilation still run without Docker, but the repository's container-backed database/Redis harness skips when Docker is unavailable; miniredis separately covers Redis TTL and atomic single consumption.
