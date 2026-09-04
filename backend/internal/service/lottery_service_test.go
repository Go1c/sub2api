package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type lotterySettingsStub struct {
	settings SiteMessageSettings
	err      error
}

func (s lotterySettingsStub) GetSiteMessageSettings(context.Context) (SiteMessageSettings, error) {
	if s.err != nil {
		return SiteMessageSettings{}, s.err
	}
	return s.settings, nil
}

type lotteryMessengerStub struct {
	nextID int64
	sent   []lotteryPrizeMessage
	err    error
}

type lotteryPrizeMessage struct {
	senderID      int64
	recipientID   int64
	campaignName  string
	code          string
	promoText     string
	promoImageURL string
}

func (s *lotteryMessengerStub) SendLotteryPrize(ctx context.Context, senderID, recipientID int64, campaignName, code, promoText, promoImageURL string) (*SiteMessage, error) {
	_ = ctx
	if s.err != nil {
		return nil, s.err
	}
	s.nextID++
	s.sent = append(s.sent, lotteryPrizeMessage{
		senderID:      senderID,
		recipientID:   recipientID,
		campaignName:  campaignName,
		code:          code,
		promoText:     promoText,
		promoImageURL: promoImageURL,
	})
	return &SiteMessage{ID: s.nextID, SenderID: senderID, RecipientID: recipientID}, nil
}

type lotteryRepoStub struct {
	nextCampaignID int64
	nextCodeID     int64
	nextDrawID     int64
	campaigns      map[int64]*LotteryCampaign
	codes          map[int64]*LotteryCode
	draws          map[int64]*LotteryDraw
	users          map[int64]*LotteryUserProfile
}

func newLotteryRepoStub() *lotteryRepoStub {
	return &lotteryRepoStub{
		nextCampaignID: 1,
		nextCodeID:     1,
		nextDrawID:     1,
		campaigns:      map[int64]*LotteryCampaign{},
		codes:          map[int64]*LotteryCode{},
		draws:          map[int64]*LotteryDraw{},
		users:          map[int64]*LotteryUserProfile{},
	}
}

func (r *lotteryRepoStub) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (r *lotteryRepoStub) FinishActiveCampaigns(_ context.Context, finishedAt time.Time) error {
	for _, c := range r.campaigns {
		if c.Status == LotteryStatusActive {
			c.Status = LotteryStatusFinished
			c.FinishedAt = &finishedAt
			c.UpdatedAt = finishedAt
		}
	}
	return nil
}

func (r *lotteryRepoStub) CreateCampaign(_ context.Context, campaign *LotteryCampaign, codes []LotteryCode) error {
	c := *campaign
	c.ID = r.nextCampaignID
	r.nextCampaignID++
	r.campaigns[c.ID] = &c
	for i := range codes {
		code := codes[i]
		code.ID = r.nextCodeID
		r.nextCodeID++
		code.CampaignID = c.ID
		r.codes[code.ID] = &code
	}
	*campaign = c
	return nil
}

func (r *lotteryRepoStub) ListCampaigns(_ context.Context, _ pagination.PaginationParams) ([]LotteryCampaign, *pagination.PaginationResult, error) {
	items := make([]LotteryCampaign, 0, len(r.campaigns))
	for _, c := range r.campaigns {
		items = append(items, *c)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	return items, &pagination.PaginationResult{Total: int64(len(items)), Page: 1, PageSize: len(items)}, nil
}

func (r *lotteryRepoStub) GetCampaign(_ context.Context, id int64) (*LotteryCampaign, error) {
	c, ok := r.campaigns[id]
	if !ok {
		return nil, ErrLotteryCampaignNotFound
	}
	out := *c
	for _, code := range r.codes {
		if code.CampaignID == id {
			out.Codes = append(out.Codes, *code)
		}
	}
	for _, draw := range r.draws {
		if draw.CampaignID == id {
			out.Draws = append(out.Draws, *draw)
		}
	}
	return &out, nil
}

func (r *lotteryRepoStub) GetActiveCampaign(_ context.Context) (*LotteryCampaign, error) {
	for _, c := range r.campaigns {
		if c.Status == LotteryStatusActive {
			out := *c
			return &out, nil
		}
	}
	return nil, ErrLotteryCampaignNotFound
}

func (r *lotteryRepoStub) GetCampaignForUpdate(ctx context.Context, id int64) (*LotteryCampaign, error) {
	return r.GetCampaign(ctx, id)
}

func (r *lotteryRepoStub) GetDrawByCampaignAndUser(_ context.Context, campaignID, userID int64) (*LotteryDraw, error) {
	for _, d := range r.draws {
		if d.CampaignID == campaignID && d.UserID == userID {
			out := *d
			return &out, nil
		}
	}
	return nil, ErrLotteryDrawNotFound
}

func (r *lotteryRepoStub) GetUserLotteryProfile(_ context.Context, userID int64) (*LotteryUserProfile, error) {
	if profile, ok := r.users[userID]; ok {
		out := *profile
		return &out, nil
	}
	return &LotteryUserProfile{UserID: userID}, nil
}

func (r *lotteryRepoStub) PickUnassignedCode(_ context.Context, campaignID int64) (*LotteryCode, error) {
	ids := make([]int64, 0, len(r.codes))
	for id := range r.codes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		code := r.codes[id]
		if code.CampaignID == campaignID && code.AssignedAt == nil {
			out := *code
			return &out, nil
		}
	}
	return nil, ErrLotteryNoCodeAvailable
}

func (r *lotteryRepoStub) CreateDraw(_ context.Context, draw *LotteryDraw) error {
	for _, existing := range r.draws {
		if existing.CampaignID == draw.CampaignID && existing.UserID == draw.UserID {
			return ErrLotteryAlreadyDrawn
		}
	}
	d := *draw
	d.ID = r.nextDrawID
	r.nextDrawID++
	r.draws[d.ID] = &d
	*draw = d
	return nil
}

func (r *lotteryRepoStub) AssignCodeToDraw(_ context.Context, codeID, userID, drawID int64, assignedAt time.Time) error {
	code, ok := r.codes[codeID]
	if !ok {
		return ErrLotteryNoCodeAvailable
	}
	if code.AssignedAt != nil {
		return ErrLotteryNoCodeAvailable
	}
	code.AssignedUserID = &userID
	code.AssignedDrawID = &drawID
	code.AssignedAt = &assignedAt
	return nil
}

func (r *lotteryRepoStub) IncrementCampaignCounters(_ context.Context, campaignID int64, joinedDelta, winnerDelta int, finish bool, finishedAt *time.Time) (*LotteryCampaign, error) {
	c, ok := r.campaigns[campaignID]
	if !ok {
		return nil, ErrLotteryCampaignNotFound
	}
	c.JoinedCount += joinedDelta
	c.WinnerCount += winnerDelta
	if finish {
		c.Status = LotteryStatusFinished
		c.FinishedAt = finishedAt
	}
	out := *c
	return &out, nil
}

func newLotteryTestService(repo *lotteryRepoStub, settings SiteMessageSettings, messenger *lotteryMessengerStub, now time.Time) *LotteryService {
	svc := NewLotteryService(repo, lotterySettingsStub{settings: settings}, messenger)
	svc.now = func() time.Time { return now }
	svc.randFloat = func() float64 { return 0.99 }
	return svc
}

func seedLotteryCampaign(repo *lotteryRepoStub, campaign LotteryCampaign, codes ...string) *LotteryCampaign {
	if campaign.ID == 0 {
		campaign.ID = repo.nextCampaignID
		repo.nextCampaignID++
	}
	repo.campaigns[campaign.ID] = &campaign
	for _, raw := range codes {
		code := &LotteryCode{
			ID:         repo.nextCodeID,
			CampaignID: campaign.ID,
			Code:       raw,
			CreatedAt:  campaign.CreatedAt,
			UpdatedAt:  campaign.UpdatedAt,
		}
		repo.nextCodeID++
		repo.codes[code.ID] = code
	}
	return repo.campaigns[campaign.ID]
}

func TestLotteryServiceCreateCampaignFinishesPreviousActive(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	old := seedLotteryCampaign(repo, LotteryCampaign{
		Name:            "old",
		Subtitle:        "old subtitle",
		Status:          LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 5,
		CreatedBy:       1,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}, "OLD-CODE")
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	created, err := svc.CreateCampaign(context.Background(), 99, CreateLotteryCampaignInput{
		Name:            "new",
		Subtitle:        "new subtitle",
		PrizeCount:      2,
		MaxParticipants: 4,
		Codes:           []string{"A", "B"},
	})

	require.NoError(t, err)
	require.Equal(t, LotteryStatusFinished, old.Status)
	require.NotNil(t, old.FinishedAt)
	require.Equal(t, LotteryStatusActive, created.Status)
	require.Equal(t, int64(99), created.CreatedBy)
	require.Equal(t, 2, created.PrizeCount)
	require.Equal(t, 25, created.EarlyBoostParticipantPercent)
	require.Equal(t, 0, created.RechargeBoostCapPercent)
}

func TestLotteryServiceCreateCampaignRejectsRechargeBoostCapAboveMaximum(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	_, err := svc.CreateCampaign(context.Background(), 99, CreateLotteryCampaignInput{
		Name:                    "too much boost",
		PrizeCount:              1,
		MaxParticipants:         2,
		Codes:                   []string{"A"},
		RechargeBoostCapPercent: 51,
	})

	require.ErrorIs(t, err, ErrLotteryInvalidCampaign)
}

func TestLotteryServiceCreateCampaignRejectsDuplicateCodes(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	_, err := svc.CreateCampaign(context.Background(), 99, CreateLotteryCampaignInput{
		Name:            "dupes",
		PrizeCount:      2,
		MaxParticipants: 3,
		Codes:           []string{"DUP", " dup "},
	})

	require.ErrorIs(t, err, ErrLotteryDuplicateCodes)
}

func TestLotteryServiceCreateCampaignRejectsDisabledSiteMessages(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: false, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	_, err := svc.CreateCampaign(context.Background(), 99, CreateLotteryCampaignInput{
		Name:            "disabled",
		PrizeCount:      1,
		MaxParticipants: 2,
		Codes:           []string{"A"},
	})

	require.ErrorIs(t, err, ErrLotterySiteMessagesDisabled)
}

func TestLotteryServiceGetActiveForUserReturnsNilAfterDraw(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:            "active",
		Subtitle:        "subtitle",
		Status:          LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 3,
		CreatedBy:       99,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, "A")
	err := repo.CreateDraw(context.Background(), &LotteryDraw{
		CampaignID:  campaign.ID,
		UserID:      7,
		Won:         false,
		ResultLabel: LotteryLoseLabel,
		CreatedAt:   now,
	})
	require.NoError(t, err)
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	active, err := svc.GetActiveForUser(context.Background(), 7)

	require.NoError(t, err)
	require.Nil(t, active)
}

func TestLotteryServiceDrawWinSendsSiteMessage(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:            "lucky",
		Subtitle:        "subtitle",
		Status:          LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 1,
		CreatedBy:       99,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, "LUCKY-CODE")
	messenger := &lotteryMessengerStub{nextID: 100}
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, messenger, now)
	svc.randFloat = func() float64 { return 0.0 }

	result, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.NoError(t, err)
	require.True(t, result.Won)
	require.NotNil(t, result.SiteMessageID)
	require.Equal(t, int64(101), *result.SiteMessageID)
	require.Equal(t, LotteryWinMessage, result.Message)
	require.Len(t, messenger.sent, 1)
	require.Equal(t, int64(99), messenger.sent[0].senderID)
	require.Equal(t, int64(7), messenger.sent[0].recipientID)
	require.Equal(t, "lucky", messenger.sent[0].campaignName)
	require.Equal(t, "LUCKY-CODE", messenger.sent[0].code)
	require.Equal(t, LotteryStatusFinished, repo.campaigns[campaign.ID].Status)
	require.Equal(t, 1, repo.campaigns[campaign.ID].JoinedCount)
	require.Equal(t, 1, repo.campaigns[campaign.ID].WinnerCount)
}

func TestLotteryServiceDrawBoostsConfiguredEarlyParticipants(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:                         "early boost",
		Subtitle:                     "subtitle",
		Status:                       LotteryStatusActive,
		PrizeCount:                   1,
		MaxParticipants:              10,
		EarlyBoostParticipantPercent: 25,
		CreatedBy:                    99,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}, "A")
	messenger := &lotteryMessengerStub{}
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, messenger, now)
	svc.randFloat = func() float64 { return 0.14 }

	result, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.NoError(t, err)
	require.True(t, result.Won)
	require.Len(t, messenger.sent, 1)
}

func TestLotteryServiceDrawReducesLateParticipantOdds(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:                         "late lower",
		Subtitle:                     "subtitle",
		Status:                       LotteryStatusActive,
		PrizeCount:                   1,
		MaxParticipants:              10,
		JoinedCount:                  8,
		EarlyBoostParticipantPercent: 25,
		CreatedBy:                    99,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}, "A")
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)
	svc.randFloat = func() float64 { return 0.4 }

	result, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.NoError(t, err)
	require.False(t, result.Won)
	require.Equal(t, 0, repo.campaigns[campaign.ID].WinnerCount)
}

func TestLotteryServiceDrawAddsRechargeWeightedOddsUpToCampaignCap(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:                         "recharge boost",
		Subtitle:                     "subtitle",
		Status:                       LotteryStatusActive,
		PrizeCount:                   1,
		MaxParticipants:              10,
		EarlyBoostParticipantPercent: 0,
		RechargeBoostCapPercent:      50,
		CreatedBy:                    99,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}, "A")
	repo.users[7] = &LotteryUserProfile{UserID: 7, TotalRecharged: 3000}
	messenger := &lotteryMessengerStub{}
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, messenger, now)
	svc.randFloat = func() float64 { return 0.35 }

	result, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.NoError(t, err)
	require.True(t, result.Won)
	require.Len(t, messenger.sent, 1)
}

func TestLotteryServiceDrawRejectsDuplicateUserDraw(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:            "duplicate",
		Subtitle:        "subtitle",
		Status:          LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 3,
		CreatedBy:       99,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, "A")
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	_, err := svc.Draw(context.Background(), 7, campaign.ID)
	require.NoError(t, err)
	_, err = svc.Draw(context.Background(), 7, campaign.ID)

	require.ErrorIs(t, err, ErrLotteryAlreadyDrawn)
}

func TestLotteryServiceDrawRollsBackWhenSiteMessageFails(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:            "rollback",
		Subtitle:        "subtitle",
		Status:          LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 1,
		CreatedBy:       99,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, "A")
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{err: errors.New("message failed")}, now)
	svc.randFloat = func() float64 { return 0.0 }

	_, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.ErrorContains(t, err, "message failed")
	require.Empty(t, repo.draws)
	require.Equal(t, 0, repo.campaigns[campaign.ID].JoinedCount)
}

func TestLotteryServiceDrawGuaranteesWinWhenPrizesCoverRemainingSlots(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:                         "full house",
		Subtitle:                     "subtitle",
		Status:                       LotteryStatusActive,
		PrizeCount:                   100,
		MaxParticipants:              100,
		JoinedCount:                  76,
		WinnerCount:                  76,
		EarlyBoostParticipantPercent: 25,
		CreatedBy:                    99,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}, "CODE-77")
	messenger := &lotteryMessengerStub{}
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, messenger, now)
	svc.randFloat = func() float64 { return 0.99 }

	result, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.NoError(t, err)
	require.True(t, result.Won)
	require.Equal(t, 77, repo.campaigns[campaign.ID].WinnerCount)
	require.Len(t, messenger.sent, 1)
}

func TestBuildLotterySegmentsStaysAtEightWhenPrizeCountIsLarge(t *testing.T) {
	segments := BuildLotterySegments(100, 8)

	require.Len(t, segments, 8)
	prizeSlots := 0
	loseSlots := 0
	for _, segment := range segments {
		if segment.IsPrize {
			prizeSlots++
		} else {
			loseSlots++
		}
	}
	require.Equal(t, 5, prizeSlots)
	require.Equal(t, 3, loseSlots)
}

func TestLotteryServiceCreateCampaignStoresHttpsPromo(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	created, err := svc.CreateCampaign(context.Background(), 99, CreateLotteryCampaignInput{
		Name:            "promo",
		PrizeCount:      1,
		MaxParticipants: 1,
		Codes:           []string{"A"},
		PromoText:       "关注公众号领福利",
		PromoImageURL:   "https://pub-xxxx.r2.dev/lottery/wechat-qr.png",
	})

	require.NoError(t, err)
	require.Equal(t, "关注公众号领福利", created.PromoText)
	require.Equal(t, "https://pub-xxxx.r2.dev/lottery/wechat-qr.png", created.PromoImageURL)
}

func TestLotteryServiceGetActiveForUserIncludesPromo(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	seedLotteryCampaign(repo, LotteryCampaign{
		Name:            "promo",
		Subtitle:        "subtitle",
		Status:          LotteryStatusActive,
		PrizeCount:      1,
		MaxParticipants: 3,
		PromoText:       "关注公众号领福利",
		PromoImageURL:   "https://cdn.example.com/qr.png",
		CreatedBy:       99,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, "A")
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	active, err := svc.GetActiveForUser(context.Background(), 7)

	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, "关注公众号领福利", active.PromoText)
	require.Equal(t, "https://cdn.example.com/qr.png", active.PromoImageURL)
	require.Len(t, active.Segments, 8)
}

func TestLotteryServiceCreateCampaignRejectsNonHttpsPromoImage(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, &lotteryMessengerStub{}, now)

	_, err := svc.CreateCampaign(context.Background(), 99, CreateLotteryCampaignInput{
		Name:            "promo",
		PrizeCount:      1,
		MaxParticipants: 1,
		Codes:           []string{"A"},
		PromoImageURL:   "http://example.com/qr.png",
	})

	require.ErrorIs(t, err, ErrLotteryInvalidCampaign)
}

func TestLotteryServiceDrawPassesPromoIntoSiteMessage(t *testing.T) {
	now := time.Unix(1779000000, 0)
	repo := newLotteryRepoStub()
	campaign := seedLotteryCampaign(repo, LotteryCampaign{
		Name:          "lucky",
		Subtitle:      "subtitle",
		Status:        LotteryStatusActive,
		PrizeCount:    1,
		MaxParticipants: 1,
		PromoText:     "关注公众号",
		PromoImageURL: "https://cdn.example.com/qr.png",
		CreatedBy:     99,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, "LUCKY-CODE")
	messenger := &lotteryMessengerStub{}
	svc := newLotteryTestService(repo, SiteMessageSettings{Enabled: true, DailySendLimit: 10, RetentionDays: 30}, messenger, now)
	svc.randFloat = func() float64 { return 0.0 }

	_, err := svc.Draw(context.Background(), 7, campaign.ID)

	require.NoError(t, err)
	require.Len(t, messenger.sent, 1)
	require.Equal(t, "关注公众号", messenger.sent[0].promoText)
	require.Equal(t, "https://cdn.example.com/qr.png", messenger.sent[0].promoImageURL)
}
