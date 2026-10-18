//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
	err    error
}

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","*.EDU.CN","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_ExposesTablePreferences(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyTableDefaultPageSize: "50",
			SettingKeyTablePageSizeOptions: "[20,50,100]",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 50, settings.TableDefaultPageSize)
	require.Equal(t, []int{20, 50, 100}, settings.TablePageSizeOptions)
}

func TestSettingService_GetPublicSettings_ExposesFrontendLocales(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyFrontendLocales: `["zh-Hant","en"]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"zh-Hant", "en"}, settings.FrontendLocales)
}

func TestSettingService_GetPublicSettings_ExposesSupportChatSettings(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeySupportChatEnabled:             "true",
			SettingKeySupportChatGatewayURL:          "https://support-gateway.example.com/",
			SettingKeySupportChatTitle:               "LumioAPI Helper",
			SettingKeySupportChatWelcomeMessage:      "Ask from the LumioAPI docs.",
			SettingKeySupportChatOfficialContactText: "Contact human support",
			SettingKeySupportChatOfficialContactURL:  "https://support.example.com/group/",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.SupportChatEnabled)
	require.Equal(t, "https://support-gateway.example.com", settings.SupportChatGatewayURL)
	require.Equal(t, "LumioAPI Helper", settings.SupportChatTitle)
	require.Equal(t, "Ask from the LumioAPI docs.", settings.SupportChatWelcomeMessage)
	require.Equal(t, "Contact human support", settings.SupportChatOfficialContactText)
	require.Equal(t, "https://support.example.com/group", settings.SupportChatOfficialContactURL)
}

func TestSettingService_ChannelMonitorShowQuotaFailsClosed(t *testing.T) {
	// 缺省（迁移插入 'false' / 老库无行）一律不展示。
	missingRuntime := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{}).GetChannelMonitorRuntime(context.Background())
	require.False(t, missingRuntime.ShowQuota)
	missingPublic, err := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{}).
		GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, missingPublic.ChannelMonitorShowQuota)

	// 仅字面 "true" 视为开启；其余值（含异常值）fail-closed。
	runtime := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyChannelMonitorShowQuota: "true",
	}}, &config.Config{}).GetChannelMonitorRuntime(context.Background())
	require.True(t, runtime.ShowQuota)

	for _, value := range []string{"false", "TRUE", "1", "yes", "on", "garbage"} {
		rt := NewSettingService(&settingPublicRepoStub{values: map[string]string{
			SettingKeyChannelMonitorShowQuota: value,
		}}, &config.Config{}).GetChannelMonitorRuntime(context.Background())
		require.False(t, rt.ShowQuota, "value=%q", value)
	}
}

func TestSettingService_GetPublicSettings_DefaultsFrontendLocales(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"en", "zh", "zh-Hant"}, settings.FrontendLocales)
}

func TestSettingService_GetPublicSettings_ExposesForceEmailOnThirdPartySignup(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyForceEmailOnThirdPartySignup: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.ForceEmailOnThirdPartySignup)
}

func TestSettingService_GetPublicSettings_ExposesUserSubscriptionsVisible(t *testing.T) {
	t.Run("defaults visible", func(t *testing.T) {
		repo := &settingPublicRepoStub{values: map[string]string{}}
		svc := NewSettingService(repo, &config.Config{})

		settings, err := svc.GetPublicSettings(context.Background())
		require.NoError(t, err)
		require.True(t, settings.UserSubscriptionsVisible)
	})

	t.Run("can be disabled", func(t *testing.T) {
		repo := &settingPublicRepoStub{
			values: map[string]string{
				SettingKeyUserSubscriptionsVisible: "false",
			},
		}
		svc := NewSettingService(repo, &config.Config{})

		settings, err := svc.GetPublicSettings(context.Background())
		require.NoError(t, err)
		require.False(t, settings.UserSubscriptionsVisible)
	})
}

func TestSettingService_GetPublicSettings_FiltersDisabledSitePages(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeySitePages: `[
				{"key":"docs","title":"Docs","slug":"doc/docs","content":"published","enabled":true},
				{"key":"draft","title":"Draft","slug":"doc/draft","content":"hidden draft","enabled":false},
				{"key":"legacy","title":"Legacy","slug":"doc/legacy","content":"legacy visible"}
			]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.NotContains(t, settings.SitePages, "hidden draft")
	require.Contains(t, settings.SitePages, "published")
	require.Contains(t, settings.SitePages, "legacy visible")

	var pages []struct {
		Key     string `json:"key"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(settings.SitePages), &pages))
	require.Len(t, pages, 2)
	require.Equal(t, []string{"docs", "legacy"}, []string{pages[0].Key, pages[1].Key})

	injection, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	rawInjection, err := json.Marshal(injection)
	require.NoError(t, err)
	require.NotContains(t, string(rawInjection), "hidden draft")
	require.Contains(t, string(rawInjection), "published")
}

func TestSettingService_GetFrameSrcOrigins_IncludesPublicLinkPages(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyHomeContent: "https://home.example.com/page",
			SettingKeySitePages: `[
				{"key":"docs","title":"Docs","slug":"doc/docs","mode":"link","content":"https://blog.lumio.games/docs/doc/api","enabled":true},
				{"key":"terms","title":"Terms","slug":"doc/terms","mode":"markdown","content":"https://not-embedded.example.com","enabled":true},
				{"key":"draft","title":"Draft","slug":"doc/draft","mode":"link","content":"https://hidden.example.com/page","enabled":false}
			]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	origins, err := svc.GetFrameSrcOrigins(context.Background())
	require.NoError(t, err)
	require.Contains(t, origins, "https://home.example.com")
	require.Contains(t, origins, "https://blog.lumio.games")
	require.NotContains(t, origins, "https://not-embedded.example.com")
	require.NotContains(t, origins, "https://hidden.example.com")
}

func TestSettingService_GetPublicSettings_ExposesWeChatOAuthModeCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectAppID:               "wx-mp-app",
			SettingKeyWeChatConnectAppSecret:           "wx-mp-secret",
			SettingKeyWeChatConnectMode:                "mp",
			SettingKeyWeChatConnectScopes:              "snsapi_base",
			SettingKeyWeChatConnectOpenEnabled:         "true",
			SettingKeyWeChatConnectMPEnabled:           "true",
			SettingKeyWeChatConnectRedirectURL:         "https://api.example.com/api/v1/auth/oauth/wechat/callback",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.True(t, settings.WeChatOAuthMPEnabled)
}

func TestSettingService_GetPublicSettings_DoesNotExposeMobileOnlyWeChatAsWebOAuthAvailable(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectMobileEnabled:       "true",
			SettingKeyWeChatConnectMode:                "mobile",
			SettingKeyWeChatConnectMobileAppID:         "wx-mobile-app",
			SettingKeyWeChatConnectMobileAppSecret:     "wx-mobile-secret",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.WeChatOAuthEnabled)
	require.False(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.True(t, settings.WeChatOAuthMobileEnabled)
}

func TestSettingService_GetPublicSettings_FallsBackToConfigForWeChatOAuthCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			OpenEnabled:         true,
			OpenAppID:           "wx-open-config",
			OpenAppSecret:       "wx-open-secret",
			FrontendRedirectURL: "/auth/wechat/config-callback",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.False(t, settings.WeChatOAuthMobileEnabled)
}

// subscription_enabled gates the user-facing "My Subscriptions" sidebar entry and
// is opt-out: only an explicit "false" hides it.
func TestSettingService_GetPublicSettings_SubscriptionEnabledOnlyExplicitFalseDisables(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   bool
	}{
		{name: "missing key defaults to enabled", values: map[string]string{}, want: true},
		{name: "empty value stays enabled", values: map[string]string{SettingKeySubscriptionEnabled: ""}, want: true},
		{name: "explicit true", values: map[string]string{SettingKeySubscriptionEnabled: "true"}, want: true},
		{name: "explicit false disables", values: map[string]string{SettingKeySubscriptionEnabled: "false"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewSettingService(&settingPublicRepoStub{values: tc.values}, &config.Config{})

			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.want, settings.SubscriptionEnabled)
		})
	}
}

// The SSR injection payload must mirror the switch so the opt-out flag does not
// flicker off on first render before the async public-settings fetch resolves.
func TestSettingService_GetPublicSettingsForInjection_MirrorsSubscriptionEnabled(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		repo := &settingPublicRepoStub{values: map[string]string{SettingKeySubscriptionEnabled: value}}
		svc := NewSettingService(repo, &config.Config{})

		raw, err := svc.GetPublicSettingsForInjection(context.Background())
		require.NoError(t, err)
		payload, ok := raw.(*PublicSettingsInjectionPayload)
		require.True(t, ok)
		require.Equal(t, value == "true", payload.SubscriptionEnabled, "value=%q", value)
	}
}

// payment_balance_disabled is exposed publicly so the user shell can derive the site
// billing mode (recharge & subscription / recharge only / subscription only) before any
// authenticated checkout call. Strict true, mirroring the payment-config parser.
func TestSettingService_GetPublicSettings_PaymentBalanceDisabledStrictTrue(t *testing.T) {
	cases := []struct {
		name  string
		value map[string]string
		want  bool
	}{
		{name: "missing key stays enabled", value: map[string]string{}, want: false},
		{name: "explicit false", value: map[string]string{SettingBalancePayDisabled: "false"}, want: false},
		{name: "explicit true disables balance recharge", value: map[string]string{SettingBalancePayDisabled: "true"}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewSettingService(&settingPublicRepoStub{values: tc.value}, &config.Config{})

			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.want, settings.PaymentBalanceDisabled)

			raw, err := svc.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)
			payload, ok := raw.(*PublicSettingsInjectionPayload)
			require.True(t, ok)
			require.Equal(t, tc.want, payload.PaymentBalanceDisabled)
		})
	}
}
