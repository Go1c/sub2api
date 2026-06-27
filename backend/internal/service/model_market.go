package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	maxModelMarketSelections = 1000
	maxModelMarketCustom     = 200
	defaultModelMarketTitle  = "模型广场"
	defaultModelMarketDesc   = "按平台、分组和计费类型查看当前可用模型。价格和倍率实时读取渠道配置。"
	modelMarketCustomChannel = "自定义"
)

type ModelMarketChannelLister interface {
	ListAvailable(ctx context.Context) ([]AvailableChannel, error)
}

type ModelMarketService struct {
	settingService *SettingService
	channelLister  ModelMarketChannelLister
}

func NewModelMarketService(settingService *SettingService, channelLister ModelMarketChannelLister) *ModelMarketService {
	return &ModelMarketService{
		settingService: settingService,
		channelLister:  channelLister,
	}
}

type ModelMarketConfig struct {
	Enabled        bool                     `json:"enabled"`
	AutoSync       bool                     `json:"auto_sync"`
	Title          string                   `json:"title"`
	Description    string                   `json:"description"`
	SelectedModels []ModelMarketSelection   `json:"selected_models"`
	CustomModels   []ModelMarketCustomModel `json:"custom_models"`
}

type ModelMarketSelection struct {
	Key       string `json:"key"`
	Platform  string `json:"platform,omitempty"`
	Model     string `json:"model,omitempty"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

type ModelMarketCustomModel struct {
	Key         string              `json:"key"`
	Platform    string              `json:"platform"`
	Model       string              `json:"model"`
	Enabled     bool                `json:"enabled"`
	SortOrder   int                 `json:"sort_order"`
	BillingMode string              `json:"billing_mode"`
	Pricing     *ModelMarketPricing `json:"pricing"`
	Groups      []ModelMarketGroup  `json:"groups"`
}

type ModelMarketResponse struct {
	Enabled     bool               `json:"enabled"`
	AutoSync    bool               `json:"auto_sync"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Models      []ModelMarketModel `json:"models"`
}

type ModelMarketAdminResponse struct {
	Config     ModelMarketConfig  `json:"config"`
	Candidates []ModelMarketModel `json:"candidates"`
	Models     []ModelMarketModel `json:"models"`
}

type ModelMarketModel struct {
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	Platform    string              `json:"platform"`
	BillingMode string              `json:"billing_mode"`
	Pricing     *ModelMarketPricing `json:"pricing"`
	Groups      []ModelMarketGroup  `json:"groups"`
	Channels    []string            `json:"channels"`
	SortOrder   int                 `json:"sort_order"`
}

type ModelMarketGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	SubscriptionType string  `json:"subscription_type"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	IsExclusive      bool    `json:"is_exclusive"`
}

type ModelMarketPricing struct {
	BillingMode      string                       `json:"billing_mode"`
	InputPrice       *float64                     `json:"input_price"`
	OutputPrice      *float64                     `json:"output_price"`
	CacheWritePrice  *float64                     `json:"cache_write_price"`
	CacheReadPrice   *float64                     `json:"cache_read_price"`
	ImageOutputPrice *float64                     `json:"image_output_price"`
	PerRequestPrice  *float64                     `json:"per_request_price"`
	Intervals        []ModelMarketPricingInterval `json:"intervals"`
}

type ModelMarketPricingInterval struct {
	MinTokens       int      `json:"min_tokens"`
	MaxTokens       *int     `json:"max_tokens"`
	TierLabel       string   `json:"tier_label,omitempty"`
	InputPrice      *float64 `json:"input_price"`
	OutputPrice     *float64 `json:"output_price"`
	CacheWritePrice *float64 `json:"cache_write_price"`
	CacheReadPrice  *float64 `json:"cache_read_price"`
	PerRequestPrice *float64 `json:"per_request_price"`
}

func DefaultModelMarketConfig() ModelMarketConfig {
	return ModelMarketConfig{
		Enabled:     true,
		AutoSync:    true,
		Title:       defaultModelMarketTitle,
		Description: defaultModelMarketDesc,
	}
}

func NormalizeModelMarketConfig(cfg ModelMarketConfig) ModelMarketConfig {
	defaults := DefaultModelMarketConfig()
	cfg.Title = strings.TrimSpace(cfg.Title)
	if cfg.Title == "" {
		cfg.Title = defaults.Title
	}
	cfg.Description = strings.TrimSpace(cfg.Description)
	if cfg.Description == "" {
		cfg.Description = defaults.Description
	}

	selections := make([]ModelMarketSelection, 0, len(cfg.SelectedModels))
	seen := make(map[string]struct{}, len(cfg.SelectedModels))
	for _, selection := range cfg.SelectedModels {
		selection.Key = normalizeModelMarketKey(selection.Key)
		selection.Platform = strings.TrimSpace(selection.Platform)
		selection.Model = strings.TrimSpace(selection.Model)
		if selection.Key == "" && selection.Platform != "" && selection.Model != "" {
			selection.Key = modelMarketKey(selection.Platform, selection.Model)
		}
		if selection.Key == "" {
			continue
		}
		if _, ok := seen[selection.Key]; ok {
			continue
		}
		seen[selection.Key] = struct{}{}
		selections = append(selections, selection)
		if len(selections) >= maxModelMarketSelections {
			break
		}
	}
	cfg.SelectedModels = selections

	customModels := make([]ModelMarketCustomModel, 0, len(cfg.CustomModels))
	seenCustom := make(map[string]struct{}, len(cfg.CustomModels))
	for _, custom := range cfg.CustomModels {
		normalized, ok := normalizeModelMarketCustomModel(custom)
		if !ok {
			continue
		}
		if _, exists := seenCustom[normalized.Key]; exists {
			continue
		}
		seenCustom[normalized.Key] = struct{}{}
		customModels = append(customModels, normalized)
		if len(customModels) >= maxModelMarketCustom {
			break
		}
	}
	cfg.CustomModels = customModels
	return cfg
}

func ValidateModelMarketConfig(cfg ModelMarketConfig) error {
	if len(cfg.SelectedModels) > maxModelMarketSelections {
		return infraerrors.BadRequest("MODEL_MARKET_TOO_MANY_SELECTIONS", fmt.Sprintf("model market supports at most %d selected models", maxModelMarketSelections))
	}
	if len(cfg.CustomModels) > maxModelMarketCustom {
		return infraerrors.BadRequest("MODEL_MARKET_TOO_MANY_CUSTOM_MODELS", fmt.Sprintf("model market supports at most %d custom models", maxModelMarketCustom))
	}
	for i, selection := range cfg.SelectedModels {
		if normalizeModelMarketKey(selection.Key) == "" && (strings.TrimSpace(selection.Platform) == "" || strings.TrimSpace(selection.Model) == "") {
			return infraerrors.BadRequest("MODEL_MARKET_SELECTION_REQUIRED", fmt.Sprintf("model market selection #%d requires key or platform/model", i+1))
		}
	}
	for i, custom := range cfg.CustomModels {
		normalized, ok := normalizeModelMarketCustomModel(custom)
		if !ok {
			return infraerrors.BadRequest("MODEL_MARKET_CUSTOM_MODEL_REQUIRED", fmt.Sprintf("model market custom model #%d requires platform and model", i+1))
		}
		if !isModelMarketBillingMode(normalized.BillingMode) {
			return infraerrors.BadRequest("MODEL_MARKET_CUSTOM_BILLING_MODE_INVALID", fmt.Sprintf("model market custom model #%d has invalid billing mode", i+1))
		}
		if normalized.Enabled && len(normalized.Groups) == 0 {
			return infraerrors.BadRequest("MODEL_MARKET_CUSTOM_GROUP_REQUIRED", fmt.Sprintf("model market custom model #%d requires at least one group", i+1))
		}
	}
	return nil
}

func (s *SettingService) GetModelMarketConfig(ctx context.Context) (*ModelMarketConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelMarket)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			cfg := DefaultModelMarketConfig()
			return &cfg, nil
		}
		return nil, fmt.Errorf("get model market config: %w", err)
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		cfg := DefaultModelMarketConfig()
		return &cfg, nil
	}

	var cfg ModelMarketConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		cfg := DefaultModelMarketConfig()
		return &cfg, nil
	}
	cfg = NormalizeModelMarketConfig(cfg)
	return &cfg, nil
}

func (s *SettingService) SetModelMarketConfig(ctx context.Context, cfg ModelMarketConfig) (*ModelMarketConfig, error) {
	if err := ValidateModelMarketConfig(cfg); err != nil {
		return nil, err
	}
	cfg = NormalizeModelMarketConfig(cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal model market config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyModelMarket, string(data)); err != nil {
		return nil, fmt.Errorf("set model market config: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return &cfg, nil
}

func (s *ModelMarketService) GetConfig(ctx context.Context) (*ModelMarketConfig, error) {
	if s == nil || s.settingService == nil {
		cfg := DefaultModelMarketConfig()
		return &cfg, nil
	}
	return s.settingService.GetModelMarketConfig(ctx)
}

func (s *ModelMarketService) SetConfig(ctx context.Context, cfg ModelMarketConfig) (*ModelMarketAdminResponse, error) {
	if s == nil || s.settingService == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_MARKET_UNAVAILABLE", "model market service unavailable")
	}
	updated, err := s.settingService.SetModelMarketConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return s.buildAdminResponse(ctx, *updated)
}

func (s *ModelMarketService) GetPublic(ctx context.Context) (*ModelMarketResponse, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	candidates, err := s.candidates(ctx, false)
	if err != nil {
		return nil, err
	}
	models := applyModelMarketConfig(candidates, *cfg)
	if !cfg.Enabled {
		models = []ModelMarketModel{}
	}
	return &ModelMarketResponse{
		Enabled:     cfg.Enabled,
		AutoSync:    cfg.AutoSync,
		Title:       cfg.Title,
		Description: cfg.Description,
		Models:      models,
	}, nil
}

func (s *ModelMarketService) GetAdmin(ctx context.Context) (*ModelMarketAdminResponse, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s.buildAdminResponse(ctx, *cfg)
}

func (s *ModelMarketService) buildAdminResponse(ctx context.Context, cfg ModelMarketConfig) (*ModelMarketAdminResponse, error) {
	adminCandidates, err := s.candidates(ctx, true)
	if err != nil {
		return nil, err
	}
	publicCandidates, err := s.candidates(ctx, false)
	if err != nil {
		return nil, err
	}
	return &ModelMarketAdminResponse{
		Config:     cfg,
		Candidates: adminCandidates,
		Models:     applyModelMarketConfig(publicCandidates, cfg),
	}, nil
}

func (s *ModelMarketService) candidates(ctx context.Context, includeWithoutPublicGroups bool) ([]ModelMarketModel, error) {
	if s == nil || s.channelLister == nil {
		return []ModelMarketModel{}, nil
	}
	channels, err := s.channelLister.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model market channels: %w", err)
	}

	byKey := make(map[string]*ModelMarketModel)
	for _, ch := range channels {
		if ch.Status != "" && ch.Status != StatusActive {
			continue
		}
		groupsByPlatform := publicGroupsByPlatform(ch.Groups)
		if len(groupsByPlatform) == 0 && !includeWithoutPublicGroups {
			continue
		}
		for _, model := range ch.SupportedModels {
			groups := groupsByPlatform[model.Platform]
			if len(groups) == 0 && !includeWithoutPublicGroups {
				continue
			}
			key := modelMarketKey(model.Platform, model.Name)
			if key == "" {
				continue
			}
			existing := byKey[key]
			if existing == nil {
				existing = &ModelMarketModel{
					Key:         key,
					Name:        model.Name,
					Platform:    model.Platform,
					BillingMode: modelMarketBillingMode(model.Pricing),
					Pricing:     toModelMarketPricing(model.Pricing),
					Groups:      []ModelMarketGroup{},
					Channels:    []string{},
				}
				byKey[key] = existing
			}
			existing.Groups = mergeModelMarketGroups(existing.Groups, groups)
			if ch.Name != "" && !containsString(existing.Channels, ch.Name) {
				existing.Channels = append(existing.Channels, ch.Name)
				sort.Strings(existing.Channels)
			}
			if existing.Pricing == nil && model.Pricing != nil {
				existing.Pricing = toModelMarketPricing(model.Pricing)
				existing.BillingMode = modelMarketBillingMode(model.Pricing)
			}
		}
	}

	out := make([]ModelMarketModel, 0, len(byKey))
	for _, model := range byKey {
		out = append(out, *model)
	}
	sortModelMarketModels(out)
	return out, nil
}

func publicGroupsByPlatform(groups []AvailableGroupRef) map[string][]ModelMarketGroup {
	out := make(map[string][]ModelMarketGroup)
	for _, group := range groups {
		if group.IsExclusive || group.Platform == "" {
			continue
		}
		out[group.Platform] = append(out[group.Platform], ModelMarketGroup{
			ID:               group.ID,
			Name:             group.Name,
			Platform:         group.Platform,
			SubscriptionType: group.SubscriptionType,
			RateMultiplier:   group.RateMultiplier,
			IsExclusive:      group.IsExclusive,
		})
	}
	for platform := range out {
		sort.SliceStable(out[platform], func(i, j int) bool {
			return strings.ToLower(out[platform][i].Name) < strings.ToLower(out[platform][j].Name)
		})
	}
	return out
}

func applyModelMarketConfig(candidates []ModelMarketModel, cfg ModelMarketConfig) []ModelMarketModel {
	cfg = NormalizeModelMarketConfig(cfg)
	customModels := modelMarketCustomModels(cfg.CustomModels)
	if cfg.AutoSync {
		out := cloneModelMarketModels(candidates)
		out = append(out, customModels...)
		sortConfiguredModelMarketModels(out)
		return out
	}

	byKey := make(map[string]ModelMarketModel, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.Key] = candidate
	}
	out := make([]ModelMarketModel, 0, len(cfg.SelectedModels))
	for _, selection := range cfg.SelectedModels {
		if !selection.Enabled {
			continue
		}
		candidate, ok := byKey[selection.Key]
		if !ok {
			continue
		}
		candidate.SortOrder = selection.SortOrder
		out = append(out, candidate)
	}
	out = append(out, customModels...)
	sortConfiguredModelMarketModels(out)
	return out
}

func modelMarketCustomModels(customModels []ModelMarketCustomModel) []ModelMarketModel {
	out := make([]ModelMarketModel, 0, len(customModels))
	for _, custom := range customModels {
		custom, ok := normalizeModelMarketCustomModel(custom)
		if !ok || !custom.Enabled || len(custom.Groups) == 0 {
			continue
		}
		pricing := cloneModelMarketPricing(custom.Pricing)
		billingMode := custom.BillingMode
		if billingMode == "" && pricing != nil {
			billingMode = pricing.BillingMode
		}
		if billingMode == "" {
			billingMode = "unknown"
		}
		if pricing != nil && pricing.BillingMode == "" && billingMode != "unknown" {
			pricing.BillingMode = billingMode
		}
		out = append(out, ModelMarketModel{
			Key:         custom.Key,
			Name:        custom.Model,
			Platform:    custom.Platform,
			BillingMode: billingMode,
			Pricing:     pricing,
			Groups:      append([]ModelMarketGroup(nil), custom.Groups...),
			Channels:    []string{modelMarketCustomChannel},
			SortOrder:   custom.SortOrder,
		})
	}
	return out
}

func cloneModelMarketModels(src []ModelMarketModel) []ModelMarketModel {
	out := make([]ModelMarketModel, len(src))
	for i := range src {
		out[i] = src[i]
		out[i].Pricing = cloneModelMarketPricing(src[i].Pricing)
		out[i].Groups = append([]ModelMarketGroup(nil), src[i].Groups...)
		out[i].Channels = append([]string(nil), src[i].Channels...)
	}
	return out
}

func cloneModelMarketPricing(src *ModelMarketPricing) *ModelMarketPricing {
	if src == nil {
		return nil
	}
	out := *src
	out.Intervals = append([]ModelMarketPricingInterval(nil), src.Intervals...)
	return &out
}

func sortModelMarketModels(models []ModelMarketModel) {
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].Platform != models[j].Platform {
			return models[i].Platform < models[j].Platform
		}
		return strings.ToLower(models[i].Name) < strings.ToLower(models[j].Name)
	})
}

func sortConfiguredModelMarketModels(models []ModelMarketModel) {
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].SortOrder != models[j].SortOrder {
			return models[i].SortOrder < models[j].SortOrder
		}
		if models[i].Platform != models[j].Platform {
			return models[i].Platform < models[j].Platform
		}
		return strings.ToLower(models[i].Name) < strings.ToLower(models[j].Name)
	})
}

func mergeModelMarketGroups(left, right []ModelMarketGroup) []ModelMarketGroup {
	byID := make(map[int64]ModelMarketGroup, len(left)+len(right))
	for _, group := range left {
		byID[group.ID] = group
	}
	for _, group := range right {
		byID[group.ID] = group
	}
	out := make([]ModelMarketGroup, 0, len(byID))
	for _, group := range byID {
		out = append(out, group)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func modelMarketKey(platform, model string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	model = strings.ToLower(strings.TrimSpace(model))
	if platform == "" || model == "" {
		return ""
	}
	return platform + ":" + model
}

func customModelMarketKey(platform, model string) string {
	key := modelMarketKey(platform, model)
	if key == "" {
		return ""
	}
	return "custom:" + key
}

func normalizeModelMarketKey(key string) string {
	parts := strings.SplitN(strings.ToLower(strings.TrimSpace(key)), ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return modelMarketKey(parts[0], parts[1])
}

func normalizeCustomModelMarketKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return ""
	}
	key = strings.TrimPrefix(key, "custom:")
	base := normalizeModelMarketKey(key)
	if base == "" {
		return ""
	}
	return "custom:" + base
}

func normalizeModelMarketCustomModel(custom ModelMarketCustomModel) (ModelMarketCustomModel, bool) {
	custom.Platform = strings.TrimSpace(custom.Platform)
	custom.Model = strings.TrimSpace(custom.Model)
	if custom.Platform == "" || custom.Model == "" {
		return ModelMarketCustomModel{}, false
	}
	custom.Key = normalizeCustomModelMarketKey(custom.Key)
	if custom.Key == "" {
		custom.Key = customModelMarketKey(custom.Platform, custom.Model)
	}
	if custom.Key == "" {
		return ModelMarketCustomModel{}, false
	}
	if custom.SortOrder < 0 {
		custom.SortOrder = 0
	}
	custom.BillingMode = strings.TrimSpace(custom.BillingMode)
	custom.Pricing = normalizeModelMarketPricing(custom.Pricing, custom.BillingMode)
	if custom.BillingMode == "" && custom.Pricing != nil {
		custom.BillingMode = custom.Pricing.BillingMode
	}
	if custom.BillingMode == "" {
		custom.BillingMode = string(BillingModeToken)
	}
	if custom.Pricing != nil {
		custom.Pricing.BillingMode = custom.BillingMode
	}
	custom.Groups = normalizeModelMarketCustomGroups(custom.Groups, custom.Platform)
	return custom, true
}

func normalizeModelMarketPricing(pricing *ModelMarketPricing, billingMode string) *ModelMarketPricing {
	if pricing == nil {
		return nil
	}
	out := *pricing
	out.BillingMode = strings.TrimSpace(out.BillingMode)
	if out.BillingMode == "" {
		out.BillingMode = strings.TrimSpace(billingMode)
	}
	if out.BillingMode == "" {
		out.BillingMode = string(BillingModeToken)
	}
	out.Intervals = append([]ModelMarketPricingInterval(nil), pricing.Intervals...)
	return &out
}

func normalizeModelMarketCustomGroups(groups []ModelMarketGroup, fallbackPlatform string) []ModelMarketGroup {
	out := make([]ModelMarketGroup, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group.Name = strings.TrimSpace(group.Name)
		group.Platform = strings.TrimSpace(group.Platform)
		if group.Platform == "" {
			group.Platform = fallbackPlatform
		}
		group.SubscriptionType = strings.TrimSpace(group.SubscriptionType)
		if group.RateMultiplier <= 0 {
			group.RateMultiplier = 1
		}
		if group.Name == "" || group.Platform == "" {
			continue
		}
		key := fmt.Sprintf("%d:%s:%s", group.ID, group.Platform, strings.ToLower(group.Name))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, group)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func isModelMarketBillingMode(mode string) bool {
	switch BillingMode(mode) {
	case BillingModeToken, BillingModePerRequest, BillingModeImage:
		return true
	default:
		return false
	}
}

func modelMarketBillingMode(pricing *ChannelModelPricing) string {
	if pricing == nil || pricing.BillingMode == "" {
		return "unknown"
	}
	return string(pricing.BillingMode)
}

func toModelMarketPricing(p *ChannelModelPricing) *ModelMarketPricing {
	if p == nil {
		return nil
	}
	mode := p.BillingMode
	if mode == "" {
		mode = BillingModeToken
	}
	intervals := make([]ModelMarketPricingInterval, 0, len(p.Intervals))
	for _, interval := range p.Intervals {
		intervals = append(intervals, ModelMarketPricingInterval{
			MinTokens:       interval.MinTokens,
			MaxTokens:       interval.MaxTokens,
			TierLabel:       interval.TierLabel,
			InputPrice:      interval.InputPrice,
			OutputPrice:     interval.OutputPrice,
			CacheWritePrice: interval.CacheWritePrice,
			CacheReadPrice:  interval.CacheReadPrice,
			PerRequestPrice: interval.PerRequestPrice,
		})
	}
	return &ModelMarketPricing{
		BillingMode:      string(mode),
		InputPrice:       p.InputPrice,
		OutputPrice:      p.OutputPrice,
		CacheWritePrice:  p.CacheWritePrice,
		CacheReadPrice:   p.CacheReadPrice,
		ImageOutputPrice: p.ImageOutputPrice,
		PerRequestPrice:  p.PerRequestPrice,
		Intervals:        intervals,
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
