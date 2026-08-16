//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestResolveRebateRatePercent_PerUserOverride verifies that per-inviter
// AffRebateRatePercent overrides the global rate, that NULL falls back to the
// global rate, and that out-of-range exclusive rates are clamped silently.
//
// SettingService is left nil here so no tier config exists; this must not fall
// back to the legacy 20% default.
func TestResolveRebateRatePercent_PerUserOverride(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}

	// nil exclusive rate + no tier config → unconfigured, not a silent 20% fallback.
	require.Nil(t,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{}, 0))

	// exclusive rate set → overrides global
	rate := 50.0
	resolved := svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &rate}, 0)
	require.NotNil(t, resolved)
	require.InDelta(t, 50.0, *resolved, 1e-9)

	// exclusive rate 0 → returns 0 (no rebate, intentional)
	zero := 0.0
	resolved = svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &zero}, 0)
	require.NotNil(t, resolved)
	require.InDelta(t, 0.0, *resolved, 1e-9)

	// exclusive rate above max → clamped to Max
	tooHigh := 250.0
	resolved = svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooHigh}, 0)
	require.NotNil(t, resolved)
	require.InDelta(t, AffiliateRebateRateMax, *resolved, 1e-9)

	// exclusive rate below min → clamped to Min
	tooLow := -5.0
	resolved = svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooLow}, 0)
	require.NotNil(t, resolved)
	require.InDelta(t, AffiliateRebateRateMin, *resolved, 1e-9)
}

func TestResolveRebateRatePercent_UsesHighestConfiguredTier(t *testing.T) {
	t.Parallel()

	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateRebateTiers: `[
			{"level":"L1","min_invitees":5,"min_recharge":500,"rebate_rate_percent":15},
			{"level":"L2","min_invitees":20,"min_recharge":5000,"rebate_rate_percent":25},
			{"level":"L3","min_invitees":50,"min_recharge":20000,"rebate_rate_percent":35},
			{"level":"L4","min_invitees":100,"min_recharge":50000,"rebate_rate_percent":45}
		]`,
	}}, nil)
	svc := NewAffiliateService(newAffiliateSignupBonusRepoStub(), settingSvc, nil, nil)

	resolved := svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffCount: 50}, 19999)
	require.NotNil(t, resolved)
	require.InDelta(t, 25.0, *resolved, 1e-9, "invite count alone must not advance to L3")

	resolved = svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffCount: 20}, 5000)
	require.NotNil(t, resolved)
	require.InDelta(t, 25.0, *resolved, 1e-9)

	resolved = svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffCount: 120}, 60000)
	require.NotNil(t, resolved)
	require.InDelta(t, 45.0, *resolved, 1e-9)
}

func TestGetAffiliateDetailIncludesTierProgress(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	repo.profiles[1].AffCount = 20
	repo.inviteeRechargeTotal = 5000
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateRebateTiers: `[
			{"level":"L1","min_invitees":5,"min_recharge":500,"rebate_rate_percent":15},
			{"level":"L2","min_invitees":20,"min_recharge":5000,"rebate_rate_percent":25},
			{"level":"L3","min_invitees":50,"min_recharge":20000,"rebate_rate_percent":35},
			{"level":"L4","min_invitees":100,"min_recharge":50000,"rebate_rate_percent":45}
		]`,
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 1)

	require.NoError(t, err)
	require.InDelta(t, 5000, detail.InviteeRechargeTotal, 1e-9)
	require.NotNil(t, detail.EffectiveRebateRatePercent)
	require.InDelta(t, 25, *detail.EffectiveRebateRatePercent, 1e-9)
	require.NotNil(t, detail.CurrentAffiliateTier)
	require.Equal(t, "L2", detail.CurrentAffiliateTier.Level)
	require.NotNil(t, detail.NextAffiliateTier)
	require.Equal(t, "L3", detail.NextAffiliateTier.Level)
	require.Len(t, detail.AffiliateTiers, 4)
}

func TestGetAffiliateDetailIncludesRuntimeRulesDefaults(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 1)

	require.NoError(t, err)
	assertAffiliateRuntimeRulesJSON(t, detail, map[string]any{
		"rebate_freeze_hours":    float64(AffiliateRebateFreezeHoursDefault),
		"rebate_duration_days":   float64(AffiliateRebateDurationDaysDefault),
		"rebate_per_invitee_cap": AffiliateRebatePerInviteeCapDefault,
		"signup_bonus_enabled":   AffiliateSignupBonusEnabledDefault,
		"signup_bonus_amount":    AffiliateSignupBonusAmountDefault,
	})
}

func TestGetAffiliateDetailIncludesRuntimeRulesFromSettings(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateRebateFreezeHours:   "24",
		SettingKeyAffiliateRebateDurationDays:  "90",
		SettingKeyAffiliateRebatePerInviteeCap: "12.5",
		SettingKeyAffiliateSignupBonusEnabled:  "true",
		SettingKeyAffiliateSignupBonusAmount:   "2.5",
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 1)

	require.NoError(t, err)
	assertAffiliateRuntimeRulesJSON(t, detail, map[string]any{
		"rebate_freeze_hours":    float64(24),
		"rebate_duration_days":   float64(90),
		"rebate_per_invitee_cap": 12.5,
		"signup_bonus_enabled":   true,
		"signup_bonus_amount":    2.5,
	})
}

func TestGetAffiliateDetailRuntimeRulesUseEffectiveClampedValues(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateRebateFreezeHours:  "99999",
		SettingKeyAffiliateRebateDurationDays: "-3",
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 1)

	require.NoError(t, err)
	assertAffiliateRuntimeRulesJSON(t, detail, map[string]any{
		"rebate_freeze_hours":    float64(AffiliateRebateFreezeHoursMax),
		"rebate_duration_days":   float64(AffiliateRebateDurationDaysDefault),
		"rebate_per_invitee_cap": AffiliateRebatePerInviteeCapDefault,
		"signup_bonus_enabled":   AffiliateSignupBonusEnabledDefault,
		"signup_bonus_amount":    AffiliateSignupBonusAmountDefault,
	})
}

func TestGetAffiliateDetailRuntimeRulesWhenSettingServiceNil(t *testing.T) {
	t.Parallel()

	svc := NewAffiliateService(newAffiliateSignupBonusRepoStub(), nil, nil, nil)

	detail, err := svc.GetAffiliateDetail(context.Background(), 1)

	require.NoError(t, err)
	assertAffiliateRuntimeRulesJSON(t, detail, map[string]any{
		"rebate_freeze_hours":    float64(AffiliateRebateFreezeHoursDefault),
		"rebate_duration_days":   float64(AffiliateRebateDurationDaysDefault),
		"rebate_per_invitee_cap": AffiliateRebatePerInviteeCapDefault,
		"signup_bonus_enabled":   AffiliateSignupBonusEnabledDefault,
		"signup_bonus_amount":    AffiliateSignupBonusAmountDefault,
	})
}

func assertAffiliateRuntimeRulesJSON(t *testing.T, detail *AffiliateDetail, want map[string]any) {
	t.Helper()
	raw, err := json.Marshal(detail)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	rules, ok := payload["rules"].(map[string]any)
	require.True(t, ok, "GET /user/aff data must include nested rules object, got %s", raw)
	require.Equal(t, want, rules)
}

func TestAccrueInviteRebateForOrderUsesProspectiveRechargeTotal(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	inviterID := int64(1)
	inviteeID := int64(2)
	repo.profiles[inviteeID].InviterID = &inviterID
	repo.profiles[inviterID].AffCount = 5
	repo.inviteeRechargeTotal = 490
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled: "true",
		SettingKeyAffiliateRebateTiers: `[
			{"level":"L1","min_invitees":5,"min_recharge":500,"rebate_rate_percent":15},
			{"level":"L2","min_invitees":20,"min_recharge":5000,"rebate_rate_percent":25},
			{"level":"L3","min_invitees":50,"min_recharge":20000,"rebate_rate_percent":35},
			{"level":"L4","min_invitees":100,"min_recharge":50000,"rebate_rate_percent":45}
		]`,
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	orderID := int64(99)
	rebate, err := svc.AccrueInviteRebateForOrder(context.Background(), inviteeID, 10, &orderID)

	require.NoError(t, err)
	require.InDelta(t, 1.5, rebate, 1e-9)
	require.Len(t, repo.accrueCalls, 1)
	require.InDelta(t, 1.5, repo.accrueCalls[0].amount, 1e-9)
}

func TestAccrueInviteRebateUsesPersistedRechargeTotal(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	inviterID := int64(1)
	inviteeID := int64(2)
	repo.profiles[inviteeID].InviterID = &inviterID
	repo.profiles[inviterID].AffCount = 5
	repo.inviteeRechargeTotal = 500
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled: "true",
		SettingKeyAffiliateRebateTiers: `[
			{"level":"L1","min_invitees":5,"min_recharge":500,"rebate_rate_percent":15},
			{"level":"L2","min_invitees":5,"min_recharge":510,"rebate_rate_percent":25},
			{"level":"L3","min_invitees":50,"min_recharge":20000,"rebate_rate_percent":35},
			{"level":"L4","min_invitees":100,"min_recharge":50000,"rebate_rate_percent":45}
		]`,
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	rebate, err := svc.AccrueInviteRebate(context.Background(), inviteeID, 10)

	require.NoError(t, err)
	require.InDelta(t, 1.5, rebate, 1e-9)
	require.Len(t, repo.accrueCalls, 1)
	require.InDelta(t, 1.5, repo.accrueCalls[0].amount, 1e-9)
}

// TestIsEnabled_NilSettingServiceReturnsDefault verifies that IsEnabled
// safely handles a nil settingService dependency by returning the default
// (off). This protects callers from nil-pointer crashes in misconfigured
// environments.
func TestIsEnabled_NilSettingServiceReturnsDefault(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}
	require.False(t, svc.IsEnabled(context.Background()))
	require.Equal(t, AffiliateEnabledDefault, svc.IsEnabled(context.Background()))
}

func TestBindInviterByCode_AwardsRegistrationBonusWhenEnabled(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled:             "true",
		SettingKeyAffiliateSignupBonusEnabled:  "true",
		SettingKeyAffiliateSignupBonusAmount:   "2.5",
		SettingKeyAffiliateSignupBonusTotalCap: "10",
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	err := svc.BindInviterByCode(context.Background(), 2, "invite123")

	require.NoError(t, err)
	require.Len(t, repo.bindWithBonusCalls, 1)
	call := repo.bindWithBonusCalls[0]
	require.Equal(t, int64(2), call.userID)
	require.Equal(t, int64(1), call.inviterID)
	require.InDelta(t, 2.5, call.bonusAmount, 1e-9)
	require.InDelta(t, 10.0, call.bonusTotalCap, 1e-9)
}

func TestBindInviterByCode_RecordsFailureReasonAndFingerprint(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled: "true",
	}}, nil)
	svc := NewAffiliateService(repo, settingSvc, nil, nil)
	ctx := ContextWithAffiliateSignupRequestMeta(
		context.Background(),
		NewAffiliateSignupRequestMeta("device-abc", "203.0.113.10", "unit-test-agent"),
	)

	err := svc.BindInviterByCode(ctx, 2, "bad code with spaces")

	require.ErrorIs(t, err, ErrAffiliateCodeInvalid)
	require.Len(t, repo.recordedLogs, 1)
	require.Equal(t, "invalid_code", repo.recordedLogs[0].FailureReason)
	require.False(t, repo.recordedLogs[0].Success)
	require.NotEmpty(t, repo.recordedLogs[0].FingerprintHash)
	require.Equal(t, "203.0.113.10", repo.recordedLogs[0].IPAddress)
	require.Equal(t, "unit-test-agent", repo.recordedLogs[0].UserAgent)
}

func TestListInviteLogs_UserScopeScrubsSensitiveAndKeepsFailureReason(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	createdAt := time.Now()
	inviterID := int64(7)
	inviteeID := int64(8)
	repo.listLogsResult = []AffiliateInviteLog{{
		ID:              11,
		InviterID:       &inviterID,
		InviterEmail:    "inviter@example.com",
		InviterUsername: "inviter",
		InviteeID:       &inviteeID,
		InviteeEmail:    "invitee@example.com",
		InviteeUsername: "invitee",
		AffiliateCode:   "INVITE123",
		Success:         false,
		FailureReason:   "fingerprint_reused",
		FingerprintHash: "secret-fingerprint",
		IPAddress:       "203.0.113.20",
		UserAgent:       "secret-agent",
		CreatedAt:       createdAt,
	}}
	repo.listLogsTotal = 1
	svc := NewAffiliateService(repo, nil, nil, nil)

	items, total, err := svc.ListInviteLogs(context.Background(), 7, 2, 10)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, int64(7), repo.lastListFilter.AccountID)
	require.False(t, repo.lastListFilter.IncludeSensitive)
	require.Equal(t, 2, repo.lastListFilter.Page)
	require.Equal(t, 10, repo.lastListFilter.PageSize)
	require.Equal(t, "fingerprint_reused", items[0].FailureReason)
	require.Contains(t, items[0].FailureMessage, "设备指纹")
	require.Empty(t, items[0].FingerprintHash)
	require.Empty(t, items[0].IPAddress)
	require.Empty(t, items[0].UserAgent)
	require.Equal(t, "i***@e***.com", items[0].InviterEmail)
	require.Equal(t, "i***@e***.com", items[0].InviteeEmail)
}

func TestAdminListInviteLogs_IncludesSensitiveFieldsAndFilters(t *testing.T) {
	t.Parallel()

	repo := newAffiliateSignupBonusRepoStub()
	repo.listLogsResult = []AffiliateInviteLog{{
		ID:              12,
		Success:         true,
		BonusAmount:     3.5,
		FingerprintHash: "secret-fingerprint",
		IPAddress:       "203.0.113.21",
		UserAgent:       "secret-agent",
	}}
	repo.listLogsTotal = 1
	svc := NewAffiliateService(repo, nil, nil, nil)

	items, total, err := svc.AdminListInviteLogs(context.Background(), AffiliateInviteLogFilter{
		AccountID: 5,
		InviterID: 6,
		InviteeID: 7,
		Page:      1,
		PageSize:  20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.True(t, repo.lastListFilter.IncludeSensitive)
	require.Equal(t, int64(5), repo.lastListFilter.AccountID)
	require.Equal(t, int64(6), repo.lastListFilter.InviterID)
	require.Equal(t, int64(7), repo.lastListFilter.InviteeID)
	require.Equal(t, "secret-fingerprint", items[0].FingerprintHash)
	require.Equal(t, "secret-agent", items[0].UserAgent)
	require.Equal(t, "203.0.113.21", items[0].IPAddress)
}

func TestAffiliateInviteFailureMessageDoesNotExposeUnknownReason(t *testing.T) {
	t.Parallel()

	message := affiliateInviteFailureMessage("internal_detail=secret")

	require.Equal(t, "邀请处理未完成，请联系管理员查看原因。", message)
	require.NotContains(t, message, "internal_detail")
	require.NotContains(t, message, "secret")
}

// TestValidateExclusiveRate_BoundaryAndInvalid covers the validator used by
// admin-facing rate setters: nil is always valid (clear), in-range values
// are accepted, NaN/Inf and out-of-range values produce a typed BadRequest.
func TestValidateExclusiveRate_BoundaryAndInvalid(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateExclusiveRate(nil))

	for _, v := range []float64{0, 0.01, 50, 99.99, 100} {
		v := v
		require.NoError(t, validateExclusiveRate(&v), "value %v should be valid", v)
	}

	for _, v := range []float64{-0.01, 100.01, -100, 200} {
		v := v
		require.Error(t, validateExclusiveRate(&v), "value %v should be rejected", v)
	}

	nan := math.NaN()
	require.Error(t, validateExclusiveRate(&nan))
	posInf := math.Inf(1)
	require.Error(t, validateExclusiveRate(&posInf))
	negInf := math.Inf(-1)
	require.Error(t, validateExclusiveRate(&negInf))
}

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	require.Equal(t, "a***@g***.com", maskEmail("alice@gmail.com"))
	require.Equal(t, "x***@d***", maskEmail("x@domain"))
	require.Equal(t, "", maskEmail(""))
}

func TestIsValidAffiliateCodeFormat(t *testing.T) {
	t.Parallel()

	// 邀请码格式校验同时服务于：
	// 1) 系统自动生成的 12 位随机码（A-Z 去 I/O，2-9 去 0/1）
	// 2) 管理员设置的自定义专属码（如 "VIP2026"、"NEW_USER-1"）
	// 因此校验放宽到 [A-Z0-9_-]{4,32}（要求调用方先 ToUpper）。
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"valid canonical 12-char", "ABCDEFGHJKLM", true},
		{"valid all digits 2-9", "234567892345", true},
		{"valid mixed", "A2B3C4D5E6F7", true},
		{"valid admin custom short", "VIP1", true},
		{"valid admin custom with hyphen", "NEW-USER", true},
		{"valid admin custom with underscore", "VIP_2026", true},
		{"valid 32-char max", "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345", true},
		// Previously-excluded chars (I/O/0/1) are now allowed since admins may use them.
		{"letter I now allowed", "IBCDEFGHJKLM", true},
		{"letter O now allowed", "OBCDEFGHJKLM", true},
		{"digit 0 now allowed", "0BCDEFGHJKLM", true},
		{"digit 1 now allowed", "1BCDEFGHJKLM", true},
		{"too short (3 chars)", "ABC", false},
		{"too long (33 chars)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456", false},
		{"lowercase rejected (caller must ToUpper first)", "abcdefghjklm", false},
		{"empty", "", false},
		{"utf8 non-ascii", "ÄÄÄÄÄÄ", false}, // bytes out of charset
		{"ascii punctuation .", "ABCDEFGHJK.M", false},
		{"whitespace", "ABCDEFGHJK M", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isValidAffiliateCodeFormat(tc.in))
		})
	}
}

type affiliateSignupBonusSettingRepoStub struct {
	values map[string]string
}

func (s *affiliateSignupBonusSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *affiliateSignupBonusSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", errors.New("setting not found")
	}
	return value, nil
}

func (s *affiliateSignupBonusSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *affiliateSignupBonusSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *affiliateSignupBonusSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *affiliateSignupBonusSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *affiliateSignupBonusSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type affiliateSignupBonusRepoStub struct {
	profiles             map[int64]*AffiliateSummary
	byCode               map[string]*AffiliateSummary
	bindWithBonusCalls   []affiliateSignupBonusBindCall
	accrueCalls          []affiliateSignupBonusAccrueCall
	recordedLogs         []AffiliateInviteLogEntry
	listLogsResult       []AffiliateInviteLog
	listLogsTotal        int64
	lastListFilter       AffiliateInviteLogFilter
	inviteeRechargeTotal float64
}

type affiliateSignupBonusBindCall struct {
	userID        int64
	inviterID     int64
	bonusAmount   float64
	bonusTotalCap float64
}

type affiliateSignupBonusAccrueCall struct {
	inviterID     int64
	inviteeUserID int64
	amount        float64
	freezeHours   int
}

func newAffiliateSignupBonusRepoStub() *affiliateSignupBonusRepoStub {
	now := time.Now()
	return &affiliateSignupBonusRepoStub{
		profiles: map[int64]*AffiliateSummary{
			1: {UserID: 1, AffCode: "INVITE123", CreatedAt: now},
			2: {UserID: 2, AffCode: "INVITEE", CreatedAt: now},
		},
		byCode: map[string]*AffiliateSummary{
			"INVITE123": {UserID: 1, AffCode: "INVITE123", CreatedAt: now},
		},
	}
}

func (r *affiliateSignupBonusRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	profile, ok := r.profiles[userID]
	if !ok {
		return nil, ErrAffiliateProfileNotFound
	}
	cp := *profile
	return &cp, nil
}

func (r *affiliateSignupBonusRepoStub) GetAffiliateByCode(_ context.Context, code string) (*AffiliateSummary, error) {
	profile, ok := r.byCode[code]
	if !ok {
		return nil, ErrAffiliateProfileNotFound
	}
	cp := *profile
	return &cp, nil
}

func (r *affiliateSignupBonusRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	profile, ok := r.profiles[userID]
	if !ok {
		return false, ErrAffiliateProfileNotFound
	}
	profile.InviterID = &inviterID
	return true, nil
}

func (r *affiliateSignupBonusRepoStub) BindInviterWithSignupBonus(_ context.Context, req AffiliateSignupBonusRequest) (*AffiliateSignupBonusResult, error) {
	r.bindWithBonusCalls = append(r.bindWithBonusCalls, affiliateSignupBonusBindCall{
		userID:        req.UserID,
		inviterID:     req.InviterID,
		bonusAmount:   req.Amount,
		bonusTotalCap: req.InviterTotalCap,
	})
	profile, ok := r.profiles[req.UserID]
	if !ok {
		return nil, ErrAffiliateProfileNotFound
	}
	profile.InviterID = &req.InviterID
	return &AffiliateSignupBonusResult{Bound: true, AwardedAmount: req.Amount}, nil
}

func (r *affiliateSignupBonusRepoStub) RecordInviteLog(_ context.Context, entry AffiliateInviteLogEntry) error {
	r.recordedLogs = append(r.recordedLogs, entry)
	return nil
}

func (r *affiliateSignupBonusRepoStub) ListInviteLogs(_ context.Context, filter AffiliateInviteLogFilter) ([]AffiliateInviteLog, int64, error) {
	r.lastListFilter = filter
	return r.listLogsResult, r.listLogsTotal, nil
}

func (r *affiliateSignupBonusRepoStub) AccrueQuota(_ context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int, _ *int64) (bool, error) {
	r.accrueCalls = append(r.accrueCalls, affiliateSignupBonusAccrueCall{
		inviterID:     inviterID,
		inviteeUserID: inviteeUserID,
		amount:        amount,
		freezeHours:   freezeHours,
	})
	return true, nil
}

func (r *affiliateSignupBonusRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}

func (r *affiliateSignupBonusRepoStub) GetInviteeRechargeTotal(context.Context, int64) (float64, error) {
	return r.inviteeRechargeTotal, nil
}

func (r *affiliateSignupBonusRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}

func (r *affiliateSignupBonusRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	panic("unexpected TransferQuotaToBalance call")
}

func (r *affiliateSignupBonusRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) {
	return nil, nil
}

func (r *affiliateSignupBonusRepoStub) GetAffiliateUserOverview(context.Context, int64) (*AffiliateUserOverview, error) {
	panic("unexpected GetAffiliateUserOverview call")
}

func (r *affiliateSignupBonusRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}

func (r *affiliateSignupBonusRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}

func (r *affiliateSignupBonusRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	panic("unexpected SetUserRebateRate call")
}

func (r *affiliateSignupBonusRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (r *affiliateSignupBonusRepoStub) ListUsersWithCustomSettings(context.Context, AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func (r *affiliateSignupBonusRepoStub) ListAffiliateInviteRecords(context.Context, AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}

func (r *affiliateSignupBonusRepoStub) ListAffiliateRebateRecords(context.Context, AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}

func (r *affiliateSignupBonusRepoStub) ListAffiliateTransferRecords(context.Context, AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}
