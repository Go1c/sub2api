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
	repo                SiteMessageRepository
	userRepo            SiteMessageUserRepository
	settings            SiteMessageSettingsReader
	emailSender         SiteMessageEmailSender
	redeemCodes         SiteMessageRedeemCodeReader
	promoCodes          SiteMessagePromoCodeValidator
	compensationBatches SiteMessageCompensationBatchRepository
	webhookNotify       *WebhookBalanceNotifyService
	now                 func() time.Time
}

func NewSiteMessageService(
	repo SiteMessageRepository,
	userRepo SiteMessageUserRepository,
	settings SiteMessageSettingsReader,
	emailSender SiteMessageEmailSender,
	redeemCodes SiteMessageRedeemCodeReader,
	promoCodes SiteMessagePromoCodeValidator,
	compensationBatches SiteMessageCompensationBatchRepository,
) *SiteMessageService {
	return &SiteMessageService{
		repo:                repo,
		userRepo:            userRepo,
		settings:            settings,
		emailSender:         emailSender,
		redeemCodes:         redeemCodes,
		promoCodes:          promoCodes,
		compensationBatches: compensationBatches,
		now:                 time.Now,
	}
}

// SetWebhookBalanceNotifyService injects optional Webhook notify on new inbox messages.
func (s *SiteMessageService) SetWebhookBalanceNotifyService(svc *WebhookBalanceNotifyService) {
	if s != nil {
		s.webhookNotify = svc
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

	input.InactiveDays = normalizeInactiveDays(input.InactiveDays)
	targets, err := s.batchRecipientTargets(ctx, input)
	if err != nil {
		return nil, err
	}
	codes := normalizeCompensationCodes(input.CompensationCodes)

	sentAt := s.now()
	messages := make([]int64, 0, len(targets))
	assignments := make([]SiteMessageCompensationCodeAssignment, 0, len(targets))
	results := make([]SiteMessageCompensationBatchResult, 0, len(targets))
	promoAssignments := make(map[int64]int)
	for i, target := range targets {
		result := SiteMessageCompensationBatchResult{
			Recipient: target.Email,
			Status:    SiteMessageBatchResultFailed,
		}

		recipient, resultErr := s.resolveBatchTarget(ctx, target)
		if resultErr != nil {
			results = append(results, *resultErr)
			continue
		}
		result.Recipient = recipient.Email
		result.UserID = recipient.ID

		code := ""
		var promoCode *PromoCode
		if input.CompensationEnabled {
			var codeErr *SiteMessageCompensationBatchResult
			code, promoCode, codeErr = s.validateCompensationCodeForTarget(ctx, recipient, i, codes, input.CompensationAmount, promoAssignments)
			result.Code = code
			if codeErr != nil {
				codeErr.UserID = recipient.ID
				results = append(results, *codeErr)
				continue
			}
		}

		messageContent := content
		if input.CompensationEnabled {
			messageContent = appendCompensationBlock(messageContent, input.CompensationAmount, code, input.CompensationFormat)
		}
		message, err := s.create(ctx, input.AdminID, recipient.ID, nil, subject, messageContent)
		if err != nil {
			result.ErrorReason = "SITE_MESSAGE_SEND_FAILED"
			result.Error = "send failed"
			results = append(results, result)
			continue
		}
		messages = append(messages, message.ID)
		result.MessageID = message.ID
		result.Status = SiteMessageBatchResultSent
		result.ErrorReason = ""
		result.Error = ""
		if input.SendEmail && s.emailSender != nil {
			if err := s.emailSender.EnqueueSiteMessage(recipient.Email, message.Subject, message.Content); err != nil {
				result.Status = SiteMessageBatchResultFailed
				result.ErrorReason = "SITE_MESSAGE_EMAIL_ENQUEUE_FAILED"
				result.Error = "email enqueue failed"
				results = append(results, result)
				continue
			}
		}
		if input.CompensationEnabled {
			assignments = append(assignments, SiteMessageCompensationCodeAssignment{
				Recipient: recipient.Email,
				Code:      code,
				Status:    StatusUnused,
			})
			if promoCode != nil {
				promoAssignments[promoCode.ID]++
			}
		}
		results = append(results, result)
	}

	amount := 0.0
	if input.CompensationEnabled {
		amount = input.CompensationAmount
	}
	successCount, failedCount := countBatchResults(results)
	batch := &SiteMessageCompensationBatch{
		ID:             fmt.Sprintf("CMP-%s", sentAt.Format("20060102-150405")),
		Subject:        subject,
		Content:        content,
		Mode:           normalizeRecipientMode(input.RecipientMode),
		Audience:       batchAudienceLabel(input.RecipientMode, input.InactiveDays, len(targets)),
		RecipientCount: len(targets),
		SuccessCount:   successCount,
		FailedCount:    failedCount,
		Amount:         amount,
		CodeCount:      len(assignments),
		Operator:       adminUser.Email,
		SentAt:         sentAt,
		Codes:          assignments,
		Results:        results,
		MessageIDs:     messages,
	}
	if s.compensationBatches != nil {
		if err := s.compensationBatches.Create(ctx, batch); err != nil {
			return nil, fmt.Errorf("record site message compensation batch: %w", err)
		}
	}
	return batch, nil
}

func (s *SiteMessageService) ListCompensationBatches(ctx context.Context, params pagination.PaginationParams) ([]SiteMessageCompensationBatch, *pagination.PaginationResult, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, nil, err
	}
	if s.compensationBatches == nil {
		return []SiteMessageCompensationBatch{}, paginationResult(0, params), nil
	}
	return s.compensationBatches.List(ctx, params)
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
	if s.webhookNotify != nil {
		s.webhookNotify.NotifySiteMessage(ctx, message.RecipientID, message.ID, message.Subject)
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

type siteMessageBatchTarget struct {
	Email    string
	Resolved *SiteMessageRecipient
}

func (s *SiteMessageService) batchRecipientTargets(ctx context.Context, input AdminSendCompensationBatchInput) ([]siteMessageBatchTarget, error) {
	mode := normalizeRecipientMode(input.RecipientMode)
	if mode == SiteMessageRecipientModeAll {
		recipients, err := s.listAllActiveRecipients(ctx, input.InactiveDays)
		if err != nil {
			return nil, err
		}
		targets := make([]siteMessageBatchTarget, 0, len(recipients))
		for i := range recipients {
			recipient := recipients[i]
			targets = append(targets, siteMessageBatchTarget{Email: recipient.Email, Resolved: &recipient})
		}
		return targets, nil
	}

	seen := make(map[string]struct{}, len(input.RecipientEmails))
	targets := make([]siteMessageBatchTarget, 0, len(input.RecipientEmails))
	for _, rawEmail := range input.RecipientEmails {
		email := strings.TrimSpace(rawEmail)
		if email == "" {
			continue
		}
		normalized := strings.ToLower(email)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		targets = append(targets, siteMessageBatchTarget{Email: email})
	}
	if len(targets) == 0 {
		return nil, ErrSiteMessageNoRecipients
	}
	return targets, nil
}

func (s *SiteMessageService) listAllActiveRecipients(ctx context.Context, inactiveDays int) ([]SiteMessageRecipient, error) {
	const pageSize = 1000
	page := 1
	includeSubscriptions := false
	recipients := make([]SiteMessageRecipient, 0)
	filters := UserListFilters{
		Status:               StatusActive,
		IncludeSubscriptions: &includeSubscriptions,
	}
	if inactiveDays > 0 {
		cutoff := s.now().AddDate(0, 0, -inactiveDays)
		filters.NoUsageSince = &cutoff
	}
	for {
		users, result, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "id",
			SortOrder: pagination.SortOrderAsc,
		}, filters)
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

func (s *SiteMessageService) resolveBatchTarget(ctx context.Context, target siteMessageBatchTarget) (*SiteMessageRecipient, *SiteMessageCompensationBatchResult) {
	if target.Resolved != nil {
		return target.Resolved, nil
	}
	email := strings.TrimSpace(target.Email)
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, failedBatchResult(email, "", "SITE_MESSAGE_RECIPIENTS_INVALID", "recipient email is invalid")
	}
	recipient, err := s.resolveRecipient(ctx, email, false)
	if err == nil {
		return recipient, nil
	}
	if errors.Is(err, ErrSiteMessageRecipientNotFound) {
		return nil, failedBatchResult(email, "", "SITE_MESSAGE_RECIPIENT_NOT_FOUND", "recipient not found")
	}
	return nil, failedBatchResult(email, "", "SITE_MESSAGE_RECIPIENT_LOOKUP_FAILED", "recipient lookup failed")
}

func (s *SiteMessageService) validateCompensationCodeForTarget(ctx context.Context, recipient *SiteMessageRecipient, index int, codes []string, amount float64, promoAssignments map[int64]int) (string, *PromoCode, *SiteMessageCompensationBatchResult) {
	recipientEmail := ""
	var userID int64
	if recipient != nil {
		recipientEmail = recipient.Email
		userID = recipient.ID
	}
	if amount <= 0 {
		return "", nil, failedBatchResult(recipientEmail, "", "SITE_MESSAGE_REDEEM_CODE_INVALID", "compensation amount is invalid")
	}
	if index >= len(codes) {
		if len(codes) == 1 {
			code := codes[0]
			if promoCode, found, result := s.validatePromoCompensationCode(ctx, userID, recipientEmail, code, amount, promoAssignments); found {
				return code, promoCode, result
			}
		}
		return "", nil, failedBatchResult(recipientEmail, "", "SITE_MESSAGE_REDEEM_CODE_SHORTAGE", "missing redeem code")
	}
	code := codes[index]
	if s.redeemCodes == nil {
		if promoCode, found, result := s.validatePromoCompensationCode(ctx, userID, recipientEmail, code, amount, promoAssignments); found {
			return code, promoCode, result
		}
		return code, nil, failedBatchResult(recipientEmail, code, "SITE_MESSAGE_REDEEM_CODE_INVALID", "redeem code validation is unavailable")
	}

	redeemCode, err := s.redeemCodes.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrRedeemCodeNotFound) {
			if promoCode, found, result := s.validatePromoCompensationCode(ctx, userID, recipientEmail, code, amount, promoAssignments); found {
				return code, promoCode, result
			}
			return code, nil, failedBatchResult(recipientEmail, code, "SITE_MESSAGE_REDEEM_CODE_NOT_FOUND", "redeem code not found")
		}
		return code, nil, failedBatchResult(recipientEmail, code, "SITE_MESSAGE_REDEEM_CODE_LOOKUP_FAILED", "redeem code lookup failed")
	}
	if redeemCode.Type != RedeemTypeBalance {
		return code, nil, failedBatchResult(recipientEmail, code, "SITE_MESSAGE_REDEEM_CODE_TYPE_INVALID", "redeem code is not a balance code")
	}
	if redeemCode.Status != StatusUnused {
		return code, nil, failedBatchResult(recipientEmail, code, "SITE_MESSAGE_REDEEM_CODE_STATUS_INVALID", "redeem code is not unused")
	}
	if !sameMoney(redeemCode.Value, amount) {
		return code, nil, failedBatchResult(recipientEmail, code, "SITE_MESSAGE_REDEEM_CODE_AMOUNT_MISMATCH", "redeem code amount does not match compensation amount")
	}
	return code, nil, nil
}

func (s *SiteMessageService) validatePromoCompensationCode(ctx context.Context, userID int64, recipient, code string, amount float64, promoAssignments map[int64]int) (*PromoCode, bool, *SiteMessageCompensationBatchResult) {
	if s.promoCodes == nil {
		return nil, false, nil
	}

	promoCode, err := s.promoCodes.ValidatePromoCodeForUser(ctx, userID, code)
	if err != nil {
		if errors.Is(err, ErrPromoCodeNotFound) {
			return nil, false, nil
		}
		return nil, true, failedBatchResult(recipient, code, promoCompensationErrorReason(err), promoCompensationErrorMessage(err))
	}
	if !sameMoney(promoCode.BonusAmount, amount) {
		return promoCode, true, failedBatchResult(recipient, code, "SITE_MESSAGE_REDEEM_CODE_AMOUNT_MISMATCH", "promo code amount does not match compensation amount")
	}
	if promoCode.MaxUses > 0 && promoCode.UsedCount+promoAssignments[promoCode.ID] >= promoCode.MaxUses {
		return promoCode, true, failedBatchResult(recipient, code, "PROMO_CODE_MAX_USED", "promo code has reached maximum uses")
	}
	return promoCode, true, nil
}

func promoCompensationErrorReason(err error) string {
	switch {
	case errors.Is(err, ErrPromoCodeExpired):
		return "PROMO_CODE_EXPIRED"
	case errors.Is(err, ErrPromoCodeDisabled):
		return "PROMO_CODE_DISABLED"
	case errors.Is(err, ErrPromoCodeMaxUsed):
		return "PROMO_CODE_MAX_USED"
	case errors.Is(err, ErrPromoCodeAlreadyUsed):
		return "PROMO_CODE_ALREADY_USED"
	case errors.Is(err, ErrPromoCodeInvalid):
		return "PROMO_CODE_INVALID"
	default:
		return "SITE_MESSAGE_PROMO_CODE_INVALID"
	}
}

func promoCompensationErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrPromoCodeExpired):
		return "promo code has expired"
	case errors.Is(err, ErrPromoCodeDisabled):
		return "promo code is disabled"
	case errors.Is(err, ErrPromoCodeMaxUsed):
		return "promo code has reached maximum uses"
	case errors.Is(err, ErrPromoCodeAlreadyUsed):
		return "recipient has already used this promo code"
	case errors.Is(err, ErrPromoCodeInvalid):
		return "promo code is invalid"
	default:
		return "promo code validation failed"
	}
}

func failedBatchResult(recipient, code, reason, message string) *SiteMessageCompensationBatchResult {
	return &SiteMessageCompensationBatchResult{
		Recipient:   recipient,
		Code:        code,
		Status:      SiteMessageBatchResultFailed,
		ErrorReason: reason,
		Error:       message,
	}
}

func countBatchResults(results []SiteMessageCompensationBatchResult) (success int, failed int) {
	for i := range results {
		if results[i].Status == SiteMessageBatchResultSent {
			success++
		} else {
			failed++
		}
	}
	return success, failed
}

func paginationResult(total int64, params pagination.PaginationParams) *pagination.PaginationResult {
	limit := params.Limit()
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	page := params.Page
	if page < 1 {
		page = 1
	}
	return &pagination.PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: limit,
		Pages:    pages,
	}
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

func normalizeInactiveDays(days int) int {
	if days <= 0 {
		return 0
	}
	if days > SiteMessagesInactiveDaysMax {
		return SiteMessagesInactiveDaysMax
	}
	return days
}

func batchAudienceLabel(mode string, inactiveDays int, count int) string {
	if normalizeRecipientMode(mode) == SiteMessageRecipientModeAll {
		if inactiveDays > 0 {
			return fmt.Sprintf("最近 %d 天未使用用户", inactiveDays)
		}
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
	return moneyCents(a) == moneyCents(b)
}

func moneyCents(v float64) int64 {
	return int64(math.Round(v * 100))
}
