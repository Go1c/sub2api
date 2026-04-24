package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	maxPublicModelPricingRows = 20
	publicPricingUSDToCNYRate = 7.0
)

func DefaultPublicModelPricingConfig() PublicModelPricingConfig {
	return PublicModelPricingConfig{
		Currency: "CNY",
		Unit:     "1M tokens",
		RateNote: "官方原价以美元（USD）标注；充值 ¥1 = $1 账户额度；人民币展示价和最终折扣自动计算。",
		Rows: []PublicModelPricingRow{
			{
				Model:          "Claude Opus 4.7",
				Group:          "Claude",
				Multiplier:     "1.4",
				InputPrice:     calculatePublicPricingPrice(5, 1.4),
				OutputPrice:    calculatePublicPricingPrice(25, 1.4),
				OfficialInput:  5,
				OfficialOutput: 25,
				Discount:       formatPublicPricingFinalDiscount(1.4),
				Enabled:        true,
			},
			{
				Model:          "Claude Sonnet 4.6",
				Group:          "Claude",
				Multiplier:     "1.4",
				InputPrice:     calculatePublicPricingPrice(3, 1.4),
				OutputPrice:    calculatePublicPricingPrice(15, 1.4),
				OfficialInput:  3,
				OfficialOutput: 15,
				Discount:       formatPublicPricingFinalDiscount(1.4),
				Enabled:        true,
			},
			{
				Model:          "GPT5.5",
				Group:          "OpenAI",
				Multiplier:     "0.2",
				InputPrice:     calculatePublicPricingPrice(5, 0.2),
				OutputPrice:    calculatePublicPricingPrice(30, 0.2),
				OfficialInput:  5,
				OfficialOutput: 30,
				Discount:       formatPublicPricingFinalDiscount(0.2),
				Enabled:        true,
			},
			{
				Model:          "GPT5.4",
				Group:          "OpenAI",
				Multiplier:     "0.2",
				InputPrice:     calculatePublicPricingPrice(2.5, 0.2),
				OutputPrice:    calculatePublicPricingPrice(15, 0.2),
				OfficialInput:  2.5,
				OfficialOutput: 15,
				Discount:       formatPublicPricingFinalDiscount(0.2),
				Enabled:        true,
			},
		},
	}
}

func parsePublicPricingMultiplier(value string) float64 {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return 0
	}
	normalized = strings.ReplaceAll(normalized, "倍率", "")
	normalized = strings.ReplaceAll(normalized, "倍", "")
	normalized = strings.ReplaceAll(normalized, "x", "")
	normalized = strings.ReplaceAll(normalized, "X", "")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return 0
	}

	if strings.HasSuffix(normalized, "%") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(normalized, "%")), 64)
		if err != nil || n <= 0 {
			return 0
		}
		return n / 100
	}
	if strings.HasSuffix(normalized, "折") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(normalized, "折")), 64)
		if err != nil || n <= 0 {
			return 0
		}
		return (n / 10) * publicPricingUSDToCNYRate
	}

	n, err := strconv.ParseFloat(normalized, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func calculatePublicPricingPrice(officialUSD float64, multiplier float64) float64 {
	if officialUSD <= 0 || multiplier <= 0 {
		return 0
	}
	return math.Round(officialUSD*multiplier*100) / 100
}

func formatPublicPricingFinalDiscount(multiplier float64) string {
	if multiplier <= 0 {
		return ""
	}
	return trimFloatForDisplay((multiplier/publicPricingUSDToCNYRate)*10) + "折"
}

func trimFloatForDisplay(value float64) string {
	precision := 1
	if value < 1 {
		precision = 2
	}
	formatted := strconv.FormatFloat(value, 'f', precision, 64)
	formatted = strings.TrimRight(formatted, "0")
	return strings.TrimRight(formatted, ".")
}

func NormalizePublicModelPricingConfig(cfg PublicModelPricingConfig) PublicModelPricingConfig {
	defaults := DefaultPublicModelPricingConfig()
	cfg.Currency = strings.TrimSpace(cfg.Currency)
	if cfg.Currency == "" {
		cfg.Currency = defaults.Currency
	}
	cfg.Unit = strings.TrimSpace(cfg.Unit)
	if cfg.Unit == "" {
		cfg.Unit = defaults.Unit
	}
	cfg.RateNote = strings.TrimSpace(cfg.RateNote)
	if cfg.RateNote == "" {
		cfg.RateNote = defaults.RateNote
	}

	rows := make([]PublicModelPricingRow, 0, len(cfg.Rows))
	for _, row := range cfg.Rows {
		row.Model = strings.TrimSpace(row.Model)
		row.Group = strings.TrimSpace(row.Group)
		row.Multiplier = strings.TrimSpace(row.Multiplier)
		row.Discount = strings.TrimSpace(row.Discount)
		if row.Model == "" {
			continue
		}
		if row.Group == "" {
			row.Group = "-"
		}
		if row.Multiplier == "" {
			row.Multiplier = row.Discount
		}
		if row.Discount == "" {
			row.Discount = row.Multiplier
		}
		multiplier := parsePublicPricingMultiplier(row.Multiplier)
		if row.InputPrice < 0 {
			row.InputPrice = 0
		}
		if row.OutputPrice < 0 {
			row.OutputPrice = 0
		}
		if row.OfficialInput < 0 {
			row.OfficialInput = 0
		}
		if row.OfficialOutput < 0 {
			row.OfficialOutput = 0
		}
		if multiplier > 0 {
			row.InputPrice = calculatePublicPricingPrice(row.OfficialInput, multiplier)
			row.OutputPrice = calculatePublicPricingPrice(row.OfficialOutput, multiplier)
			row.Discount = formatPublicPricingFinalDiscount(multiplier)
		}
		rows = append(rows, row)
		if len(rows) >= maxPublicModelPricingRows {
			break
		}
	}
	if len(rows) == 0 {
		rows = defaults.Rows
	}
	cfg.Rows = rows
	return cfg
}

func ValidatePublicModelPricingConfig(cfg PublicModelPricingConfig) error {
	if len(cfg.Rows) > maxPublicModelPricingRows {
		return infraerrors.BadRequest("PUBLIC_MODEL_PRICING_TOO_MANY_ROWS", fmt.Sprintf("public model pricing supports at most %d rows", maxPublicModelPricingRows))
	}
	for i, row := range cfg.Rows {
		if strings.TrimSpace(row.Model) == "" {
			return infraerrors.BadRequest("PUBLIC_MODEL_PRICING_MODEL_REQUIRED", fmt.Sprintf("model is required for pricing row #%d", i+1))
		}
		if row.InputPrice < 0 || row.OutputPrice < 0 || row.OfficialInput < 0 || row.OfficialOutput < 0 {
			return infraerrors.BadRequest("PUBLIC_MODEL_PRICING_PRICE_INVALID", fmt.Sprintf("prices must be non-negative for pricing row #%d", i+1))
		}
	}
	return nil
}

func (s *SettingService) GetPublicModelPricing(ctx context.Context) (*PublicModelPricingConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyPublicModelPricing)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			cfg := DefaultPublicModelPricingConfig()
			return &cfg, nil
		}
		return nil, fmt.Errorf("get public model pricing: %w", err)
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		cfg := DefaultPublicModelPricingConfig()
		return &cfg, nil
	}

	var cfg PublicModelPricingConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		cfg := DefaultPublicModelPricingConfig()
		return &cfg, nil
	}
	cfg = NormalizePublicModelPricingConfig(cfg)
	return &cfg, nil
}

func (s *SettingService) SetPublicModelPricing(ctx context.Context, cfg PublicModelPricingConfig) (*PublicModelPricingConfig, error) {
	if err := ValidatePublicModelPricingConfig(cfg); err != nil {
		return nil, err
	}
	cfg = NormalizePublicModelPricingConfig(cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal public model pricing: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyPublicModelPricing, string(data)); err != nil {
		return nil, fmt.Errorf("set public model pricing: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return &cfg, nil
}

func (cfg PublicModelPricingConfig) EnabledRows() PublicModelPricingConfig {
	cfg = NormalizePublicModelPricingConfig(cfg)
	rows := make([]PublicModelPricingRow, 0, len(cfg.Rows))
	for _, row := range cfg.Rows {
		if row.Enabled {
			rows = append(rows, row)
		}
	}
	cfg.Rows = rows
	return cfg
}
