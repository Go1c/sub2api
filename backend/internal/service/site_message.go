package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SiteMessagesEnabledDefault        = false
	SiteMessagesDailySendLimitDefault = 10
	SiteMessagesRetentionDaysDefault  = 30
	SiteMessagesDailySendLimitMax     = 1000
	SiteMessagesRetentionDaysMin      = 1
	SiteMessagesRetentionDaysMax      = 365
	SiteMessagesInactiveDaysMax       = 3650
)

var (
	ErrSiteMessageNotFound           = domain.ErrSiteMessageNotFound
	ErrSiteMessagesDisabled          = domain.ErrSiteMessagesDisabled
	ErrSiteMessageRecipientNotFound  = domain.ErrSiteMessageRecipientNotFound
	ErrSiteMessageDailyLimitExceeded = domain.ErrSiteMessageDailyLimitExceeded
	ErrSiteMessageInvalidSubject     = domain.ErrSiteMessageInvalidSubject
	ErrSiteMessageContentRequired    = domain.ErrSiteMessageContentRequired
	ErrSiteMessageInvalidRecipients  = infraerrors.BadRequest("SITE_MESSAGE_RECIPIENTS_INVALID", "site message recipients are invalid")
	ErrSiteMessageNoRecipients       = infraerrors.BadRequest("SITE_MESSAGE_RECIPIENTS_EMPTY", "site message recipients are empty")
	ErrSiteMessageInvalidRedeemCode  = infraerrors.BadRequest("SITE_MESSAGE_REDEEM_CODE_INVALID", "site message redeem code is invalid")
	ErrSiteMessageRedeemCodeShortage = infraerrors.BadRequest("SITE_MESSAGE_REDEEM_CODE_SHORTAGE", "not enough site message redeem codes")
)

type SiteMessage struct {
	ID          int64
	SenderID    int64
	RecipientID int64
	ParentID    *int64
	Subject     string
	Content     string
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Sender      *User
	Recipient   *User
	Replies     []SiteMessage
}

type SiteMessageSettings struct {
	Enabled               bool
	DailySendLimit        int
	RetentionDays         int
	DefaultRecipientEmail string
}

type SiteMessageRecipient struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type SendSiteMessageInput struct {
	SenderID       int64
	RecipientQuery string
	Subject        string
	Content        string
}

type AdminSendSiteMessageInput struct {
	AdminID     int64
	RecipientID int64
	Subject     string
	Content     string
	SendEmail   bool
}

const (
	SiteMessageRecipientModeSelected = "selected"
	SiteMessageRecipientModeAll      = "all"

	SiteMessageCompensationFormatBlock   = "block"
	SiteMessageCompensationFormatCompact = "compact"

	SiteMessageBatchResultSent   = "sent"
	SiteMessageBatchResultFailed = "failed"
)

type AdminSendCompensationBatchInput struct {
	AdminID             int64
	RecipientMode       string
	RecipientEmails     []string
	Subject             string
	Content             string
	CompensationEnabled bool
	CompensationAmount  float64
	CompensationCodes   []string
	CompensationFormat  string
	SendEmail           bool
	InactiveDays        int
}

type SiteMessageCompensationCodeAssignment struct {
	Recipient string
	Code      string
	Status    string
}

type SiteMessageCompensationBatchResult struct {
	Recipient   string
	UserID      int64
	Code        string
	MessageID   int64
	Status      string
	ErrorReason string
	Error       string
}

type SiteMessageCompensationBatch struct {
	ID             string
	Subject        string
	Content        string
	Mode           string
	Audience       string
	RecipientCount int
	SuccessCount   int
	FailedCount    int
	Amount         float64 // batch total of successfully issued compensation, not per-user unit.
	CodeCount      int
	Operator       string
	SentAt         time.Time
	Codes          []SiteMessageCompensationCodeAssignment
	Results        []SiteMessageCompensationBatchResult
	MessageIDs     []int64
}

type SiteMessageEmailSender interface {
	EnqueueSiteMessage(email, subject, content string) error
}

type SiteMessageRedeemCodeReader interface {
	GetByCode(ctx context.Context, code string) (*RedeemCode, error)
}

type SiteMessagePromoCodeValidator interface {
	ValidatePromoCodeForUser(ctx context.Context, userID int64, code string) (*PromoCode, error)
}

type SiteMessageCompensationBatchRepository interface {
	Create(ctx context.Context, batch *SiteMessageCompensationBatch) error
	List(ctx context.Context, params pagination.PaginationParams) ([]SiteMessageCompensationBatch, *pagination.PaginationResult, error)
}

type SiteMessageRepository interface {
	Create(ctx context.Context, message *SiteMessage) error
	GetVisibleByID(ctx context.Context, messageID, userID int64, retentionCutoff time.Time) (*SiteMessage, error)
	ListInbox(ctx context.Context, userID int64, params pagination.PaginationParams, retentionCutoff time.Time) ([]SiteMessage, *pagination.PaginationResult, error)
	ListSent(ctx context.Context, userID int64, params pagination.PaginationParams, retentionCutoff time.Time) ([]SiteMessage, *pagination.PaginationResult, error)
	MarkRead(ctx context.Context, messageID, userID int64, readAt time.Time) error
	CountUnread(ctx context.Context, userID int64, retentionCutoff time.Time) (int64, error)
	CountSentSince(ctx context.Context, userID int64, since time.Time) (int64, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type SiteMessageUserRepository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error)
}

type SiteMessageSettingsReader interface {
	GetSiteMessageSettings(ctx context.Context) (SiteMessageSettings, error)
}
