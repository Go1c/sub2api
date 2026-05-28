//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

// 订阅额度池扩展方法的集成测试。
// 与原 UserSubscriptionRepoSuite 分文件，避免单文件过大；共用 testEntTx 和工具。

type UserSubscriptionCreditSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	repo   *userSubscriptionRepository
}

func (s *UserSubscriptionCreditSuite) SetupTest() {
	s.ctx = context.Background()
	tx := testEntTx(s.T())
	s.client = tx.Client()
	s.repo = NewUserSubscriptionRepository(s.client, integrationDB).(*userSubscriptionRepository)
}

func TestUserSubscriptionCreditSuite(t *testing.T) {
	suite.Run(t, new(UserSubscriptionCreditSuite))
}

// 创建一个测试用户。复用现有 mustCreateUser 风格。
func (s *UserSubscriptionCreditSuite) mustCreateUser(email string) *service.User {
	s.T().Helper()
	u, err := s.client.User.Create().
		SetEmail(email).
		SetPasswordHash("test").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(s.ctx)
	s.Require().NoError(err)
	return userEntityToService(u)
}

// 创建一个 active credit-pool 订阅（group_id NULL，scope=all_available_groups）。
func (s *UserSubscriptionCreditSuite) mustCreateCreditSub(userID int64, mutate func(*dbent.UserSubscriptionCreate)) *dbent.UserSubscription {
	s.T().Helper()
	now := time.Now()
	c := s.client.UserSubscription.Create().
		SetUserID(userID).
		SetScopeType(service.SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetQuotaLimitUsd(10).
		SetQuotaUsedUsd(0).
		SetStartsAt(now.Add(-1 * time.Hour)).
		SetExpiresAt(now.Add(24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("")
	if mutate != nil {
		mutate(c)
	}
	sub, err := c.Save(s.ctx)
	s.Require().NoError(err)
	return sub
}

// ====================================================================================
// GetUsableCreditSubscription
// ====================================================================================

func (s *UserSubscriptionCreditSuite) TestGetUsableCreditSubscription_HappyPath() {
	user := s.mustCreateUser("credit-get-usable@test.com")
	sub := s.mustCreateCreditSub(user.ID, nil)

	got, err := s.repo.GetUsableCreditSubscription(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Equal(sub.ID, got.ID)
	s.Require().Nil(got.GroupID, "credit-pool sub should have nil GroupID")
	s.Require().Equal(service.SubscriptionScopeAllAvailableGroups, got.ScopeType)
	s.Require().InDelta(10.0, got.QuotaLimitUSD, 1e-9)
	s.Require().Nil(got.ExhaustedAt)
}

func (s *UserSubscriptionCreditSuite) TestGetUsableCreditSubscription_ExhaustedExcluded() {
	user := s.mustCreateUser("credit-get-exhausted@test.com")
	now := time.Now()
	s.mustCreateCreditSub(user.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetExhaustedAt(now)
	})

	_, err := s.repo.GetUsableCreditSubscription(s.ctx, user.ID)
	s.Require().ErrorIs(err, service.ErrSubscriptionNotFound)
}

func (s *UserSubscriptionCreditSuite) TestGetUsableCreditSubscription_ExpiredExcluded() {
	user := s.mustCreateUser("credit-get-expired@test.com")
	now := time.Now()
	s.mustCreateCreditSub(user.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetStartsAt(now.Add(-48 * time.Hour))
		c.SetExpiresAt(now.Add(-1 * time.Hour))
	})

	_, err := s.repo.GetUsableCreditSubscription(s.ctx, user.ID)
	s.Require().ErrorIs(err, service.ErrSubscriptionNotFound)
}

func (s *UserSubscriptionCreditSuite) TestGetUsableCreditSubscription_NotActiveExcluded() {
	user := s.mustCreateUser("credit-get-suspended@test.com")
	s.mustCreateCreditSub(user.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetStatus(service.SubscriptionStatusSuspended)
	})

	_, err := s.repo.GetUsableCreditSubscription(s.ctx, user.ID)
	s.Require().ErrorIs(err, service.ErrSubscriptionNotFound)
}

// ====================================================================================
// HasUsableCreditSubscription
// ====================================================================================

func (s *UserSubscriptionCreditSuite) TestHasUsableCreditSubscription_True() {
	user := s.mustCreateUser("credit-has-true@test.com")
	s.mustCreateCreditSub(user.ID, nil)

	has, err := s.repo.HasUsableCreditSubscription(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().True(has)
}

func (s *UserSubscriptionCreditSuite) TestHasUsableCreditSubscription_FalseWhenExhausted() {
	user := s.mustCreateUser("credit-has-exhausted@test.com")
	now := time.Now()
	s.mustCreateCreditSub(user.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetExhaustedAt(now)
	})

	has, err := s.repo.HasUsableCreditSubscription(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().False(has)
}

func (s *UserSubscriptionCreditSuite) TestHasUsableCreditSubscription_FalseWhenNone() {
	user := s.mustCreateUser("credit-has-none@test.com")

	has, err := s.repo.HasUsableCreditSubscription(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().False(has)
}

// ====================================================================================
// GetRenewalEligibility
// ====================================================================================

func (s *UserSubscriptionCreditSuite) TestGetRenewalEligibility_NoSubscription() {
	user := s.mustCreateUser("credit-renew-none@test.com")

	r, err := s.repo.GetRenewalEligibility(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().True(r.Allowed)
	s.Require().Equal(service.RenewalReasonNoSubscription, r.Reason)
	s.Require().Nil(r.Subscription)
}

func (s *UserSubscriptionCreditSuite) TestGetRenewalEligibility_NotExhaustedRejected() {
	user := s.mustCreateUser("credit-renew-notexhausted@test.com")
	sub := s.mustCreateCreditSub(user.ID, nil) // 可消费

	r, err := s.repo.GetRenewalEligibility(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().False(r.Allowed)
	s.Require().Equal(service.RenewalReasonNotExhausted, r.Reason)
	s.Require().NotNil(r.Subscription)
	s.Require().Equal(sub.ID, r.Subscription.ID)
}

func (s *UserSubscriptionCreditSuite) TestGetRenewalEligibility_ExhaustedAllowed() {
	user := s.mustCreateUser("credit-renew-exhausted@test.com")
	now := time.Now()
	sub := s.mustCreateCreditSub(user.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetExhaustedAt(now).SetQuotaUsedUsd(10)
	})

	r, err := s.repo.GetRenewalEligibility(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().True(r.Allowed)
	s.Require().Equal(service.RenewalReasonExhausted, r.Reason)
	s.Require().NotNil(r.Subscription)
	s.Require().Equal(sub.ID, r.Subscription.ID)
}

func (s *UserSubscriptionCreditSuite) TestGetRenewalEligibility_ExpiredAllowed() {
	user := s.mustCreateUser("credit-renew-expired@test.com")
	now := time.Now()
	sub := s.mustCreateCreditSub(user.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetStartsAt(now.Add(-48 * time.Hour))
		c.SetExpiresAt(now.Add(-1 * time.Hour))
	})

	r, err := s.repo.GetRenewalEligibility(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().True(r.Allowed)
	s.Require().Equal(service.RenewalReasonExpired, r.Reason)
	s.Require().NotNil(r.Subscription)
	s.Require().Equal(sub.ID, r.Subscription.ID)
}

// ====================================================================================
// LockUserForSubscriptionWrite
// ====================================================================================

func (s *UserSubscriptionCreditSuite) TestLockUserForSubscriptionWrite_RequiresTx() {
	err := s.repo.LockUserForSubscriptionWrite(s.ctx, nil, 1)
	s.Require().Error(err)
}

// 注意：在事务测试 harness 下 LockUserForSubscriptionWrite 难以独立验证 FOR UPDATE
// 语义（会与 testEntTx 的外层事务嵌套）；这里仅校验函数签名与 nil tx 防御。
// 真实并发行为通过 Task 6 履约集成测试验证（多个并发履约只成功一个）。

// ====================================================================================
// MarkExpiredCreditLogged
// ====================================================================================

func (s *UserSubscriptionCreditSuite) TestMarkExpiredCreditLogged() {
	user := s.mustCreateUser("credit-marklogged@test.com")
	sub := s.mustCreateCreditSub(user.ID, nil)
	loggedAt := time.Now().UTC()

	err := s.repo.MarkExpiredCreditLogged(s.ctx, sub.ID, loggedAt)
	s.Require().NoError(err)

	got, err := s.repo.GetByID(s.ctx, sub.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.ExpiredCreditLoggedAt)
	s.Require().WithinDuration(loggedAt, *got.ExpiredCreditLoggedAt, time.Second)
}

func (s *UserSubscriptionCreditSuite) TestMarkExpiredCreditLogged_NotFound() {
	err := s.repo.MarkExpiredCreditLogged(s.ctx, 9999999, time.Now())
	s.Require().Error(err)
}
