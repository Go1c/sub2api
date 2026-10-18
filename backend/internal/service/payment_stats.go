package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// --- Dashboard & Analytics ---

func (s *PaymentService) GetDashboardStats(ctx context.Context, days int) (*DashboardStats, error) {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	since := todayStart.AddDate(0, 0, -(days - 1))
	until := todayStart.AddDate(0, 0, 1)
	return s.dashboardStatsForRange(ctx, since, until, since, days)
}

// GetDashboardStatsByMonth returns dashboard statistics for a specific calendar month.
// year/month follow Go time semantics (month 1-12).
func (s *PaymentService) GetDashboardStatsByMonth(ctx context.Context, year int, month time.Month) (*DashboardStats, error) {
	loc := time.Now().Location()
	since := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	until := since.AddDate(0, 1, 0)
	days := int(until.Sub(since).Hours() / 24)
	return s.dashboardStatsForRange(ctx, since, until, since, days)
}

func (s *PaymentService) dashboardStatsForRange(ctx context.Context, since, until, firstDay time.Time, days int) (*DashboardStats, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	paidStatuses := []string{OrderStatusCompleted, OrderStatusPaid, OrderStatusRecharging, OrderStatusFulfillmentFailed}

	orders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(paidStatuses...),
			paymentorder.PaidAtGTE(since),
			paymentorder.PaidAtLT(until),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	st := &DashboardStats{}
	revenueOrders := filterRevenueOrders(orders)
	computeBasicStats(st, revenueOrders, todayStart)

	st.PendingOrders, err = s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusEQ(OrderStatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	st.DailySeries = buildDailySeries(revenueOrders, firstDay, days)
	st.PaymentMethods = buildMethodDistribution(revenueOrders)
	st.TopUsers = buildTopUsers(revenueOrders)

	return st, nil
}

func filterRevenueOrders(orders []*dbent.PaymentOrder) []*dbent.PaymentOrder {
	if len(orders) == 0 {
		return nil
	}
	filtered := make([]*dbent.PaymentOrder, 0, len(orders))
	for _, order := range orders {
		if order == nil || order.PaymentType == payment.TypeBalance {
			continue
		}
		filtered = append(filtered, order)
	}
	return filtered
}

func computeBasicStats(st *DashboardStats, orders []*dbent.PaymentOrder, todayStart time.Time) {
	st.TotalAmount = make(CurrencyAmounts)
	st.TodayAmount = make(CurrencyAmounts)
	st.AvgAmount = make(CurrencyAmounts)
	currencyCounts := make(map[string]int)
	var todayCount int
	for _, o := range orders {
		currency := PaymentOrderCurrency(o)
		st.TotalAmount[currency] += o.PayAmount
		currencyCounts[currency]++
		if o.PaidAt != nil && !o.PaidAt.Before(todayStart) {
			st.TodayAmount[currency] += o.PayAmount
			todayCount++
		}
	}
	st.TotalCount = len(orders)
	st.TodayCount = todayCount
	for currency, totalAmount := range st.TotalAmount {
		st.AvgAmount[currency] = roundAmount(totalAmount / float64(currencyCounts[currency]))
	}
	roundCurrencyAmounts(st.TotalAmount)
	roundCurrencyAmounts(st.TodayAmount)
}

// buildDailySeries builds a per-day series covering [firstDay, firstDay+days).
// firstDay is the first day (inclusive, midnight local).
func buildDailySeries(orders []*dbent.PaymentOrder, firstDay time.Time, days int) []DailyStats {
	dailyMap := make(map[string]*DailyStats)
	for _, o := range orders {
		if o.PaidAt == nil {
			continue
		}
		date := o.PaidAt.Format("2006-01-02")
		ds, ok := dailyMap[date]
		if !ok {
			ds = &DailyStats{Date: date, Amount: make(CurrencyAmounts)}
			dailyMap[date] = ds
		}
		ds.Amount[PaymentOrderCurrency(o)] += o.PayAmount
		ds.Count++
	}
	series := make([]DailyStats, 0, days)
	for i := 0; i < days; i++ {
		date := firstDay.AddDate(0, 0, i).Format("2006-01-02")
		if ds, ok := dailyMap[date]; ok {
			roundCurrencyAmounts(ds.Amount)
			series = append(series, *ds)
		} else {
			series = append(series, DailyStats{Date: date, Amount: make(CurrencyAmounts)})
		}
	}
	return series
}

func buildMethodDistribution(orders []*dbent.PaymentOrder) []PaymentMethodStat {
	methodMap := make(map[string]*PaymentMethodStat)
	for _, o := range orders {
		ms, ok := methodMap[o.PaymentType]
		if !ok {
			ms = &PaymentMethodStat{Type: o.PaymentType, Amount: make(CurrencyAmounts)}
			methodMap[o.PaymentType] = ms
		}
		ms.Amount[PaymentOrderCurrency(o)] += o.PayAmount
		ms.Count++
	}
	methods := make([]PaymentMethodStat, 0, len(methodMap))
	for _, ms := range methodMap {
		roundCurrencyAmounts(ms.Amount)
		methods = append(methods, *ms)
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Type < methods[j].Type
	})
	return methods
}

func buildTopUsers(orders []*dbent.PaymentOrder) TopUsersByCurrency {
	userMap := make(map[string]map[int64]*TopUserStat)
	for _, o := range orders {
		currency := PaymentOrderCurrency(o)
		users, ok := userMap[currency]
		if !ok {
			users = make(map[int64]*TopUserStat)
			userMap[currency] = users
		}
		us, ok := users[o.UserID]
		if !ok {
			us = &TopUserStat{UserID: o.UserID, Email: o.UserEmail}
			users[o.UserID] = us
		}
		us.Amount += o.PayAmount
	}
	result := make(TopUsersByCurrency, len(userMap))
	for currency, users := range userMap {
		userList := make([]*TopUserStat, 0, len(users))
		for _, us := range users {
			us.Amount = roundAmount(us.Amount)
			userList = append(userList, us)
		}
		sort.Slice(userList, func(i, j int) bool {
			return userList[i].Amount > userList[j].Amount
		})
		limit := topUsersLimit
		if len(userList) < limit {
			limit = len(userList)
		}
		result[currency] = make([]TopUserStat, 0, limit)
		for i := 0; i < limit; i++ {
			result[currency] = append(result[currency], *userList[i])
		}
	}
	return result
}

func roundCurrencyAmounts(amounts CurrencyAmounts) {
	for currency, amount := range amounts {
		amounts[currency] = roundAmount(amount)
	}
}

func roundAmount(amount float64) float64 {
	return math.Round(amount*100) / 100
}

// --- Audit Logs ---

func (s *PaymentService) writeAuditLog(ctx context.Context, oid int64, action, op string, detail map[string]any) {
	dj, _ := json.Marshal(detail)
	_, err := s.entClient.PaymentAuditLog.Create().SetOrderID(strconv.FormatInt(oid, 10)).SetAction(action).SetDetail(string(dj)).SetOperator(op).Save(ctx)
	if err != nil {
		slog.Error("audit log failed", "orderID", oid, "action", action, "error", err)
	}
}

func (s *PaymentService) GetOrderAuditLogs(ctx context.Context, oid int64) ([]*dbent.PaymentAuditLog, error) {
	return s.entClient.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(oid, 10))).Order(paymentauditlog.ByCreatedAt()).All(ctx)
}
