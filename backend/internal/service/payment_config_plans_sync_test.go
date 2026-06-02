package service

import (
	"context"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func TestPaymentConfigServicePreviewPlanLimitSyncCountsOnlyActiveUnexpiredPlanSubscriptions(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}
	now := time.Now().UTC()
	daily := 20.0
	weekly := 80.0
	oldDaily := 10.0
	oldWeekly := 40.0

	plan := createLimitSyncPlan(t, ctx, client, "Plan A", &daily, &weekly)
	otherPlan := createLimitSyncPlan(t, ctx, client, "Plan B", &daily, &weekly)
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour)})
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &daily, WeeklyLimit: &weekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour)})
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour), ExhaustedAt: &now})
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(-24 * time.Hour)})
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusSuspended, ExpiresAt: now.Add(24 * time.Hour)})
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour), DeletedAt: &now})
	createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: otherPlan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour)})

	preview, err := svc.PreviewPlanLimitSync(ctx, plan.ID)
	if err != nil {
		t.Fatalf("PreviewPlanLimitSync returned error: %v", err)
	}

	if preview.PlanID != plan.ID {
		t.Fatalf("PlanID = %d, want %d", preview.PlanID, plan.ID)
	}
	if preview.DailyLimitUSD == nil || *preview.DailyLimitUSD != daily {
		t.Fatalf("DailyLimitUSD = %v, want %v", preview.DailyLimitUSD, daily)
	}
	if preview.WeeklyLimitUSD == nil || *preview.WeeklyLimitUSD != weekly {
		t.Fatalf("WeeklyLimitUSD = %v, want %v", preview.WeeklyLimitUSD, weekly)
	}
	if preview.MatchedCount != 3 {
		t.Fatalf("MatchedCount = %d, want 3", preview.MatchedCount)
	}
	if preview.ChangedCount != 2 {
		t.Fatalf("ChangedCount = %d, want 2", preview.ChangedCount)
	}
}

func TestPaymentConfigServiceSyncPlanLimitsUpdatesOnlyDailyAndWeeklyLimits(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}
	now := time.Now().UTC()
	daily := 20.0
	weekly := 80.0
	oldDaily := 10.0
	oldWeekly := 40.0

	plan := createLimitSyncPlan(t, ctx, client, "Plan A", &daily, &weekly)
	changed := createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour), QuotaLimit: 100, QuotaUsed: 7})
	unchanged := createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &daily, WeeklyLimit: &weekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour), QuotaLimit: 200, QuotaUsed: 11})
	expired := createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(-24 * time.Hour), QuotaLimit: 300, QuotaUsed: 13})

	result, err := svc.SyncPlanLimits(ctx, plan.ID)
	if err != nil {
		t.Fatalf("SyncPlanLimits returned error: %v", err)
	}

	if result.MatchedCount != 2 {
		t.Fatalf("MatchedCount = %d, want 2", result.MatchedCount)
	}
	if result.ChangedCount != 1 {
		t.Fatalf("ChangedCount = %d, want 1", result.ChangedCount)
	}
	if result.UpdatedCount != 1 {
		t.Fatalf("UpdatedCount = %d, want 1", result.UpdatedCount)
	}

	reloadedChanged := getLimitSyncSubscription(t, ctx, client, changed.ID)
	if reloadedChanged.DailyLimitUsd == nil || *reloadedChanged.DailyLimitUsd != daily {
		t.Fatalf("changed DailyLimitUsd = %v, want %v", reloadedChanged.DailyLimitUsd, daily)
	}
	if reloadedChanged.WeeklyLimitUsd == nil || *reloadedChanged.WeeklyLimitUsd != weekly {
		t.Fatalf("changed WeeklyLimitUsd = %v, want %v", reloadedChanged.WeeklyLimitUsd, weekly)
	}
	if reloadedChanged.QuotaLimitUsd != 100 || reloadedChanged.QuotaUsedUsd != 7 {
		t.Fatalf("changed quota fields = (%v, %v), want (100, 7)", reloadedChanged.QuotaLimitUsd, reloadedChanged.QuotaUsedUsd)
	}

	reloadedUnchanged := getLimitSyncSubscription(t, ctx, client, unchanged.ID)
	if reloadedUnchanged.DailyLimitUsd == nil || *reloadedUnchanged.DailyLimitUsd != daily {
		t.Fatalf("unchanged DailyLimitUsd = %v, want %v", reloadedUnchanged.DailyLimitUsd, daily)
	}
	if reloadedUnchanged.WeeklyLimitUsd == nil || *reloadedUnchanged.WeeklyLimitUsd != weekly {
		t.Fatalf("unchanged WeeklyLimitUsd = %v, want %v", reloadedUnchanged.WeeklyLimitUsd, weekly)
	}
	if reloadedUnchanged.QuotaLimitUsd != 200 || reloadedUnchanged.QuotaUsedUsd != 11 {
		t.Fatalf("unchanged quota fields = (%v, %v), want (200, 11)", reloadedUnchanged.QuotaLimitUsd, reloadedUnchanged.QuotaUsedUsd)
	}

	reloadedExpired := getLimitSyncSubscription(t, ctx, client, expired.ID)
	if reloadedExpired.DailyLimitUsd == nil || *reloadedExpired.DailyLimitUsd != oldDaily {
		t.Fatalf("expired DailyLimitUsd = %v, want %v", reloadedExpired.DailyLimitUsd, oldDaily)
	}
	if reloadedExpired.WeeklyLimitUsd == nil || *reloadedExpired.WeeklyLimitUsd != oldWeekly {
		t.Fatalf("expired WeeklyLimitUsd = %v, want %v", reloadedExpired.WeeklyLimitUsd, oldWeekly)
	}
	if reloadedExpired.QuotaLimitUsd != 300 || reloadedExpired.QuotaUsedUsd != 13 {
		t.Fatalf("expired quota fields = (%v, %v), want (300, 13)", reloadedExpired.QuotaLimitUsd, reloadedExpired.QuotaUsedUsd)
	}
}

func TestPaymentConfigServiceSyncPlanLimitsClearsSubscriptionLimitsWhenPlanLimitsAreNil(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}
	now := time.Now().UTC()
	oldDaily := 10.0
	oldWeekly := 40.0

	plan := createLimitSyncPlan(t, ctx, client, "Plan A", nil, nil)
	sub := createLimitSyncSubscription(t, ctx, client, limitSyncSubSeed{PlanID: plan.ID, DailyLimit: &oldDaily, WeeklyLimit: &oldWeekly, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour)})

	result, err := svc.SyncPlanLimits(ctx, plan.ID)
	if err != nil {
		t.Fatalf("SyncPlanLimits returned error: %v", err)
	}

	if result.MatchedCount != 1 {
		t.Fatalf("MatchedCount = %d, want 1", result.MatchedCount)
	}
	if result.ChangedCount != 1 {
		t.Fatalf("ChangedCount = %d, want 1", result.ChangedCount)
	}
	if result.UpdatedCount != 1 {
		t.Fatalf("UpdatedCount = %d, want 1", result.UpdatedCount)
	}

	reloaded := getLimitSyncSubscription(t, ctx, client, sub.ID)
	if reloaded.DailyLimitUsd != nil {
		t.Fatalf("DailyLimitUsd = %v, want nil", reloaded.DailyLimitUsd)
	}
	if reloaded.WeeklyLimitUsd != nil {
		t.Fatalf("WeeklyLimitUsd = %v, want nil", reloaded.WeeklyLimitUsd)
	}
}

type limitSyncSubSeed struct {
	PlanID      int64
	DailyLimit  *float64
	WeeklyLimit *float64
	Status      string
	ExpiresAt   time.Time
	ExhaustedAt *time.Time
	DeletedAt   *time.Time
	QuotaLimit  float64
	QuotaUsed   float64
}

func createLimitSyncPlan(t *testing.T, ctx context.Context, client *dbent.Client, name string, dailyLimit, weeklyLimit *float64) *dbent.SubscriptionPlan {
	t.Helper()

	plan, err := client.SubscriptionPlan.Create().
		SetName(name).
		SetDescription(name).
		SetPrice(9.9).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetQuotaUsd(100).
		SetNillableDailyLimitUsd(dailyLimit).
		SetNillableWeeklyLimitUsd(weeklyLimit).
		SetFeatures("").
		SetPurchaseNotice("").
		SetProductName("").
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	return plan
}

func createLimitSyncSubscription(t *testing.T, ctx context.Context, client *dbent.Client, seed limitSyncSubSeed) *dbent.UserSubscription {
	t.Helper()

	user, err := client.User.Create().
		SetEmail(limitSyncUserEmail(t)).
		SetPasswordHash("hash").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	quotaLimit := seed.QuotaLimit
	if quotaLimit == 0 {
		quotaLimit = 100
	}
	start := seed.ExpiresAt.Add(-24 * time.Hour)
	builder := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetPlanID(seed.PlanID).
		SetScopeType(SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetQuotaLimitUsd(quotaLimit).
		SetQuotaUsedUsd(seed.QuotaUsed).
		SetNillableDailyLimitUsd(seed.DailyLimit).
		SetNillableWeeklyLimitUsd(seed.WeeklyLimit).
		SetStartsAt(start).
		SetExpiresAt(seed.ExpiresAt).
		SetStatus(seed.Status).
		SetAssignedAt(start)
	if seed.ExhaustedAt != nil {
		builder.SetExhaustedAt(*seed.ExhaustedAt)
	}
	if seed.DeletedAt != nil {
		builder.SetDeletedAt(*seed.DeletedAt)
	}
	sub, err := builder.Save(ctx)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	return sub
}

func getLimitSyncSubscription(t *testing.T, ctx context.Context, client *dbent.Client, id int64) *dbent.UserSubscription {
	t.Helper()

	sub, err := client.UserSubscription.Get(ctx, id)
	if err != nil {
		t.Fatalf("get subscription %d: %v", id, err)
	}
	return sub
}

func limitSyncUserEmail(t *testing.T) string {
	t.Helper()
	return strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-").Replace(t.Name()) + "-" + time.Now().Format("150405.000000") + "@example.com"
}
