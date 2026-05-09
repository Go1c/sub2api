# support-gateway

Small Go service that exposes a browser-safe support chat API and forwards requests to DocsGPT Agent API.

## Endpoints

- `GET /healthz`
- `GET /widget-config?locale=zh-CN|zh-Hant|en-US`
- `POST /chat/stream`

`POST /chat/stream` accepts:

```json
{
  "message": "How do I recharge?",
  "conversationId": "optional-docsgpt-conversation-id",
  "locale": "en-US",
  "user": { "id": "optional-user-id", "email": "optional-user-email" }
}
```

The gateway injects `DOCSGPT_AGENT_API_KEY` into the upstream DocsGPT `/stream` request and never returns it to the browser.

## Design

```text
Browser SupportChatWidget
  -> support-gateway /chat/stream
  -> DocsGPT /stream
  -> model endpoint configured inside DocsGPT
```

The frontend sends the current UI locale on every request. The gateway maps supported locales before forwarding them through `passthrough`:

- `en-US` -> `English`
- `zh-CN` -> `Simplified Chinese`
- `zh-Hant` -> `Traditional Chinese`

The same locale can be passed to `/widget-config?locale=...` for localized default title, welcome message, and contact text.

## Configuration

Copy `.env.example` into your deployment platform variables. Required variables:

- `DOCSGPT_API_BASE_URL`
- `DOCSGPT_AGENT_API_KEY`
- `ALLOWED_ORIGINS`

Optional variables:

- `SUPPORT_EMAIL`
- `SUPPORT_URL`
- `WIDGET_TITLE`
- `WELCOME_MESSAGE`
- `OFFICIAL_CONTACT_TEXT`
- `RATE_LIMIT_WINDOW_SECONDS`
- `RATE_LIMIT_MAX_REQUESTS`

When widget text overrides are omitted, `/widget-config` localizes default text by `locale`.

## Local Verification

```bash
go test ./...
go run .
```

## Key Files

- `main.go`: process entrypoint and `PORT` binding.
- `config.go`: environment variable parsing and validation.
- `server.go`: CORS, rate limiting, public widget config, DocsGPT request building, SSE forwarding.
- `server_test.go`: regression coverage for secret isolation, locale passthrough, CORS, SSE forwarding, and rate limiting.
