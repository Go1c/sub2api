package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type SiteMessageService struct {
	repo        SiteMessageRepository
	userRepo    SiteMessageUserRepository
	settings    SiteMessageSettingsReader
	emailSender SiteMessageEmailSender
	now         func() time.Time
}

func NewSiteMessageService(
	repo SiteMessageRepository,
	userRepo SiteMessageUserRepository,
	settings SiteMessageSettingsReader,
	emailSender SiteMessageEmailSender,
) *SiteMessageService {
	return &SiteMessageService{
		repo:        repo,
		userRepo:    userRepo,
		settings:    settings,
		emailSender: emailSender,
		now:         time.Now,
	}
}

func (s *SiteMessageService) Send(ctx context.Context, input SendSiteMessageInput) (*SiteMessage, error) {
	settings, err := s.enabledSettings(ctx)
	if err != nil {
		return nil, err
	}
	sender, err := s.userRepo.GetByID(ctx, input.SenderID)
	if err != nil {
		return nil, err
	}
	recipient, err := s.resolveRecipient(ctx, input.RecipientQuery, false)
	if err != nil {
		return nil, err
	}
	if !sender.IsAdmin() {
		if err := s.checkDailyLimit(ctx, sender.ID, settings); err != nil {
			return nil, err
		}
	}
	return s.create(ctx, sender.ID, recipient.ID, nil, input.Subject, input.Content)
}

func (s *SiteMessageService) AdminSendToUser(ctx context.Context, input AdminSendSiteMessageInput) (*SiteMessage, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, err
	}
	adminUser, err := s.userRepo.GetByID(ctx, input.AdminID)
	if err != nil {
		return nil, err
	}
	if !adminUser.IsAdmin() {
		return nil, ErrSiteMessageNotFound
	}
	recipient, err := s.userRepo.GetByID(ctx, input.RecipientID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrSiteMessageRecipientNotFound
		}
		return nil, err
	}
	message, err := s.create(ctx, input.AdminID, input.RecipientID, nil, input.Subject, input.Content)
	if err != nil {
		return nil, err
	}
	if input.SendEmail && s.emailSender != nil {
		if err := s.emailSender.EnqueueSiteMessage(recipient.Email, message.Subject, message.Content); err != nil {
			return nil, fmt.Errorf("enqueue site message email: %w", err)
		}
	}
	return message, nil
}

func (s *SiteMessageService) Reply(ctx context.Context, senderID, parentID int64, content string) (*SiteMessage, error) {
	settings, err := s.enabledSettings(ctx)
	if err != nil {
		return nil, err
	}
	parent, err := s.Get(ctx, senderID, parentID)
	if err != nil {
		return nil, err
	}
	sender, err := s.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, err
	}
	if !sender.IsAdmin() {
		if err := s.checkDailyLimit(ctx, sender.ID, settings); err != nil {
			return nil, err
		}
	}
	recipientID := parent.SenderID
	if parent.SenderID == senderID {
		recipientID = parent.RecipientID
	}
	replySubject := parent.Subject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}
	return s.create(ctx, senderID, recipientID, &parent.ID, replySubject, content)
}

func (s *SiteMessageService) Get(ctx context.Context, userID, messageID int64) (*SiteMessage, error) {
	settings, err := s.enabledSettings(ctx)
	if err != nil {
		return nil, err
	}
	message, err := s.repo.GetVisibleByID(ctx, messageID, userID, s.retentionCutoff(settings))
	if err != nil {
		return nil, err
	}
	if message.RecipientID == userID && message.ReadAt == nil {
		readAt := s.now()
		if err := s.repo.MarkRead(ctx, messageID, userID, readAt); err != nil {
			return nil, err
		}
		message.ReadAt = &readAt
	}
	return message, nil
}

func (s *SiteMessageService) ListInbox(ctx context.Context, userID int64, params pagination.PaginationParams) ([]SiteMessage, *pagination.PaginationResult, error) {
	settings, err := s.enabledSettings(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.repo.ListInbox(ctx, userID, params, s.retentionCutoff(settings))
}

func (s *SiteMessageService) ListSent(ctx context.Context, userID int64, params pagination.PaginationParams) ([]SiteMessage, *pagination.PaginationResult, error) {
	settings, err := s.enabledSettings(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.repo.ListSent(ctx, userID, params, s.retentionCutoff(settings))
}

func (s *SiteMessageService) MarkRead(ctx context.Context, userID, messageID int64) error {
	if _, err := s.Get(ctx, userID, messageID); err != nil {
		return err
	}
	return nil
}

func (s *SiteMessageService) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	settings, err := s.enabledSettings(ctx)
	if err != nil {
		return 0, err
	}
	return s.repo.CountUnread(ctx, userID, s.retentionCutoff(settings))
}

func (s *SiteMessageService) ResolveRecipient(ctx context.Context, query string, fuzzy bool) (*SiteMessageRecipient, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, err
	}
	return s.resolveRecipient(ctx, query, fuzzy)
}

func (s *SiteMessageService) resolveRecipient(ctx context.Context, query string, fuzzy bool) (*SiteMessageRecipient, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrSiteMessageRecipientNotFound
	}
	if id, err := strconv.ParseInt(query, 10, 64); err == nil && id > 0 {
		user, err := s.userRepo.GetByID(ctx, id)
		return siteMessageRecipientFromUser(user, err)
	}
	if !fuzzy {
		if !strings.Contains(query, "@") {
			return nil, ErrSiteMessageRecipientNotFound
		}
		user, err := s.userRepo.GetByEmail(ctx, query)
		return siteMessageRecipientFromUser(user, err)
	}
	users, _, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20, SortBy: "email", SortOrder: pagination.SortOrderAsc}, UserListFilters{Search: query})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrSiteMessageRecipientNotFound
	}
	return siteMessageRecipientFromUser(&users[0], nil)
}

func (s *SiteMessageService) SearchRecipients(ctx context.Context, query string, limit int) ([]SiteMessageRecipient, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []SiteMessageRecipient{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	users, _, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: limit, SortBy: "email", SortOrder: pagination.SortOrderAsc}, UserListFilters{Search: query})
	if err != nil {
		return nil, err
	}
	out := make([]SiteMessageRecipient, 0, len(users))
	for i := range users {
		out = append(out, SiteMessageRecipient{ID: users[i].ID, Email: users[i].Email, Username: users[i].Username})
	}
	return out, nil
}

func (s *SiteMessageService) CleanupExpired(ctx context.Context) (int64, error) {
	settings, err := s.settings.GetSiteMessageSettings(ctx)
	if err != nil {
		return 0, err
	}
	return s.repo.DeleteOlderThan(ctx, s.retentionCutoff(normalizeSiteMessageSettings(settings)))
}

func (s *SiteMessageService) create(ctx context.Context, senderID, recipientID int64, parentID *int64, subject, content string) (*SiteMessage, error) {
	subject = strings.TrimSpace(subject)
	content = strings.TrimSpace(content)
	if subject == "" || len([]rune(subject)) > 200 {
		return nil, ErrSiteMessageInvalidSubject
	}
	if content == "" {
		return nil, ErrSiteMessageContentRequired
	}
	now := s.now()
	message := &SiteMessage{
		SenderID:    senderID,
		RecipientID: recipientID,
		ParentID:    parentID,
		Subject:     subject,
		Content:     content,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("create site message: %w", err)
	}
	return message, nil
}

func (s *SiteMessageService) enabledSettings(ctx context.Context) (SiteMessageSettings, error) {
	settings, err := s.settings.GetSiteMessageSettings(ctx)
	if err != nil {
		return SiteMessageSettings{}, err
	}
	settings = normalizeSiteMessageSettings(settings)
	if !settings.Enabled {
		return SiteMessageSettings{}, ErrSiteMessagesDisabled
	}
	return settings, nil
}

func (s *SiteMessageService) checkDailyLimit(ctx context.Context, userID int64, settings SiteMessageSettings) error {
	if settings.DailySendLimit <= 0 {
		return nil
	}
	now := s.now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count, err := s.repo.CountSentSince(ctx, userID, since)
	if err != nil {
		return err
	}
	if count >= int64(settings.DailySendLimit) {
		return ErrSiteMessageDailyLimitExceeded.WithMetadata(map[string]string{
			"limit": strconv.Itoa(settings.DailySendLimit),
		})
	}
	return nil
}

func (s *SiteMessageService) retentionCutoff(settings SiteMessageSettings) time.Time {
	return s.now().AddDate(0, 0, -settings.RetentionDays)
}

func normalizeSiteMessageSettings(settings SiteMessageSettings) SiteMessageSettings {
	if settings.DailySendLimit < 0 {
		settings.DailySendLimit = SiteMessagesDailySendLimitDefault
	}
	if settings.DailySendLimit > SiteMessagesDailySendLimitMax {
		settings.DailySendLimit = SiteMessagesDailySendLimitMax
	}
	if settings.RetentionDays < SiteMessagesRetentionDaysMin {
		settings.RetentionDays = SiteMessagesRetentionDaysDefault
	}
	if settings.RetentionDays > SiteMessagesRetentionDaysMax {
		settings.RetentionDays = SiteMessagesRetentionDaysMax
	}
	return settings
}

func siteMessageRecipientFromUser(user *User, err error) (*SiteMessageRecipient, error) {
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrSiteMessageRecipientNotFound
		}
		return nil, err
	}
	if user == nil {
		return nil, ErrSiteMessageRecipientNotFound
	}
	return &SiteMessageRecipient{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}, nil
}
