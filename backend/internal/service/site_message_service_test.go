package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type siteMessageSettingsStub struct {
	settings SiteMessageSettings
	err      error
}

func (s siteMessageSettingsStub) GetSiteMessageSettings(context.Context) (SiteMessageSettings, error) {
	if s.err != nil {
		return SiteMessageSettings{}, s.err
	}
	return s.settings, nil
}

type siteMessageUserRepoStub struct {
	users map[int64]*User
}

type siteMessageEmailCall struct {
	email   string
	subject string
	content string
}

type siteMessageEmailSenderStub struct {
	calls []siteMessageEmailCall
	err   error
}

func (s *siteMessageEmailSenderStub) EnqueueSiteMessage(email, subject, content string) error {
	s.calls = append(s.calls, siteMessageEmailCall{email: email, subject: subject, content: content})
	return s.err
}

type siteMessageRedeemCodeReaderStub struct {
	codes map[string]*RedeemCode
}

type siteMessagePromoCodeValidatorStub struct {
	codes  map[string]*PromoCode
	usages map[string]map[int64]bool
}

type siteMessageCompensationBatchRepoStub struct {
	created []*SiteMessageCompensationBatch
	items   []SiteMessageCompensationBatch
	err     error
}

func (s *siteMessageCompensationBatchRepoStub) Create(_ context.Context, batch *SiteMessageCompensationBatch) error {
	if s.err != nil {
		return s.err
	}
	copy := *batch
	copy.Codes = append([]SiteMessageCompensationCodeAssignment(nil), batch.Codes...)
	copy.Results = append([]SiteMessageCompensationBatchResult(nil), batch.Results...)
	copy.MessageIDs = append([]int64(nil), batch.MessageIDs...)
	s.created = append(s.created, &copy)
	return nil
}

func (s *siteMessageCompensationBatchRepoStub) List(_ context.Context, params pagination.PaginationParams) ([]SiteMessageCompensationBatch, *pagination.PaginationResult, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	total := int64(len(s.items))
	items := paginateServiceSlice(s.items, params)
	return items, paginationResult(total, params), nil
}

func (s siteMessageRedeemCodeReaderStub) GetByCode(_ context.Context, code string) (*RedeemCode, error) {
	if item, ok := s.codes[code]; ok {
		copy := *item
		return &copy, nil
	}
	return nil, ErrRedeemCodeNotFound
}

func (s siteMessagePromoCodeValidatorStub) ValidatePromoCodeForUser(_ context.Context, userID int64, code string) (*PromoCode, error) {
	if item, ok := s.codes[code]; ok {
		if s.usages != nil && s.usages[code][userID] {
			return nil, ErrPromoCodeAlreadyUsed
		}
		copy := *item
		return &copy, nil
	}
	return nil, ErrPromoCodeNotFound
}

func (s siteMessageUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if user, ok := s.users[id]; ok {
		copy := *user
		return &copy, nil
	}
	return nil, ErrUserNotFound
}

func (s siteMessageUserRepoStub) GetByEmail(_ context.Context, email string) (*User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	for _, user := range s.users {
		if strings.ToLower(strings.TrimSpace(user.Email)) == normalized {
			copy := *user
			return &copy, nil
		}
	}
	return nil, ErrUserNotFound
}

func (s siteMessageUserRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	search := strings.ToLower(strings.TrimSpace(filters.Search))
	out := make([]User, 0)
	for _, user := range s.users {
		if filters.Status != "" && user.Status != filters.Status {
			continue
		}
		if filters.Role != "" && user.Role != filters.Role {
			continue
		}
		if filters.NoUsageSince != nil && user.LastUsedAt != nil && !user.LastUsedAt.Before(*filters.NoUsageSince) {
			continue
		}
		if search == "" ||
			strings.Contains(strings.ToLower(user.Email), search) ||
			strings.Contains(strings.ToLower(user.Username), search) {
			copy := *user
			out = append(out, copy)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, &pagination.PaginationResult{
		Total:    int64(len(out)),
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

type siteMessageRepoStub struct {
	nextID  int64
	items   map[int64]*SiteMessage
	created []*SiteMessage
}

func newSiteMessageRepoStub() *siteMessageRepoStub {
	return &siteMessageRepoStub{
		nextID: 1,
		items:  make(map[int64]*SiteMessage),
	}
}

func (s *siteMessageRepoStub) Create(_ context.Context, message *SiteMessage) error {
	copy := *message
	if copy.ID == 0 {
		copy.ID = s.nextID
		s.nextID++
	}
	s.items[copy.ID] = &copy
	s.created = append(s.created, &copy)
	message.ID = copy.ID
	message.CreatedAt = copy.CreatedAt
	message.UpdatedAt = copy.UpdatedAt
	return nil
}

func (s *siteMessageRepoStub) GetVisibleByID(_ context.Context, messageID, userID int64, cutoff time.Time) (*SiteMessage, error) {
	item, ok := s.items[messageID]
	if !ok || item.CreatedAt.Before(cutoff) {
		return nil, ErrSiteMessageNotFound
	}
	if item.SenderID != userID && item.RecipientID != userID {
		return nil, ErrSiteMessageNotFound
	}
	copy := *item
	return &copy, nil
}

func (s *siteMessageRepoStub) ListInbox(_ context.Context, userID int64, params pagination.PaginationParams, cutoff time.Time) ([]SiteMessage, *pagination.PaginationResult, error) {
	out := make([]SiteMessage, 0)
	for _, item := range s.items {
		if item.RecipientID == userID && !item.CreatedAt.Before(cutoff) {
			out = append(out, *item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.PageSize}, nil
}

func (s *siteMessageRepoStub) ListSent(_ context.Context, userID int64, params pagination.PaginationParams, cutoff time.Time) ([]SiteMessage, *pagination.PaginationResult, error) {
	out := make([]SiteMessage, 0)
	for _, item := range s.items {
		if item.SenderID == userID && !item.CreatedAt.Before(cutoff) {
			out = append(out, *item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.PageSize}, nil
}

func (s *siteMessageRepoStub) MarkRead(_ context.Context, messageID, userID int64, readAt time.Time) error {
	item, ok := s.items[messageID]
	if !ok || item.RecipientID != userID {
		return ErrSiteMessageNotFound
	}
	if item.ReadAt == nil {
		item.ReadAt = &readAt
	}
	return nil
}

func (s *siteMessageRepoStub) CountUnread(_ context.Context, userID int64, cutoff time.Time) (int64, error) {
	var count int64
	for _, item := range s.items {
		if item.RecipientID == userID && item.ReadAt == nil && !item.CreatedAt.Before(cutoff) {
			count++
		}
	}
	return count, nil
}

func (s *siteMessageRepoStub) CountSentSince(_ context.Context, userID int64, since time.Time) (int64, error) {
	var count int64
	for _, item := range s.items {
		if item.SenderID == userID && !item.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (s *siteMessageRepoStub) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	var deleted int64
	for id, item := range s.items {
		if item.CreatedAt.Before(cutoff) {
			delete(s.items, id)
			deleted++
		}
	}
	return deleted, nil
}

func newSiteMessageTestService(repo *siteMessageRepoStub, settings SiteMessageSettings, users map[int64]*User, now time.Time) *SiteMessageService {
	svc := NewSiteMessageService(repo, siteMessageUserRepoStub{users: users}, siteMessageSettingsStub{settings: settings}, nil, nil, nil, nil)
	svc.now = func() time.Time { return now }
	return svc
}

func newSiteMessageTestServiceWithRedeem(repo *siteMessageRepoStub, settings SiteMessageSettings, users map[int64]*User, redeemCodes map[string]*RedeemCode, now time.Time) *SiteMessageService {
	svc := NewSiteMessageService(
		repo,
		siteMessageUserRepoStub{users: users},
		siteMessageSettingsStub{settings: settings},
		nil,
		siteMessageRedeemCodeReaderStub{codes: redeemCodes},
		nil,
		nil,
	)
	svc.now = func() time.Time { return now }
	return svc
}

func paginateServiceSlice[T any](items []T, params pagination.PaginationParams) []T {
	offset := params.Offset()
	if offset >= len(items) {
		return []T{}
	}
	end := offset + params.Limit()
	if end > len(items) {
		end = len(items)
	}
	return append([]T(nil), items[offset:end]...)
}

func siteMessageTestUsers() map[int64]*User {
	return map[int64]*User{
		1: {ID: 1, Email: "admin@example.com", Username: "admin", Role: RoleAdmin, Status: StatusActive},
		2: {ID: 2, Email: "alice@example.com", Username: "alice", Role: RoleUser, Status: StatusActive},
		3: {ID: 3, Email: "bob@example.com", Username: "bob", Role: RoleUser, Status: StatusActive},
	}
}

func TestSiteMessageServiceDisabledBlocksSend(t *testing.T) {
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: false, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), time.Unix(1778000000, 0))

	_, err := svc.Send(context.Background(), SendSiteMessageInput{
		SenderID:       2,
		RecipientQuery: "bob@example.com",
		Subject:        "hello",
		Content:        "body",
	})

	require.ErrorIs(t, err, ErrSiteMessagesDisabled)
	require.Empty(t, repo.created)
}

func TestSiteMessageServiceResolvesExactRecipientOnlyForUsers(t *testing.T) {
	svc := newSiteMessageTestService(newSiteMessageRepoStub(), SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), time.Unix(1778000000, 0))

	byID, err := svc.ResolveRecipient(context.Background(), "3", false)
	require.NoError(t, err)
	require.Equal(t, int64(3), byID.ID)

	byEmail, err := svc.ResolveRecipient(context.Background(), "BOB@example.com", false)
	require.NoError(t, err)
	require.Equal(t, int64(3), byEmail.ID)

	_, err = svc.ResolveRecipient(context.Background(), "bob", false)
	require.ErrorIs(t, err, ErrSiteMessageRecipientNotFound)
}

func TestSiteMessageServiceResolveRecipientReturnsDisabledWhenFeatureOff(t *testing.T) {
	svc := newSiteMessageTestService(newSiteMessageRepoStub(), SiteMessageSettings{Enabled: false, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), time.Unix(1778000000, 0))

	_, err := svc.ResolveRecipient(context.Background(), "bob@example.com", false)

	require.ErrorIs(t, err, ErrSiteMessagesDisabled)
}

func TestSiteMessageServiceEnforcesDailyLimitForRegularUsers(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	for i := 0; i < 10; i++ {
		id := int64(i + 1)
		repo.items[id] = &SiteMessage{
			ID:          id,
			SenderID:    2,
			RecipientID: 3,
			Subject:     "sent " + strconv.Itoa(i),
			Content:     "body",
			CreatedAt:   now.Add(-time.Duration(i) * time.Hour),
		}
	}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)

	_, err := svc.Send(context.Background(), SendSiteMessageInput{
		SenderID:       2,
		RecipientQuery: "bob@example.com",
		Subject:        "limit",
		Content:        "body",
	})

	require.ErrorIs(t, err, ErrSiteMessageDailyLimitExceeded)
}

func TestSiteMessageServiceAdminBypassesDailyLimit(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	for i := 0; i < 10; i++ {
		id := int64(i + 1)
		repo.items[id] = &SiteMessage{
			ID:          id,
			SenderID:    1,
			RecipientID: 3,
			Subject:     "sent " + strconv.Itoa(i),
			Content:     "body",
			CreatedAt:   now.Add(-time.Duration(i) * time.Hour),
		}
	}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)

	message, err := svc.AdminSendToUser(context.Background(), AdminSendSiteMessageInput{
		AdminID:     1,
		RecipientID: 3,
		Subject:     "admin note",
		Content:     "body",
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), message.SenderID)
	require.Equal(t, int64(3), message.RecipientID)
}

func TestSiteMessageServiceAdminSendToUserEnqueuesEmailCopyWhenRequested(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	emailSender := &siteMessageEmailSenderStub{}
	svc := newSiteMessageTestService(newSiteMessageRepoStub(), SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)
	svc.emailSender = emailSender

	message, err := svc.AdminSendToUser(context.Background(), AdminSendSiteMessageInput{
		AdminID:     1,
		RecipientID: 3,
		Subject:     "admin note",
		Content:     "body",
		SendEmail:   true,
	})

	require.NoError(t, err)
	require.Equal(t, int64(3), message.RecipientID)
	require.Len(t, emailSender.calls, 1)
	require.Equal(t, "bob@example.com", emailSender.calls[0].email)
	require.Equal(t, "admin note", emailSender.calls[0].subject)
	require.Equal(t, "body", emailSender.calls[0].content)
}

func TestSiteMessageServiceAdminSendToUserSkipsEmailCopyWhenNotRequested(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	emailSender := &siteMessageEmailSenderStub{}
	svc := newSiteMessageTestService(newSiteMessageRepoStub(), SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)
	svc.emailSender = emailSender

	_, err := svc.AdminSendToUser(context.Background(), AdminSendSiteMessageInput{
		AdminID:     1,
		RecipientID: 3,
		Subject:     "admin note",
		Content:     "body",
		SendEmail:   false,
	})

	require.NoError(t, err)
	require.Empty(t, emailSender.calls)
}

func TestSiteMessageServiceAdminSendCompensationBatchSendsRealMessages(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestServiceWithRedeem(
		repo,
		SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30},
		siteMessageTestUsers(),
		map[string]*RedeemCode{
			"CMP-A": {Code: "CMP-A", Type: RedeemTypeBalance, Value: 5, Status: StatusUnused},
			"CMP-B": {Code: "CMP-B", Type: RedeemTypeBalance, Value: 5, Status: StatusUnused},
		},
		now,
	)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"ALICE@example.com", "bob@example.com"},
		Subject:             "测试补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  5,
		CompensationCodes:   []string{"CMP-A", "CMP-B"},
		CompensationFormat:  SiteMessageCompensationFormatCompact,
	})

	require.NoError(t, err)
	require.NotNil(t, batch)
	require.Equal(t, "CMP-20260629-111100", batch.ID)
	require.Equal(t, "admin@example.com", batch.Operator)
	require.Equal(t, 2, batch.RecipientCount)
	require.Equal(t, 2, batch.SuccessCount)
	require.Equal(t, 0, batch.FailedCount)
	require.Equal(t, 10.0, batch.Amount)
	require.Equal(t, 2, batch.CodeCount)
	require.Len(t, batch.MessageIDs, 2)
	require.Len(t, batch.Codes, 2)
	require.Len(t, batch.Results, 2)
	require.Equal(t, SiteMessageBatchResultSent, batch.Results[0].Status)
	require.Equal(t, SiteMessageBatchResultSent, batch.Results[1].Status)
	require.Len(t, repo.created, 2)
	require.Equal(t, int64(2), repo.created[0].RecipientID)
	require.Contains(t, repo.created[0].Content, "CMP-A")
	require.Equal(t, int64(3), repo.created[1].RecipientID)
	require.Contains(t, repo.created[1].Content, "CMP-B")
}

func TestSiteMessageServiceAdminSendCompensationBatchAcceptsReusablePromoCode(t *testing.T) {
	now := time.Date(2026, 7, 3, 11, 11, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	svc := NewSiteMessageService(
		repo,
		siteMessageUserRepoStub{users: siteMessageTestUsers()},
		siteMessageSettingsStub{settings: SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}},
		nil,
		siteMessageRedeemCodeReaderStub{codes: map[string]*RedeemCode{}},
		siteMessagePromoCodeValidatorStub{codes: map[string]*PromoCode{
			"LUMIOAPI": {
				Code:        "LUMIOAPI",
				BonusAmount: 1.88,
				MaxUses:     100,
				UsedCount:   0,
				Status:      PromoCodeStatusActive,
			},
		}},
		nil,
	)
	svc.now = func() time.Time { return now }

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com", "bob@example.com"},
		Subject:             "优惠码补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  1.88,
		CompensationCodes:   []string{"LUMIOAPI"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Equal(t, 2, batch.SuccessCount)
	require.Equal(t, 0, batch.FailedCount)
	require.Equal(t, 2, batch.CodeCount)
	require.Len(t, batch.Codes, 2)
	require.Equal(t, "LUMIOAPI", batch.Codes[0].Code)
	require.Equal(t, "LUMIOAPI", batch.Codes[1].Code)
	require.Len(t, repo.created, 2)
	require.Contains(t, repo.created[0].Content, "LUMIOAPI")
	require.Contains(t, repo.created[1].Content, "LUMIOAPI")
}

func TestSiteMessageServiceAdminSendCompensationBatchEnforcesPromoRemainingUses(t *testing.T) {
	now := time.Date(2026, 7, 3, 11, 11, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	users := siteMessageTestUsers()
	svc := NewSiteMessageService(
		repo,
		siteMessageUserRepoStub{users: users},
		siteMessageSettingsStub{settings: SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}},
		nil,
		siteMessageRedeemCodeReaderStub{codes: map[string]*RedeemCode{}},
		siteMessagePromoCodeValidatorStub{codes: map[string]*PromoCode{
			"LUMIOAPI": {
				Code:        "LUMIOAPI",
				BonusAmount: 1.88,
				MaxUses:     2,
				UsedCount:   1,
				Status:      PromoCodeStatusActive,
			},
		}},
		nil,
	)
	svc.now = func() time.Time { return now }

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com", "bob@example.com"},
		Subject:             "优惠码补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  1.88,
		CompensationCodes:   []string{"LUMIOAPI"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Equal(t, 1, batch.SuccessCount)
	require.Equal(t, 1, batch.FailedCount)
	require.Equal(t, 1, batch.CodeCount)
	require.Len(t, repo.created, 1)
	require.Len(t, batch.Results, 2)
	require.Equal(t, SiteMessageBatchResultSent, batch.Results[0].Status)
	require.Equal(t, SiteMessageBatchResultFailed, batch.Results[1].Status)
	require.Equal(t, "PROMO_CODE_MAX_USED", batch.Results[1].ErrorReason)
}

func TestSiteMessageServiceAdminSendCompensationBatchRejectsPromoAlreadyUsedByRecipient(t *testing.T) {
	now := time.Date(2026, 7, 3, 11, 11, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	svc := NewSiteMessageService(
		repo,
		siteMessageUserRepoStub{users: siteMessageTestUsers()},
		siteMessageSettingsStub{settings: SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}},
		nil,
		siteMessageRedeemCodeReaderStub{codes: map[string]*RedeemCode{}},
		siteMessagePromoCodeValidatorStub{
			codes: map[string]*PromoCode{
				"LUMIOAPI": {
					ID:          10,
					Code:        "LUMIOAPI",
					BonusAmount: 1.88,
					MaxUses:     100,
					UsedCount:   0,
					Status:      PromoCodeStatusActive,
				},
			},
			usages: map[string]map[int64]bool{
				"LUMIOAPI": {2: true},
			},
		},
		nil,
	)
	svc.now = func() time.Time { return now }

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com", "bob@example.com"},
		Subject:             "优惠码补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  1.88,
		CompensationCodes:   []string{"LUMIOAPI"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Equal(t, 1, batch.SuccessCount)
	require.Equal(t, 1, batch.FailedCount)
	require.Len(t, repo.created, 1)
	require.Equal(t, int64(3), repo.created[0].RecipientID)
	require.Len(t, batch.Results, 2)
	require.Equal(t, SiteMessageBatchResultFailed, batch.Results[0].Status)
	require.Equal(t, "PROMO_CODE_ALREADY_USED", batch.Results[0].ErrorReason)
	require.Equal(t, SiteMessageBatchResultSent, batch.Results[1].Status)
}

func TestSiteMessageServiceAdminSendCompensationBatchEnqueuesEmailCopiesWhenRequested(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	emailSender := &siteMessageEmailSenderStub{}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)
	svc.emailSender = emailSender

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:         1,
		RecipientMode:   SiteMessageRecipientModeSelected,
		RecipientEmails: []string{"alice@example.com", "bob@example.com"},
		Subject:         "回归邀请",
		Content:         "欢迎回来",
		SendEmail:       true,
	})

	require.NoError(t, err)
	require.Equal(t, 2, batch.SuccessCount)
	require.Len(t, repo.created, 2)
	require.Len(t, emailSender.calls, 2)
	require.Equal(t, "alice@example.com", emailSender.calls[0].email)
	require.Equal(t, "回归邀请", emailSender.calls[0].subject)
	require.Equal(t, "欢迎回来", emailSender.calls[0].content)
	require.Equal(t, "bob@example.com", emailSender.calls[1].email)
}

func TestSiteMessageServiceAdminSendCompensationBatchPersistsHistory(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	historyRepo := &siteMessageCompensationBatchRepoStub{}
	svc := NewSiteMessageService(
		repo,
		siteMessageUserRepoStub{users: siteMessageTestUsers()},
		siteMessageSettingsStub{settings: SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}},
		nil,
		siteMessageRedeemCodeReaderStub{codes: map[string]*RedeemCode{
			"CMP-A": {Code: "CMP-A", Type: RedeemTypeBalance, Value: 5, Status: StatusUnused},
		}},
		nil,
		historyRepo,
	)
	svc.now = func() time.Time { return now }

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com"},
		Subject:             "测试补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  5,
		CompensationCodes:   []string{"CMP-A"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Len(t, historyRepo.created, 1)
	require.Equal(t, batch.ID, historyRepo.created[0].ID)
	require.Equal(t, 1, historyRepo.created[0].SuccessCount)
	require.Equal(t, 0, historyRepo.created[0].FailedCount)
	require.Equal(t, 5.0, historyRepo.created[0].Amount)
	require.Len(t, historyRepo.created[0].Codes, 1)
	require.Len(t, historyRepo.created[0].Results, 1)
}

func TestSiteMessageServiceListCompensationBatchesReturnsPersistedHistory(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC)
	historyRepo := &siteMessageCompensationBatchRepoStub{
		items: []SiteMessageCompensationBatch{
			{
				ID:             "CMP-20260629-111100",
				Subject:        "测试补偿",
				Content:        "基础内容",
				Mode:           SiteMessageRecipientModeSelected,
				Audience:       "指定 1 个用户",
				RecipientCount: 1,
				SuccessCount:   1,
				Amount:         5,
				CodeCount:      1,
				Operator:       "admin@example.com",
				SentAt:         now,
				Codes:          []SiteMessageCompensationCodeAssignment{{Recipient: "alice@example.com", Code: "CMP-A", Status: StatusUnused}},
				Results:        []SiteMessageCompensationBatchResult{{Recipient: "alice@example.com", Code: "CMP-A", Status: SiteMessageBatchResultSent, MessageID: 1}},
				MessageIDs:     []int64{1},
			},
		},
	}
	svc := NewSiteMessageService(
		newSiteMessageRepoStub(),
		siteMessageUserRepoStub{users: siteMessageTestUsers()},
		siteMessageSettingsStub{settings: SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}},
		nil,
		nil,
		nil,
		historyRepo,
	)

	items, page, err := svc.ListCompensationBatches(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, items, 1)
	require.Equal(t, "CMP-20260629-111100", items[0].ID)
	require.Len(t, items[0].Codes, 1)
	require.Len(t, items[0].Results, 1)
}

func TestSiteMessageServiceAdminSendCompensationBatchRecordsInvalidRedeemCode(t *testing.T) {
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestServiceWithRedeem(
		repo,
		SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30},
		siteMessageTestUsers(),
		map[string]*RedeemCode{
			"CMP-USED": {Code: "CMP-USED", Type: RedeemTypeBalance, Value: 5, Status: StatusUsed},
		},
		time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC),
	)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com"},
		Subject:             "测试补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  5,
		CompensationCodes:   []string{"CMP-USED"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Equal(t, 1, batch.RecipientCount)
	require.Equal(t, 0, batch.SuccessCount)
	require.Equal(t, 1, batch.FailedCount)
	require.Equal(t, 0.0, batch.Amount)
	require.Len(t, batch.Results, 1)
	require.Equal(t, SiteMessageBatchResultFailed, batch.Results[0].Status)
	require.Equal(t, "SITE_MESSAGE_REDEEM_CODE_STATUS_INVALID", batch.Results[0].ErrorReason)
	require.Empty(t, repo.created)
}

func TestSiteMessageServiceAdminSendCompensationBatchAllowsMinorMoneyPrecisionDiff(t *testing.T) {
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestServiceWithRedeem(
		repo,
		SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30},
		siteMessageTestUsers(),
		map[string]*RedeemCode{
			"CMP-PRECISION": {Code: "CMP-PRECISION", Type: RedeemTypeBalance, Value: 5.000000004, Status: StatusUnused},
		},
		time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC),
	)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com"},
		Subject:             "测试补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  5,
		CompensationCodes:   []string{"CMP-PRECISION"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Equal(t, 1, batch.SuccessCount)
	require.Equal(t, 0, batch.FailedCount)
	require.Len(t, repo.created, 1)
}

func TestSiteMessageServiceAdminSendCompensationBatchReportsRedeemAmountMismatch(t *testing.T) {
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestServiceWithRedeem(
		repo,
		SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30},
		siteMessageTestUsers(),
		map[string]*RedeemCode{
			"CMP-TEN": {Code: "CMP-TEN", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused},
		},
		time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC),
	)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"alice@example.com"},
		Subject:             "测试补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  5,
		CompensationCodes:   []string{"CMP-TEN"},
		CompensationFormat:  SiteMessageCompensationFormatBlock,
	})

	require.NoError(t, err)
	require.Equal(t, 0, batch.SuccessCount)
	require.Equal(t, 1, batch.FailedCount)
	require.Len(t, batch.Results, 1)
	require.Equal(t, "SITE_MESSAGE_REDEEM_CODE_AMOUNT_MISMATCH", batch.Results[0].ErrorReason)
	require.Empty(t, repo.created)
}

func TestSiteMessageServiceAdminSendCompensationBatchDoesNotBlockOnMissingRecipient(t *testing.T) {
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestServiceWithRedeem(
		repo,
		SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30},
		siteMessageTestUsers(),
		map[string]*RedeemCode{
			"CMP-A": {Code: "CMP-A", Type: RedeemTypeBalance, Value: 5, Status: StatusUnused},
			"CMP-B": {Code: "CMP-B", Type: RedeemTypeBalance, Value: 5, Status: StatusUnused},
		},
		time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC),
	)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:             1,
		RecipientMode:       SiteMessageRecipientModeSelected,
		RecipientEmails:     []string{"missing@example.com", "alice@example.com"},
		Subject:             "测试补偿",
		Content:             "基础内容",
		CompensationEnabled: true,
		CompensationAmount:  5,
		CompensationCodes:   []string{"CMP-A", "CMP-B"},
		CompensationFormat:  SiteMessageCompensationFormatCompact,
	})

	require.NoError(t, err)
	require.Equal(t, 2, batch.RecipientCount)
	require.Equal(t, 1, batch.SuccessCount)
	require.Equal(t, 1, batch.FailedCount)
	require.Len(t, batch.Results, 2)
	require.Equal(t, SiteMessageBatchResultFailed, batch.Results[0].Status)
	require.Equal(t, "missing@example.com", batch.Results[0].Recipient)
	require.Equal(t, "SITE_MESSAGE_RECIPIENT_NOT_FOUND", batch.Results[0].ErrorReason)
	require.Equal(t, SiteMessageBatchResultSent, batch.Results[1].Status)
	require.Equal(t, "alice@example.com", batch.Results[1].Recipient)
	require.Len(t, repo.created, 1)
	require.Equal(t, int64(2), repo.created[0].RecipientID)
	require.Contains(t, repo.created[0].Content, "CMP-B")
}

func TestSiteMessageServiceAdminSendCompensationBatchAllUsersLoopsActiveUsers(t *testing.T) {
	repo := newSiteMessageRepoStub()
	users := siteMessageTestUsers()
	users[4] = &User{ID: 4, Email: "disabled@example.com", Username: "disabled", Role: RoleUser, Status: StatusDisabled}
	svc := newSiteMessageTestServiceWithRedeem(
		repo,
		SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30},
		users,
		nil,
		time.Date(2026, 6, 29, 11, 11, 0, 0, time.UTC),
	)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:       1,
		RecipientMode: SiteMessageRecipientModeAll,
		Subject:       "全员通知",
		Content:       "基础内容",
	})

	require.NoError(t, err)
	require.Equal(t, 3, batch.RecipientCount)
	require.Equal(t, 3, batch.SuccessCount)
	require.Equal(t, 0, batch.FailedCount)
	require.Equal(t, 0.0, batch.Amount)
	require.Len(t, repo.created, 3)
	for _, message := range repo.created {
		require.NotEqual(t, int64(4), message.RecipientID)
	}
}

func TestSiteMessageServiceAdminSendCompensationBatchAllUsersFiltersInactiveDays(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	cutoff := now.AddDate(0, 0, -3)
	recent := cutoff.Add(time.Hour)
	old := cutoff.Add(-time.Hour)
	repo := newSiteMessageRepoStub()
	users := siteMessageTestUsers()
	users[1].LastUsedAt = &recent
	users[2].LastUsedAt = &old
	users[3].LastUsedAt = &recent
	users[4] = &User{ID: 4, Email: "never-used@example.com", Username: "never", Role: RoleUser, Status: StatusActive}
	users[5] = &User{ID: 5, Email: "disabled-old@example.com", Username: "disabled", Role: RoleUser, Status: StatusDisabled, LastUsedAt: &old}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, users, now)

	batch, err := svc.AdminSendCompensationBatch(context.Background(), AdminSendCompensationBatchInput{
		AdminID:       1,
		RecipientMode: SiteMessageRecipientModeAll,
		Subject:       "回归邀请",
		Content:       "好久不见",
		InactiveDays:  3,
	})

	require.NoError(t, err)
	require.Equal(t, "最近 3 天未使用用户", batch.Audience)
	require.Equal(t, 2, batch.RecipientCount)
	require.Equal(t, 2, batch.SuccessCount)
	require.Len(t, repo.created, 2)
	require.Equal(t, int64(2), repo.created[0].RecipientID)
	require.Equal(t, int64(4), repo.created[1].RecipientID)
}

func TestSiteMessageServiceOpenMarksRecipientMessageRead(t *testing.T) {
	now := time.Unix(1778000000, 0)
	repo := newSiteMessageRepoStub()
	repo.items[1] = &SiteMessage{
		ID:          1,
		SenderID:    2,
		RecipientID: 3,
		Subject:     "hello",
		Content:     "body",
		CreatedAt:   now,
	}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)

	message, err := svc.Get(context.Background(), 3, 1)

	require.NoError(t, err)
	require.Equal(t, int64(1), message.ID)
	require.NotNil(t, repo.items[1].ReadAt)
}

func TestSiteMessageServiceRejectsUnauthorizedDetail(t *testing.T) {
	now := time.Unix(1778000000, 0)
	repo := newSiteMessageRepoStub()
	repo.items[1] = &SiteMessage{
		ID:          1,
		SenderID:    1,
		RecipientID: 3,
		Subject:     "secret",
		Content:     "body",
		CreatedAt:   now,
	}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)

	_, err := svc.Get(context.Background(), 2, 1)

	require.ErrorIs(t, err, ErrSiteMessageNotFound)
}

func TestSiteMessageServiceReplyPreservesParentAndRequiresAccess(t *testing.T) {
	now := time.Unix(1778000000, 0)
	repo := newSiteMessageRepoStub()
	repo.items[1] = &SiteMessage{
		ID:          1,
		SenderID:    2,
		RecipientID: 3,
		Subject:     "question",
		Content:     "body",
		CreatedAt:   now,
	}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)

	reply, err := svc.Reply(context.Background(), 3, 1, "answer")
	require.NoError(t, err)
	require.NotNil(t, reply.ParentID)
	require.Equal(t, int64(1), *reply.ParentID)
	require.Equal(t, int64(3), reply.SenderID)
	require.Equal(t, int64(2), reply.RecipientID)

	_, err = svc.Reply(context.Background(), 1, 1, "not allowed")
	require.ErrorIs(t, err, ErrSiteMessageNotFound)
}

func TestSiteMessageServiceReplyCountsTowardDailyLimit(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	repo := newSiteMessageRepoStub()
	repo.items[1] = &SiteMessage{
		ID:          1,
		SenderID:    2,
		RecipientID: 3,
		Subject:     "question",
		Content:     "body",
		CreatedAt:   now.Add(-time.Hour),
	}
	repo.items[2] = &SiteMessage{
		ID:          2,
		SenderID:    3,
		RecipientID: 2,
		Subject:     "already sent",
		Content:     "body",
		CreatedAt:   now.Add(-time.Minute),
	}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 1, RetentionDays: 30}, siteMessageTestUsers(), now)

	_, err := svc.Reply(context.Background(), 3, 1, "answer")

	require.ErrorIs(t, err, ErrSiteMessageDailyLimitExceeded)
}

func TestSiteMessageServiceUnreadCountIgnoresReadAndExpired(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	readAt := now.Add(-time.Hour)
	repo := newSiteMessageRepoStub()
	repo.items[1] = &SiteMessage{ID: 1, SenderID: 2, RecipientID: 3, Subject: "unread", Content: "body", CreatedAt: now.Add(-time.Hour)}
	repo.items[2] = &SiteMessage{ID: 2, SenderID: 2, RecipientID: 3, Subject: "read", Content: "body", ReadAt: &readAt, CreatedAt: now.Add(-time.Hour)}
	repo.items[3] = &SiteMessage{ID: 3, SenderID: 2, RecipientID: 3, Subject: "expired", Content: "body", CreatedAt: now.AddDate(0, 0, -31)}
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), now)

	count, err := svc.UnreadCount(context.Background(), 3)

	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestSiteMessageServicePropagatesSettingsError(t *testing.T) {
	svc := NewSiteMessageService(newSiteMessageRepoStub(), siteMessageUserRepoStub{users: siteMessageTestUsers()}, siteMessageSettingsStub{err: errors.New("settings down")}, nil, nil, nil, nil)

	_, err := svc.UnreadCount(context.Background(), 3)

	require.ErrorContains(t, err, "settings down")
}

func TestSiteMessageServiceSendLotteryPrizeAppendsPromo(t *testing.T) {
	repo := newSiteMessageRepoStub()
	svc := newSiteMessageTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, siteMessageTestUsers(), time.Unix(1778000000, 0))

	msg, err := svc.SendLotteryPrize(
		context.Background(),
		1,
		2,
		"五月幸运转盘",
		"LUCK-001",
		"关注公众号领福利",
		"https://cdn.example.com/qr.png",
	)

	require.NoError(t, err)
	require.NotNil(t, msg)
	require.Equal(t, "恭喜中奖：五月幸运转盘", repo.created[0].Subject)
	require.Contains(t, repo.created[0].Content, "LUCK-001")
	require.Contains(t, repo.created[0].Content, "关注公众号领福利")
	require.Contains(t, repo.created[0].Content, "https://cdn.example.com/qr.png")
}
