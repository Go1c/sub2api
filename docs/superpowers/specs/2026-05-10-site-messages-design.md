# Site Messages Design

## Goal

Add an internal site-message feature that behaves like lightweight email: users can receive, send, read, and reply to messages inside Sub2API. Administrators can enable or disable the feature globally and can send a message to a selected user from the admin user-management table.

## Requirements

- Users see a new "站内信" entry in the account/sidebar navigation when the feature is enabled.
- The navigation entry shows a red unread indicator when the current user has unread messages.
- Users can view inbox and sent messages.
- Users can open a message, see its content and sender/recipient metadata, and reply.
- Users can send a new message to another user by entering a full email address or numeric user ID. Regular users cannot fuzzy-search or enumerate users.
- Messages have read/unread state per recipient. Opening a received message marks it read.
- Administrators can enable or disable site messages in system settings.
- When site messages are disabled, the frontend hides the entry and the backend returns a feature-disabled error from all site-message APIs.
- Administrators can send a site message from the admin users table "更多" menu. The dialog has title and content fields; the recipient is fixed to the selected user.
- Message content is retained for 30 days by default. Expired messages are not returned by list/detail APIs and can be cleaned up by a repository cleanup method.
- Non-admin users can send at most 10 messages per day by default. This daily limit is configurable from admin settings. Administrators are not rate-limited by the daily send limit.

## Non-Goals

- Real-time push or websocket notifications.
- Attachments, rich HTML email rendering, drafts, labels, folders, or bulk selection.
- Multi-recipient messages.
- User fuzzy-search for regular users.
- A full admin message management inbox.

## Settings

Add these DB-backed settings:

- `site_messages_enabled`: boolean, default `false`.
- `site_messages_daily_send_limit`: integer, default `10`.
- `site_messages_retention_days`: integer, default `30`.

`site_messages_enabled` is exposed through public settings so the sidebar and route guard can hide the user UI before the async settings request finishes. It should be registered as an opt-in feature flag.

The daily send limit and retention days are admin/system settings. They are not needed in public settings.

## Data Model

Add an Ent schema backed by `site_messages`.

Fields:

- `id`: generated primary key.
- `sender_id`: required user ID.
- `recipient_id`: required user ID.
- `parent_id`: optional message ID for replies.
- `subject`: string, max 200, required.
- `content`: text, required.
- `read_at`: nullable timestamptz. `null` means unread for the recipient.
- `created_at`: immutable timestamptz.
- `updated_at`: timestamptz.

Edges:

- `sender` from `User`.
- `recipient` from `User`.
- `parent` self-edge.
- `replies` self-edge.

Indexes:

- `recipient_id, created_at` for inbox listing.
- `sender_id, created_at` for sent listing.
- `recipient_id, read_at` for unread counts.
- `parent_id, created_at` for reply-chain loading.
- `created_at` for cleanup.

Retention is based on `created_at`. Messages older than the configured retention window are excluded from normal reads and deleted by cleanup.

## Backend Architecture

Add a `SiteMessageService` with these responsibilities:

- Resolve recipients.
- Enforce feature switch.
- Enforce regular-user daily send limit.
- Create messages and replies.
- List inbox/sent messages with pagination.
- Load a message detail only when the current user is sender or recipient.
- Mark a received message as read.
- Count unread messages for the current user.
- Clean up expired messages.

Repository interface:

- `Create(ctx, message)`.
- `GetVisibleByID(ctx, messageID, userID, retentionCutoff)`.
- `ListInbox(ctx, userID, page, pageSize, retentionCutoff)`.
- `ListSent(ctx, userID, page, pageSize, retentionCutoff)`.
- `ListThread(ctx, rootOrParentID, userID, retentionCutoff)`.
- `MarkRead(ctx, messageID, userID, readAt)`.
- `CountUnread(ctx, userID, retentionCutoff)`.
- `CountSentSince(ctx, userID, since)`.
- `DeleteOlderThan(ctx, cutoff)`.

The service should use the existing setting service for all setting reads. Feature-disabled responses use a stable domain error such as `SITE_MESSAGES_DISABLED`.

## API Design

User APIs:

- `GET /api/v1/site-messages/inbox?page=&page_size=`
- `GET /api/v1/site-messages/sent?page=&page_size=`
- `GET /api/v1/site-messages/unread-count`
- `GET /api/v1/site-messages/:id`
- `POST /api/v1/site-messages`
- `POST /api/v1/site-messages/:id/reply`
- `POST /api/v1/site-messages/:id/read`
- `GET /api/v1/site-messages/recipient/resolve?query=`

Regular recipient resolution accepts only:

- Exact numeric user ID.
- Exact email address, case-insensitive after normal email normalization.

Admin APIs:

- `POST /api/v1/admin/site-messages/users/:id`
- `GET /api/v1/admin/site-messages/recipients?q=`

The user-management "send" flow uses `POST /api/v1/admin/site-messages/users/:id` because the row already fixes the recipient. The admin recipient-search endpoint supports future admin compose flows and uses fuzzy email/username matching.

## Frontend Design

User navigation:

- Add a `站内信` nav item in `AppSidebar.vue`.
- Register `site_messages_enabled` in `featureFlags.ts`.
- Render a small red dot when unread count is greater than zero.
- Fetch unread count after public settings and auth state are available. Refetch when the user reads or sends/replies.

User page:

- Add `frontend/src/views/user/SiteMessagesView.vue`.
- Use tabs for inbox and sent messages.
- Inbox rows show unread state, sender, subject, preview, and created time.
- Sent rows show recipient, subject, preview, and created time.
- Detail view shows subject, sender, recipient, content, read state, created time, and reply action.
- Compose form validates recipient, subject, and content. Recipient input resolves only exact email or user ID.

Admin user-management flow:

- Add a "发送站内信" item to the existing "更多" menu in `UsersView.vue`.
- Open a modal with the selected user's email/ID shown as fixed recipient.
- Submit title and content to `POST /api/v1/admin/site-messages/users/:id`.

Settings page:

- Add a card for site messages.
- Controls:
  - enable/disable switch.
  - daily send limit number input, default 10.
  - retention days number input, default 30.

## Error Handling

- Disabled feature: backend returns a typed feature-disabled error. Frontend shows a concise "站内信功能未开启" message if a stale page calls the API.
- Recipient not found: regular user sees "收件人不存在或不可用" without exposing partial-match data.
- Self-send is allowed unless implementation discovers an existing project rule against it; this keeps admin testing and user note-to-self behavior simple.
- Empty/oversized subject or empty content returns validation errors.
- Daily limit exceeded returns a typed error containing the configured limit.

## Testing

Backend tests:

- Creating a message trims and validates subject/content.
- Regular user recipient resolution only supports exact email or user ID.
- Regular user cannot read another user's message.
- Opening/marking a received message sets `read_at`.
- Unread count ignores read and expired messages.
- Reply preserves `parent_id` and enforces visibility.
- Disabled feature blocks all site-message actions.
- Regular user daily send limit defaults to 10 and is configurable.
- Admin sends are not blocked by the daily send limit.
- Expired messages are excluded and cleanup deletes them.

Frontend tests:

- Sidebar hides site messages when public setting is disabled.
- Sidebar shows a red dot when unread count is nonzero.
- User page sends a message only after exact recipient resolution.
- Detail view marks inbox messages read and allows reply.
- Admin users "更多" menu opens the send-site-message dialog with fixed recipient and submits title/content.

## Rollout Notes

The feature defaults off. Existing deployments receive the new tables and settings, but no user-facing navigation appears until administrators enable site messages.
