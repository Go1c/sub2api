package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

var affiliateRebateTierLevels = []string{"L1", "L2", "L3", "L4"}

type AffiliateRebateTier struct {
	Level             string   `json:"level"`
	MinInvitees       int      `json:"min_invitees"`
	MinRecharge       float64  `json:"min_recharge"`
	RebateRatePercent *float64 `json:"rebate_rate_percent"`
}

func defaultAffiliateRebateTiers() []AffiliateRebateTier {
	tiers := make([]AffiliateRebateTier, 0, len(affiliateRebateTierLevels))
	for _, level := range affiliateRebateTierLevels {
		tiers = append(tiers, AffiliateRebateTier{Level: level})
	}
	return tiers
}

func normalizeAffiliateRebateTiers(input []AffiliateRebateTier) []AffiliateRebateTier {
	byLevel := make(map[string]AffiliateRebateTier, len(input))
	for _, tier := range input {
		level := strings.ToUpper(strings.TrimSpace(tier.Level))
		if level == "" {
			continue
		}
		tier.Level = level
		byLevel[level] = tier
	}

	normalized := defaultAffiliateRebateTiers()
	for i := range normalized {
		if tier, ok := byLevel[normalized[i].Level]; ok {
			normalized[i] = normalizeAffiliateRebateTier(normalized[i].Level, tier)
		}
	}
	return normalized
}

func normalizeAffiliateRebateTier(level string, tier AffiliateRebateTier) AffiliateRebateTier {
	if tier.MinInvitees < 0 {
		tier.MinInvitees = 0
	}
	if tier.MinRecharge < 0 || math.IsNaN(tier.MinRecharge) || math.IsInf(tier.MinRecharge, 0) {
		tier.MinRecharge = 0
	}
	if tier.RebateRatePercent != nil {
		rate := clampAffiliateRebateRate(*tier.RebateRatePercent)
		tier.RebateRatePercent = &rate
	}
	tier.Level = level
	return tier
}

func marshalAffiliateRebateTiers(tiers []AffiliateRebateTier) (string, error) {
	payload, err := json.Marshal(normalizeAffiliateRebateTiers(tiers))
	if err != nil {
		return "", fmt.Errorf("marshal affiliate rebate tiers: %w", err)
	}
	return string(payload), nil
}

func parseAffiliateRebateTiers(raw string) []AffiliateRebateTier {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAffiliateRebateTiers()
	}
	var tiers []AffiliateRebateTier
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return defaultAffiliateRebateTiers()
	}
	return normalizeAffiliateRebateTiers(tiers)
}

func affiliateTierRateConfigured(tier AffiliateRebateTier) bool {
	return tier.RebateRatePercent != nil
}

func resolveAffiliateTier(tiers []AffiliateRebateTier, invitees int, rechargeTotal float64) *AffiliateRebateTier {
	normalized := normalizeAffiliateRebateTiers(tiers)
	var current *AffiliateRebateTier
	for i := range normalized {
		tier := normalized[i]
		if !affiliateTierRateConfigured(tier) {
			continue
		}
		if invitees >= tier.MinInvitees && rechargeTotal+1e-9 >= tier.MinRecharge {
			cp := tier
			current = &cp
		}
	}
	return current
}

func nextAffiliateTier(tiers []AffiliateRebateTier, current *AffiliateRebateTier) *AffiliateRebateTier {
	normalized := normalizeAffiliateRebateTiers(tiers)
	currentIndex := -1
	if current != nil {
		for i := range normalized {
			if normalized[i].Level == current.Level {
				currentIndex = i
				break
			}
		}
	}
	for i := currentIndex + 1; i < len(normalized); i++ {
		if !affiliateTierRateConfigured(normalized[i]) {
			continue
		}
		cp := normalized[i]
		return &cp
	}
	return nil
}
