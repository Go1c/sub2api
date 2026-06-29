package service

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	redeemCodes SiteMessageRedeemCodeReader
	now         func() time.Time
}

func NewSiteMessageService(
	repo SiteMessageRepository,
	userRepo SiteMessageUserRepository,
	settings SiteMessageSettingsReader,
	emailSender SiteMessageEmailSender,
	redeemCodes SiteMessageRedeemCodeReader,
) *SiteMessageService {
	return &SiteMessageService{
		repo:        repo,
		userRepo:    userRepo,
		settings:    settings,
		emailSender: emailSender,
		redeemCodes: redeemCodes,
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

func (s *SiteMessageService) AdminSendCompensationBatch(ctx context.Context, input AdminSendCompensationBatchInput) (*SiteMessageCompensationBatch, error) {
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

	subject, content, err := normalizeSiteMessagePayload(input.Subject, input.Content)
	if err != nil {
		return nil, err
	}

	recipients, err := s.resolveBatchRecipients(ctx, input)
	if err != nil {
		return nil, err
	}
	assignments, err := s.validateCompensationAssignments(ctx, recipients, input)
	if err != nil {
		return nil, err
	}

	sentAt := s.now()
	messages := make([]int64, 0, len(recipients))
	for i := range recipients {
		messageContent := content
		if input.CompensationEnabled {
			messageContent = appendCompensationBlock(messageContent, input.CompensationAmount, assignments[i].Code, input.CompensationFormat)
		}
		message, err := s.create(ctx, input.AdminID, recipients[i].ID, nil, subject, messageContent)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message.ID)
		if input.SendEmail && s.emailSender != nil {
			if err := s.emailSender.EnqueueSiteMessage(recipients[i].Email, message.Subject, message.Content); err != nil {
				return nil, fmt.Errorf("enqueue site message email: %w", err)
			}
		}
	}

	amount := 0.0
	if input.CompensationEnabled {
		amount = input.CompensationAmount
	}
	return &SiteMessageCompensationBatch{
		ID:             fmt.Sprintf("CMP-%s", sentAt.Format("20060102-150405")),
		Subject:        subject,
		Content:        content,
		Mode:           normalizeRecipientMode(input.RecipientMode),
		Audience:       batchAudienceLabel(input.RecipientMode, len(recipients)),
		RecipientCount: len(recipients),
		Amount:         amount,
		CodeCount:      len(assignments),
		Operator:       adminUser.Email,
		SentAt:         sentAt,
		Codes:          assignments,
		MessageIDs:     messages,
	}, nil
}

func (s *SiteMessageService) SendLotteryPrize(ctx context.Context, senderID, recipientID int64, campaignName, code string) (*SiteMessage, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, err
	}
	campaignName = strings.TrimSpace(campaignName)
	if campaignName == "" {
		campaignName = "抽奖活动"
	}
	subject := fmt.Sprintf("恭喜中奖：%s", campaignName)
	content := fmt.Sprintf("你在「%s」中中奖。\n\n兑换码：%s\n\n请复制该兑换码前往兑换页面使用。", campaignName, strings.TrimSpace(code))
	return s.create(ctx, senderID, recipientID, nil, subject, content)
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
	normalizedSubject, normalizedContent, err := normalizeSiteMessagePayload(subject, content)
	if err != nil {
		return nil, err
	}
	now := s.now()
	message := &SiteMessage{
		SenderID:    senderID,
		RecipientID: recipientID,
		ParentID:    parentID,
		Subject:     normalizedSubject,
		Content:     normalizedContent,
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
	settings.DefaultRecipientEmail = strings.TrimSpace(settings.DefaultRecipientEmail)
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

func normalizeSiteMessagePayload(subject, content string) (string, string, error) {
	subject = strings.TrimSpace(subject)
	content = strings.TrimSpace(content)
	if subject == "" || len([]rune(subject)) > 200 {
		return "", "", ErrSiteMessageInvalidSubject
	}
	if content == "" {
		return "", "", ErrSiteMessageContentRequired
	}
	return subject, content, nil
}

func (s *SiteMessageService) resolveBatchRecipients(ctx context.Context, input AdminSendCompensationBatchInput) ([]SiteMessageRecipient, error) {
	mode := normalizeRecipientMode(input.RecipientMode)
	if mode == SiteMessageRecipientModeAll {
		return s.listAllActiveRecipients(ctx)
	}

	seen := make(map[string]struct{}, len(input.RecipientEmails))
	recipients := make([]SiteMessageRecipient, 0, len(input.RecipientEmails))
	for _, rawEmail := range input.RecipientEmails {
		email := strings.TrimSpace(rawEmail)
		if email == "" {
			continue
		}
		normalized := strings.ToLower(email)
		if !strings.Contains(normalized, "@") || !strings.Contains(normalized, ".") {
			return nil, ErrSiteMessageInvalidRecipients.WithMetadata(map[string]string{"recipient": email})
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		recipient, err := s.resolveRecipient(ctx, email, false)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, *recipient)
	}
	if len(recipients) == 0 {
		return nil, ErrSiteMessageNoRecipients
	}
	return recipients, nil
}

func (s *SiteMessageService) listAllActiveRecipients(ctx context.Context) ([]SiteMessageRecipient, error) {
	const pageSize = 1000
	page := 1
	includeSubscriptions := false
	recipients := make([]SiteMessageRecipient, 0)
	for {
		users, result, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "id",
			SortOrder: pagination.SortOrderAsc,
		}, UserListFilters{
			Status:               StatusActive,
			IncludeSubscriptions: &includeSubscriptions,
		})
		if err != nil {
			return nil, err
		}
		for i := range users {
			recipients = append(recipients, SiteMessageRecipient{
				ID:       users[i].ID,
				Email:    users[i].Email,
				Username: users[i].Username,
			})
		}
		if len(users) == 0 || result == nil || page >= result.Pages {
			break
		}
		page++
	}
	if len(recipients) == 0 {
		return nil, ErrSiteMessageNoRecipients
	}
	return recipients, nil
}

func (s *SiteMessageService) validateCompensationAssignments(ctx context.Context, recipients []SiteMessageRecipient, input AdminSendCompensationBatchInput) ([]SiteMessageCompensationCodeAssignment, error) {
	if !input.CompensationEnabled {
		return []SiteMessageCompensationCodeAssignment{}, nil
	}
	if input.CompensationAmount <= 0 {
		return nil, ErrSiteMessageInvalidRedeemCode.WithMetadata(map[string]string{"reason": "amount_required"})
	}
	codes := normalizeCompensationCodes(input.CompensationCodes)
	if len(codes) < len(recipients) {
		return nil, ErrSiteMessageRedeemCodeShortage.WithMetadata(map[string]string{
			"required": strconv.Itoa(len(recipients)),
			"provided": strconv.Itoa(len(codes)),
		})
	}
	if s.redeemCodes == nil {
		return nil, ErrSiteMessageInvalidRedeemCode.WithMetadata(map[string]string{"reason": "redeem_reader_missing"})
	}

	assignments := make([]SiteMessageCompensationCodeAssignment, 0, len(recipients))
	for i := range recipients {
		code := codes[i]
		redeemCode, err := s.redeemCodes.GetByCode(ctx, code)
		if err != nil {
			return nil, ErrSiteMessageInvalidRedeemCode.WithMetadata(map[string]string{"code": code})
		}
		if redeemCode.Type != RedeemTypeBalance || redeemCode.Status != StatusUnused || !sameMoney(redeemCode.Value, input.CompensationAmount) {
			return nil, ErrSiteMessageInvalidRedeemCode.WithMetadata(map[string]string{"code": code})
		}
		assignments = append(assignments, SiteMessageCompensationCodeAssignment{
			Recipient: recipients[i].Email,
			Code:      code,
			Status:    redeemCode.Status,
		})
	}
	return assignments, nil
}

func normalizeCompensationCodes(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	codes := make([]string, 0, len(input))
	for _, rawCode := range input {
		code := strings.TrimSpace(rawCode)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes
}

func normalizeRecipientMode(mode string) string {
	if strings.TrimSpace(mode) == SiteMessageRecipientModeAll {
		return SiteMessageRecipientModeAll
	}
	return SiteMessageRecipientModeSelected
}

func batchAudienceLabel(mode string, count int) string {
	if normalizeRecipientMode(mode) == SiteMessageRecipientModeAll {
		return "全员用户"
	}
	return fmt.Sprintf("指定 %d 个用户", count)
}

func appendCompensationBlock(content string, amount float64, code, format string) string {
	amountText := fmt.Sprintf("%.2f", amount)
	if strings.TrimSpace(format) == SiteMessageCompensationFormatCompact {
		return fmt.Sprintf("%s\n\n补偿 %s 元，兑换码：%s", content, amountText, code)
	}
	return fmt.Sprintf("%s\n\n补偿金额：%s 元\n兑换码：%s\n请复制兑换码前往兑换码页面使用。", content, amountText, code)
}

func sameMoney(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}
