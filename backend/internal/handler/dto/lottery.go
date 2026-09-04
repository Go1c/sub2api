package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type LotterySegment struct {
	Label   string `json:"label"`
	IsPrize bool   `json:"is_prize"`
}

type LotteryActiveCampaign struct {
	ID                           int64            `json:"id"`
	Name                         string           `json:"name"`
	Subtitle                     string           `json:"subtitle"`
	PrizeCount                   int              `json:"prize_count"`
	MaxParticipants              int              `json:"max_participants"`
	JoinedCount                  int              `json:"joined_count"`
	EarlyBoostParticipantPercent int              `json:"early_boost_participant_percent"`
	RechargeBoostCapPercent      int              `json:"recharge_boost_cap_percent"`
	PromoText                    string           `json:"promo_text"`
	PromoImageURL                string           `json:"promo_image_url"`
	Segments                     []LotterySegment `json:"segments"`
}

type LotteryActiveResponse struct {
	Campaign *LotteryActiveCampaign `json:"campaign"`
}

type LotteryDrawResult struct {
	Won           bool   `json:"won"`
	Index         int    `json:"index"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	SiteMessageID *int64 `json:"site_message_id,omitempty"`
}

type LotteryCampaign struct {
	ID                           int64         `json:"id"`
	Name                         string        `json:"name"`
	Subtitle                     string        `json:"subtitle"`
	Status                       string        `json:"status"`
	PrizeCount                   int           `json:"prize_count"`
	MaxParticipants              int           `json:"max_participants"`
	JoinedCount                  int           `json:"joined_count"`
	WinnerCount                  int           `json:"winner_count"`
	EarlyBoostParticipantPercent int           `json:"early_boost_participant_percent"`
	RechargeBoostCapPercent      int           `json:"recharge_boost_cap_percent"`
	PromoText                    string        `json:"promo_text"`
	PromoImageURL                string        `json:"promo_image_url"`
	CreatedBy                    int64         `json:"created_by"`
	CreatedAt                    time.Time     `json:"created_at"`
	UpdatedAt                    time.Time     `json:"updated_at"`
	FinishedAt                   *time.Time    `json:"finished_at,omitempty"`
	Codes                        []LotteryCode `json:"codes,omitempty"`
	Draws                        []LotteryDraw `json:"draws,omitempty"`
}

type LotteryCode struct {
	ID             int64      `json:"id"`
	CampaignID     int64      `json:"campaign_id"`
	Code           string     `json:"code"`
	AssignedUserID *int64     `json:"assigned_user_id,omitempty"`
	AssignedDrawID *int64     `json:"assigned_draw_id,omitempty"`
	AssignedAt     *time.Time `json:"assigned_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type LotteryDraw struct {
	ID            int64     `json:"id"`
	CampaignID    int64     `json:"campaign_id"`
	UserID        int64     `json:"user_id"`
	UserEmail     string    `json:"user_email,omitempty"`
	Won           bool      `json:"won"`
	LotteryCodeID *int64    `json:"lottery_code_id,omitempty"`
	SiteMessageID *int64    `json:"site_message_id,omitempty"`
	ResultLabel   string    `json:"result_label"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateLotteryCampaignRequest struct {
	Name                         string   `json:"name" binding:"required"`
	Subtitle                     string   `json:"subtitle"`
	PrizeCount                   int      `json:"prize_count" binding:"required"`
	MaxParticipants              int      `json:"max_participants" binding:"required"`
	EarlyBoostParticipantPercent *int     `json:"early_boost_participant_percent"`
	RechargeBoostCapPercent      int      `json:"recharge_boost_cap_percent"`
	PromoText                    string   `json:"promo_text"`
	PromoImageURL                string   `json:"promo_image_url"`
	Codes                        []string `json:"codes" binding:"required"`
}

func LotteryActiveCampaignFromService(c *service.LotteryActiveCampaign) *LotteryActiveCampaign {
	if c == nil {
		return nil
	}
	return &LotteryActiveCampaign{
		ID:                           c.ID,
		Name:                         c.Name,
		Subtitle:                     c.Subtitle,
		PrizeCount:                   c.PrizeCount,
		MaxParticipants:              c.MaxParticipants,
		JoinedCount:                  c.JoinedCount,
		EarlyBoostParticipantPercent: c.EarlyBoostParticipantPercent,
		RechargeBoostCapPercent:      c.RechargeBoostCapPercent,
		PromoText:                    c.PromoText,
		PromoImageURL:                c.PromoImageURL,
		Segments:                     LotterySegmentsFromService(c.Segments),
	}
}

func LotterySegmentsFromService(items []service.LotterySegment) []LotterySegment {
	out := make([]LotterySegment, 0, len(items))
	for _, item := range items {
		out = append(out, LotterySegment{Label: item.Label, IsPrize: item.IsPrize})
	}
	return out
}

func LotteryDrawResultFromService(r *service.LotteryDrawResult) *LotteryDrawResult {
	if r == nil {
		return nil
	}
	return &LotteryDrawResult{
		Won:           r.Won,
		Index:         r.Index,
		Label:         r.Label,
		Message:       r.Message,
		SiteMessageID: r.SiteMessageID,
	}
}

func LotteryCampaignFromService(c *service.LotteryCampaign) *LotteryCampaign {
	if c == nil {
		return nil
	}
	return &LotteryCampaign{
		ID:                           c.ID,
		Name:                         c.Name,
		Subtitle:                     c.Subtitle,
		Status:                       c.Status,
		PrizeCount:                   c.PrizeCount,
		MaxParticipants:              c.MaxParticipants,
		JoinedCount:                  c.JoinedCount,
		WinnerCount:                  c.WinnerCount,
		EarlyBoostParticipantPercent: c.EarlyBoostParticipantPercent,
		RechargeBoostCapPercent:      c.RechargeBoostCapPercent,
		PromoText:                    c.PromoText,
		PromoImageURL:                c.PromoImageURL,
		CreatedBy:                    c.CreatedBy,
		CreatedAt:                    c.CreatedAt,
		UpdatedAt:                    c.UpdatedAt,
		FinishedAt:                   c.FinishedAt,
		Codes:                        LotteryCodesFromService(c.Codes),
		Draws:                        LotteryDrawsFromService(c.Draws),
	}
}

func LotteryCampaignsFromService(items []service.LotteryCampaign) []LotteryCampaign {
	out := make([]LotteryCampaign, 0, len(items))
	for i := range items {
		if converted := LotteryCampaignFromService(&items[i]); converted != nil {
			converted.Codes = nil
			converted.Draws = nil
			out = append(out, *converted)
		}
	}
	return out
}

func LotteryCodesFromService(items []service.LotteryCode) []LotteryCode {
	out := make([]LotteryCode, 0, len(items))
	for _, item := range items {
		out = append(out, LotteryCode{
			ID:             item.ID,
			CampaignID:     item.CampaignID,
			Code:           item.Code,
			AssignedUserID: item.AssignedUserID,
			AssignedDrawID: item.AssignedDrawID,
			AssignedAt:     item.AssignedAt,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}
	return out
}

func LotteryDrawsFromService(items []service.LotteryDraw) []LotteryDraw {
	out := make([]LotteryDraw, 0, len(items))
	for _, item := range items {
		converted := LotteryDraw{
			ID:            item.ID,
			CampaignID:    item.CampaignID,
			UserID:        item.UserID,
			Won:           item.Won,
			LotteryCodeID: item.LotteryCodeID,
			SiteMessageID: item.SiteMessageID,
			ResultLabel:   item.ResultLabel,
			CreatedAt:     item.CreatedAt,
		}
		if item.User != nil {
			converted.UserEmail = item.User.Email
		}
		out = append(out, converted)
	}
	return out
}
