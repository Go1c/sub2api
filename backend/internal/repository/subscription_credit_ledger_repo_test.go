//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

// SubscriptionCreditLedgerRepoSuite 测试订阅额度流水仓储。
// 使用 integrationDB（真实 PG，非事务）以便测试 ON CONFLICT 幂等语义。
type SubscriptionCreditLedgerRepoSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	repo   service.SubscriptionCreditLedgerRepository
}

func (s *SubscriptionCreditLedgerRepoSuite) SetupTest() {
	s.ctx = context.Background()
	tx := testEntTx(s.T())
	s.client = tx.Client()
	// ledger repo 直接走 integrationDB（绕过 tx），保证 ON CONFLICT 真实生效
	s.repo = NewSubscriptionCreditLedgerRepository(integrationDB)
	// 清理本测试用的 ledger 行（防止上一次失败 PASS 残留）
	_, _ = integrationDB.ExecContext(s.ctx, `DELETE FROM subscription_credit_ledger WHERE reason LIKE 'ledger-test:%'`)
}

func TestSubscriptionCreditLedgerRepoSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionCreditLedgerRepoSuite))
}

func (s *SubscriptionCreditLedgerRepoSuite) mustCreateUserAndSub() (int64, int64) {
	s.T().Helper()
	u, err := s.client.User.Create().
		SetEmail("ledger-" + time.Now().Format("150405.000000000") + "@test.com").
		SetPasswordHash("test").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(s.ctx)
	s.Require().NoError(err)

	now := time.Now()
	sub, err := s.client.UserSubscription.Create().
		SetUserID(u.ID).
		SetScopeType(service.SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetQuotaLimitUsd(10).
		SetQuotaUsedUsd(0).
		SetStartsAt(now.Add(-1 * time.Hour)).
		SetExpiresAt(now.Add(24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("").
		Save(s.ctx)
	s.Require().NoError(err)
	return u.ID, sub.ID
}

func strPtr(s string) *string { return &s }

// ====================================================================================
// Create
// ====================================================================================

func (s *SubscriptionCreditLedgerRepoSuite) TestCreate_Purchase() {
	userID, subID := s.mustCreateUserAndSub()

	err := s.repo.Create(s.ctx, nil, &service.SubscriptionCreditLedgerEntry{
		UserID:            userID,
		SubscriptionID:    subID,
		Type:              service.SubscriptionCreditLedgerPurchase,
		DeltaUSD:          10,
		RemainingAfterUSD: 10,
		Reason:            "ledger-test: purchase",
	})
	s.Require().NoError(err)

	entries, _, err := s.repo.ListBySubscriptionID(s.ctx, subID, "", pagination.PaginationParams{Page: 1, PageSize: 20})
	s.Require().NoError(err)
	s.Require().Len(entries, 1)
	s.Require().Equal(service.SubscriptionCreditLedgerPurchase, entries[0].Type)
	s.Require().InDelta(10.0, entries[0].DeltaUSD, 1e-9)
	s.Require().InDelta(10.0, entries[0].RemainingAfterUSD, 1e-9)
}

func (s *SubscriptionCreditLedgerRepoSuite) TestCreate_WithMetadata() {
	userID, subID := s.mustCreateUserAndSub()

	err := s.repo.Create(s.ctx, nil, &service.SubscriptionCreditLedgerEntry{
		UserID:            userID,
		SubscriptionID:    subID,
		Type:              service.SubscriptionCreditLedgerWindowReset,
		DeltaUSD:          0,
		RemainingAfterUSD: 10,
		Reason:            "ledger-test: daily window reset",
		EventKey:          strPtr("window_reset_daily:" + s.formatSubKey(subID)),
		Metadata: map[string]any{
			"window":                "daily",
			"limit_usd":             5.0,
			"used_before_reset_usd": 3.5,
			"wasted_usd":            1.5,
			"wasted_ratio":          0.3,
		},
	})
	s.Require().NoError(err)

	entries, _, err := s.repo.ListBySubscriptionID(s.ctx, subID, "", pagination.PaginationParams{Page: 1, PageSize: 20})
	s.Require().NoError(err)
	s.Require().Len(entries, 1)
	s.Require().Equal("daily", entries[0].Metadata["window"])
	s.Require().InDelta(0.3, entries[0].Metadata["wasted_ratio"].(float64), 1e-9)
}

func (s *SubscriptionCreditLedgerRepoSuite) formatSubKey(subID int64) string {
	return time.Now().UTC().Format(time.RFC3339Nano) + ":" + intToStr(subID)
}

// ====================================================================================
// CreateLimitReachedEvent 幂等
// ====================================================================================

func (s *SubscriptionCreditLedgerRepoSuite) TestCreateLimitReachedEvent_Idempotent() {
	userID, subID := s.mustCreateUserAndSub()

	entry := &service.SubscriptionCreditLedgerEntry{
		UserID:            userID,
		SubscriptionID:    subID,
		Type:              service.SubscriptionCreditLedgerLimitReached,
		DeltaUSD:          0,
		RemainingAfterUSD: 0,
		Reason:            "ledger-test: total exhausted",
		EventKey:          strPtr("total:" + intToStr(subID)),
	}

	// 首次写入 → created=true
	created, err := s.repo.CreateLimitReachedEvent(s.ctx, nil, entry)
	s.Require().NoError(err)
	s.Require().True(created, "first write must be created")

	// 二次写入相同 event_key → created=false
	created2, err := s.repo.CreateLimitReachedEvent(s.ctx, nil, entry)
	s.Require().NoError(err)
	s.Require().False(created2, "second write must be deduplicated")

	// ledger 中只有 1 条
	entries, _, err := s.repo.ListBySubscriptionID(s.ctx, subID, "", pagination.PaginationParams{Page: 1, PageSize: 20})
	s.Require().NoError(err)
	s.Require().Len(entries, 1)
}

func (s *SubscriptionCreditLedgerRepoSuite) TestCreateLimitReachedEvent_RequiresEventKey() {
	userID, subID := s.mustCreateUserAndSub()

	_, err := s.repo.CreateLimitReachedEvent(s.ctx, nil, &service.SubscriptionCreditLedgerEntry{
		UserID:            userID,
		SubscriptionID:    subID,
		Type:              service.SubscriptionCreditLedgerLimitReached,
		DeltaUSD:          0,
		RemainingAfterUSD: 0,
		Reason:            "ledger-test: missing key",
		EventKey:          nil,
	})
	s.Require().Error(err, "must reject nil event_key")
}

// ====================================================================================
// ListByUserID / ListBySubscriptionID 排序 + 分页
// ====================================================================================

func (s *SubscriptionCreditLedgerRepoSuite) TestList_OrderByCreatedAtDescAndPaginate() {
	userID, subID := s.mustCreateUserAndSub()

	// 写 5 条 ledger（不同 event_key 避免冲突）
	for i := 1; i <= 5; i++ {
		err := s.repo.Create(s.ctx, nil, &service.SubscriptionCreditLedgerEntry{
			UserID:            userID,
			SubscriptionID:    subID,
			Type:              service.SubscriptionCreditLedgerConsume,
			DeltaUSD:          -float64(i),
			RemainingAfterUSD: 10 - float64(i),
			Reason:            "ledger-test: consume " + intToStr(int64(i)),
		})
		s.Require().NoError(err)
		time.Sleep(2 * time.Millisecond) // 拉开 created_at 顺序
	}

	// 第 1 页（PageSize=2）：拿最新 2 条
	entries, page, err := s.repo.ListByUserID(s.ctx, userID, "", pagination.PaginationParams{Page: 1, PageSize: 2})
	s.Require().NoError(err)
	s.Require().Len(entries, 2)
	s.Require().InDelta(-5.0, entries[0].DeltaUSD, 1e-9)
	s.Require().InDelta(-4.0, entries[1].DeltaUSD, 1e-9)
	s.Require().Equal(int64(5), page.Total)
	s.Require().Equal(3, page.Pages)

	// 第 3 页（PageSize=2）：只剩 1 条（最早的）
	entries3, _, err := s.repo.ListByUserID(s.ctx, userID, "", pagination.PaginationParams{Page: 3, PageSize: 2})
	s.Require().NoError(err)
	s.Require().Len(entries3, 1)
	s.Require().InDelta(-1.0, entries3[0].DeltaUSD, 1e-9)
}

// 辅助：int64 → string（避免 strconv 占用）
func intToStr(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
