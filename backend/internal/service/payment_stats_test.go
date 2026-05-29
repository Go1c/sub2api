package service

import (
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestFilterRevenueOrdersExcludesBalancePayments(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	todayStart := time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)

	paidAtToday := now
	paidAtYesterday := now.Add(-24 * time.Hour)
	orders := []*dbent.PaymentOrder{
		{ID: 1, PaymentType: payment.TypeBalance, PayAmount: 15, UserID: 9, UserEmail: "balance@example.com", PaidAt: &paidAtToday},
		{ID: 2, PaymentType: payment.TypeWxpay, PayAmount: 20, UserID: 8, UserEmail: "wx@example.com", PaidAt: &paidAtToday},
		{ID: 3, PaymentType: payment.TypeAlipay, PayAmount: 10, UserID: 7, UserEmail: "ali@example.com", PaidAt: &paidAtYesterday},
	}

	revenueOrders := filterRevenueOrders(orders)
	require.Len(t, revenueOrders, 2)

	stats := &DashboardStats{}
	computeBasicStats(stats, revenueOrders, todayStart)
	require.InDelta(t, 30, stats.TotalAmount, 1e-9)
	require.InDelta(t, 20, stats.TodayAmount, 1e-9)
	require.Equal(t, 2, stats.TotalCount)
	require.Equal(t, 1, stats.TodayCount)

	methods := buildMethodDistribution(revenueOrders)
	require.Len(t, methods, 2)
	require.ElementsMatch(t, []PaymentMethodStat{
		{Type: payment.TypeWxpay, Amount: 20, Count: 1},
		{Type: payment.TypeAlipay, Amount: 10, Count: 1},
	}, methods)

	topUsers := buildTopUsers(revenueOrders)
	require.Len(t, topUsers, 2)
	require.Equal(t, int64(8), topUsers[0].UserID)
	require.InDelta(t, 20, topUsers[0].Amount, 1e-9)
}
