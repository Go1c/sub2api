package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type LotteryService struct {
	repo         LotteryRepository
	settings     LotterySettingsReader
	siteMessages LotteryPrizeMessenger
	now          func() time.Time
	randFloat    func() float64
}

func NewLotteryService(
	repo LotteryRepository,
	settings LotterySettingsReader,
	siteMessages LotteryPrizeMessenger,
) *LotteryService {
	return &LotteryService{
		repo:         repo,
		settings:     settings,
		siteMessages: siteMessages,
		now:          time.Now,
		randFloat:    rand.Float64,
	}
}

func (s *LotteryService) CreateCampaign(ctx context.Context, adminID int64, input CreateLotteryCampaignInput) (*LotteryCampaign, error) {
	if err := s.ensureSiteMessagesEnabled(ctx); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	subtitle := strings.TrimSpace(input.Subtitle)
	if subtitle == "" {
		subtitle = LotteryDefaultSubtitle
	}
	codes, err := normalizeLotteryCodes(input.Codes, input.PrizeCount)
	if err != nil {
		return nil, err
	}
	if name == "" || len([]rune(name)) > 120 ||
		len([]rune(subtitle)) > 240 ||
		input.PrizeCount < 1 ||
		input.MaxParticipants < input.PrizeCount {
		return nil, ErrLotteryInvalidCampaign
	}

	now := s.now()
	var created *LotteryCampaign
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.FinishActiveCampaigns(txCtx, now); err != nil {
			return err
		}
		campaign := &LotteryCampaign{
			Name:            name,
			Subtitle:        subtitle,
			Status:          LotteryStatusActive,
			PrizeCount:      input.PrizeCount,
			MaxParticipants: input.MaxParticipants,
			CreatedBy:       adminID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		codeRows := make([]LotteryCode, 0, len(codes))
		for _, code := range codes {
			codeRows = append(codeRows, LotteryCode{
				Code:      code,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if err := s.repo.CreateCampaign(txCtx, campaign, codeRows); err != nil {
			return err
		}
		created = campaign
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *LotteryService) ListCampaigns(ctx context.Context, params pagination.PaginationParams) ([]LotteryCampaign, *pagination.PaginationResult, error) {
	return s.repo.ListCampaigns(ctx, params)
}

func (s *LotteryService) GetCampaign(ctx context.Context, id int64) (*LotteryCampaign, error) {
	return s.repo.GetCampaign(ctx, id)
}

func (s *LotteryService) FinishCampaign(ctx context.Context, _ int64, id int64) (*LotteryCampaign, error) {
	now := s.now()
	var out *LotteryCampaign
	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		campaign, err := s.repo.GetCampaignForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if campaign.Status == LotteryStatusFinished {
			out = campaign
			return nil
		}
		finished, err := s.repo.IncrementCampaignCounters(txCtx, id, 0, 0, true, &now)
		if err != nil {
			return err
		}
		out = finished
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *LotteryService) GetActiveForUser(ctx context.Context, userID int64) (*LotteryActiveCampaign, error) {
	campaign, err := s.repo.GetActiveCampaign(ctx)
	if err != nil {
		if errors.Is(err, ErrLotteryCampaignNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if campaign.JoinedCount >= campaign.MaxParticipants {
		return nil, nil
	}
	if _, err := s.repo.GetDrawByCampaignAndUser(ctx, campaign.ID, userID); err == nil {
		return nil, nil
	} else if !errors.Is(err, ErrLotteryDrawNotFound) {
		return nil, err
	}
	return &LotteryActiveCampaign{
		ID:              campaign.ID,
		Name:            campaign.Name,
		Subtitle:        campaign.Subtitle,
		PrizeCount:      campaign.PrizeCount,
		MaxParticipants: campaign.MaxParticipants,
		JoinedCount:     campaign.JoinedCount,
		Segments:        BuildLotterySegments(campaign.PrizeCount, 8),
	}, nil
}

func (s *LotteryService) Draw(ctx context.Context, userID, campaignID int64) (*LotteryDrawResult, error) {
	var result *LotteryDrawResult
	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		campaign, err := s.repo.GetCampaignForUpdate(txCtx, campaignID)
		if err != nil {
			return err
		}
		if campaign.Status != LotteryStatusActive || campaign.JoinedCount >= campaign.MaxParticipants {
			return ErrLotteryCampaignClosed
		}
		if _, err := s.repo.GetDrawByCampaignAndUser(txCtx, campaignID, userID); err == nil {
			return ErrLotteryAlreadyDrawn
		} else if !errors.Is(err, ErrLotteryDrawNotFound) {
			return err
		}

		segments := BuildLotterySegments(campaign.PrizeCount, 8)
		remainingPrizes := campaign.PrizeCount - campaign.WinnerCount
		remainingSlots := campaign.MaxParticipants - campaign.JoinedCount
		won := remainingPrizes > 0 && remainingSlots > 0 && s.randFloat() < float64(remainingPrizes)/float64(remainingSlots)

		var code *LotteryCode
		var codeID *int64
		var siteMessageID *int64
		label := LotteryLoseLabel
		message := LotteryLoseMessage
		index := firstLotterySegmentIndex(segments, false)

		if won {
			code, err = s.repo.PickUnassignedCode(txCtx, campaignID)
			if err != nil {
				if !errors.Is(err, ErrLotteryNoCodeAvailable) {
					return err
				}
				won = false
			}
		}
		if won && code != nil {
			if s.siteMessages == nil {
				return fmt.Errorf("lottery prize messenger is not configured")
			}
			msg, err := s.siteMessages.SendLotteryPrize(txCtx, campaign.CreatedBy, userID, campaign.Name, code.Code)
			if err != nil {
				return err
			}
			codeID = &code.ID
			if msg != nil {
				siteMessageID = &msg.ID
			}
			label = firstLotteryPrizeLabel(segments)
			message = LotteryWinMessage
			index = firstLotterySegmentIndex(segments, true)
		}

		now := s.now()
		draw := &LotteryDraw{
			CampaignID:    campaignID,
			UserID:        userID,
			Won:           won,
			LotteryCodeID: codeID,
			SiteMessageID: siteMessageID,
			ResultLabel:   label,
			CreatedAt:     now,
		}
		if err := s.repo.CreateDraw(txCtx, draw); err != nil {
			if errors.Is(err, ErrLotteryAlreadyDrawn) {
				return ErrLotteryAlreadyDrawn
			}
			return err
		}
		if won && code != nil {
			if err := s.repo.AssignCodeToDraw(txCtx, code.ID, userID, draw.ID, now); err != nil {
				return err
			}
		}
		joinedCount := campaign.JoinedCount + 1
		finish := joinedCount >= campaign.MaxParticipants
		var finishedAt *time.Time
		if finish {
			finishedAt = &now
		}
		winnerDelta := 0
		if won {
			winnerDelta = 1
		}
		if _, err := s.repo.IncrementCampaignCounters(txCtx, campaignID, 1, winnerDelta, finish, finishedAt); err != nil {
			return err
		}
		result = &LotteryDrawResult{
			Won:           won,
			Index:         index,
			Label:         label,
			Message:       message,
			SiteMessageID: siteMessageID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *LotteryService) ensureSiteMessagesEnabled(ctx context.Context) error {
	if s.settings == nil {
		return ErrLotterySiteMessagesDisabled
	}
	settings, err := s.settings.GetSiteMessageSettings(ctx)
	if err != nil {
		return err
	}
	if !normalizeSiteMessageSettings(settings).Enabled {
		return ErrLotterySiteMessagesDisabled
	}
	return nil
}

func normalizeLotteryCodes(rawCodes []string, prizeCount int) ([]string, error) {
	if prizeCount < 1 || len(rawCodes) < prizeCount {
		return nil, ErrLotteryInvalidCampaign
	}
	out := make([]string, 0, prizeCount)
	seen := make(map[string]struct{}, prizeCount)
	for _, raw := range rawCodes {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		key := strings.ToUpper(code)
		if _, ok := seen[key]; ok {
			return nil, ErrLotteryDuplicateCodes
		}
		seen[key] = struct{}{}
		out = append(out, code)
		if len(out) == prizeCount {
			break
		}
	}
	if len(out) < prizeCount {
		return nil, ErrLotteryInvalidCampaign
	}
	return out, nil
}

func BuildLotterySegments(prizeCount int, totalSlots int) []LotterySegment {
	slots := totalSlots
	if prizeCount+2 > slots {
		slots = prizeCount + 2
	}
	segs := make([]LotterySegment, 0, slots)
	for i := 0; i < slots; i++ {
		if i < prizeCount {
			segs = append(segs, LotterySegment{Label: fmt.Sprintf("%s %d", LotteryPrizeLabel, i+1), IsPrize: true})
		} else {
			segs = append(segs, LotterySegment{Label: LotteryLoseLabel, IsPrize: false})
		}
	}
	order := make([]int, slots)
	for i := range order {
		order[i] = i
	}
	for i := len(order) - 1; i > 0; i-- {
		j := int(float64((i*9301+49297)%233280) / 233280.0 * float64(i+1))
		order[i], order[j] = order[j], order[i]
	}
	out := make([]LotterySegment, 0, slots)
	for _, idx := range order {
		out = append(out, segs[idx])
	}
	return out
}

func firstLotterySegmentIndex(segments []LotterySegment, prize bool) int {
	for i, seg := range segments {
		if seg.IsPrize == prize {
			return i
		}
	}
	return 0
}

func firstLotteryPrizeLabel(segments []LotterySegment) string {
	for _, seg := range segments {
		if seg.IsPrize {
			return seg.Label
		}
	}
	return LotteryPrizeLabel
}
