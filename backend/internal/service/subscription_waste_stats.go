package service

import (
	"context"
	"math"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	WasteStatsWindowDaily  = "daily"
	WasteStatsWindowWeekly = "weekly"
	WasteStatsWindowTotal  = "total"
	WasteStatsWindowAll    = "all"
)

// WasteStatsQuery scopes subscription credit waste analytics.
type WasteStatsQuery struct {
	StartTime time.Time
	EndTime   time.Time
	PlanID    *int64
	UserID    *int64
	Window    string
}

// WasteStatsResult contains the summary plus plan and time projections.
type WasteStatsResult struct {
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Window            string    `json:"window"`
	PurchasedUSD      float64   `json:"purchased_usd"`
	ConsumedUSD       float64   `json:"consumed_usd"`
	ExpiredWastedUSD  float64   `json:"expired_wasted_usd"`
	WindowWastedUSD   float64   `json:"window_wasted_usd"`
	TotalWastedUSD    float64   `json:"total_wasted_usd"`
	WasteRatio        float64   `json:"waste_ratio"`
	TotalWastedRatio  float64   `json:"total_wasted_ratio"`
	AverageWasteRatio float64   `json:"average_waste_ratio"`
	ResetCount        int64     `json:"reset_count"`

	// Plan-contract aliases kept explicit for admin UI/API consumers.
	TotalSubscriptionsPurchased int64   `json:"total_subscriptions_purchased"`
	TotalQuotaPurchasedUSD      float64 `json:"total_quota_purchased_usd"`
	TotalQuotaConsumedUSD       float64 `json:"total_quota_consumed_usd"`
	TotalQuotaWastedUSD         float64 `json:"total_quota_wasted_usd"`
	DailyResetCount             int64   `json:"daily_reset_count"`
	DailyAverageWasteRatio      float64 `json:"daily_average_waste_ratio"`
	DailyTotalWastedUSD         float64 `json:"daily_total_wasted_usd"`
	WeeklyResetCount            int64   `json:"weekly_reset_count"`
	WeeklyAverageWasteRatio     float64 `json:"weekly_average_waste_ratio"`
	WeeklyTotalWastedUSD        float64 `json:"weekly_total_wasted_usd"`

	ByPlan     []PlanWasteBucket `json:"by_plan"`
	TimeSeries []WasteTimeBucket `json:"time_series"`
}

// PlanWasteBucket is the plan-level waste projection.
type PlanWasteBucket struct {
	PlanID            *int64  `json:"plan_id"`
	PlanName          string  `json:"plan_name"`
	PurchasedUSD      float64 `json:"purchased_usd"`
	ConsumedUSD       float64 `json:"consumed_usd"`
	ExpiredWastedUSD  float64 `json:"expired_wasted_usd"`
	WindowWastedUSD   float64 `json:"window_wasted_usd"`
	TotalWastedUSD    float64 `json:"total_wasted_usd"`
	WasteRatio        float64 `json:"waste_ratio"`
	TotalWastedRatio  float64 `json:"total_wasted_ratio"`
	AverageWasteRatio float64 `json:"average_waste_ratio"`
	ResetCount        int64   `json:"reset_count"`

	PurchaseCount           int64   `json:"purchase_count"`
	AverageDailyWasteRatio  float64 `json:"average_daily_waste_ratio"`
	AverageWeeklyWasteRatio float64 `json:"average_weekly_waste_ratio"`
	TotalQuotaWastedRatio   float64 `json:"total_quota_wasted_ratio"`
}

// WasteTimeBucket is the time-series waste projection.
type WasteTimeBucket struct {
	BucketStart             time.Time `json:"bucket_start"`
	BucketEnd               time.Time `json:"bucket_end"`
	Window                  string    `json:"window"`
	WastedUSD               float64   `json:"wasted_usd"`
	TotalWastedUSD          float64   `json:"total_wasted_usd"`
	ExpiredWastedUSD        float64   `json:"expired_wasted_usd"`
	WindowWastedUSD         float64   `json:"window_wasted_usd"`
	AverageWasteRatio       float64   `json:"average_waste_ratio"`
	DailyAverageWasteRatio  float64   `json:"daily_average_waste_ratio"`
	WeeklyAverageWasteRatio float64   `json:"weekly_average_waste_ratio"`
	TotalWastedRatio        float64   `json:"total_wasted_ratio"`
	ResetCount              int64     `json:"reset_count"`
}

type SubscriptionWasteStatsRepository interface {
	GetWasteStats(ctx context.Context, query WasteStatsQuery) (WasteStatsResult, error)
}

type SubscriptionWasteStatsService struct {
	repo SubscriptionWasteStatsRepository
}

func NewSubscriptionWasteStatsService(repo SubscriptionWasteStatsRepository) *SubscriptionWasteStatsService {
	return &SubscriptionWasteStatsService{repo: repo}
}

func (s *SubscriptionWasteStatsService) GetWasteStats(ctx context.Context, query WasteStatsQuery) (WasteStatsResult, error) {
	query.StartTime = query.StartTime.UTC()
	query.EndTime = query.EndTime.UTC()
	if !query.StartTime.Before(query.EndTime) {
		return WasteStatsResult{}, infraerrors.BadRequest("INVALID_WASTE_STATS_RANGE", "start must be before end")
	}
	if query.Window == "" {
		query.Window = WasteStatsWindowAll
	}
	if !isValidWasteStatsWindow(query.Window) {
		return WasteStatsResult{}, infraerrors.BadRequest("INVALID_WASTE_STATS_WINDOW", "window must be daily, weekly, total, or all")
	}
	if s == nil || s.repo == nil {
		return WasteStatsResult{}, infraerrors.InternalServer("WASTE_STATS_REPOSITORY_UNAVAILABLE", "subscription waste stats repository is not configured")
	}

	result, err := s.repo.GetWasteStats(ctx, query)
	if err != nil {
		return WasteStatsResult{}, err
	}
	result.StartTime = query.StartTime
	result.EndTime = query.EndTime
	result.Window = query.Window
	normalizeWasteStatsResult(&result)
	return result, nil
}

func isValidWasteStatsWindow(window string) bool {
	switch window {
	case WasteStatsWindowDaily, WasteStatsWindowWeekly, WasteStatsWindowTotal, WasteStatsWindowAll:
		return true
	default:
		return false
	}
}

func normalizeWasteStatsResult(result *WasteStatsResult) {
	if result == nil {
		return
	}
	result.PurchasedUSD = finiteOrZero(result.PurchasedUSD)
	result.ConsumedUSD = finiteOrZero(result.ConsumedUSD)
	result.ExpiredWastedUSD = finiteOrZero(result.ExpiredWastedUSD)
	result.WindowWastedUSD = finiteOrZero(result.WindowWastedUSD)
	result.TotalWastedUSD = finiteOrZero(result.TotalWastedUSD)
	result.WasteRatio = finiteOrZero(result.WasteRatio)
	result.TotalWastedRatio = finiteOrZero(result.TotalWastedRatio)
	result.AverageWasteRatio = finiteOrZero(result.AverageWasteRatio)
	if result.TotalQuotaPurchasedUSD == 0 {
		result.TotalQuotaPurchasedUSD = result.PurchasedUSD
	} else if result.PurchasedUSD == 0 {
		result.PurchasedUSD = result.TotalQuotaPurchasedUSD
	}
	if result.TotalQuotaConsumedUSD == 0 {
		result.TotalQuotaConsumedUSD = result.ConsumedUSD
	} else if result.ConsumedUSD == 0 {
		result.ConsumedUSD = result.TotalQuotaConsumedUSD
	}
	if result.TotalQuotaWastedUSD == 0 {
		result.TotalQuotaWastedUSD = result.ExpiredWastedUSD
	} else if result.ExpiredWastedUSD == 0 {
		result.ExpiredWastedUSD = result.TotalQuotaWastedUSD
	}
	result.TotalQuotaPurchasedUSD = finiteOrZero(result.TotalQuotaPurchasedUSD)
	result.TotalQuotaConsumedUSD = finiteOrZero(result.TotalQuotaConsumedUSD)
	result.TotalQuotaWastedUSD = finiteOrZero(result.TotalQuotaWastedUSD)
	result.DailyAverageWasteRatio = finiteOrZero(result.DailyAverageWasteRatio)
	result.DailyTotalWastedUSD = finiteOrZero(result.DailyTotalWastedUSD)
	result.WeeklyAverageWasteRatio = finiteOrZero(result.WeeklyAverageWasteRatio)
	result.WeeklyTotalWastedUSD = finiteOrZero(result.WeeklyTotalWastedUSD)
	for i := range result.ByPlan {
		result.ByPlan[i].PurchasedUSD = finiteOrZero(result.ByPlan[i].PurchasedUSD)
		result.ByPlan[i].ConsumedUSD = finiteOrZero(result.ByPlan[i].ConsumedUSD)
		result.ByPlan[i].ExpiredWastedUSD = finiteOrZero(result.ByPlan[i].ExpiredWastedUSD)
		result.ByPlan[i].WindowWastedUSD = finiteOrZero(result.ByPlan[i].WindowWastedUSD)
		result.ByPlan[i].TotalWastedUSD = finiteOrZero(result.ByPlan[i].TotalWastedUSD)
		result.ByPlan[i].WasteRatio = finiteOrZero(result.ByPlan[i].WasteRatio)
		result.ByPlan[i].TotalWastedRatio = finiteOrZero(result.ByPlan[i].TotalWastedRatio)
		result.ByPlan[i].AverageWasteRatio = finiteOrZero(result.ByPlan[i].AverageWasteRatio)
		result.ByPlan[i].AverageDailyWasteRatio = finiteOrZero(result.ByPlan[i].AverageDailyWasteRatio)
		result.ByPlan[i].AverageWeeklyWasteRatio = finiteOrZero(result.ByPlan[i].AverageWeeklyWasteRatio)
		if result.ByPlan[i].TotalQuotaWastedRatio == 0 {
			result.ByPlan[i].TotalQuotaWastedRatio = result.ByPlan[i].TotalWastedRatio
		}
		if result.ByPlan[i].TotalQuotaWastedRatio == 0 {
			result.ByPlan[i].TotalQuotaWastedRatio = finiteOrZero(result.ByPlan[i].ExpiredWastedUSD / result.ByPlan[i].PurchasedUSD)
		}
		result.ByPlan[i].TotalQuotaWastedRatio = finiteOrZero(result.ByPlan[i].TotalQuotaWastedRatio)
	}
	for i := range result.TimeSeries {
		result.TimeSeries[i].WastedUSD = finiteOrZero(result.TimeSeries[i].WastedUSD)
		if result.TimeSeries[i].TotalWastedUSD == 0 {
			result.TimeSeries[i].TotalWastedUSD = result.TimeSeries[i].WastedUSD
		} else if result.TimeSeries[i].WastedUSD == 0 {
			result.TimeSeries[i].WastedUSD = result.TimeSeries[i].TotalWastedUSD
		}
		result.TimeSeries[i].TotalWastedUSD = finiteOrZero(result.TimeSeries[i].TotalWastedUSD)
		result.TimeSeries[i].ExpiredWastedUSD = finiteOrZero(result.TimeSeries[i].ExpiredWastedUSD)
		result.TimeSeries[i].WindowWastedUSD = finiteOrZero(result.TimeSeries[i].WindowWastedUSD)
		result.TimeSeries[i].AverageWasteRatio = finiteOrZero(result.TimeSeries[i].AverageWasteRatio)
		result.TimeSeries[i].DailyAverageWasteRatio = finiteOrZero(result.TimeSeries[i].DailyAverageWasteRatio)
		result.TimeSeries[i].WeeklyAverageWasteRatio = finiteOrZero(result.TimeSeries[i].WeeklyAverageWasteRatio)
		result.TimeSeries[i].TotalWastedRatio = finiteOrZero(result.TimeSeries[i].TotalWastedRatio)
	}
}

func finiteOrZero(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}
