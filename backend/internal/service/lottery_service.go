package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
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
	earlyBoostParticipantPercent := LotteryDefaultEarlyBoostParticipantPercent
	if input.EarlyBoostParticipantPercent != nil {
		earlyBoostParticipantPercent = *input.EarlyBoostParticipantPercent
	}
	if earlyBoostParticipantPercent < 0 ||
		earlyBoostParticipantPercent > 100 ||
		input.RechargeBoostCapPercent < 0 ||
		input.RechargeBoostCapPercent > LotteryMaxRechargeBoostCapPercent {
		return nil, ErrLotteryInvalidCampaign
	}
	promoText, promoImageURL, err := normalizeLotteryPromo(input.PromoText, input.PromoImageURL)
	if err != nil {
		return nil, err
	}

	now := s.now()
	var created *LotteryCampaign
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.FinishActiveCampaigns(txCtx, now); err != nil {
			return err
		}
		campaign := &LotteryCampaign{
			Name:                         name,
			Subtitle:                     subtitle,
			Status:                       LotteryStatusActive,
			PrizeCount:                   input.PrizeCount,
			MaxParticipants:              input.MaxParticipants,
			EarlyBoostParticipantPercent: earlyBoostParticipantPercent,
			RechargeBoostCapPercent:      input.RechargeBoostCapPercent,
			PromoText:                    promoText,
			PromoImageURL:                promoImageURL,
			CreatedBy:                    adminID,
			CreatedAt:                    now,
			UpdatedAt:                    now,
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
		ID:                           campaign.ID,
		Name:                         campaign.Name,
		Subtitle:                     campaign.Subtitle,
		PrizeCount:                   campaign.PrizeCount,
		MaxParticipants:              campaign.MaxParticipants,
		JoinedCount:                  campaign.JoinedCount,
		EarlyBoostParticipantPercent: campaign.EarlyBoostParticipantPercent,
		RechargeBoostCapPercent:      campaign.RechargeBoostCapPercent,
		PromoText:                    campaign.PromoText,
		PromoImageURL:                campaign.PromoImageURL,
		Segments:                     BuildLotterySegments(campaign.PrizeCount, 8),
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
		winProbability, err := s.winProbability(txCtx, campaign, userID, remainingPrizes, remainingSlots)
		if err != nil {
			return err
		}
		won := remainingPrizes > 0 && remainingSlots > 0 && s.randFloat() < winProbability

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
			msg, err := s.siteMessages.SendLotteryPrize(txCtx, campaign.CreatedBy, userID, campaign.Name, code.Code, campaign.PromoText, campaign.PromoImageURL)
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

func (s *LotteryService) winProbability(ctx context.Context, campaign *LotteryCampaign, userID int64, remainingPrizes, remainingSlots int) (float64, error) {
	if campaign == nil || remainingPrizes <= 0 || remainingSlots <= 0 {
		return 0, nil
	}
	if remainingPrizes >= remainingSlots {
		return 1, nil
	}

	probability := float64(remainingPrizes) / float64(remainingSlots)
	probability *= lotteryParticipantPhaseMultiplier(
		campaign.JoinedCount,
		campaign.MaxParticipants,
		campaign.EarlyBoostParticipantPercent,
	)

	if campaign.RechargeBoostCapPercent > 0 {
		profile, err := s.repo.GetUserLotteryProfile(ctx, userID)
		if err != nil {
			return 0, err
		}
		if profile != nil {
			probability += lotteryRechargeBoost(profile.TotalRecharged, campaign.RechargeBoostCapPercent)
		}
	}

	return clampLotteryProbability(probability), nil
}

func lotteryParticipantPhaseMultiplier(joinedCount, maxParticipants, earlyPercent int) float64 {
	if maxParticipants <= 0 || earlyPercent <= 0 {
		return 1
	}
	if earlyPercent >= 100 {
		return 1.5
	}

	progress := float64(joinedCount) / float64(maxParticipants)
	earlyCutoff := float64(earlyPercent) / 100
	if progress < earlyCutoff {
		return 1.5
	}

	lateProgress := (progress - earlyCutoff) / (1 - earlyCutoff)
	if lateProgress < 0 {
		lateProgress = 0
	}
	if lateProgress > 1 {
		lateProgress = 1
	}
	return 1 - 0.5*lateProgress
}

func lotteryRechargeBoost(totalRecharged float64, capPercent int) float64 {
	if totalRecharged <= 0 || capPercent <= 0 {
		return 0
	}
	if capPercent > LotteryMaxRechargeBoostCapPercent {
		capPercent = LotteryMaxRechargeBoostCapPercent
	}
	boost := totalRecharged / 10000
	capProbability := float64(capPercent) / 100
	if boost > capProbability {
		return capProbability
	}
	return boost
}

func clampLotteryProbability(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func normalizeLotteryPromo(rawText, rawImageURL string) (string, string, error) {
	text := strings.TrimSpace(rawText)
	if len([]rune(text)) > 240 {
		return "", "", ErrLotteryInvalidCampaign
	}
	imageURL := strings.TrimSpace(rawImageURL)
	if imageURL == "" {
		return text, "", nil
	}
	if len(imageURL) > 2048 {
		return "", "", ErrLotteryInvalidCampaign
	}
	parsed, err := url.Parse(imageURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", "", ErrLotteryInvalidCampaign
	}
	return text, parsed.String(), nil
}

func BuildLotterySegments(prizeCount int, totalSlots int) []LotterySegment {
	_ = prizeCount
	slots := totalSlots
	if slots != 8 {
		slots = 8
	}
	pattern := []bool{true, false, true, true, false, true, false, true}
	segs := make([]LotterySegment, 0, slots)
	for i := 0; i < slots; i++ {
		if pattern[i] {
			segs = append(segs, LotterySegment{Label: LotteryPrizeLabel, IsPrize: true})
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
