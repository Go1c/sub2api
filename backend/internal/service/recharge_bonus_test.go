//go:build unit

package service

import (
	"context"
	"math"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// computeRechargeBonusAmount — 按支付金额命中最高档，在到账基数上按百分比赠送
// ---------------------------------------------------------------------------

func TestSnapshotRechargeBonusAmountSkipsBannedUsers(t *testing.T) {
	t.Parallel()
	cfg := &PaymentConfig{
		RechargeBonusEnabled: true,
		RechargeBonusTiers:   []RechargeBonusTier{{MinAmount: 100, BonusPercent: 10}},
	}
	assert.InDelta(t, 10.0, snapshotRechargeBonusAmount(payment.OrderTypeBalance, 100, 100, cfg, false), 1e-9)
	assert.Zero(t, snapshotRechargeBonusAmount(payment.OrderTypeBalance, 100, 100, cfg, true))
	assert.Zero(t, snapshotRechargeBonusAmount(payment.OrderTypeSubscription, 100, 100, cfg, false))
}

func TestVisibleRechargeBonusHidesTagsWhenBanned(t *testing.T) {
	t.Parallel()
	cfg := &PaymentConfig{
		RechargeBonusEnabled: true,
		RechargeBonusTiers:   []RechargeBonusTier{{MinAmount: 100, BonusPercent: 10}},
	}
	enabled, tiers := VisibleRechargeBonus(cfg, false)
	assert.True(t, enabled)
	require.Len(t, tiers, 1)

	enabled, tiers = VisibleRechargeBonus(cfg, true)
	assert.False(t, enabled)
	assert.Empty(t, tiers)
}

func TestComputeRechargeBonusAmount(t *testing.T) {
	t.Parallel()

	tiers := []RechargeBonusTier{
		{MinAmount: 100, BonusPercent: 10},
		{MinAmount: 500, BonusPercent: 16},
		{MinAmount: 1000, BonusPercent: 20},
	}

	tests := []struct {
		name          string
		paymentAmount float64
		baseCredited  float64
		cfg           *PaymentConfig
		want          float64
	}{
		{name: "nil config returns 0", paymentAmount: 1000, baseCredited: 1000, cfg: nil, want: 0},
		{name: "disabled returns 0", paymentAmount: 1000, baseCredited: 1000, cfg: &PaymentConfig{RechargeBonusEnabled: false, RechargeBonusTiers: tiers}, want: 0},
		{name: "empty tiers returns 0", paymentAmount: 1000, baseCredited: 1000, cfg: &PaymentConfig{RechargeBonusEnabled: true}, want: 0},
		{name: "below lowest tier returns 0", paymentAmount: 99.99, baseCredited: 99.99, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 0},
		{name: "exactly at lowest tier hits", paymentAmount: 100, baseCredited: 100, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 10},
		{name: "between tiers takes lower percent on full base", paymentAmount: 499.99, baseCredited: 499.99, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 50},
		{name: "middle tier", paymentAmount: 500, baseCredited: 500, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 80},
		{name: "multiple matches take highest tier percent", paymentAmount: 1500, baseCredited: 1500, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 300},
		{name: "percent applies to credited base not payment amount", paymentAmount: 100, baseCredited: 150, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 15},
		{name: "tier matched by payment amount not credited base", paymentAmount: 100, baseCredited: 50, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 5},
		{name: "discount mode not wired into order flow returns 0", paymentAmount: 1000, baseCredited: 1000, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusMode: RechargeBonusModeDiscount, RechargeBonusTiers: tiers}, want: 0},
		{name: "zero percent tier returns 0", paymentAmount: 100, baseCredited: 100, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: []RechargeBonusTier{{MinAmount: 100}}}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeRechargeBonusAmount(tt.paymentAmount, tt.baseCredited, tt.cfg)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestComputeRechargeBonusAmountHighestMatchWins(t *testing.T) {
	t.Parallel()

	// parseRechargeBonusTiers 输出保证按 MinAmount 升序；命中取不超过支付金额的最大档。
	cfg := &PaymentConfig{
		RechargeBonusEnabled: true,
		RechargeBonusTiers: []RechargeBonusTier{
			{MinAmount: 100, BonusPercent: 10},
			{MinAmount: 500, BonusPercent: 16},
			{MinAmount: 1000, BonusPercent: 20},
		},
	}

	assert.InDelta(t, 96.0, computeRechargeBonusAmount(600, 600, cfg), 1e-9)
	assert.InDelta(t, 19999.8, computeRechargeBonusAmount(99999, 99999, cfg), 1e-9)
}

// ---------------------------------------------------------------------------
// parsePaymentConfig — enabled 开关与 tiers JSON 解析
// ---------------------------------------------------------------------------

func TestParsePaymentConfigRechargeBonusTiers(t *testing.T) {
	t.Parallel()

	svc := &PaymentConfigService{}

	t.Run("enabled parsed from settings key", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusEnabled: "true",
		})
		assert.True(t, cfg.RechargeBonusEnabled)

		cfg = svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusEnabled: "false",
		})
		assert.False(t, cfg.RechargeBonusEnabled)

		// 缺省视为关闭（未配置过开关的部署）
		cfg = svc.parsePaymentConfig(map[string]string{})
		assert.False(t, cfg.RechargeBonusEnabled)
	})

	t.Run("empty string yields empty slice", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusEnabled: "true",
			SettingRechargeBonusTiers:   "",
		})
		assert.True(t, cfg.RechargeBonusEnabled)
		require.NotNil(t, cfg.RechargeBonusTiers)
		assert.Empty(t, cfg.RechargeBonusTiers)
	})

	t.Run("bad JSON yields empty slice without panic", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusTiers: `{"not":"an array"`,
		})
		require.NotNil(t, cfg.RechargeBonusTiers)
		assert.Empty(t, cfg.RechargeBonusTiers)
	})

	t.Run("valid JSON parsed sorted ascending by min_amount", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusEnabled: "true",
			SettingRechargeBonusTiers:   `[{"min_amount":100,"bonus_percent":10},{"min_amount":1000,"bonus_percent":20},{"min_amount":500,"bonus_percent":16}]`,
		})
		assert.True(t, cfg.RechargeBonusEnabled)
		require.Len(t, cfg.RechargeBonusTiers, 3)
		assert.InDelta(t, 100, cfg.RechargeBonusTiers[0].MinAmount, 1e-9)
		assert.InDelta(t, 500, cfg.RechargeBonusTiers[1].MinAmount, 1e-9)
		assert.InDelta(t, 1000, cfg.RechargeBonusTiers[2].MinAmount, 1e-9)
	})

	t.Run("legacy threshold/bonus JSON parses to inert tiers", func(t *testing.T) {
		t.Parallel()
		// 旧固定金额格式的存量配置：解析不报错；旧行的 MinAmount/BonusPercent 均为 0，
		// 去重后只剩一条 0/0 档，下单与展示都按无优惠处理（fail-safe），需要在后台按百分比重新配置。
		cfg := svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusTiers: `[{"threshold":1000,"bonus":200},{"threshold":100,"bonus":10}]`,
		})
		require.Len(t, cfg.RechargeBonusTiers, 1)
		assert.InDelta(t, 0.0, cfg.RechargeBonusTiers[0].BonusPercent, 1e-9)
		assert.Zero(t, computeRechargeBonusAmount(1000, 1000, cfg))
	})
}

// ---------------------------------------------------------------------------
// UpdatePaymentConfig — enabled 开关、折扣模式拒绝、tiers 序列化
// ---------------------------------------------------------------------------

func TestUpdatePaymentConfigRechargeBonusEnabledPersists(t *testing.T) {
	t.Parallel()

	repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
	svc := &PaymentConfigService{settingRepo: repo}

	value := true
	require.NoError(t, svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusEnabled: &value,
	}))
	assert.Equal(t, "true", repo.values[SettingRechargeBonusEnabled])

	value = false
	require.NoError(t, svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusEnabled: &value,
	}))
	assert.Equal(t, "false", repo.values[SettingRechargeBonusEnabled])

	// nil 不写
	repo2 := &paymentConfigSettingRepoStub{values: map[string]string{SettingRechargeBonusEnabled: "true"}}
	svc2 := &PaymentConfigService{settingRepo: repo2}
	require.NoError(t, svc2.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{}))
	assert.Equal(t, "true", repo2.values[SettingRechargeBonusEnabled])
	if _, ok := repo2.updates[SettingRechargeBonusEnabled]; ok {
		t.Fatal("nil enabled must not be written")
	}
}

func TestUpdatePaymentConfigRejectsDiscountMode(t *testing.T) {
	t.Parallel()

	repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
	svc := &PaymentConfigService{settingRepo: repo}

	discount := RechargeBonusModeDiscount
	err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusMode: &discount,
	})
	require.Error(t, err)
	assert.Equal(t, "INVALID_RECHARGE_BONUS_MODE", infraerrors.Reason(err))

	tiers := []RechargeBonusTier{{MinAmount: 100, BonusPercent: 10}}
	err = svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusTiers: &tiers,
	})
	require.NoError(t, err)
	assert.JSONEq(t,
		`[{"min_amount":100,"bonus_percent":10}]`,
		repo.values[SettingRechargeBonusTiers],
	)
}

func TestUpdatePaymentConfigRechargeBonusTiersValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		tiers []RechargeBonusTier
	}{
		{name: "negative min amount rejected", tiers: []RechargeBonusTier{{MinAmount: -100, BonusPercent: 10}}},
		{name: "negative percent rejected", tiers: []RechargeBonusTier{{MinAmount: 100, BonusPercent: -1}}},
		{name: "percent above 1000 rejected", tiers: []RechargeBonusTier{{MinAmount: 100, BonusPercent: 1001}}},
		{name: "min amount with 3 decimals rejected", tiers: []RechargeBonusTier{{MinAmount: 100.123, BonusPercent: 10}}},
		{name: "percent with 3 decimals rejected", tiers: []RechargeBonusTier{{MinAmount: 100, BonusPercent: 10.123}}},
		{name: "NaN min amount rejected", tiers: []RechargeBonusTier{{MinAmount: math.NaN(), BonusPercent: 10}}},
		{name: "Inf percent rejected", tiers: []RechargeBonusTier{{MinAmount: 100, BonusPercent: math.Inf(1)}}},
		{name: "duplicate min amount rejected", tiers: []RechargeBonusTier{{MinAmount: 100, BonusPercent: 10}, {MinAmount: 100, BonusPercent: 20}}},
		{name: "more than 20 tiers rejected", tiers: func() []RechargeBonusTier {
			tiers := make([]RechargeBonusTier, 21)
			for i := range tiers {
				tiers[i] = RechargeBonusTier{MinAmount: float64(i + 1), BonusPercent: 1}
			}
			return tiers
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
			svc := &PaymentConfigService{settingRepo: repo}
			tiers := tt.tiers
			err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
				RechargeBonusTiers: &tiers,
			})
			require.Error(t, err)
			assert.Equal(t, "INVALID_RECHARGE_BONUS_TIERS", infraerrors.Reason(err))
		})
	}
}

func TestUpdatePaymentConfigRechargeBonusTiersPersists(t *testing.T) {
	t.Parallel()

	repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
	svc := &PaymentConfigService{settingRepo: repo}

	tiers := []RechargeBonusTier{
		{MinAmount: 100, BonusPercent: 10},
		{MinAmount: 1000, BonusPercent: 20},
	}
	err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusTiers: &tiers,
	})
	require.NoError(t, err)
	assert.JSONEq(t,
		`[{"min_amount":100,"bonus_percent":10},{"min_amount":1000,"bonus_percent":20}]`,
		repo.values[SettingRechargeBonusTiers],
	)

	// 空数组清空阶梯（存空串，与未配置同形态）
	empty := []RechargeBonusTier{}
	err = svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusTiers: &empty,
	})
	require.NoError(t, err)
	assert.Equal(t, "", repo.values[SettingRechargeBonusTiers])

	// nil slice 不写（与 EnabledTypes 同语义）
	repo2 := &paymentConfigSettingRepoStub{values: map[string]string{SettingRechargeBonusTiers: `[{"min_amount":100,"bonus_percent":10}]`}}
	svc2 := &PaymentConfigService{settingRepo: repo2}
	require.NoError(t, svc2.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{}))
	assert.Equal(t, `[{"min_amount":100,"bonus_percent":10}]`, repo2.values[SettingRechargeBonusTiers])
	if _, ok := repo2.updates[SettingRechargeBonusTiers]; ok {
		t.Fatal("nil tiers must not be written")
	}
}

// ---------------------------------------------------------------------------
// 履约：bonus>0 发放 recharge_bonus 码、幂等、主码已用补发
// ---------------------------------------------------------------------------

func createPaidBonusBalanceOrderForFulfillmentTest(t *testing.T, ctx context.Context, client *dbent.Client, bonus float64) *dbent.PaymentOrder {
	t.Helper()
	order := createPaidBalanceOrderForFulfillmentTest(t, ctx, client)
	if bonus > 0 {
		client.PaymentOrder.UpdateOneID(order.ID).SetBonusAmount(bonus).SaveX(ctx)
	}
	return order
}

func TestExecuteBalanceFulfillmentGrantsRechargeBonus(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	order := createPaidBonusBalanceOrderForFulfillmentTest(t, ctx, client, 25)

	userRepo := &mockUserRepo{getByIDUser: &User{ID: order.UserID}}
	userRepo.updateBalanceFn = func(_ context.Context, _ int64, amount float64) error {
		userRepo.getByIDUser.Balance += amount
		return nil
	}
	redeemRepo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{}}
	redeemService := NewRedeemService(redeemRepo, userRepo, nil, nil, nil, client, nil, nil)
	svc := &PaymentService{entClient: client, redeemService: redeemService, userRepo: userRepo}

	err := svc.ExecuteBalanceFulfillment(ctx, order.ID)
	require.NoError(t, err)

	// 存在 type=recharge_bonus 的已用兑换码且 value=bonus
	bonusCode := redeemRepo.codesByCode[order.RechargeCode+"-B"]
	require.NotNil(t, bonusCode, "bonus redeem code must exist")
	require.Equal(t, RedeemTypeRechargeBonus, bonusCode.Type)
	require.Equal(t, StatusUsed, bonusCode.Status)
	require.InDelta(t, 25.0, bonusCode.Value, 1e-9)
	require.NotNil(t, bonusCode.UsedBy)
	require.Equal(t, order.UserID, *bonusCode.UsedBy)

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status)
	require.InDelta(t, order.Amount+25.0, userRepo.getByIDUser.Balance, 1e-9)

	// 重放履约（模拟重试）不重复加余额
	_, err = client.PaymentOrder.UpdateOneID(order.ID).
		SetStatus(OrderStatusFulfillmentFailed).
		ClearCompletedAt().
		Save(ctx)
	require.NoError(t, err)

	err = svc.ExecuteBalanceFulfillment(ctx, order.ID)
	require.NoError(t, err)
	require.InDelta(t, order.Amount+25.0, userRepo.getByIDUser.Balance, 1e-9)
	require.Equal(t, 2, redeemRepo.createCalls, "main + bonus code each created exactly once")
	require.Equal(t, 2, redeemRepo.useCallCount())
}

func TestExecuteBalanceFulfillmentWithoutBonusSkipsBonusCode(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	order := createPaidBonusBalanceOrderForFulfillmentTest(t, ctx, client, 0)

	userRepo := &mockUserRepo{getByIDUser: &User{ID: order.UserID}}
	userRepo.updateBalanceFn = func(_ context.Context, _ int64, amount float64) error {
		userRepo.getByIDUser.Balance += amount
		return nil
	}
	redeemRepo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{}}
	redeemService := NewRedeemService(redeemRepo, userRepo, nil, nil, nil, client, nil, nil)
	svc := &PaymentService{entClient: client, redeemService: redeemService, userRepo: userRepo}

	err := svc.ExecuteBalanceFulfillment(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, redeemRepo.codesByCode[order.RechargeCode+"-B"])
	assert.Equal(t, 1, redeemRepo.createCalls)
	assert.InDelta(t, order.Amount, userRepo.getByIDUser.Balance, 1e-9)
}

func TestExecuteBalanceFulfillmentBackfillsBonusWhenMainCodeAlreadyUsed(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	order := createPaidBonusBalanceOrderForFulfillmentTest(t, ctx, client, 25)

	userRepo := &mockUserRepo{getByIDUser: &User{ID: order.UserID}}
	userRepo.updateBalanceFn = func(_ context.Context, _ int64, amount float64) error {
		userRepo.getByIDUser.Balance += amount
		return nil
	}
	// 主码已使用（上次履约成功入账但 bonus 发放失败），bonus 码缺失。
	// 已用主码必须带上本单用户，否则会被兑换码归属校验拒绝。
	usedBy := order.UserID
	redeemRepo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{
		order.RechargeCode: {
			ID:     1,
			Code:   order.RechargeCode,
			Type:   RedeemTypeBalance,
			Value:  order.Amount,
			Status: StatusUsed,
			UsedBy: &usedBy,
		},
	}}
	redeemService := NewRedeemService(redeemRepo, userRepo, nil, nil, nil, client, nil, nil)
	svc := &PaymentService{entClient: client, redeemService: redeemService, userRepo: userRepo}

	err := svc.ExecuteBalanceFulfillment(ctx, order.ID)
	require.NoError(t, err)

	bonusCode := redeemRepo.codesByCode[order.RechargeCode+"-B"]
	require.NotNil(t, bonusCode, "bonus must be backfilled on the SkipCompleted path")
	require.Equal(t, RedeemTypeRechargeBonus, bonusCode.Type)
	require.Equal(t, StatusUsed, bonusCode.Status)
	// 主码不再重复入账，只有 bonus 加余额
	require.InDelta(t, 25.0, userRepo.getByIDUser.Balance, 1e-9)
	require.Equal(t, 1, redeemRepo.createCalls)

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status)
}
