package domain

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

var (
	ErrSiteMessageNotFound = infraerrors.NotFound(
		"SITE_MESSAGE_NOT_FOUND",
		"site message not found",
	)
	ErrSiteMessagesDisabled = infraerrors.Forbidden(
		"SITE_MESSAGES_DISABLED",
		"site messages are disabled",
	)
	ErrSiteMessageRecipientNotFound = infraerrors.NotFound(
		"SITE_MESSAGE_RECIPIENT_NOT_FOUND",
		"recipient not found",
	)
	ErrSiteMessageDailyLimitExceeded = infraerrors.Forbidden(
		"SITE_MESSAGE_DAILY_LIMIT_EXCEEDED",
		"daily site message send limit exceeded",
	)
	ErrSiteMessageInvalidSubject = infraerrors.BadRequest(
		"SITE_MESSAGE_SUBJECT_INVALID",
		"site message subject is invalid",
	)
	ErrSiteMessageContentRequired = infraerrors.BadRequest(
		"SITE_MESSAGE_CONTENT_REQUIRED",
		"site message content is required",
	)
)
