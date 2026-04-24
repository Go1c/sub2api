package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const maxPublicModelPricingRows = 20

func DefaultPublicModelPricingConfig() PublicModelPricingConfig {
	return PublicModelPricingConfig{
		Currency: "CNY",
		Unit:     "1M tokens",
		RateNote: "价格以人民币（¥）计价，单位为百万 tokens；折扣展示可在管理员后台调整。",
		Rows: []PublicModelPricingRow{
			{
				Model:          "Claude Opus 4.7",
				Group:          "企业稳定版",
				Multiplier:     "3.3%",
				InputPrice:     3.50,
				OutputPrice:    17.50,
				OfficialInput:  15,
				OfficialOutput: 75,
				Discount:       "3.3%",
				Enabled:        true,
			},
			{
				Model:          "Claude Sonnet 4.6",
				Group:          "企业稳定版",
				Multiplier:     "3.3%",
				InputPrice:     0.70,
				OutputPrice:    3.50,
				OfficialInput:  3,
				OfficialOutput: 15,
				Discount:       "3.3%",
				Enabled:        true,
			},
			{
				Model:          "GPT5.5",
				Group:          "OpenAI",
				Multiplier:     "1.0%",
				InputPrice:     0.18,
				OutputPrice:    0.70,
				OfficialInput:  2.5,
				OfficialOutput: 10,
				Discount:       "1.0%",
				Enabled:        true,
			},
			{
				Model:          "GPT5.4",
				Group:          "OpenAI",
				Multiplier:     "1.0%",
				InputPrice:     0.18,
				OutputPrice:    1.05,
				OfficialInput:  2.5,
				OfficialOutput: 15,
				Discount:       "1.0%",
				Enabled:        true,
			},
		},
	}
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
