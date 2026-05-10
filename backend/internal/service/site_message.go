package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SiteMessagesEnabledDefault        = false
	SiteMessagesDailySendLimitDefault = 10
	SiteMessagesRetentionDaysDefault  = 30
	SiteMessagesDailySendLimitMax     = 1000
	SiteMessagesRetentionDaysMin      = 1
	SiteMessagesRetentionDaysMax      = 365
)

var (
	ErrSiteMessageNotFound           = domain.ErrSiteMessageNotFound
	ErrSiteMessagesDisabled          = domain.ErrSiteMessagesDisabled
	ErrSiteMessageRecipientNotFound  = domain.ErrSiteMessageRecipientNotFound
	ErrSiteMessageDailyLimitExceeded = domain.ErrSiteMessageDailyLimitExceeded
	ErrSiteMessageInvalidSubject     = domain.ErrSiteMessageInvalidSubject
	ErrSiteMessageContentRequired    = domain.ErrSiteMessageContentRequired
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
}

type SiteMessageSettings struct {
	Enabled        bool
	DailySendLimit int
	RetentionDays  int
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

type SiteMessageEmailSender interface {
	EnqueueSiteMessage(email, subject, content string) error
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
