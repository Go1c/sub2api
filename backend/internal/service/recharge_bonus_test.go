//go:build unit

package service

import (
	"context"
	"math"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// computeRechargeBonusAmount — 单档满赠、取满足条件的最高档
// ---------------------------------------------------------------------------

func TestComputeRechargeBonusAmount(t *testing.T) {
	t.Parallel()

	tiers := []RechargeBonusTier{
		{Threshold: 1000, Bonus: 200},
		{Threshold: 500, Bonus: 80},
		{Threshold: 100, Bonus: 10},
	}

	tests := []struct {
		name    string
		amount  float64
		cfg     *PaymentConfig
		want    float64
		wantHit bool
	}{
		{name: "nil config returns 0", amount: 1000, cfg: nil, want: 0},
		{name: "disabled returns 0", amount: 1000, cfg: &PaymentConfig{RechargeBonusEnabled: false, RechargeBonusTiers: tiers}, want: 0},
		{name: "empty tiers returns 0", amount: 1000, cfg: &PaymentConfig{RechargeBonusEnabled: true}, want: 0},
		{name: "below lowest threshold returns 0", amount: 99.99, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 0},
		{name: "exactly at lowest threshold hits", amount: 100, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 10},
		{name: "between tiers takes lower", amount: 499.99, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 10},
		{name: "middle tier", amount: 500, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 80},
		{name: "multiple matches take highest", amount: 1500, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 200},
		{name: "decimal amount rounds to 2 places", amount: 1000.005, cfg: &PaymentConfig{RechargeBonusEnabled: true, RechargeBonusTiers: tiers}, want: 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeRechargeBonusAmount(tt.amount, tt.cfg)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestComputeRechargeBonusAmountUnsortedTiersStillFindsHighestMatch(t *testing.T) {
	t.Parallel()

	// parsePaymentConfig 输出的 tiers 保证降序；这里验证降序输入的检索语义：
	// 找第一个 threshold <= creditedAmount 的档（即满足条件的最高档）。
	cfg := &PaymentConfig{
		RechargeBonusEnabled: true,
		RechargeBonusTiers: []RechargeBonusTier{
			{Threshold: 1000, Bonus: 200},
			{Threshold: 500, Bonus: 80},
			{Threshold: 100, Bonus: 10},
		},
	}

	assert.InDelta(t, 80.0, computeRechargeBonusAmount(600, cfg), 1e-9)
	assert.InDelta(t, 200.0, computeRechargeBonusAmount(99999, cfg), 1e-9)
}

// ---------------------------------------------------------------------------
// normalizeRechargeBonusTiers — 非法项剔除、去重、降序
// ---------------------------------------------------------------------------

func TestNormalizeRechargeBonusTiers(t *testing.T) {
	t.Parallel()

	t.Run("drops invalid entries", func(t *testing.T) {
		t.Parallel()
		got := normalizeRechargeBonusTiers([]RechargeBonusTier{
			{Threshold: 100, Bonus: 10},
			{Threshold: 0, Bonus: 5},                       // threshold <= 0
			{Threshold: -1, Bonus: 5},                      // threshold < 0
			{Threshold: 200, Bonus: 0},                     // bonus <= 0
			{Threshold: 300, Bonus: -5},                    // bonus < 0
			{Threshold: math.NaN(), Bonus: 5},              // NaN threshold
			{Threshold: 400, Bonus: math.Inf(1)},           // Inf bonus
			{Threshold: math.Inf(-1), Bonus: math.Inf(-1)}, // both invalid
			{Threshold: 500, Bonus: 50},
		})
		require.Len(t, got, 2)
		assert.InDelta(t, 500, got[0].Threshold, 1e-9)
		assert.InDelta(t, 100, got[1].Threshold, 1e-9)
	})

	t.Run("keeps first tier for duplicate thresholds", func(t *testing.T) {
		t.Parallel()
		got := normalizeRechargeBonusTiers([]RechargeBonusTier{
			{Threshold: 100, Bonus: 10},
			{Threshold: 100, Bonus: 20},
			{Threshold: 200, Bonus: 30},
		})
		require.Len(t, got, 2)
		assert.InDelta(t, 200, got[0].Threshold, 1e-9)
		assert.InDelta(t, 30, got[0].Bonus, 1e-9)
		assert.InDelta(t, 100, got[1].Threshold, 1e-9)
		assert.InDelta(t, 10, got[1].Bonus, 1e-9) // 首个保留
	})

	t.Run("sorts descending by threshold", func(t *testing.T) {
		t.Parallel()
		got := normalizeRechargeBonusTiers([]RechargeBonusTier{
			{Threshold: 100, Bonus: 10},
			{Threshold: 1000, Bonus: 200},
			{Threshold: 500, Bonus: 80},
		})
		require.Len(t, got, 3)
		assert.InDelta(t, 1000, got[0].Threshold, 1e-9)
		assert.InDelta(t, 500, got[1].Threshold, 1e-9)
		assert.InDelta(t, 100, got[2].Threshold, 1e-9)
	})

	t.Run("nil input returns empty slice", func(t *testing.T) {
		t.Parallel()
		got := normalizeRechargeBonusTiers(nil)
		require.NotNil(t, got)
		assert.Empty(t, got)
	})
}

// ---------------------------------------------------------------------------
// parsePaymentConfig — tiers JSON 解析容错
// ---------------------------------------------------------------------------

func TestParsePaymentConfigRechargeBonusTiers(t *testing.T) {
	t.Parallel()

	svc := &PaymentConfigService{}

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

	t.Run("valid JSON parsed normalized descending", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{
			SettingRechargeBonusEnabled: "true",
			SettingRechargeBonusTiers:   `[{"threshold":100,"bonus":10},{"threshold":1000,"bonus":200},{"threshold":0,"bonus":5}]`,
		})
		assert.True(t, cfg.RechargeBonusEnabled)
		require.Len(t, cfg.RechargeBonusTiers, 2)
		assert.InDelta(t, 1000, cfg.RechargeBonusTiers[0].Threshold, 1e-9)
		assert.InDelta(t, 100, cfg.RechargeBonusTiers[1].Threshold, 1e-9)
	})
}

// ---------------------------------------------------------------------------
// UpdatePaymentConfig — tiers 校验与序列化
// ---------------------------------------------------------------------------

func TestUpdatePaymentConfigRechargeBonusTiersValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		tiers []RechargeBonusTier
	}{
		{name: "zero threshold rejected", tiers: []RechargeBonusTier{{Threshold: 0, Bonus: 10}}},
		{name: "negative threshold rejected", tiers: []RechargeBonusTier{{Threshold: -100, Bonus: 10}}},
		{name: "zero bonus rejected", tiers: []RechargeBonusTier{{Threshold: 100, Bonus: 0}}},
		{name: "negative bonus rejected", tiers: []RechargeBonusTier{{Threshold: 100, Bonus: -1}}},
		{name: "threshold with 3 decimals rejected", tiers: []RechargeBonusTier{{Threshold: 100.123, Bonus: 10}}},
		{name: "bonus with 3 decimals rejected", tiers: []RechargeBonusTier{{Threshold: 100, Bonus: 10.123}}},
		{name: "NaN threshold rejected", tiers: []RechargeBonusTier{{Threshold: math.NaN(), Bonus: 10}}},
		{name: "Inf bonus rejected", tiers: []RechargeBonusTier{{Threshold: 100, Bonus: math.Inf(1)}}},
		{name: "duplicate threshold rejected", tiers: []RechargeBonusTier{{Threshold: 100, Bonus: 10}, {Threshold: 100, Bonus: 20}}},
		{name: "more than 20 tiers rejected", tiers: func() []RechargeBonusTier {
			tiers := make([]RechargeBonusTier, 21)
			for i := range tiers {
				tiers[i] = RechargeBonusTier{Threshold: float64(i + 1), Bonus: 1}
			}
			return tiers
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
			svc := &PaymentConfigService{settingRepo: repo}
			err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
				RechargeBonusEnabled: new(bool),
				RechargeBonusTiers:   tt.tiers,
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

	enabled := true
	err := svc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		RechargeBonusEnabled: &enabled,
		RechargeBonusTiers: []RechargeBonusTier{
			{Threshold: 100, Bonus: 10},
			{Threshold: 1000, Bonus: 200},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "true", repo.values[SettingRechargeBonusEnabled])
	assert.JSONEq(t,
		`[{"threshold":100,"bonus":10},{"threshold":1000,"bonus":200}]`,
		repo.values[SettingRechargeBonusTiers],
	)

	// nil slice 不写（与 EnabledTypes 同语义）
	repo2 := &paymentConfigSettingRepoStub{values: map[string]string{SettingRechargeBonusTiers: `[{"threshold":100,"bonus":10}]`}}
	svc2 := &PaymentConfigService{settingRepo: repo2}
	require.NoError(t, svc2.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{}))
	assert.Equal(t, `[{"threshold":100,"bonus":10}]`, repo2.values[SettingRechargeBonusTiers])
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
	require.Nil(t, redeemRepo.codesByCode[order.RechargeCode+"-B"])
	require.Equal(t, 1, redeemRepo.createCalls)
	require.InDelta(t, order.Amount, userRepo.getByIDUser.Balance, 1e-9)
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
	// 主码已使用（上次履约成功入账但 bonus 发放失败），bonus 码缺失
	redeemRepo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{
		order.RechargeCode: {
			ID:     1,
			Code:   order.RechargeCode,
			Type:   RedeemTypeBalance,
			Value:  order.Amount,
			Status: StatusUsed,
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
