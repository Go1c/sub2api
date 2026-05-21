package domain

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

const (
	LotteryStatusActive   = "active"
	LotteryStatusFinished = "finished"
)

var (
	ErrLotteryCampaignNotFound = infraerrors.NotFound(
		"LOTTERY_CAMPAIGN_NOT_FOUND",
		"lottery campaign not found",
	)
	ErrLotteryDrawNotFound = infraerrors.NotFound(
		"LOTTERY_DRAW_NOT_FOUND",
		"lottery draw not found",
	)
	ErrLotteryInvalidCampaign = infraerrors.BadRequest(
		"LOTTERY_CAMPAIGN_INVALID",
		"lottery campaign is invalid",
	)
	ErrLotteryDuplicateCodes = infraerrors.BadRequest(
		"LOTTERY_DUPLICATE_CODES",
		"lottery redeem codes must be unique",
	)
	ErrLotterySiteMessagesDisabled = infraerrors.BadRequest(
		"LOTTERY_SITE_MESSAGES_DISABLED",
		"site messages must be enabled before starting a lottery campaign",
	)
	ErrLotteryCampaignClosed = infraerrors.Conflict(
		"LOTTERY_CAMPAIGN_CLOSED",
		"lottery campaign is closed",
	)
	ErrLotteryAlreadyDrawn = infraerrors.Conflict(
		"LOTTERY_ALREADY_DRAWN",
		"user has already drawn in this campaign",
	)
	ErrLotteryNoCodeAvailable = infraerrors.Conflict(
		"LOTTERY_NO_CODE_AVAILABLE",
		"no lottery code is available",
	)
)
