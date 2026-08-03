package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// normalizePlanCurrency validates and normalizes the display-only currency label.
// Empty means "no label" and is kept as-is so existing plans stay unchanged.
// Unlike payment settlement currency helpers, empty is valid and does not default to CNY.
func normalizePlanCurrency(raw string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(raw))
	if currency == "" {
		return "", nil
	}
	if len(currency) != 3 {
		return "", infraerrors.BadRequest("PLAN_CURRENCY_INVALID", "currency must be a 3-letter ISO currency code")
	}
	for _, ch := range currency {
		if ch < 'A' || ch > 'Z' {
			return "", infraerrors.BadRequest("PLAN_CURRENCY_INVALID", "currency must be a 3-letter ISO currency code")
		}
	}
	return currency, nil
}

// validatePlanRequired checks that all required fields for a plan are provided.
func validatePlanRequired(name string, groupID int64, price float64, validityDays int, validityUnit string, originalPrice *float64) error {
	if strings.TrimSpace(name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if groupID < 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if validityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if strings.TrimSpace(validityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if originalPrice != nil && *originalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	return nil
}

func validatePlanCreate(req CreatePlanRequest) error {
	if err := validatePlanRequired(req.Name, req.GroupID, req.Price, req.ValidityDays, req.ValidityUnit, req.OriginalPrice); err != nil {
		return err
	}
	if err := validatePlanValidityLimit(req.ValidityDays, req.ValidityUnit); err != nil {
		return err
	}
	if req.DailyLimitUSD != nil && *req.DailyLimitUSD == 0 {
		return infraerrors.BadRequest("PLAN_DAILY_LIMIT_INVALID", "daily limit must be > 0")
	}
	if req.WeeklyLimitUSD != nil && *req.WeeklyLimitUSD == 0 {
		return infraerrors.BadRequest("PLAN_WEEKLY_LIMIT_INVALID", "weekly limit must be > 0")
	}
	return validatePlanCreditFields(&req.QuotaUSD, req.DailyLimitUSD, req.WeeklyLimitUSD, &req.ScopeType)
}

// validatePlanPatch validates only the non-nil fields in a patch update.
func validatePlanPatch(req UpdatePlanRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if req.GroupID != nil && *req.GroupID < 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if req.Price != nil && *req.Price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if req.ValidityDays != nil && *req.ValidityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if req.ValidityUnit != nil && strings.TrimSpace(*req.ValidityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if req.ValidityDays != nil {
		unit := "day"
		if req.ValidityUnit != nil {
			unit = *req.ValidityUnit
		}
		if err := validatePlanValidityLimit(*req.ValidityDays, unit); err != nil {
			return err
		}
	}
	if err := validatePlanCreditFields(req.QuotaUSD, req.DailyLimitUSD, req.WeeklyLimitUSD, req.ScopeType); err != nil {
		return err
	}
	return nil
}

func validatePlanValidityLimit(days int, unit string) error {
	if psComputeValidityDays(days, strings.TrimSpace(unit)) > 30 {
		return infraerrors.BadRequest("PLAN_VALIDITY_TOO_LONG", "validity days must be <= 30 days")
	}
	return nil
}

func validatePlanCreditFields(quota *float64, daily *float64, weekly *float64, scopeType *string) error {
	if quota != nil && (*quota <= 0 || math.IsNaN(*quota) || math.IsInf(*quota, 0)) {
		return infraerrors.BadRequest("PLAN_QUOTA_INVALID", "quota must be > 0")
	}
	if daily != nil && (*daily < 0 || math.IsNaN(*daily) || math.IsInf(*daily, 0)) {
		return infraerrors.BadRequest("PLAN_DAILY_LIMIT_INVALID", "daily limit must be > 0")
	}
	if weekly != nil && (*weekly < 0 || math.IsNaN(*weekly) || math.IsInf(*weekly, 0)) {
		return infraerrors.BadRequest("PLAN_WEEKLY_LIMIT_INVALID", "weekly limit must be > 0")
	}
	if scopeType != nil {
		switch strings.TrimSpace(*scopeType) {
		case "", SubscriptionScopeAllAvailableGroups, SubscriptionScopeSelectedGroups, SubscriptionScopePlatforms:
			return nil
		default:
			return infraerrors.BadRequest("PLAN_SCOPE_TYPE_INVALID", "scope type is invalid")
		}
	}
	return nil
}

// --- Plan CRUD ---

// PlanGroupInfo holds the group details needed for subscription plan display.
type PlanGroupInfo struct {
	Platform        string   `json:"platform"`
	Name            string   `json:"name"`
	RateMultiplier  float64  `json:"rate_multiplier"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	ModelScopes     []string `json:"supported_model_scopes"`
}

// PlanLimitSyncPreview describes how many existing subscriptions would receive
// the current daily/weekly limits from a plan.
type PlanLimitSyncPreview struct {
	PlanID         int64    `json:"plan_id"`
	DailyLimitUSD  *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD *float64 `json:"weekly_limit_usd"`
	MatchedCount   int      `json:"matched_count"`
	ChangedCount   int      `json:"changed_count"`
}

// PlanLimitSyncResult is returned after applying a plan limit sync.
type PlanLimitSyncResult struct {
	PlanLimitSyncPreview
	UpdatedCount int `json:"updated_count"`
}

// GetGroupPlatformMap returns a map of group_id → platform for the given plans.
func (s *PaymentConfigService) GetGroupPlatformMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]string {
	info := s.GetGroupInfoMap(ctx, plans)
	m := make(map[int64]string, len(info))
	for id, gi := range info {
		m[id] = gi.Platform
	}
	return m
}

// GetGroupInfoMap returns a map of group_id → PlanGroupInfo for the given plans.
func (s *PaymentConfigService) GetGroupInfoMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]PlanGroupInfo {
	ids := make([]int64, 0, len(plans))
	seen := make(map[int64]bool)
	for _, p := range plans {
		// 额度池套餐 group_id 可空（不绑定具体分组）；跳过未绑定分组的套餐
		if p.GroupID == nil {
			continue
		}
		gid := *p.GroupID
		if !seen[gid] {
			seen[gid] = true
			ids = append(ids, gid)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	groups, err := s.entClient.Group.Query().Where(group.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil
	}
	m := make(map[int64]PlanGroupInfo, len(groups))
	for _, g := range groups {
		m[int64(g.ID)] = PlanGroupInfo{
			Platform:        g.Platform,
			Name:            g.Name,
			RateMultiplier:  g.RateMultiplier,
			DailyLimitUSD:   g.DailyLimitUsd,
			WeeklyLimitUSD:  g.WeeklyLimitUsd,
			MonthlyLimitUSD: g.MonthlyLimitUsd,
			ModelScopes:     g.SupportedModelScopes,
		}
	}
	return m
}

func (s *PaymentConfigService) ListPlans(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) ListPlansForSale(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Where(subscriptionplan.ForSaleEQ(true)).Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) CreatePlan(ctx context.Context, req CreatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanCreate(req); err != nil {
		return nil, err
	}
	currency, err := normalizePlanCurrency(req.Currency)
	if err != nil {
		return nil, err
	}
	b := s.entClient.SubscriptionPlan.Create().
		SetName(req.Name).SetDescription(req.Description).
		SetPrice(req.Price).SetCurrency(currency).SetValidityDays(req.ValidityDays).SetValidityUnit(req.ValidityUnit).
		SetFeatures(req.Features).SetPurchaseNotice(req.PurchaseNotice).SetProductName(req.ProductName).
		SetQuotaUsd(req.QuotaUSD).
		SetNillableDailyLimitUsd(req.DailyLimitUSD).
		SetNillableWeeklyLimitUsd(req.WeeklyLimitUSD).
		SetScopeType(normalizePlanScopeType(req.ScopeType)).
		SetScopeConfig(planScopeConfigOrDefault(req.ScopeConfig)).
		SetForSale(req.ForSale).SetSortOrder(req.SortOrder)
	if req.GroupID > 0 {
		b.SetGroupID(req.GroupID)
	}
	if req.OriginalPrice != nil {
		b.SetOriginalPrice(*req.OriginalPrice)
	}
	return b.Save(ctx)
}

// UpdatePlan updates a subscription plan by ID (patch semantics).
// NOTE: This function exceeds 30 lines due to per-field nil-check patch update boilerplate
// plus a validation guard for non-nil fields.
func (s *PaymentConfigService) UpdatePlan(ctx context.Context, id int64, req UpdatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanPatch(req); err != nil {
		return nil, err
	}
	u := s.entClient.SubscriptionPlan.UpdateOneID(id)
	if req.GroupID != nil {
		if *req.GroupID > 0 {
			u.SetGroupID(*req.GroupID)
		} else {
			u.ClearGroupID()
		}
	}
	if req.Name != nil {
		u.SetName(*req.Name)
	}
	if req.Description != nil {
		u.SetDescription(*req.Description)
	}
	if req.Price != nil {
		u.SetPrice(*req.Price)
	}
	if req.OriginalPrice != nil {
		u.SetOriginalPrice(*req.OriginalPrice)
	}
	if req.Currency != nil {
		currency, err := normalizePlanCurrency(*req.Currency)
		if err != nil {
			return nil, err
		}
		u.SetCurrency(currency)
	}
	if req.ValidityDays != nil {
		u.SetValidityDays(*req.ValidityDays)
	}
	if req.ValidityUnit != nil {
		u.SetValidityUnit(*req.ValidityUnit)
	}
	if req.QuotaUSD != nil {
		u.SetQuotaUsd(*req.QuotaUSD)
	}
	if req.DailyLimitUSD != nil {
		if *req.DailyLimitUSD > 0 {
			u.SetDailyLimitUsd(*req.DailyLimitUSD)
		} else {
			u.ClearDailyLimitUsd()
		}
	}
	if req.WeeklyLimitUSD != nil {
		if *req.WeeklyLimitUSD > 0 {
			u.SetWeeklyLimitUsd(*req.WeeklyLimitUSD)
		} else {
			u.ClearWeeklyLimitUsd()
		}
	}
	if req.ScopeType != nil {
		u.SetScopeType(normalizePlanScopeType(*req.ScopeType))
	}
	if req.ScopeConfig != nil {
		u.SetScopeConfig(planScopeConfigOrDefault(req.ScopeConfig))
	}
	if req.Features != nil {
		u.SetFeatures(*req.Features)
	}
	if req.PurchaseNotice != nil {
		u.SetPurchaseNotice(*req.PurchaseNotice)
	}
	if req.ProductName != nil {
		u.SetProductName(*req.ProductName)
	}
	if req.ForSale != nil {
		u.SetForSale(*req.ForSale)
	}
	if req.SortOrder != nil {
		u.SetSortOrder(*req.SortOrder)
	}
	return u.Save(ctx)
}

// PreviewPlanLimitSync returns the active, unexpired subscription count that
// would be affected by syncing a plan's daily/weekly limits.
func (s *PaymentConfigService) PreviewPlanLimitSync(ctx context.Context, planID int64) (*PlanLimitSyncPreview, error) {
	plan, subscriptions, err := s.getPlanLimitSyncCandidates(ctx, planID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	preview, _ := buildPlanLimitSyncPreview(plan, subscriptions)
	return preview, nil
}

// SyncPlanLimits copies only daily_limit_usd and weekly_limit_usd from the plan
// to currently active, unexpired purchased subscriptions for that plan.
func (s *PaymentConfigService) SyncPlanLimits(ctx context.Context, planID int64) (*PlanLimitSyncResult, error) {
	plan, subscriptions, err := s.getPlanLimitSyncCandidates(ctx, planID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	preview, changedIDs := buildPlanLimitSyncPreview(plan, subscriptions)
	result := &PlanLimitSyncResult{
		PlanLimitSyncPreview: *preview,
	}
	if len(changedIDs) == 0 {
		return result, nil
	}

	update := s.entClient.UserSubscription.Update().
		Where(usersubscription.IDIn(changedIDs...)).
		SetUpdatedAt(time.Now().UTC())
	if plan.DailyLimitUsd != nil {
		update.SetDailyLimitUsd(*plan.DailyLimitUsd)
	} else {
		update.ClearDailyLimitUsd()
	}
	if plan.WeeklyLimitUsd != nil {
		update.SetWeeklyLimitUsd(*plan.WeeklyLimitUsd)
	} else {
		update.ClearWeeklyLimitUsd()
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("sync plan subscription limits: %w", err)
	}
	result.UpdatedCount = updated
	return result, nil
}

func (s *PaymentConfigService) getPlanLimitSyncCandidates(ctx context.Context, planID int64, now time.Time) (*dbent.SubscriptionPlan, []*dbent.UserSubscription, error) {
	plan, err := s.GetPlan(ctx, planID)
	if err != nil {
		return nil, nil, err
	}
	subscriptions, err := s.entClient.UserSubscription.Query().
		Where(
			usersubscription.PlanIDEQ(planID),
			usersubscription.StatusEQ(SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(now),
			usersubscription.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("query plan subscription limits: %w", err)
	}
	return plan, subscriptions, nil
}

func buildPlanLimitSyncPreview(plan *dbent.SubscriptionPlan, subscriptions []*dbent.UserSubscription) (*PlanLimitSyncPreview, []int64) {
	changedIDs := make([]int64, 0, len(subscriptions))
	for _, sub := range subscriptions {
		if limitUSDChanged(sub.DailyLimitUsd, plan.DailyLimitUsd) || limitUSDChanged(sub.WeeklyLimitUsd, plan.WeeklyLimitUsd) {
			changedIDs = append(changedIDs, sub.ID)
		}
	}
	return &PlanLimitSyncPreview{
		PlanID:         plan.ID,
		DailyLimitUSD:  plan.DailyLimitUsd,
		WeeklyLimitUSD: plan.WeeklyLimitUsd,
		MatchedCount:   len(subscriptions),
		ChangedCount:   len(changedIDs),
	}, changedIDs
}

func limitUSDChanged(current, target *float64) bool {
	if current == nil && target == nil {
		return false
	}
	if current == nil || target == nil {
		return true
	}
	return *current != *target
}

func normalizePlanScopeType(scopeType string) string {
	scopeType = strings.TrimSpace(scopeType)
	if scopeType == "" {
		return SubscriptionScopeAllAvailableGroups
	}
	return scopeType
}

func planScopeConfigOrDefault(scopeConfig map[string]any) map[string]any {
	if scopeConfig == nil {
		return map[string]any{}
	}
	return scopeConfig
}

func (s *PaymentConfigService) DeletePlan(ctx context.Context, id int64) error {
	count, err := s.countPendingOrdersByPlan(ctx, id)
	if err != nil {
		return fmt.Errorf("check pending orders: %w", err)
	}
	if count > 0 {
		return infraerrors.Conflict("PENDING_ORDERS",
			fmt.Sprintf("this plan has %d in-progress orders and cannot be deleted — wait for orders to complete first", count))
	}
	return s.entClient.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
}

// GetPlan returns a subscription plan by ID.
func (s *PaymentConfigService) GetPlan(ctx context.Context, id int64) (*dbent.SubscriptionPlan, error) {
	plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	return plan, nil
}
