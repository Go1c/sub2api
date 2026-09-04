package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	LotteryStatusActive   = domain.LotteryStatusActive
	LotteryStatusFinished = domain.LotteryStatusFinished

	LotteryDefaultSubtitle = "登录就有机会，转一转赢取兑换码"
	LotteryPrizeLabel      = "奖品"
	LotteryLoseLabel       = "谢谢参与"
	LotteryWinMessage      = "恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。"
	LotteryLoseMessage     = "很遗憾，这次没有中奖。"

	LotteryDefaultEarlyBoostParticipantPercent = 25
	LotteryMaxRechargeBoostCapPercent          = 50
)

var (
	ErrLotteryCampaignNotFound     = domain.ErrLotteryCampaignNotFound
	ErrLotteryDrawNotFound         = domain.ErrLotteryDrawNotFound
	ErrLotteryInvalidCampaign      = domain.ErrLotteryInvalidCampaign
	ErrLotteryDuplicateCodes       = domain.ErrLotteryDuplicateCodes
	ErrLotterySiteMessagesDisabled = domain.ErrLotterySiteMessagesDisabled
	ErrLotteryCampaignClosed       = domain.ErrLotteryCampaignClosed
	ErrLotteryAlreadyDrawn         = domain.ErrLotteryAlreadyDrawn
	ErrLotteryNoCodeAvailable      = domain.ErrLotteryNoCodeAvailable
)

type LotterySegment struct {
	Label   string `json:"label"`
	IsPrize bool   `json:"is_prize"`
}

type LotteryCampaign struct {
	ID              int64
	Name            string
	Subtitle        string
	Status          string
	PrizeCount      int
	MaxParticipants int
	JoinedCount     int
	WinnerCount     int
	// EarlyBoostParticipantPercent is the first N percent of participants who receive boosted odds.
	EarlyBoostParticipantPercent int
	RechargeBoostCapPercent      int
	PromoText                    string
	PromoImageURL                string
	CreatedBy                    int64
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
	FinishedAt                   *time.Time
	Codes                        []LotteryCode
	Draws                        []LotteryDraw
}

type LotteryCode struct {
	ID             int64
	CampaignID     int64
	Code           string
	AssignedUserID *int64
	AssignedDrawID *int64
	AssignedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LotteryDraw struct {
	ID            int64
	CampaignID    int64
	UserID        int64
	Won           bool
	LotteryCodeID *int64
	SiteMessageID *int64
	ResultLabel   string
	CreatedAt     time.Time
	User          *User
	Code          *LotteryCode
}

type CreateLotteryCampaignInput struct {
	Name                         string
	Subtitle                     string
	PrizeCount                   int
	MaxParticipants              int
	EarlyBoostParticipantPercent *int
	RechargeBoostCapPercent      int
	PromoText                    string
	PromoImageURL                string
	Codes                        []string
}

type LotteryActiveCampaign struct {
	ID                           int64
	Name                         string
	Subtitle                     string
	PrizeCount                   int
	MaxParticipants              int
	JoinedCount                  int
	EarlyBoostParticipantPercent int
	RechargeBoostCapPercent      int
	PromoText                    string
	PromoImageURL                string
	Segments                     []LotterySegment
}

type LotteryDrawResult struct {
	Won           bool
	Index         int
	Label         string
	Message       string
	SiteMessageID *int64
}

type LotteryUserProfile struct {
	UserID         int64
	TotalRecharged float64
}

type LotteryRepository interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
	FinishActiveCampaigns(ctx context.Context, finishedAt time.Time) error
	CreateCampaign(ctx context.Context, campaign *LotteryCampaign, codes []LotteryCode) error
	ListCampaigns(ctx context.Context, params pagination.PaginationParams) ([]LotteryCampaign, *pagination.PaginationResult, error)
	GetCampaign(ctx context.Context, id int64) (*LotteryCampaign, error)
	GetActiveCampaign(ctx context.Context) (*LotteryCampaign, error)
	GetCampaignForUpdate(ctx context.Context, id int64) (*LotteryCampaign, error)
	GetDrawByCampaignAndUser(ctx context.Context, campaignID, userID int64) (*LotteryDraw, error)
	GetUserLotteryProfile(ctx context.Context, userID int64) (*LotteryUserProfile, error)
	PickUnassignedCode(ctx context.Context, campaignID int64) (*LotteryCode, error)
	CreateDraw(ctx context.Context, draw *LotteryDraw) error
	AssignCodeToDraw(ctx context.Context, codeID, userID, drawID int64, assignedAt time.Time) error
	IncrementCampaignCounters(ctx context.Context, campaignID int64, joinedDelta, winnerDelta int, finish bool, finishedAt *time.Time) (*LotteryCampaign, error)
}

type LotterySettingsReader interface {
	GetSiteMessageSettings(ctx context.Context) (SiteMessageSettings, error)
}

type LotteryPrizeMessenger interface {
	SendLotteryPrize(ctx context.Context, senderID, recipientID int64, campaignName, code, promoText, promoImageURL string) (*SiteMessage, error)
}
