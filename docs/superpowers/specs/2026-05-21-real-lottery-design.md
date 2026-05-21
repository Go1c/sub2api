# Real Lottery Campaign Design

## Goal

Replace the frontend-only lottery mock with a backend-backed lottery system. When one active campaign has remaining participant slots, every authenticated user sees a login popup until the user draws or closes the popup for the current browser session. Winners receive their redeem code through site messages, and the wheel result tells them to open site messages to claim it.

## Current State

The current lottery code lives only in the frontend:

- `frontend/src/stores/lottery.ts` stores campaigns, draw state, and winners in `localStorage`.
- `frontend/src/components/lottery/LotteryPromptManager.vue` reads that local state and opens the popup.
- `frontend/src/views/admin/LotteryView.vue` creates a mock campaign in the same browser.
- No backend route, repository, migration, or API file exists for lottery.

This makes the admin-created campaign visible only in the admin's browser. Other users cannot see it.

## Product Rules

- The system supports at most one active campaign.
- An active campaign appears to users only while `joined_count < max_participants`.
- A logged-in user sees the popup when:
  - one campaign is active,
  - the campaign has remaining participant slots,
  - the user has not drawn in that campaign,
  - the user has not dismissed that campaign in the current browser session.
- Closing the popup stores a session-only dismissal in `sessionStorage`; it does not create a backend record.
- Drawing creates a backend record. After a draw, the user never sees that campaign again from any browser.
- A winner receives a site message that contains the redeem code.
- The result popup does not show the redeem code. It says the code was sent by site message.
- If all participant slots are used, the backend finishes the campaign.
- If all prize codes are used before all participant slots are used, the backend can keep the campaign active until slots run out, but all later draws lose.
- Creating a new campaign finishes any existing active campaign.
- Creating or activating a campaign requires site messages to be enabled, because winning delivery depends on them.

## Data Model

Add three Ent schemas and one SQL migration.

### `lottery_campaigns`

Fields:

- `id bigint primary key`
- `name varchar(120) not null`
- `subtitle varchar(240) not null`
- `status varchar(20) not null` with values `active` and `finished`
- `prize_count int not null`
- `max_participants int not null`
- `joined_count int not null default 0`
- `winner_count int not null default 0`
- `created_by bigint not null`
- `created_at timestamptz not null`
- `updated_at timestamptz not null`
- `finished_at timestamptz null`

Indexes:

- `status`
- `created_at desc`

Application invariant:

- Only one campaign can have `status = active`.
- The service enforces this by updating existing active campaigns to `finished` before creating a new active campaign.

### `lottery_codes`

Fields:

- `id bigint primary key`
- `campaign_id bigint not null`
- `code varchar(128) not null`
- `assigned_user_id bigint null`
- `assigned_draw_id bigint null`
- `assigned_at timestamptz null`
- `created_at timestamptz not null`
- `updated_at timestamptz not null`

Indexes:

- `(campaign_id, code)` unique
- `(campaign_id, assigned_at)`

### `lottery_draws`

Fields:

- `id bigint primary key`
- `campaign_id bigint not null`
- `user_id bigint not null`
- `won boolean not null`
- `lottery_code_id bigint null`
- `site_message_id bigint null`
- `result_label varchar(80) not null`
- `created_at timestamptz not null`

Indexes:

- `(campaign_id, user_id)` unique
- `(campaign_id, created_at)`
- `(user_id, created_at)`

This unique index prevents duplicate draws across devices and concurrent tabs.

## Backend Services

Create `LotteryService` with these responsibilities:

- Admin lifecycle:
  - `CreateCampaign(ctx, adminID, input)` validates input, verifies site messages are enabled, finishes old active campaigns, creates the campaign, and creates code rows.
  - `ListCampaigns(ctx, params)` returns campaigns with counts.
  - `GetCampaign(ctx, id)` returns one campaign with code and winner details.
  - `FinishCampaign(ctx, adminID, id)` marks a campaign finished.
- User flow:
  - `GetActiveForUser(ctx, userID)` returns a popup payload when a campaign is eligible for that user.
  - `Draw(ctx, userID, campaignID)` performs the draw in a transaction.

### Draw Algorithm

`Draw` must run in a database transaction.

Steps:

1. Lock the campaign row with `FOR UPDATE`.
2. Reject if the campaign is not active.
3. Reject if `joined_count >= max_participants`.
4. Query for an existing draw by `(campaign_id, user_id)`. If it exists, return an already-drawn response.
5. Compute:
   - `remainingPrizes = prize_count - winner_count`
   - `remainingSlots = max_participants - joined_count`
   - `winProbability = remainingPrizes / remainingSlots`
6. Randomly decide the outcome.
7. If the user wins, select one unassigned code for the campaign with `FOR UPDATE SKIP LOCKED`, assign it to the user, and create a site message.
8. Create the draw row.
9. Increment `joined_count`; increment `winner_count` only for wins.
10. If `joined_count >= max_participants`, mark the campaign finished.
11. Commit and return the draw result.

If the random result wins but no code remains, treat the draw as a loss. This guards against corrupted campaign data and keeps the transaction safe.

### Site Message Delivery

The lottery service should use `SiteMessageService.SystemSendToUser` or an equivalent internal method that:

- bypasses regular user's daily send limits,
- validates that site messages are enabled,
- creates a message from an admin or system sender,
- returns the created `site_message_id`.

The message copy:

- Subject: `恭喜中奖：{活动名}`
- Content:
  - `你在「{活动名}」中中奖。`
  - `兑换码：{code}`
  - `请复制该兑换码前往兑换页面使用。`

If message creation fails, the draw transaction rolls back. A winner should never be recorded without a deliverable code message.

## Backend API

### User API

`GET /api/v1/lottery/active`

Returns:

```json
{
  "campaign": {
    "id": 1,
    "name": "五月幸运转盘",
    "subtitle": "登录就有机会，转一转赢取兑换码",
    "prize_count": 5,
    "max_participants": 20,
    "joined_count": 3,
    "segments": [
      { "label": "奖品 1", "is_prize": true },
      { "label": "谢谢参与", "is_prize": false }
    ]
  }
}
```

When no popup should appear:

```json
{ "campaign": null }
```

`POST /api/v1/lottery/:id/draw`

Returns a win:

```json
{
  "won": true,
  "index": 0,
  "label": "奖品 1",
  "message": "恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。",
  "site_message_id": 123
}
```

Returns a loss:

```json
{
  "won": false,
  "index": 3,
  "label": "谢谢参与",
  "message": "很遗憾，这次没有中奖。"
}
```

### Admin API

`GET /api/v1/admin/lottery/campaigns`

Lists campaigns, newest first.

`POST /api/v1/admin/lottery/campaigns`

Request:

```json
{
  "name": "五月幸运转盘",
  "subtitle": "登录就有机会，转一转赢取兑换码",
  "prize_count": 5,
  "max_participants": 20,
  "codes": ["CODE-1", "CODE-2", "CODE-3", "CODE-4", "CODE-5"]
}
```

Validation:

- `name` is required and at most 120 runes.
- `subtitle` defaults to `登录就有机会，转一转赢取兑换码`.
- `prize_count >= 1`.
- `max_participants >= prize_count`.
- `len(codes) >= prize_count`.
- codes are trimmed, non-empty, and unique within the request.
- site messages must be enabled.

`GET /api/v1/admin/lottery/campaigns/:id`

Returns campaign details, codes, winners, and draw counts.

`POST /api/v1/admin/lottery/campaigns/:id/finish`

Marks a campaign finished.

## Frontend Changes

### API Layer

Add:

- `frontend/src/api/lottery.ts`
- `frontend/src/api/admin/lottery.ts`

These files map the user and admin endpoints.

### Store

Rewrite `frontend/src/stores/lottery.ts` as an API-backed store.

State:

- `activeCampaign`
- `loadingActive`
- `drawing`
- `lastResult`

Actions:

- `fetchActive()`
- `draw(campaignId)`
- `clearActive()`
- admin actions for create, list, detail, and finish

The store no longer writes campaigns or draw results to `localStorage`.

### Popup Manager

`LotteryPromptManager.vue` should:

- watch authentication and user ID,
- fetch active campaign after login,
- check `sessionStorage` dismissal key `lottery_dismissed_v2`,
- open the dialog only when the backend returns a campaign and the session key does not include that campaign ID,
- write dismissal only on close before draw,
- clear active campaign after a completed draw.

### Dialog

`LotteryDialog.vue` should keep the wheel animation. It should change the winner result panel:

- Do not render the code.
- Show `恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。`
- Provide a button to open `/site-messages`.

Loss copy remains short and neutral.

### Admin Page

`LotteryView.vue` should:

- call the admin API to create campaigns,
- load campaign history from backend,
- show active/finished status,
- show participants and winners,
- show unassigned codes and winner records in the details panel,
- disable create and show a clear error if site messages are disabled.

## Error Handling

User-facing errors:

- No active campaign: `campaign: null`, not an error.
- Already drawn: HTTP 409 with code `LOTTERY_ALREADY_DRAWN`; frontend closes the popup.
- Campaign full or finished: HTTP 409 with code `LOTTERY_CAMPAIGN_CLOSED`; frontend closes the popup and shows a toast.
- Site messages disabled during create: HTTP 400 with code `LOTTERY_SITE_MESSAGES_DISABLED`.

Backend logs should include campaign ID and user ID for draw failures, but never log redeem code values except in debug-local tests.

## Testing

Backend tests:

- Creating a campaign finishes the previous active campaign.
- Creating a campaign rejects duplicate codes.
- Creating a campaign rejects disabled site messages.
- `GetActiveForUser` returns nil after a user draws.
- `Draw` records one draw per user.
- `Draw` sends a site message and stores `site_message_id` on win.
- Concurrent draw attempts for the same user produce one draw.
- Full campaigns return no active campaign.

Frontend tests:

- Popup fetches the active campaign after login.
- Session dismissal hides the popup only for the current session.
- Draw win shows the site-message tip and link.
- Draw loss shows the loss copy.
- Admin create calls the backend API and reloads history.

## Out Of Scope

- Weighted prizes.
- Multiple active campaigns.
- Public non-login lottery.
- Email delivery for lottery prizes.
- Auto-redeeming the code for the user.
- Persisting close/dismiss state across devices.

## Self Review

- No placeholder requirements remain.
- The design uses backend persistence instead of frontend storage.
- The design preserves one active campaign.
- The draw transaction covers duplicate draws, participant limits, code assignment, and site message delivery.
- The frontend popup behavior matches the product rules.
