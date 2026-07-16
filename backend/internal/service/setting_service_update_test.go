//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type settingUpdateRepoStub struct {
	updates map[string]string
}

func (s *settingUpdateRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingUpdateRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingUpdateRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingUpdateRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingUpdateRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for k, v := range settings {
		s.updates[k] = v
	}
	return nil
}

func (s *settingUpdateRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingUpdateRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type settingAntigravityUARepoStub struct {
	values map[string]string
}

func (s *settingAntigravityUARepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingAntigravityUARepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingAntigravityUARepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingAntigravityUARepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingAntigravityUARepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingAntigravityUARepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingAntigravityUARepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type defaultSubGroupReaderStub struct {
	byID  map[int64]*Group
	errBy map[int64]error
	calls []int64
}

func (s *defaultSubGroupReaderStub) GetByID(ctx context.Context, id int64) (*Group, error) {
	s.calls = append(s.calls, id)
	if err, ok := s.errBy[id]; ok {
		return nil, err
	}
	if g, ok := s.byID[id]; ok {
		return g, nil
	}
	return nil, ErrGroupNotFound
}

func TestSettingService_UpdateSettings_DefaultSubscriptions_ValidGroup(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	groupReader := &defaultSubGroupReaderStub{
		byID: map[int64]*Group{
			11: {ID: 11, SubscriptionType: SubscriptionTypeSubscription},
		},
	}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetDefaultSubscriptionGroupReader(groupReader)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		DefaultSubscriptions: []DefaultSubscriptionSetting{
			{GroupID: 11, ValidityDays: 30},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{11}, groupReader.calls)

	raw, ok := repo.updates[SettingKeyDefaultSubscriptions]
	require.True(t, ok)

	var got []DefaultSubscriptionSetting
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	require.Equal(t, []DefaultSubscriptionSetting{
		{GroupID: 11, ValidityDays: 30},
	}, got)
}

func TestSettingService_UpdateSettings_DefaultSubscriptions_RejectsNonSubscriptionGroup(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	groupReader := &defaultSubGroupReaderStub{
		byID: map[int64]*Group{
			12: {ID: 12, SubscriptionType: SubscriptionTypeStandard},
		},
	}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetDefaultSubscriptionGroupReader(groupReader)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		DefaultSubscriptions: []DefaultSubscriptionSetting{
			{GroupID: 12, ValidityDays: 7},
		},
	})
	require.Error(t, err)
	require.Equal(t, "DEFAULT_SUBSCRIPTION_GROUP_INVALID", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_DefaultSubscriptions_RejectsNotFoundGroup(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	groupReader := &defaultSubGroupReaderStub{
		errBy: map[int64]error{
			13: ErrGroupNotFound,
		},
	}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetDefaultSubscriptionGroupReader(groupReader)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		DefaultSubscriptions: []DefaultSubscriptionSetting{
			{GroupID: 13, ValidityDays: 7},
		},
	})
	require.Error(t, err)
	require.Equal(t, "DEFAULT_SUBSCRIPTION_GROUP_INVALID", infraerrors.Reason(err))
	require.Equal(t, "13", infraerrors.FromError(err).Metadata["group_id"])
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_DefaultSubscriptions_RejectsDuplicateGroup(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	groupReader := &defaultSubGroupReaderStub{
		byID: map[int64]*Group{
			11: {ID: 11, SubscriptionType: SubscriptionTypeSubscription},
		},
	}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetDefaultSubscriptionGroupReader(groupReader)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		DefaultSubscriptions: []DefaultSubscriptionSetting{
			{GroupID: 11, ValidityDays: 30},
			{GroupID: 11, ValidityDays: 60},
		},
	})
	require.Error(t, err)
	require.Equal(t, "DEFAULT_SUBSCRIPTION_GROUP_DUPLICATE", infraerrors.Reason(err))
	require.Equal(t, "11", infraerrors.FromError(err).Metadata["group_id"])
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_DefaultSubscriptions_RejectsDuplicateGroupWithoutGroupReader(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		DefaultSubscriptions: []DefaultSubscriptionSetting{
			{GroupID: 11, ValidityDays: 30},
			{GroupID: 11, ValidityDays: 60},
		},
	})
	require.Error(t, err)
	require.Equal(t, "DEFAULT_SUBSCRIPTION_GROUP_DUPLICATE", infraerrors.Reason(err))
	require.Equal(t, "11", infraerrors.FromError(err).Metadata["group_id"])
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_RegistrationEmailSuffixWhitelist_Normalized(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		RegistrationEmailSuffixWhitelist: []string{"example.com", "@EXAMPLE.com", " @foo.bar "},
	})
	require.NoError(t, err)
	require.Equal(t, `["@example.com","@foo.bar"]`, repo.updates[SettingKeyRegistrationEmailSuffixWhitelist])
}

func TestSettingService_UpdateSettings_RegistrationEmailSuffixWhitelist_Invalid(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		RegistrationEmailSuffixWhitelist: []string{"@invalid_domain"},
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_REGISTRATION_EMAIL_SUFFIX_WHITELIST", infraerrors.Reason(err))
}

func TestParseDefaultSubscriptions_NormalizesValues(t *testing.T) {
	got := parseDefaultSubscriptions(`[{"group_id":11,"validity_days":30},{"group_id":11,"validity_days":60},{"group_id":0,"validity_days":10},{"group_id":12,"validity_days":99999}]`)
	require.Equal(t, []DefaultSubscriptionSetting{
		{GroupID: 11, ValidityDays: 30},
		{GroupID: 11, ValidityDays: 60},
		{GroupID: 12, ValidityDays: MaxValidityDays},
	}, got)
}

func TestSettingService_UpdateSettings_TablePreferences(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		TableDefaultPageSize: 50,
		TablePageSizeOptions: []int{20, 50, 100},
	})
	require.NoError(t, err)
	require.Equal(t, "50", repo.updates[SettingKeyTableDefaultPageSize])
	require.Equal(t, "[20,50,100]", repo.updates[SettingKeyTablePageSizeOptions])

	err = svc.UpdateSettings(context.Background(), &SystemSettings{
		TableDefaultPageSize: 1000,
		TablePageSizeOptions: []int{20, 100},
	})
	require.NoError(t, err)
	require.Equal(t, "1000", repo.updates[SettingKeyTableDefaultPageSize])
	require.Equal(t, "[20,100]", repo.updates[SettingKeyTablePageSizeOptions])
}

func TestSettingService_UpdateSettings_AffiliateRebateTiersNormalized(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	tooHigh := 150.0
	l2Rate := 25.0

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AffiliateRebateTiers: []AffiliateRebateTier{
			{Level: "L2", MinInvitees: 20, MinRecharge: 5000, RebateRatePercent: &l2Rate},
			{Level: "L1", MinInvitees: -5, MinRecharge: -10, RebateRatePercent: &tooHigh},
		},
	})

	require.NoError(t, err)
	require.JSONEq(t, `[
		{"level":"L1","min_invitees":0,"min_recharge":0,"rebate_rate_percent":100},
		{"level":"L2","min_invitees":20,"min_recharge":5000,"rebate_rate_percent":25},
		{"level":"L3","min_invitees":0,"min_recharge":0,"rebate_rate_percent":null},
		{"level":"L4","min_invitees":0,"min_recharge":0,"rebate_rate_percent":null}
	]`, repo.updates[SettingKeyAffiliateRebateTiers])
}

func TestSettingService_UpdateSettings_SubscriptionQuotaResetSettings(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SubscriptionQuotaResetUTCOffsetMinutes: 480,
		SubscriptionQuotaResetHour:             4,
	})

	require.NoError(t, err)
	require.Equal(t, "480", repo.updates[SettingKeySubscriptionQuotaResetUTCOffsetMinutes])
	require.Equal(t, "4", repo.updates[SettingKeySubscriptionQuotaResetHour])
}

func TestSettingService_UpdateSettings_SubscriptionNotifyEmailEnabled(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SubscriptionNotifyEmailEnabled: true,
	})

	require.NoError(t, err)
	require.Equal(t, "true", repo.updates[SettingKeySubscriptionNotifyEmailEnabled])
}

func TestSettingService_UpdateSettings_SubscriptionMultiplePurchasesEnabled(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SubscriptionMultiplePurchasesEnabled: true,
	})

	require.NoError(t, err)
	require.Equal(t, "true", repo.updates[SettingKeySubscriptionMultiplePurchasesEnabled])
}

func TestSettingService_UpdateSettings_RejectsInvalidSubscriptionQuotaResetSettings(t *testing.T) {
	svc := NewSettingService(&settingUpdateRepoStub{}, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{SubscriptionQuotaResetUTCOffsetMinutes: 481})
	require.Error(t, err)
	require.Equal(t, "INVALID_SUBSCRIPTION_QUOTA_RESET_UTC_OFFSET", infraerrors.Reason(err))

	err = svc.UpdateSettings(context.Background(), &SystemSettings{SubscriptionQuotaResetHour: 24})
	require.Error(t, err)
	require.Equal(t, "INVALID_SUBSCRIPTION_QUOTA_RESET_HOUR", infraerrors.Reason(err))
}

func TestSettingService_ParseSettings_SubscriptionQuotaResetDefaultsAndValues(t *testing.T) {
	svc := NewSettingService(&settingUpdateRepoStub{}, &config.Config{})

	defaults := svc.parseSettings(map[string]string{})
	require.Equal(t, 0, defaults.SubscriptionQuotaResetUTCOffsetMinutes)
	require.Equal(t, 0, defaults.SubscriptionQuotaResetHour)

	parsed := svc.parseSettings(map[string]string{
		SettingKeySubscriptionQuotaResetUTCOffsetMinutes: "480",
		SettingKeySubscriptionQuotaResetHour:             "4",
	})
	require.Equal(t, 480, parsed.SubscriptionQuotaResetUTCOffsetMinutes)
	require.Equal(t, 4, parsed.SubscriptionQuotaResetHour)
}

func TestSettingService_ParseSettings_SubscriptionNotifyEmailDefaultsAndValues(t *testing.T) {
	svc := NewSettingService(&settingUpdateRepoStub{}, &config.Config{})

	defaults := svc.parseSettings(map[string]string{})
	require.False(t, defaults.SubscriptionNotifyEmailEnabled)
	require.False(t, defaults.SubscriptionMultiplePurchasesEnabled)

	parsed := svc.parseSettings(map[string]string{
		SettingKeySubscriptionNotifyEmailEnabled:       "true",
		SettingKeySubscriptionMultiplePurchasesEnabled: "true",
	})
	require.True(t, parsed.SubscriptionNotifyEmailEnabled)
	require.True(t, parsed.SubscriptionMultiplePurchasesEnabled)
}

func TestSettingService_GetSubscriptionMultiplePurchasesEnabled(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeySubscriptionMultiplePurchasesEnabled: "true",
	}}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.GetSubscriptionMultiplePurchasesEnabled(context.Background()))
}

func TestSettingService_GetSubscriptionQuotaResetConfig(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeySubscriptionQuotaResetUTCOffsetMinutes: "480",
		SettingKeySubscriptionQuotaResetHour:             "4",
	}}
	svc := NewSettingService(repo, &config.Config{})

	got := svc.GetSubscriptionQuotaResetConfig(context.Background())

	require.Equal(t, SubscriptionQuotaResetConfig{UTCOffsetMinutes: 480, ResetHour: 4}, got)
}

func TestSettingService_UpdateSettings_FrontendLocales(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		FrontendLocales: []string{"zh-Hant", "zh-CN", "en", "zh-Hant"},
	})
	require.NoError(t, err)
	require.Equal(t, `["zh-Hant","zh","en"]`, repo.updates[SettingKeyFrontendLocales])
}

func TestSettingService_UpdateSettings_SupportChatSettings(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SupportChatEnabled:             true,
		SupportChatGatewayURL:          " https://support-gateway.example.com/ ",
		SupportChatTitle:               " LumioAPI Helper ",
		SupportChatWelcomeMessage:      " Ask from the LumioAPI docs. ",
		SupportChatOfficialContactText: " Contact human support ",
		SupportChatOfficialContactURL:  " https://support.example.com/group/ ",
	})
	require.NoError(t, err)
	require.Equal(t, "true", repo.updates[SettingKeySupportChatEnabled])
	require.Equal(t, "https://support-gateway.example.com", repo.updates[SettingKeySupportChatGatewayURL])
	require.Equal(t, "LumioAPI Helper", repo.updates[SettingKeySupportChatTitle])
	require.Equal(t, "Ask from the LumioAPI docs.", repo.updates[SettingKeySupportChatWelcomeMessage])
	require.Equal(t, "Contact human support", repo.updates[SettingKeySupportChatOfficialContactText])
	require.Equal(t, "https://support.example.com/group", repo.updates[SettingKeySupportChatOfficialContactURL])
}

func TestSettingService_UpdateSettings_SupportChatEnabledRequiresHTTPGatewayURL(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SupportChatEnabled:    true,
		SupportChatGatewayURL: "javascript:alert(1)",
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_SUPPORT_CHAT_GATEWAY_URL", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_RejectsInvalidSupportChatOfficialContactURL(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SupportChatOfficialContactURL: "javascript:alert(1)",
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_SUPPORT_CHAT_OFFICIAL_CONTACT_URL", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_RejectsInvalidFrontendLocale(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		FrontendLocales: []string{"en", "fr"},
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_FRONTEND_LOCALE", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_PaymentVisibleMethodsAndAdvancedScheduler(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		PaymentVisibleMethodAlipaySource:  "alipay",
		PaymentVisibleMethodWxpaySource:   "easypay",
		PaymentVisibleMethodAlipayEnabled: true,
		PaymentVisibleMethodWxpayEnabled:  false,
		OpenAIAdvancedSchedulerEnabled:    true,
	})
	require.NoError(t, err)
	require.Equal(t, VisibleMethodSourceOfficialAlipay, repo.updates[SettingPaymentVisibleMethodAlipaySource])
	require.Equal(t, VisibleMethodSourceEasyPayWechat, repo.updates[SettingPaymentVisibleMethodWxpaySource])
	require.Equal(t, "true", repo.updates[SettingPaymentVisibleMethodAlipayEnabled])
	require.Equal(t, "false", repo.updates[SettingPaymentVisibleMethodWxpayEnabled])
	require.Equal(t, "true", repo.updates[openAIAdvancedSchedulerSettingKey])
}

func TestSettingService_UpdateSettings_AntigravityUserAgentVersion(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AntigravityUserAgentVersion: "1.23.2",
	})
	require.NoError(t, err)
	require.Equal(t, "1.23.2", repo.updates[SettingKeyAntigravityUserAgentVersion])
}

func TestSettingService_GetAntigravityUserAgentVersion_Precedence(t *testing.T) {
	t.Run("后台设置优先", func(t *testing.T) {
		svc := NewSettingService(&settingAntigravityUARepoStub{values: map[string]string{
			SettingKeyAntigravityUserAgentVersion: "1.24.0",
		}}, &config.Config{})

		require.Equal(t, "1.24.0", svc.GetAntigravityUserAgentVersion(context.Background()))
	})

	t.Run("空值回退配置默认值", func(t *testing.T) {
		svc := NewSettingService(&settingAntigravityUARepoStub{values: map[string]string{
			SettingKeyAntigravityUserAgentVersion: "",
		}}, &config.Config{})

		require.Equal(t, antigravity.GetDefaultUserAgentVersion(), svc.GetAntigravityUserAgentVersion(context.Background()))
	})

	t.Run("缺失回退配置默认值", func(t *testing.T) {
		svc := NewSettingService(&settingAntigravityUARepoStub{values: map[string]string{}}, &config.Config{})

		require.Equal(t, antigravity.GetDefaultUserAgentVersion(), svc.GetAntigravityUserAgentVersion(context.Background()))
	})
}

func TestSettingService_UpdateSettings_RejectsInvalidPaymentVisibleMethodSource(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		PaymentVisibleMethodAlipaySource: "not-a-provider",
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_PAYMENT_VISIBLE_METHOD_SOURCE", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}
