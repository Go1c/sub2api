package service

import (
	"context"
	"math"
	"testing"
	"time"
)

type wasteStatsRepoStub struct {
	result WasteStatsResult
	query  WasteStatsQuery
	called bool
}

func (s *wasteStatsRepoStub) GetWasteStats(_ context.Context, query WasteStatsQuery) (WasteStatsResult, error) {
	s.called = true
	s.query = query
	return s.result, nil
}

func TestSubscriptionWasteStatsRejectsInvalidRange(t *testing.T) {
	svc := NewSubscriptionWasteStatsService(&wasteStatsRepoStub{})
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	_, err := svc.GetWasteStats(context.Background(), WasteStatsQuery{
		StartTime: now,
		EndTime:   now,
		Window:    "daily",
	})
	if err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestSubscriptionWasteStatsRejectsInvalidWindow(t *testing.T) {
	svc := NewSubscriptionWasteStatsService(&wasteStatsRepoStub{})
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	_, err := svc.GetWasteStats(context.Background(), WasteStatsQuery{
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now,
		Window:    "monthly",
	})
	if err == nil {
		t.Fatal("expected invalid window error")
	}
}

func TestSubscriptionWasteStatsNormalizesEmptyAndNaNResult(t *testing.T) {
	start := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	repo := &wasteStatsRepoStub{result: WasteStatsResult{
		WasteRatio:       math.NaN(),
		TotalWastedRatio: math.Inf(1),
		ByPlan: []PlanWasteBucket{{
			WasteRatio:       math.NaN(),
			TotalWastedRatio: math.Inf(-1),
		}},
		TimeSeries: []WasteTimeBucket{{
			AverageWasteRatio:       math.NaN(),
			DailyAverageWasteRatio:  math.Inf(1),
			WeeklyAverageWasteRatio: math.Inf(-1),
			TotalWastedRatio:        math.NaN(),
		}},
	}}
	svc := NewSubscriptionWasteStatsService(repo)

	got, err := svc.GetWasteStats(context.Background(), WasteStatsQuery{
		StartTime: start,
		EndTime:   end,
		Window:    "",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.Window != "all" {
		t.Fatalf("expected default window all, got %q", got.Window)
	}
	if !got.StartTime.Equal(start) || !got.EndTime.Equal(end) {
		t.Fatalf("expected service to preserve query range, got %s - %s", got.StartTime, got.EndTime)
	}
	if got.WasteRatio != 0 || got.TotalWastedRatio != 0 {
		t.Fatalf("expected normalized summary ratios, got waste=%v total=%v", got.WasteRatio, got.TotalWastedRatio)
	}
	if got.ByPlan[0].WasteRatio != 0 || got.ByPlan[0].TotalWastedRatio != 0 {
		t.Fatalf("expected normalized plan ratios, got %+v", got.ByPlan[0])
	}
	if got.TimeSeries[0].AverageWasteRatio != 0 || got.TimeSeries[0].DailyAverageWasteRatio != 0 || got.TimeSeries[0].WeeklyAverageWasteRatio != 0 || got.TimeSeries[0].TotalWastedRatio != 0 {
		t.Fatalf("expected normalized time-series ratios, got %+v", got.TimeSeries[0])
	}
	if !repo.called {
		t.Fatal("expected repository to be called")
	}
}

func TestSubscriptionWasteStatsPopulatesPlanContractAliases(t *testing.T) {
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	repo := &wasteStatsRepoStub{result: WasteStatsResult{
		PurchasedUSD:     100,
		ConsumedUSD:      70,
		ExpiredWastedUSD: 20,
		ByPlan: []PlanWasteBucket{{
			PurchasedUSD:     50,
			ExpiredWastedUSD: 10,
		}},
		TimeSeries: []WasteTimeBucket{{
			WastedUSD: 3,
		}},
	}}
	svc := NewSubscriptionWasteStatsService(repo)

	got, err := svc.GetWasteStats(context.Background(), WasteStatsQuery{
		StartTime: start,
		EndTime:   end,
		Window:    WasteStatsWindowAll,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.TotalQuotaPurchasedUSD != 100 || got.TotalQuotaConsumedUSD != 70 || got.TotalQuotaWastedUSD != 20 {
		t.Fatalf("expected plan summary aliases from canonical fields, got %+v", got)
	}
	if got.ByPlan[0].TotalQuotaWastedRatio != 0.2 {
		t.Fatalf("expected plan waste ratio alias 0.2, got %+v", got.ByPlan[0])
	}
	if got.TimeSeries[0].TotalWastedUSD != 3 {
		t.Fatalf("expected time-series total_wasted_usd alias, got %+v", got.TimeSeries[0])
	}
}
